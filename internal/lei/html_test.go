package lei

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// lerFixture abre testdata/nome, que está em windows-1252 como no Planalto,
// e devolve as linhas já extraídas.
func lerFixture(t *testing.T, nome string) []Linha {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", nome))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	linhas, err := extrairLinhas(charmap.Windows1252.NewDecoder().Reader(f))
	if err != nil {
		t.Fatalf("extrairLinhas(%s): %v", nome, err)
	}
	return linhas
}

// desenharLinhas mostra uma linha por linha de texto, com "**" na frente
// das que começam em negrito (a mesma convenção de paraLinhas).
func desenharLinhas(linhas []Linha) string {
	var b strings.Builder
	for _, l := range linhas {
		if l.Negrito {
			b.WriteString("**")
		}
		b.WriteString(l.Texto)
		b.WriteString("\n")
	}
	return b.String()
}

func TestExtrairLinhas(t *testing.T) {
	testes := []struct {
		nome     string
		html     string
		esperado string
	}{
		{
			nome:     "parágrafos",
			html:     "<p>Art. 1º Primeiro.</p><p>Art. 2º Segundo.</p>",
			esperado: "Art. 1º Primeiro.\nArt. 2º Segundo.\n",
		},
		{
			nome:     "riscado no meio",
			html:     "<p>Art. 5º <strike>velho</strike> novo</p>",
			esperado: "Art. 5º novo\n",
		},
		{
			nome:     "s e del também riscam",
			html:     "<p>a<s>X</s>b<del>Y</del>c</p>",
			esperado: "abc\n",
		},
		{
			nome:     "riscado por CSS",
			html:     `<p><span style="text-decoration: line-through">X</span>Y</p><p><span style="TEXT-DECORATION:LINE-THROUGH">Z</span>W</p>`,
			esperado: "Y\nW\n",
		},
		{
			nome:     "parágrafo todo riscado some",
			html:     "<p><strike>Art. 6º Versão antiga.</strike></p><p>Art. 6º Versão nova.</p>",
			esperado: "Art. 6º Versão nova.\n",
		},
		{
			nome:     "br quebra linha",
			html:     "<p>TÍTULO II<br>DO CRIME</p>",
			esperado: "TÍTULO II\nDO CRIME\n",
		},
		{
			// No CP: "<strike><br></strike>" separa a rubrica do artigo.
			nome:     "br riscado ainda quebra linha",
			html:     "<p>A<strike><br></strike>B</p>",
			esperado: "A\nB\n",
		},
		{
			nome:     "nbsp e quebras do código-fonte",
			html:     "<p>&nbsp;&nbsp;Art\n187.&nbsp;&nbsp;\t(Revogado)</p>",
			esperado: "Art 187. (Revogado)\n",
		},
		{
			// O CP tem "<b>A</b>rt. 107" e "Art. 1<b>º</b>": nada de espaço entre tags.
			nome:     "sem espaço entre tags",
			html:     "<p><b>A</b>rt. 107 - Extingue-se</p><p>Art. 1<b>º</b> - Não há crime</p>",
			esperado: "**Art. 107 - Extingue-se\nArt. 1º - Não há crime\n",
		},
		{
			// Na CF o ordinal vem num <sup> cercado de quebras de linha.
			nome:     "sup gruda no texto anterior",
			html:     "<p>§ 1\n<sup>\no\n</sup>\nA Câmara</p><p>§ 2<u><sup>o</sup></u>-A Texto</p>",
			esperado: "§ 1o A Câmara\n§ 2o-A Texto\n",
		},
		{
			nome:     "rubrica com anotação sem negrito",
			html:     `<p><b>Lugar do crime</b> <a href="x">(Redação dada pela Lei nº 7.209, de 1984)</a></p>`,
			esperado: "**Lugar do crime (Redação dada pela Lei nº 7.209, de 1984)\n",
		},
		{
			nome:     "negrito no meio da frase",
			html:     "<p>o <b>caput</b> deste artigo</p><p><strong>Homicídio simples</strong></p>",
			esperado: "o caput deste artigo\n**Homicídio simples\n",
		},
		{
			nome:     "font-weight no style",
			html:     `<p><b><span style="font-weight: normal">Art. 1º Texto</span></b></p><p><span style="font-weight:bold">Rubrica</span></p>`,
			esperado: "Art. 1º Texto\n**Rubrica\n",
		},
		{
			nome:     "travessão windows-1252 em referência numérica",
			html:     "<p>I &#150; texto</p>",
			esperado: "I – texto\n",
		},
		{
			// A página do Planalto tem um script de monitoramento no fim.
			nome:     "script, style, comentário, head e linhas vazias",
			html:     "<html><head><title>L2848</title><style>p{}</style></head><body><!-- nota --><p> </p><script>var f5=1;</script><p>Texto</p></body></html>",
			esperado: "Texto\n",
		},
		{
			// Tags mal aninhadas: o parser corrige como o navegador.
			nome:     "a sem fechar e b atravessando p",
			html:     `<b><p>Rubrica</b> <a href="x">(Revogado</p><p>Art. 2º Texto.</p>`,
			esperado: "**Rubrica (Revogado\nArt. 2º Texto.\n",
		},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			linhas, err := extrairLinhas(strings.NewReader(tc.html))
			if err != nil {
				t.Fatal(err)
			}
			if got := desenharLinhas(linhas); got != tc.esperado {
				t.Errorf("extrairLinhas() =\n%s\nesperado:\n%s", got, tc.esperado)
			}
		})
	}
}

func TestExtrairLinhasFixtureCPArts187a191(t *testing.T) {
	esperado := `**CAPÍTULO II
**DOS CRIMES CONTRA O PRIVILÉGIO DE INVENÇÃO
**Violação de privilégio de invenção
Art 187. (Revogado pela Lei nº 9.279, de 14.5.1996)
**Falsa atribuição de privilégio
Art 188. (Revogado pela Lei nº 9.279, de 14.5.1996)
**Usurpação ou indevida exploração de modelo ou desenho privilegiado
Art. 189. (Revogado pela Lei nº 9.279, de 14.5.1996)
**Falsa declaração de depósito em modelo ou desenho
Art. 190. (Revogado pela Lei nº 9.279, de 14.5.1996)
Art. 191. (Revogado pela Lei nº 9.279, de 14.5.1996)
`
	if got := desenharLinhas(lerFixture(t, "cp_arts187a191.html")); got != esperado {
		t.Errorf("linhas =\n%s\nesperado:\n%s", got, esperado)
	}
}

// Nas fixtures maiores, confere só as linhas que importam: as que precisam
// aparecer (exatamente iguais) e trechos que não podem aparecer em nenhuma.
func TestExtrairLinhasFixtures(t *testing.T) {
	testes := []struct {
		arquivo   string
		presentes []string
		ausentes  []string
	}{
		{
			arquivo: "cp_art121.html",
			presentes: []string{
				"**TÍTULO I",
				"**DOS CRIMES CONTRA A PESSOA",
				"**Homicídio simples",
				"Art. 121. Matar alguem:",
				"**Caso de diminuição de pena",
				"Art. 121-A. Matar mulher por razões da condição do sexo feminino: (Incluído pela Lei nº 14.994, de 2024)",
			},
		},
		{
			arquivo:   "cf_art100_strike_meio.html",
			presentes: []string{"I - quitação de débitos parcelados ou débitos inscritos em dívida ativa do ente federativo devedor, inclusive em transação resolutiva de litígio, e, subsidiariamente, débitos com a administração autárquica e fundacional do mesmo ente; (Incluído pela Emenda Constitucional nº 113, de 2021)"},
			ausentes:  []string{"com auto aplicabilidade", "lei da entidade federativa devedora"},
		},
		{
			arquivo:   "cf_art115_separador_riscado.html",
			presentes: []string{"III (Revogado pela Emenda Constitucional nº 24, de 1999)"},
			ausentes:  []string{"advogados e membros do Ministério Público", "classistas"},
		},
		{
			arquivo:   "cf_secao_line_through.html",
			presentes: []string{"Seção III", "DOS MILITARES DOS ESTADOS, DO DISTRITO FEDERAL E DOS TERRITÓRIOS"},
			ausentes:  []string{"DOS SERVIDORES PÚBLICOS MILITARES"},
		},
		{
			arquivo:   "cf_fim_adct.html",
			presentes: []string{"Brasília, 5 de outubro de 1988.", "**ATO DAS DISPOSIÇÕES CONSTITUCIONAIS TRANSITÓRIAS"},
		},
	}

	for _, tc := range testes {
		t.Run(tc.arquivo, func(t *testing.T) {
			linhas := strings.Split(desenharLinhas(lerFixture(t, tc.arquivo)), "\n")
			for _, p := range tc.presentes {
				if !slices.Contains(linhas, p) {
					t.Errorf("falta a linha %q", p)
				}
			}
			tudo := strings.Join(linhas, "\n")
			for _, a := range tc.ausentes {
				if strings.Contains(tudo, a) {
					t.Errorf("texto riscado apareceu: %q", a)
				}
			}
			if t.Failed() {
				t.Logf("linhas extraídas:\n%s", tudo)
			}
		})
	}
}
