package lei

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Fase 2: linhas → árvore de dispositivos.

// sufixo casa a letra de um dispositivo incluído depois ("-A" em "§ 1º-A").
// A letra precisa terminar ali (espaço, separador ou fim de linha): em
// "§ 3º-Aplica-se", o "-" é o separador e o "A" já é o texto. Uma letra
// solta continua ambígua ("§ 1º-A Os débitos" é sufixo, como na CF).
const sufixo = `(?:(?P<sufixo>-[A-Z])(?:[\s.\-–]|$))?`

// numeroArtigo casa o número do artigo, com ou sem ponto de milhar:
// "1.337" e "1337" (os dois aparecem no CC).
const numeroArtigo = `(?P<rotulo>\d{1,3}(?:\.\d{3})+|\d+)`

// rotuloAgrupador casa o rótulo de um agrupador: romano, "Único"/"Única",
// ordinal por extenso ("LIVRO PRIMEIRO", no CTN) ou "COMPLEMENTAR" (CC e
// CPC). normalizarAgrupador o deixa na forma canônica.
const rotuloAgrupador = `[IVXLCDM]+|(?i:únic[oa]|primeiro|segundo|terceiro|quarto|quinto|sexto|sétimo|oitavo|nono|décimo|complementar)`

// agrupador monta a regex de um agrupador: palavra (sem distinção de
// maiúsculas: "TÍTULO", "Título", "TITULO"), rótulo, sufixo de letra
// ("TÍTULO II-A", na CLT) e o nome opcional.
func agrupador(palavra, rotulo string) *regexp.Regexp {
	return regexp.MustCompile(`^(?i:` + palavra + `)\s+(?P<rotulo>` + rotulo + `)(?P<sufixo>-[A-Z])?` +
		`(?:\s*[–-]\s*(?P<texto>.*)|\s+(?P<nome>\p{Lu}.*))?$`)
}

// ordinais converte o ordinal por extenso do rótulo em romano.
var ordinais = map[string]string{
	"primeiro": "I", "segundo": "II", "terceiro": "III", "quarto": "IV", "quinto": "V",
	"sexto": "VI", "sétimo": "VII", "oitavo": "VIII", "nono": "IX", "décimo": "X",
}

// normalizarAgrupador deixa o rótulo do agrupador na forma canônica:
// romano como está, ordinal por extenso em romano, "Única" como "unico" e
// as palavras ("GERAL", "COMPLEMENTAR") em minúsculas.
func normalizarAgrupador(rotulo string) string {
	minusculo := strings.ToLower(strings.ReplaceAll(rotulo, " ", ""))
	switch {
	case ordinais[minusculo] != "":
		return ordinais[minusculo]
	case minusculo == "único" || minusculo == "única":
		return "unico"
	case strings.Trim(rotulo, "IVXLCDM") == "":
		return rotulo
	}
	return minusculo
}

// padroes lista, em ordem de prioridade, como reconhecer cada tipo.
// Toda regex tem os grupos "rotulo" e "texto", e pode ter "sufixo", que é
// colado ao rótulo ("1" + "-A"). Quando fixo não é vazio, ele substitui o
// grupo "rotulo".
// As linhas devem chegar normalizadas: \s só casa espaço ASCII, não &nbsp;.
var padroes = []struct {
	tipo Tipo
	re   *regexp.Regexp
	fixo string
}{
	// Letra sem hífen, colada ao número: "Art. 401A." (CLT). Vira "401-A".
	{Artigo, regexp.MustCompile(`^Art(?:\.[\s.]*|\s+)` + numeroArtigo + `(?P<sufixo>[A-Z])(?:[\s.\-–]+|$)(?P<texto>.*)$`), ""},
	// "Art.194" e "Art. . 189" (CLT) também valem.
	{Artigo, regexp.MustCompile(`^Art(?:\.[\s.]*|\s+)` + numeroArtigo + `[º°o]?` + sufixo + `\s*(?:[.\-–]\s*)?(?P<texto>.*)$`), ""},
	{Paragrafo, regexp.MustCompile(`^§\s*(?P<rotulo>\d+)[º°o]?` + sufixo + `\s*(?:[.\-–]\s*)?(?P<texto>.*)$`), ""},
	{Paragrafo, regexp.MustCompile(`^Parágrafo\s+(?P<rotulo>único)\s*(?:[.\-–]\s*)?(?P<texto>.*)$`), "unico"},
	// O separador obrigatório é o que barra "Incumbe…" (inciso) e "A soberania…" (alínea).
	{Inciso, regexp.MustCompile(`^(?P<rotulo>[IVXLCDM]+)(?P<sufixo>-[A-Z])?\s*[–-]\s*(?P<texto>.*)$`), ""},
	// Sem separador, só quando o texto começa com minúscula ("XI nos
	// tribunais…", 49 vezes na CF) ou com anotação, porque o separador
	// estava riscado no original ("III (Revogado pela…)").
	{Inciso, regexp.MustCompile(`^(?P<rotulo>[IVXLCDM]+)(?P<sufixo>-[A-Z])?\s+(?P<texto>[\p{Ll}(].*)$`), ""},
	{Alinea, regexp.MustCompile(`^(?P<rotulo>[a-z])\)\s*(?P<texto>.*)$`), ""},
	// Agrupadores: o rótulo encerra a linha ou vem seguido do nome, depois
	// de "–" ou de espaço ("CAPÍTULO I DA PERSONALIDADE", no CC). Sem
	// separador, o nome precisa começar com maiúscula: "Título II desta
	// Lei" não é agrupador. Veja agrupador. A Parte aceita as letras
	// espaçadas do CC ("P A R T E G E R A L").
	{Parte, agrupador(`p ?a ?r ?t ?e`, `(?i:g ?e ?r ?a ?l|e ?s ?p ?e ?c ?i ?a ?l)`), ""},
	{Livro, agrupador(`livro`, rotuloAgrupador), ""},
	{Titulo, agrupador(`t[íi]tulo`, rotuloAgrupador), ""},
	{Subtitulo, agrupador(`subt[íi]tulo`, rotuloAgrupador), ""},
	{Capitulo, agrupador(`cap[íi]tulo`, rotuloAgrupador), ""},
	{Secao, agrupador(`sec?[çc][ãa]o`, rotuloAgrupador), ""},
	{Subsecao, agrupador(`subsec?[çc][ãa]o`, rotuloAgrupador), ""},
}

// reSufixoSeparado casa o sufixo de letra separado do número por espaço,
// "§ 2o -A." (ECA). O ponto depois da letra é o que o distingue do
// separador seguido de texto ("§ 2º - A pessoa…").
var reSufixoSeparado = regexp.MustCompile(`^((?:Art\.?|§)\s*[\d.]+[º°o]?)\s+-([A-Z])\.`)

// separarRotulo verifica se a linha inicia com um dispositivo e em caso positivo retorna o rotulo separado do texto.
func separarRotulo(linha string) (Dispositivo, bool) {
	linha = reSufixoSeparado.ReplaceAllString(linha, "$1-$2.")
	for _, p := range padroes {
		if m := p.re.FindStringSubmatch(linha); m != nil {
			rotulo := p.fixo
			if rotulo == "" {
				rotulo = strings.ReplaceAll(m[p.re.SubexpIndex("rotulo")], ".", "")
			}
			if p.tipo.Agrupador() {
				rotulo = normalizarAgrupador(rotulo)
			}
			if i := p.re.SubexpIndex("sufixo"); i >= 0 && m[i] != "" {
				rotulo += "-" + strings.TrimPrefix(m[i], "-")
			}
			texto := m[p.re.SubexpIndex("texto")]
			if i := p.re.SubexpIndex("nome"); i >= 0 && m[i] != "" {
				texto = m[i]
			}
			return Dispositivo{Tipo: p.tipo, Rotulo: rotulo, Texto: texto}, true
		}
	}

	return Dispositivo{}, false
}

// cabecalhoADCT abre o Ato das Disposições Constitucionais Transitórias,
// que recomeça a numeração dos artigos e fica fora da árvore da CF.
const cabecalhoADCT = "ATO DAS DISPOSIÇÕES CONSTITUCIONAIS TRANSITÓRIAS"

// reAnexo casa o cabeçalho dos anexos ("ANEXO", "ANEXO I"), que vêm depois
// do último artigo: tabelas e quadros, não texto normativo. Na CLT ele vem
// sem data de assinatura antes.
var reAnexo = regexp.MustCompile(`^ANEXO(?:\s+[IVXLCDM\d]+)?$`)

// reDataAssinatura casa o fecho da lei ("Brasília, 5 de outubro de 1988.",
// "Rio de Janeiro, 7 de dezembro de 1940; 119º da Independência…"). Depois
// dele vêm assinaturas e avisos, que não são texto normativo.
var reDataAssinatura = regexp.MustCompile(`^\p{Lu}[\p{L} ]*, (?:em )?\d{1,2}[º°o]? de \p{Ll}+ de \d{4}\b`)

// recomeca informa se d abre uma numeração nova: "Art. 1º" ou o primeiro
// agrupador ("TÍTULO I", "PARTE GERAL"). Depois do fecho, é o sinal de que
// o que veio antes era só o decreto que aprova um código anexo, como na
// CLT (Decreto-Lei nº 5.452, de 1943).
func recomeca(d Dispositivo) bool {
	switch {
	case d.Tipo == Artigo:
		return d.Rotulo == "1"
	case d.Tipo.Agrupador():
		return d.Rotulo == "I" || d.Rotulo == "geral" || d.Rotulo == "unico"
	}
	return false
}

// reAnotacao casa o conteúdo de um parêntese que é nota do Planalto, não
// texto da norma: "Redação dada…", "Incluído…", "Vide…", "Revogado…",
// "VETADO" etc.
var reAnotacao = regexp.MustCompile(`(?i)^(?:parágrafo\s+(?:único\s+)?)?(?:redação|incluíd[oa]|acrescid[oa]|revogad[oa]|vetad[oa]|vide|vigência|regulamento|produção de efeito|renumerad[oa]|numerad[oa]|transformad[oa]|suprimid[oa])`)

// reRevogacao e reVeto casam a anotação que revoga ou veta o dispositivo.
var (
	reRevogacao = regexp.MustCompile(`(?i)^\(\s*revogad[oa]`)
	reVeto      = regexp.MustCompile(`(?i)^\(\s*vetad[oa]`)
)

// sobraVazia é o que pode restar do texto de um dispositivo revogado ou
// vetado depois de tirar as anotações: "Art. 359-U. (VETADO).”" deixa ".”".
const sobraVazia = " .,;:–-\"“”"

// reVigenciaEncerrada casa o selo "Vigência encerrada", sem parênteses, que
// o Planalto põe no texto restaurado quando uma MP caduca (99 vezes na CLT,
// pelas MPs 905 e 955).
var reVigenciaEncerrada = regexp.MustCompile(`\s*\bVigência encerrada\b\.?`)

// reEspacoAntesPontuacao casa o espaço que sobra entre o texto e a
// pontuação quando uma anotação é removida do meio ("o lazer (…);").
var reEspacoAntesPontuacao = regexp.MustCompile(`\s+([.,;:])`)

// separarAnotacoes tira de texto os parênteses que são anotações e os
// devolve à parte, na ordem em que aparecem e com os espaços normalizados.
// Parênteses que fazem parte da norma, como "(vinte)", ficam no texto.
func separarAnotacoes(texto string) (string, []string) {
	var anotacoes []string
	var resto strings.Builder

	for {
		inicio := strings.IndexByte(texto, '(')
		if inicio < 0 {
			break
		}
		fim := fechaParentese(texto, inicio)
		if fim < 0 {
			break
		}

		resto.WriteString(texto[:inicio])
		conteudo := strings.Join(strings.Fields(texto[inicio+1:fim]), " ")
		if reAnotacao.MatchString(conteudo) {
			anotacoes = append(anotacoes, "("+conteudo+")")
		} else {
			resto.WriteString(texto[inicio : fim+1])
		}
		texto = texto[fim+1:]
	}
	resto.WriteString(texto)

	limpo := strings.Join(strings.Fields(resto.String()), " ")
	if len(anotacoes) > 0 {
		limpo = reEspacoAntesPontuacao.ReplaceAllString(limpo, "$1")
	}
	return limpo, anotacoes
}

// fechaParentese devolve o índice do ")" que fecha o "(" em s[inicio],
// contando os aninhados, ou -1 se ele não fecha.
func fechaParentese(s string, inicio int) int {
	nivel := 0
	for i := inicio; i < len(s); i++ {
		switch s[i] {
		case '(':
			nivel++
		case ')':
			nivel--
			if nivel == 0 {
				return i
			}
		}
	}
	return -1
}

// limpar separa as anotações do texto de cada nó e marca como revogado
// (ou vetado) o que fica sem texto, só pontuação, e tem uma anotação de
// revogação (ou veto). "Art. 88 - Revogado o livramento…" tem texto, então
// não é revogação.
func limpar(ds []*Dispositivo) {
	for _, d := range ds {
		d.Texto, d.Anotacoes = separarAnotacoes(d.Texto)
		if reVigenciaEncerrada.MatchString(d.Texto) {
			d.Texto = strings.TrimSpace(reVigenciaEncerrada.ReplaceAllString(d.Texto, ""))
			d.Anotacoes = append(d.Anotacoes, "(Vigência encerrada)")
		}

		if strings.Trim(d.Texto, sobraVazia) == "" {
			for _, a := range d.Anotacoes {
				switch {
				case reRevogacao.MatchString(a):
					d.Revogado = true
				case reVeto.MatchString(a):
					d.Vetado = true
				}
			}
			if d.Revogado || d.Vetado {
				d.Texto = ""
			}
		}

		limpar(d.Filhos)
	}
}

// removerRepetidos tira o irmão idêntico (mesmo tipo, rótulo e texto) a
// um anterior: o Planalto às vezes repete um dispositivo sem riscar a cópia
// (parágrafo único do art. 153 da CLT). Os filhos da cópia vão para o
// original.
func removerRepetidos(ds []*Dispositivo) []*Dispositivo {
	saida := ds[:0]
	for _, d := range ds {
		i := slices.IndexFunc(saida, func(o *Dispositivo) bool {
			return o.Tipo == d.Tipo && o.Rotulo == d.Rotulo && o.Texto == d.Texto
		})
		if i >= 0 && d.Texto != "" {
			saida[i].Filhos = append(saida[i].Filhos, d.Filhos...)
			continue
		}
		saida = append(saida, d)
	}
	for _, d := range saida {
		d.Filhos = removerRepetidos(d.Filhos)
	}
	return saida
}

// semAnotacoes devolve o texto sem as anotações e sem espaços nas pontas.
func semAnotacoes(texto string) string {
	t, _ := separarAnotacoes(texto)
	return strings.TrimSpace(t)
}

// contarArtigos conta os artigos da árvore.
func contarArtigos(ds []*Dispositivo) int {
	n := 0
	for _, d := range ds {
		if d.Tipo == Artigo {
			n++
		}
		n += contarArtigos(d.Filhos)
	}
	return n
}

// Linha é uma linha de texto entregue pela Fase 1. Negrito indica que a
// linha começa em negrito (o primeiro trecho com texto): é o que distingue
// as rubricas do CP ("Homicídio qualificado") do texto normativo. Não é
// "a linha inteira" porque a rubrica costuma vir seguida de uma anotação
// sem negrito: "Lugar do crime (Redação dada pela Lei nº 7.209, de 1984)".
type Linha struct {
	Texto   string
	Negrito bool
}

// juntar emenda uma linha de continuação ao texto, separada por espaço.
func juntar(texto, linha string) string {
	if texto == "" {
		return linha
	}
	return texto + " " + linha
}

// maiusculas informa se s tem letras e todas são maiúsculas, como nos
// nomes de agrupador do CP ("DOS CRIMES CONTRA AS").
func maiusculas(s string) bool {
	return strings.ContainsFunc(s, unicode.IsLetter) && strings.ToUpper(s) == s
}

// rePena casa a linha da pena ("Pena - reclusão…"), o preceito secundário
// do tipo penal. Ela vale para o artigo ou parágrafo inteiro, não para o
// último inciso antes dela.
var rePena = regexp.MustCompile(`^Pena\s*[-–:]`)

// reParagrafoEmendado casa o "Parágrafo único." que o original deixou na
// mesma linha do caput, logo depois do ponto final (art. 625-A da CLT).
// Com maiúscula, depois de ponto e seguido de separador, é rótulo: no meio
// da frase a remissão vem em minúscula ("o parágrafo único deste artigo").
var reParagrafoEmendado = regexp.MustCompile(`[.;:]\s+(Parágrafo\s+único\s*[.\-–]\s)`)

// separarEmendas divide as linhas em que um rótulo ficou emendado ao
// texto anterior (veja reParagrafoEmendado).
func separarEmendas(linhas []Linha) []Linha {
	var saida []Linha
	for _, l := range linhas {
		for {
			m := reParagrafoEmendado.FindStringSubmatchIndex(l.Texto)
			if m == nil {
				break
			}
			saida = append(saida, Linha{Texto: l.Texto[:m[2]-1], Negrito: l.Negrito})
			l = Linha{Texto: l.Texto[m[2]:]}
		}
		saida = append(saida, l)
	}
	return saida
}

// reAlteracao casa a fórmula do artigo que altera outra lei ("passa a
// vigorar com as seguintes alterações:", "acrescido do seguinte art.
// 337-E:"). As linhas seguintes citam a outra lei: veja proximoArtigo.
var reAlteracao = regexp.MustCompile(`(?i)(?:passa(?:m)? a vigorar|fica(?:m)? acrescid|acrescid[oa]s? d[oa]s? seguintes?|seguintes? alteraç|com a seguinte redação)[^:]*:$`)

// proximoArtigo informa se novo pode ser o artigo seguinte a atual na
// mesma lei: o mesmo número com sufixo maior ("178" → "178-A") ou um
// número até 5 acima (os revogados riscados somem do texto). Um artigo
// citado de outra lei ("Art. 337-E" depois do art. 178 da Lei 14.133)
// raramente cai nessa faixa.
func proximoArtigo(atual, novo string) bool {
	na, sa, _ := strings.Cut(atual, "-")
	nn, sn, _ := strings.Cut(novo, "-")
	a, err1 := strconv.Atoi(na)
	n, err2 := strconv.Atoi(nn)
	if err1 != nil || err2 != nil {
		return false
	}
	return (n == a && sn > sa) || (n > a && n <= a+5)
}

// fimDaCitacao devolve quantas das linhas citadas são mesmo citação: os
// agrupadores no fim, sem dispositivo depois deles, são da própria lei
// (o capítulo que começa antes do próximo artigo).
func fimDaCitacao(linhas []Linha) int {
	fim := len(linhas)
	for i := len(linhas) - 1; i >= 0; i-- {
		d, valido := separarRotulo(linhas[i].Texto)
		if valido && !d.Tipo.Agrupador() {
			break
		}
		if valido {
			fim = i
		}
	}
	return fim
}

// montarArvore monta a árvore a partir das linhas já limpas da Fase 1.
// Linhas antes do primeiro rótulo são ignoradas, e a leitura para no
// cabeçalho do ADCT ou de um anexo. A data de assinatura também encerra o
// texto, a menos que depois dela a numeração recomece (veja recomeca). Aí
// há dois candidatos, e fica o que tiver mais artigos: o código anexo ao
// decreto que o aprova (CLT), e não o índice depois da assinatura (CC).
//
// O artigo que altera outra lei (veja reAlteracao) leva no texto as linhas
// citadas, até o próximo artigo da própria lei: os artigos e parágrafos
// citados não são dispositivos desta.
//
// Linha sem rótulo continua o texto do nó aberto, com duas exceções para
// linhas em negrito: a primeira linha depois de um agrupador é sempre o seu
// nome (na CF o nome é negrito), e as seguintes só continuam o nome se
// estiverem em maiúsculas. Qualquer outra linha em negrito é rubrica do
// próximo dispositivo; se vier um agrupador antes, ela é descartada.
func montarArvore(linhas []Linha) []*Dispositivo {
	var raiz = []*Dispositivo{}
	var pilha = []*Dispositivo{}
	var rubrica string          // à espera do próximo dispositivo
	var assinada bool           // passou do fecho; só um novo começo reabre a leitura
	var anterior []*Dispositivo // o texto lido antes do último recomeço
	var citando *Dispositivo    // artigo que altera outra lei
	var inicioCitacao int       // índice da primeira linha citada
	var citado *Dispositivo     // último artigo cuja citação já terminou

	linhas = separarEmendas(linhas)
	for i := 0; i < len(linhas); i++ {
		l := linhas[i]
		linha := l.Texto
		fecho := len(raiz) > 0 && (strings.EqualFold(linha, cabecalhoADCT) || reAnexo.MatchString(linha) ||
			reDataAssinatura.MatchString(linha))
		d, valido := separarRotulo(linha)

		if citando != nil {
			if !fecho && !(valido && d.Tipo == Artigo && proximoArtigo(citando.Rotulo, d.Rotulo)) {
				continue
			}
			citadas := linhas[inicioCitacao:i]
			fim := fimDaCitacao(citadas)
			for _, c := range citadas[:fim] {
				citando.Texto = juntar(citando.Texto, c.Texto)
			}
			citado, citando = citando, nil
			if fim < len(citadas) {
				i = inicioCitacao + fim - 1 // reprocessa os agrupadores da própria lei
				continue
			}
		}

		if len(raiz) > 0 && (strings.EqualFold(linha, cabecalhoADCT) || reAnexo.MatchString(linha)) {
			break
		}
		if len(raiz) > 0 && reDataAssinatura.MatchString(linha) {
			assinada = true
			continue
		}
		if assinada {
			if !valido || !recomeca(d) {
				continue
			}
			if contarArtigos(raiz) > contarArtigos(anterior) {
				anterior = raiz
			}
			raiz, pilha, rubrica, assinada = []*Dispositivo{}, nil, "", false
		}
		if !valido && len(raiz) == 0 {
			continue
		}

		if valido && d.Tipo.Agrupador() && len(pilha) > 0 {
			// Cabeçalho repetido no original ("Subseção IV" duas vezes na
			// Lei 8.112): o primeiro ainda está vazio.
			if topo := pilha[len(pilha)-1]; topo.Tipo == d.Tipo && topo.Rotulo == d.Rotulo && topo.Texto == "" && len(topo.Filhos) == 0 {
				continue
			}
		}

		if valido {
			if !d.Tipo.Agrupador() {
				d.Rubrica, _ = separarAnotacoes(rubrica)
			}
			rubrica = ""

			for len(pilha) > 0 && pilha[len(pilha)-1].Tipo >= d.Tipo {
				pilha = pilha[:len(pilha)-1]
			}

			if len(pilha) == 0 {
				raiz = append(raiz, &d)
			} else {
				topo := pilha[len(pilha)-1]

				topo.Filhos = append(topo.Filhos, &d)
			}

			pilha = append(pilha, &d)
		} else {
			topo := pilha[len(pilha)-1]
			nomeDeAgrupador := topo.Tipo.Agrupador() && (topo.Texto == "" || maiusculas(linha))
			switch {
			case l.Negrito && !nomeDeAgrupador:
				rubrica = juntar(rubrica, linha)
			case rePena.MatchString(linha):
				// A pena fecha o artigo ou parágrafo: o que estiver aberto
				// abaixo dele (incisos, alíneas) é fechado também.
				for len(pilha) > 1 && pilha[len(pilha)-1].Tipo > Paragrafo {
					pilha = pilha[:len(pilha)-1]
				}
				topo = pilha[len(pilha)-1]
				topo.Texto = juntar(topo.Texto, linha)
			default:
				topo.Texto = juntar(topo.Texto, linha)
			}
		}

		if topo := pilha[len(pilha)-1]; topo.Tipo == Artigo && topo != citado && reAlteracao.MatchString(semAnotacoes(topo.Texto)) {
			citando, inicioCitacao = topo, i+1
		}
	}
	if citando != nil {
		for _, c := range linhas[inicioCitacao:] {
			citando.Texto = juntar(citando.Texto, c.Texto)
		}
	}

	if contarArtigos(anterior) > contarArtigos(raiz) {
		raiz = anterior
	}
	limpar(raiz)
	return removerRepetidos(raiz)
}
