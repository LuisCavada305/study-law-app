package lei

import (
	"fmt"
	"strings"
	"testing"
)

func TestSepararRotulo(t *testing.T) {
	testes := []struct {
		linha  string
		tipo   Tipo
		rotulo string
		texto  string
		ok     bool
	}{
		// Artigo: separador ".", " -" ou nenhum; ordinal até o 9º, cardinal depois.
		{linha: "Art. 5º Todos são iguais perante a lei", tipo: Artigo, rotulo: "5", texto: "Todos são iguais perante a lei", ok: true},
		{linha: "Art. 10. É assegurada a participação", tipo: Artigo, rotulo: "10", texto: "É assegurada a participação", ok: true},
		{linha: "Art. 121. Matar alguem:", tipo: Artigo, rotulo: "121", texto: "Matar alguem:", ok: true},
		{linha: "Art. 81 - A suspensão será revogada", tipo: Artigo, rotulo: "81", texto: "A suspensão será revogada", ok: true},
		{linha: "Art 188.", tipo: Artigo, rotulo: "188", texto: "", ok: true},
		{linha: "Art. 121-A. Matar mulher", tipo: Artigo, rotulo: "121-A", texto: "Matar mulher", ok: true},
		{linha: "Art. 1.000. A obrigação", tipo: Artigo, rotulo: "1000", texto: "A obrigação", ok: true},
		// Variantes do ordinal no Planalto: º (ordinal), ° (grau) e a letra o.
		{linha: "Art. 5° Todos", tipo: Artigo, rotulo: "5", texto: "Todos", ok: true},
		{linha: "Art. 5o Todos", tipo: Artigo, rotulo: "5", texto: "Todos", ok: true},
		// Sufixo de letra depois do ordinal.
		{linha: "Art. 5º-A Todos", tipo: Artigo, rotulo: "5-A", texto: "Todos", ok: true},

		// Parágrafo.
		{linha: "§ 1º As normas definidoras", tipo: Paragrafo, rotulo: "1", texto: "As normas definidoras", ok: true},
		{linha: "§ 10. Aplica-se", tipo: Paragrafo, rotulo: "10", texto: "Aplica-se", ok: true},
		{linha: "Parágrafo único. Todo o poder emana do povo", tipo: Paragrafo, rotulo: "unico", texto: "Todo o poder emana do povo", ok: true},
		// Sufixo de letra: com ponto, sem separador, ordinal "o" e sem texto.
		{linha: "§ 5º-A. O requisito de idade", tipo: Paragrafo, rotulo: "5-A", texto: "O requisito de idade", ok: true},
		{linha: "§ 1º-A Os débitos de natureza alimentícia", tipo: Paragrafo, rotulo: "1-A", texto: "Os débitos de natureza alimentícia", ok: true},
		{linha: "§ 2o-B A pena", tipo: Paragrafo, rotulo: "2-B", texto: "A pena", ok: true},
		{linha: "§ 4º-B", tipo: Paragrafo, rotulo: "4-B", texto: "", ok: true},
		// Hífen colado a uma palavra não é sufixo: a letra é o começo do texto.
		{linha: "§ 3º-Aplica-se o disposto", tipo: Paragrafo, rotulo: "3", texto: "Aplica-se o disposto", ok: true},

		// Inciso: travessão (–) ou hífen (-).
		{linha: "LXVII – não haverá prisão civil por dívida", tipo: Inciso, rotulo: "LXVII", texto: "não haverá prisão civil por dívida", ok: true},
		{linha: "I - obrigatórios para os maiores de dezoito anos;", tipo: Inciso, rotulo: "I", texto: "obrigatórios para os maiores de dezoito anos;", ok: true},
		{linha: "II-A - texto incluído", tipo: Inciso, rotulo: "II-A", texto: "texto incluído", ok: true},
		{linha: "I -A soberania", tipo: Inciso, rotulo: "I", texto: "A soberania", ok: true},
		// Separador riscado no original (CF, art. 115, III): sobra o rótulo e a anotação.
		{linha: "III (Revogado pela Emenda Constitucional nº 24, de 1999)", tipo: Inciso, rotulo: "III", texto: "(Revogado pela Emenda Constitucional nº 24, de 1999)", ok: true},
		// Sem separador no original (CF, art. 93).
		{linha: "XI nos tribunais com número superior a vinte e cinco julgadores", tipo: Inciso, rotulo: "XI", texto: "nos tribunais com número superior a vinte e cinco julgadores", ok: true},

		// Alínea.
		{linha: "a) os analfabetos;", tipo: Alinea, rotulo: "a", texto: "os analfabetos;", ok: true},

		// Agrupadores: o nome vem na linha seguinte, então o texto fica vazio.
		{linha: "PARTE ESPECIAL", tipo: Parte, rotulo: "especial", texto: "", ok: true},
		{linha: "LIVRO I", tipo: Livro, rotulo: "I", texto: "", ok: true},
		{linha: "TÍTULO II", tipo: Titulo, rotulo: "II", texto: "", ok: true},
		{linha: "CAPÍTULO I", tipo: Capitulo, rotulo: "I", texto: "", ok: true},
		{linha: "Seção II", tipo: Secao, rotulo: "II", texto: "", ok: true},
		{linha: "Subseção I", tipo: Subsecao, rotulo: "I", texto: "", ok: true},
		{linha: "PARTE GERAL", tipo: Parte, rotulo: "geral", texto: "", ok: true},
		{linha: "SEÇÃO III", tipo: Secao, rotulo: "III", texto: "", ok: true},
		{linha: "CAPITULO II", tipo: Capitulo, rotulo: "II", texto: "", ok: true}, // sem acento, em textos antigos
		{linha: "CAPÍTULO I - DOS CRIMES CONTRA A VIDA", tipo: Capitulo, rotulo: "I", texto: "DOS CRIMES CONTRA A VIDA", ok: true},

		// Não são rótulos.
		{linha: "", ok: false},
		{linha: "nos termos do art. 3º", ok: false},
		{linha: "Vide art. 5º", ok: false},
		{linha: "Arts. 5º e 6º", ok: false},
		{linha: "Incumbe ao Poder Público:", ok: false},         // começa com "I", mas não é inciso
		{linha: "A soberania popular será exercida", ok: false}, // começa com "A", mas não é alínea
		{linha: "DOS CRIMES CONTRA A VIDA", ok: false},          // nome de agrupador, não rótulo
		{linha: "Título II desta Lei", ok: false},               // remissão, não agrupador
		{linha: "Seção I deste Capítulo", ok: false},
		{linha: "Incumbe (nos termos da lei)", ok: false}, // sem separador, o numeral precisa de espaço depois
		{linha: "CIVIL E MILITAR", ok: false},             // só letras romanas, mas seguido de maiúscula
	}

	for _, tc := range testes {
		t.Run(tc.linha, func(t *testing.T) {
			d, ok := separarRotulo(tc.linha)

			if ok != tc.ok {
				t.Fatalf("separarRotulo(%q) ok = %v; esperado %v", tc.linha, ok, tc.ok)
			}
			if !ok {
				return
			}
			if d.Tipo != tc.tipo || d.Rotulo != tc.rotulo || d.Texto != tc.texto {
				t.Errorf("separarRotulo(%q) = {%v %q %q}; esperado {%v %q %q}",
					tc.linha, d.Tipo, d.Rotulo, d.Texto, tc.tipo, tc.rotulo, tc.texto)
			}
		})
	}
}

func TestSepararAnotacoes(t *testing.T) {
	testes := []struct {
		nome      string
		texto     string
		esperado  string
		anotacoes []string
	}{
		{
			nome:     "sem anotação",
			texto:    "Todos são iguais perante a lei.",
			esperado: "Todos são iguais perante a lei.",
		},
		{
			nome:      "no fim",
			texto:     "Matar alguém: (Redação dada pela Lei nº 14.994, de 2024)",
			esperado:  "Matar alguém:",
			anotacoes: []string{"(Redação dada pela Lei nº 14.994, de 2024)"},
		},
		{
			nome:      "várias, na ordem",
			texto:     "para: (Redação dada pela Emenda Constitucional nº 113, de 2021) (Vide ADI 7047) (Vide ADI 7064)",
			esperado:  "para:",
			anotacoes: []string{"(Redação dada pela Emenda Constitucional nº 113, de 2021)", "(Vide ADI 7047)", "(Vide ADI 7064)"},
		},
		{
			// Parênteses que fazem parte da norma ficam no texto.
			nome:     "parênteses do texto",
			texto:    "Pena – reclusão, de 20 (vinte) a 40 (quarenta) anos, nos termos da Lei nº 11.340 (Lei Maria da Penha).",
			esperado: "Pena – reclusão, de 20 (vinte) a 40 (quarenta) anos, nos termos da Lei nº 11.340 (Lei Maria da Penha).",
		},
		{
			// Sem o espaço que sobraria antes da pontuação.
			nome:      "no meio, antes de pontuação",
			texto:     "o lazer (Incluído pela Emenda Constitucional nº 26, de 2000); a segurança",
			esperado:  "o lazer; a segurança",
			anotacoes: []string{"(Incluído pela Emenda Constitucional nº 26, de 2000)"},
		},
		{
			nome:      "maiúsculas, minúsculas e variações",
			texto:     "(revogada) (Parágrafo único renumerado pela Lei nº 9.983, de 2000) (Vigência) (Produção de efeito) (VETADO)",
			esperado:  "",
			anotacoes: []string{"(revogada)", "(Parágrafo único renumerado pela Lei nº 9.983, de 2000)", "(Vigência)", "(Produção de efeito)", "(VETADO)"},
		},
		{
			// Anotação quebrada em duas linhas chega com espaços extras.
			nome:      "espaços dentro da anotação",
			texto:     "(Revogado   pela Lei nº 9.279, de 14.5.1996)",
			esperado:  "",
			anotacoes: []string{"(Revogado pela Lei nº 9.279, de 14.5.1996)"},
		},
		{
			// Parêntese aberto sem fechar: nada é removido.
			nome:     "parêntese sem fechar",
			texto:    "texto (Redação dada pela Lei",
			esperado: "texto (Redação dada pela Lei",
		},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			texto, anotacoes := separarAnotacoes(tc.texto)
			if texto != tc.esperado {
				t.Errorf("texto = %q; esperado %q", texto, tc.esperado)
			}
			if strings.Join(anotacoes, "|") != strings.Join(tc.anotacoes, "|") {
				t.Errorf("anotacoes = %q; esperado %q", anotacoes, tc.anotacoes)
			}
		})
	}
}

// formatar desenha a árvore como texto indentado, um nó por linha
// ("Tipo Rotulo: Texto", sem ": " quando não há texto). A rubrica aparece
// entre « » depois do rótulo, revogado e vetado ganham " [revogado]" e
// " [vetado]", e cada anotação vira uma linha "nota: …" logo abaixo do nó.
// Comparar o desenho em vez dos structs dá mensagens de falha legíveis e
// não depende de Filhos ser nil ou vazio.
func formatar(ds []*Dispositivo) string {
	var b strings.Builder
	var desenhar func(ds []*Dispositivo, nivel int)
	desenhar = func(ds []*Dispositivo, nivel int) {
		for _, d := range ds {
			recuo := strings.Repeat("  ", nivel)
			fmt.Fprintf(&b, "%s%v %s", recuo, d.Tipo, d.Rotulo)
			if d.Rubrica != "" {
				fmt.Fprintf(&b, " «%s»", d.Rubrica)
			}
			if d.Texto != "" {
				fmt.Fprintf(&b, ": %s", d.Texto)
			}
			if d.Revogado {
				b.WriteString(" [revogado]")
			}
			if d.Vetado {
				b.WriteString(" [vetado]")
			}
			b.WriteString("\n")
			for _, a := range d.Anotacoes {
				fmt.Fprintf(&b, "%s  nota: %s\n", recuo, a)
			}
			desenhar(d.Filhos, nivel+1)
		}
	}
	desenhar(ds, 0)
	return b.String()
}

// paraLinhas converte as linhas dos testes: "**texto**" é uma linha em
// negrito, o resto é texto normal.
func paraLinhas(ss []string) []Linha {
	var linhas []Linha
	for _, s := range ss {
		if texto, ok := strings.CutPrefix(s, "**"); ok {
			linhas = append(linhas, Linha{Texto: strings.TrimSuffix(texto, "**"), Negrito: true})
		} else {
			linhas = append(linhas, Linha{Texto: s})
		}
	}
	return linhas
}

func TestMontarArvore(t *testing.T) {
	testes := []struct {
		nome     string
		linhas   []string
		esperado string
	}{
		{
			nome:     "vazia",
			linhas:   nil,
			esperado: "",
		},
		{
			nome: "lei sem agrupadores",
			linhas: []string{
				"Art. 1º Esta Lei regula os direitos autorais.",
				"Art. 2º Os estrangeiros gozarão da proteção.",
			},
			esperado: `
Artigo 1: Esta Lei regula os direitos autorais.
Artigo 2: Os estrangeiros gozarão da proteção.
`,
		},
		{
			// Linhas antes do primeiro rótulo (cabeçalho, preâmbulo) são ignoradas.
			// O nome do agrupador vem na linha seguinte e vira o seu Texto.
			// Parágrafo único depois de incisos volta a ser filho do artigo.
			nome: "início da CF",
			linhas: []string{
				"PREÂMBULO",
				"Nós, representantes do povo brasileiro,",
				"TÍTULO I",
				"Dos Princípios Fundamentais",
				"Art. 1º A República Federativa do Brasil tem como fundamentos:",
				"I - a soberania;",
				"II - a cidadania;",
				"Parágrafo único. Todo o poder emana do povo.",
				"Art. 2º São Poderes da União o Legislativo, o Executivo e o Judiciário.",
			},
			esperado: `
Título I: Dos Princípios Fundamentais
  Artigo 1: A República Federativa do Brasil tem como fundamentos:
    Inciso I: a soberania;
    Inciso II: a cidadania;
    Parágrafo unico: Todo o poder emana do povo.
  Artigo 2: São Poderes da União o Legislativo, o Executivo e o Judiciário.
`,
		},
		{
			// Incisos de um parágrafo, alíneas de um inciso, e a volta ao artigo.
			nome: "art. 14 da CF",
			linhas: []string{
				"Art. 14. A soberania popular será exercida mediante:",
				"I - plebiscito;",
				"§ 1º O alistamento eleitoral e o voto são:",
				"I - obrigatórios para os maiores de dezoito anos;",
				"II - facultativos para:",
				"a) os analfabetos;",
				"b) os maiores de setenta anos;",
				"§ 2º Não podem alistar-se como eleitores os estrangeiros.",
			},
			esperado: `
Artigo 14: A soberania popular será exercida mediante:
  Inciso I: plebiscito;
  Parágrafo 1: O alistamento eleitoral e o voto são:
    Inciso I: obrigatórios para os maiores de dezoito anos;
    Inciso II: facultativos para:
      Alínea a: os analfabetos;
      Alínea b: os maiores de setenta anos;
  Parágrafo 2: Não podem alistar-se como eleitores os estrangeiros.
`,
		},
		{
			// A hierarquia pula níveis: Seção direto sob Título, artigo direto
			// sob Capítulo. Um Título novo fecha tudo que estava aberto.
			nome: "níveis pulados",
			linhas: []string{
				"TÍTULO I",
				"Seção I",
				"Art. 1º Primeiro.",
				"TÍTULO II",
				"CAPÍTULO I",
				"Art. 2º Segundo.",
			},
			esperado: `
Título I
  Seção I
    Artigo 1: Primeiro.
Título II
  Capítulo I
    Artigo 2: Segundo.
`,
		},
		{
			// Linha sem rótulo continua o texto do dispositivo anterior,
			// separada por um espaço. Vale também para nome de agrupador
			// quebrado em duas linhas.
			nome: "continuação de texto",
			linhas: []string{
				"CAPÍTULO I",
				"DOS DIREITOS E DEVERES",
				"INDIVIDUAIS E COLETIVOS",
				"Art. 5º Todos são iguais perante a lei,",
				"sem distinção de qualquer natureza.",
			},
			esperado: `
Capítulo I: DOS DIREITOS E DEVERES INDIVIDUAIS E COLETIVOS
  Artigo 5: Todos são iguais perante a lei, sem distinção de qualquer natureza.
`,
		},
		{
			// A anotação pode vir na linha seguinte ou quebrada entre linhas:
			// ela só é separada depois de o texto inteiro estar montado.
			nome: "anotações",
			linhas: []string{
				"Art. 121-A. Matar mulher por razões da condição do sexo feminino:",
				"(Incluído pela",
				"Lei nº 14.994, de 2024)",
				"§ 1º Considera-se que há razões da condição do sexo feminino quando o crime envolve: (Incluído pela Lei nº 14.994, de 2024)",
			},
			esperado: `
Artigo 121-A: Matar mulher por razões da condição do sexo feminino:
  nota: (Incluído pela Lei nº 14.994, de 2024)
  Parágrafo 1: Considera-se que há razões da condição do sexo feminino quando o crime envolve:
    nota: (Incluído pela Lei nº 14.994, de 2024)
`,
		},
		{
			// Revogado é quem fica sem texto depois de tirar as anotações.
			// "Art. 88 - Revogado o livramento…" é texto da norma, não revogação.
			nome: "revogados sem riscado",
			linhas: []string{
				"Art. 88 - Revogado o livramento, não poderá ser novamente concedido.",
				"Art. 187. (Revogado pela Lei nº 9.279, de 14.5.1996)",
				"Art 188. (Revogado",
				"pela Lei nº 9.279, de 14.5.1996)",
				"Art. 189. Texto. (Vide Lei nº 9.279, de 1996)",
				"§ 1º - . (Revogado pela Emenda Constitucional nº 6, de 1995)",
				"III (Revogado pela Emenda Constitucional nº 24, de 1999)",
				"IV - (Revogado).",
			},
			esperado: `
Artigo 88: Revogado o livramento, não poderá ser novamente concedido.
Artigo 187 [revogado]
  nota: (Revogado pela Lei nº 9.279, de 14.5.1996)
Artigo 188 [revogado]
  nota: (Revogado pela Lei nº 9.279, de 14.5.1996)
Artigo 189: Texto.
  nota: (Vide Lei nº 9.279, de 1996)
  Parágrafo 1 [revogado]
    nota: (Revogado pela Emenda Constitucional nº 6, de 1995)
    Inciso III [revogado]
      nota: (Revogado pela Emenda Constitucional nº 24, de 1999)
    Inciso IV [revogado]
      nota: (Revogado)
`,
		},
		{
			// Agrupador também perde a anotação do nome.
			nome: "anotação em agrupador",
			linhas: []string{
				"Seção III",
				"DOS MILITARES DOS ESTADOS, DO DISTRITO FEDERAL E DOS TERRITÓRIOS",
				"(Redação dada pela Emenda Constitucional nº 18, de 1998)",
			},
			esperado: `
Seção III: DOS MILITARES DOS ESTADOS, DO DISTRITO FEDERAL E DOS TERRITÓRIOS
  nota: (Redação dada pela Emenda Constitucional nº 18, de 1998)
`,
		},
		{
			// A data de assinatura encerra o texto: assinaturas, avisos e o
			// ADCT (que recomeça a numeração) não entram na árvore.
			nome: "fim da CF",
			linhas: []string{
				"Art. 250. Com o objetivo de assegurar recursos, a União poderá constituir fundo. (Incluído pela Emenda Constitucional nº 20, de 1998)",
				"Brasília, 5 de outubro de 1988.",
				"Ulysses Guimarães , Presidente - Mauro Benevides , 1º Vice-Presidente",
				"Este texto não substitui o publicado no D.O.U. de 5.10.1988.",
				"ATO DAS DISPOSIÇÕES CONSTITUCIONAIS TRANSITÓRIAS",
				"Art. 1º. O Presidente da República prestará o compromisso.",
			},
			esperado: `
Artigo 250: Com o objetivo de assegurar recursos, a União poderá constituir fundo.
  nota: (Incluído pela Emenda Constitucional nº 20, de 1998)
`,
		},
		{
			// Mesmo sem a linha da data, o cabeçalho do ADCT encerra o texto.
			nome: "ADCT sem data antes",
			linhas: []string{
				"Art. 250. Último.",
				"ATO DAS DISPOSIÇÕES CONSTITUCIONAIS TRANSITÓRIAS",
				"Art. 1º. Primeiro do ADCT.",
			},
			esperado: `
Artigo 250: Último.
`,
		},
		{
			// Fecho do CP: data com "º" e o resto da frase na mesma linha.
			nome: "fim do CP",
			linhas: []string{
				"Art. 361. Este Código entrará em vigor no dia 1º de janeiro de 1942.",
				"Rio de Janeiro, 7 de dezembro de 1940; 119º da Independência e 52º da República.",
				"GETÚLIO VARGAS",
			},
			esperado: `
Artigo 361: Este Código entrará em vigor no dia 1º de janeiro de 1942.
`,
		},
		{
			// Rubrica (negrito, sem rótulo) vai para o próximo dispositivo,
			// sem a anotação. O nome do agrupador também é negrito: a primeira
			// linha depois do rótulo é sempre o nome, e linha em maiúsculas
			// continua o nome; o resto em negrito é rubrica.
			nome: "rubricas do CP",
			linhas: []string{
				"**CAPÍTULO III**",
				"**DOS CRIMES CONTRA AS**",
				"**MARCAS DE INDÚSTRIA E COMÉRCIO**",
				"**Homicídio simples**",
				"Art. 121. Matar alguem:",
				"Pena - reclusão, de seis a vinte anos.",
				"**Caso de diminuição de pena**",
				"§ 1º Se o agente comete o crime impelido por motivo de relevante valor social.",
				"**Feminicídio (Incluído pela Lei nº 13.104, de 2015)**",
				"VI - (Revogado pela Lei nº 14.994, de 2024)",
			},
			esperado: `
Capítulo III: DOS CRIMES CONTRA AS MARCAS DE INDÚSTRIA E COMÉRCIO
  Artigo 121 «Homicídio simples»: Matar alguem: Pena - reclusão, de seis a vinte anos.
    Parágrafo 1 «Caso de diminuição de pena»: Se o agente comete o crime impelido por motivo de relevante valor social.
      Inciso VI «Feminicídio» [revogado]
        nota: (Revogado pela Lei nº 14.994, de 2024)
`,
		},
		{
			// Na CF o nome do agrupador vem em negrito e com maiúsculas e
			// minúsculas: por ser a primeira linha, ainda é nome.
			nome: "nome de agrupador da CF em negrito",
			linhas: []string{
				"**TÍTULO I**",
				"**Dos Princípios Fundamentais**",
				"Art. 1º A República Federativa do Brasil tem como fundamentos:",
			},
			esperado: `
Título I: Dos Princípios Fundamentais
  Artigo 1: A República Federativa do Brasil tem como fundamentos:
`,
		},
		{
			// Rubrica seguida de agrupador é descartada: não há dispositivo
			// para ela.
			nome: "rubrica antes de agrupador",
			linhas: []string{
				"Art. 1º Primeiro.",
				"**Rubrica solta**",
				"CAPÍTULO II",
				"Art. 2º Segundo.",
			},
			esperado: `
Artigo 1: Primeiro.
Capítulo II
  Artigo 2: Segundo.
`,
		},
		{
			// A pena (preceito secundário) é do artigo ou do parágrafo, não do
			// último inciso: no CP, "Pena - reclusão…" vem depois do inciso V
			// do § 2º do art. 121 e vale para o § 2º inteiro.
			nome: "pena volta ao parágrafo",
			linhas: []string{
				"Art. 121. Matar alguem:",
				"Pena - reclusão, de seis a vinte anos.",
				"§ 2° Se o homicídio é cometido:",
				"I - mediante paga;",
				"V - para assegurar a execução de outro crime:",
				"Pena - reclusão, de doze a trinta anos.",
				"§ 3º Se o homicídio é culposo:",
				"Pena – detenção, de um a três anos.",
			},
			esperado: `
Artigo 121: Matar alguem: Pena - reclusão, de seis a vinte anos.
  Parágrafo 2: Se o homicídio é cometido: Pena - reclusão, de doze a trinta anos.
    Inciso I: mediante paga;
    Inciso V: para assegurar a execução de outro crime:
  Parágrafo 3: Se o homicídio é culposo: Pena – detenção, de um a três anos.
`,
		},
		{
			// Vetado é como revogado: sem texto depois de tirar as anotações.
			nome: "vetados",
			linhas: []string{
				"Art. 359-O. (VETADO). (Incluído pela Lei nº 10.028, de 2000)",
				"Art. 359-U. (VETADO).” (Incluído pela Lei nº 14.197, de 2021)",
				"Art. 360. Ressalvada a legislação especial.",
				"I – (VETADO); (Incluído pela Lei nº 13.964, de 2019)",
			},
			esperado: `
Artigo 359-O [vetado]
  nota: (VETADO)
  nota: (Incluído pela Lei nº 10.028, de 2000)
Artigo 359-U [vetado]
  nota: (VETADO)
  nota: (Incluído pela Lei nº 14.197, de 2021)
Artigo 360: Ressalvada a legislação especial.
  Inciso I [vetado]
    nota: (VETADO)
    nota: (Incluído pela Lei nº 13.964, de 2019)
`,
		},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			got := formatar(montarArvore(paraLinhas(tc.linhas)))
			esperado := strings.TrimPrefix(tc.esperado, "\n")
			if got != esperado {
				t.Errorf("montarArvore() =\n%s\nesperado:\n%s", got, esperado)
			}
		})
	}
}
