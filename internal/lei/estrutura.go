package lei

import (
	"regexp"
	"strings"
	"unicode"
)

// Fase 2: linhas → árvore de dispositivos.

// sufixo casa a letra de um dispositivo incluído depois ("-A" em "§ 1º-A").
// A letra precisa terminar ali (espaço, separador ou fim de linha): em
// "§ 3º-Aplica-se", o "-" é o separador e o "A" já é o texto. Uma letra
// solta continua ambígua ("§ 1º-A Os débitos" é sufixo, como na CF).
const sufixo = `(?:(?P<sufixo>-[A-Z])(?:[\s.\-–]|$))?`

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
	{Artigo, regexp.MustCompile(`^Art\.?\s+(?P<rotulo>\d{1,3}(?:\.\d{3})*)[º°o]?` + sufixo + `\s*(?:[.\-–]\s*)?(?P<texto>.*)$`), ""},
	{Paragrafo, regexp.MustCompile(`^§\s*(?P<rotulo>\d+)[º°o]?` + sufixo + `\s*(?:[.\-–]\s*)?(?P<texto>.*)$`), ""},
	{Paragrafo, regexp.MustCompile(`^Parágrafo\s+(?P<rotulo>único)\s*(?:[.\-–]\s*)?(?P<texto>.*)$`), "unico"},
	// O separador obrigatório é o que barra "Incumbe…" (inciso) e "A soberania…" (alínea).
	{Inciso, regexp.MustCompile(`^(?P<rotulo>[IVXLCDM]+)(?P<sufixo>-[A-Z])?\s*[–-]\s*(?P<texto>.*)$`), ""},
	// Sem separador, só quando o texto começa com minúscula ("XI nos
	// tribunais…", 49 vezes na CF) ou com anotação, porque o separador
	// estava riscado no original ("III (Revogado pela…)").
	{Inciso, regexp.MustCompile(`^(?P<rotulo>[IVXLCDM]+)(?P<sufixo>-[A-Z])?\s+(?P<texto>[\p{Ll}(].*)$`), ""},
	{Alinea, regexp.MustCompile(`^(?P<rotulo>[a-z])\)\s*(?P<texto>.*)$`), ""},
	// Agrupadores: o rótulo encerra a linha, ou vem seguido de "–" e o nome.
	// Sem isso, "Título II desta Lei" viraria agrupador.
	{Parte, regexp.MustCompile(`^PARTE\s+(?P<rotulo>GERAL)(?:\s*[–-]\s*(?P<texto>.*))?$`), "geral"},
	{Parte, regexp.MustCompile(`^PARTE\s+(?P<rotulo>ESPECIAL)(?:\s*[–-]\s*(?P<texto>.*))?$`), "especial"},
	{Livro, regexp.MustCompile(`^LIVRO\s+(?P<rotulo>[IVXLCDM]+)(?:\s*[–-]\s*(?P<texto>.*))?$`), ""},
	{Titulo, regexp.MustCompile(`^T[ÍI]TULO\s+(?P<rotulo>[IVXLCDM]+)(?:\s*[–-]\s*(?P<texto>.*))?$`), ""},
	{Capitulo, regexp.MustCompile(`^CAP[ÍI]TULO\s+(?P<rotulo>[IVXLCDM]+)(?:\s*[–-]\s*(?P<texto>.*))?$`), ""},
	{Secao, regexp.MustCompile(`^(?:Seção|SEÇÃO)\s+(?P<rotulo>[IVXLCDM]+)(?:\s*[–-]\s*(?P<texto>.*))?$`), ""},
	{Subsecao, regexp.MustCompile(`^(?:Subseção|SUBSEÇÃO)\s+(?P<rotulo>[IVXLCDM]+)(?:\s*[–-]\s*(?P<texto>.*))?$`), ""},
}

// separarRotulo verifica se a linha inicia com um dispositivo e em caso positivo retorna o rotulo separado do texto.
func separarRotulo(linha string) (Dispositivo, bool) {
	for _, p := range padroes {
		if m := p.re.FindStringSubmatch(linha); m != nil {
			rotulo := p.fixo
			if rotulo == "" {
				rotulo = strings.ReplaceAll(m[p.re.SubexpIndex("rotulo")], ".", "")
			}
			if i := p.re.SubexpIndex("sufixo"); i >= 0 {
				rotulo += m[i]
			}
			return Dispositivo{Tipo: p.tipo, Rotulo: rotulo, Texto: m[p.re.SubexpIndex("texto")]}, true
		}
	}

	return Dispositivo{}, false
}

// cabecalhoADCT abre o Ato das Disposições Constitucionais Transitórias,
// que recomeça a numeração dos artigos e fica fora da árvore da CF.
const cabecalhoADCT = "ATO DAS DISPOSIÇÕES CONSTITUCIONAIS TRANSITÓRIAS"

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
		return d.Rotulo == "I" || d.Rotulo == "geral"
	}
	return false
}

// reAnotacao casa o conteúdo de um parêntese que é nota do Planalto, não
// texto da norma: "Redação dada…", "Incluído…", "Vide…", "Revogado…",
// "VETADO" etc.
var reAnotacao = regexp.MustCompile(`(?i)^(?:parágrafo\s+(?:único\s+)?)?(?:redação|incluíd[oa]|acrescid[oa]|revogad[oa]|vetad[oa]|vide|vigência|regulamento|produção de efeito|renumerad[oa]|numerad[oa]|transformad[oa])`)

// reRevogacao e reVeto casam a anotação que revoga ou veta o dispositivo.
var (
	reRevogacao = regexp.MustCompile(`(?i)^\(\s*revogad[oa]`)
	reVeto      = regexp.MustCompile(`(?i)^\(\s*vetad[oa]`)
)

// sobraVazia é o que pode restar do texto de um dispositivo revogado ou
// vetado depois de tirar as anotações: "Art. 359-U. (VETADO).”" deixa ".”".
const sobraVazia = " .,;:–-\"“”"

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

// montarArvore monta a árvore a partir das linhas já limpas da Fase 1.
// Linhas antes do primeiro rótulo são ignoradas, e a leitura para no
// cabeçalho do ADCT. A data de assinatura também encerra o texto, a menos
// que depois dela a numeração recomece (veja recomeca): aí o que foi lido
// era o decreto de aprovação, que é descartado em favor do código anexo.
//
// Linha sem rótulo continua o texto do nó aberto, com duas exceções para
// linhas em negrito: a primeira linha depois de um agrupador é sempre o seu
// nome (na CF o nome é negrito), e as seguintes só continuam o nome se
// estiverem em maiúsculas. Qualquer outra linha em negrito é rubrica do
// próximo dispositivo; se vier um agrupador antes, ela é descartada.
func montarArvore(linhas []Linha) []*Dispositivo {
	var raiz = []*Dispositivo{}
	var pilha = []*Dispositivo{}
	var rubrica string // à espera do próximo dispositivo
	var assinada bool  // passou do fecho; só um novo começo reabre a leitura

	for _, l := range linhas {
		linha := l.Texto
		if len(raiz) > 0 && strings.EqualFold(linha, cabecalhoADCT) {
			break
		}
		if len(raiz) > 0 && reDataAssinatura.MatchString(linha) {
			assinada = true
			continue
		}

		d, valido := separarRotulo(linha)
		if assinada {
			if !valido || !recomeca(d) {
				continue
			}
			raiz, pilha, rubrica, assinada = []*Dispositivo{}, nil, "", false
		}
		if !valido && len(raiz) == 0 {
			continue
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
	}

	limpar(raiz)
	return raiz
}
