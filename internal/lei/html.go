package lei

// Fase 1: HTML → linhas de texto limpas (sem <strike>/<s>/<del>).

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// blocos são os elementos que começam e terminam uma linha. <br> também
// quebra linha, mas é tratado à parte porque não tem conteúdo.
var blocos = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Blockquote: true, atom.Center: true,
	atom.Li: true, atom.Ul: true, atom.Ol: true,
	atom.Table: true, atom.Tr: true, atom.Td: true, atom.Th: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
}

// ignorados são os elementos cujo conteúdo nunca é texto da lei.
var ignorados = map[atom.Atom]bool{
	atom.Head: true, atom.Script: true, atom.Style: true, atom.Noscript: true,
}

// extrairLinhas lê o HTML (em UTF-8) e devolve as linhas de texto visíveis,
// sem o texto riscado.
//
// Usa html.Parse, que corrige tags mal aninhadas (<a> sem fechar, <b>
// atravessando <p>) do mesmo jeito que o navegador: riscado e negrito valem
// para o que aparece riscado e em negrito na tela. O parser também troca
// referências como &#150; pelo caractere do windows-1252 ("–").
func extrairLinhas(r io.Reader) ([]Linha, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var e extrator
	e.visitar(doc, false, false)
	e.fecharLinha()
	return e.linhas, nil
}

// extrator acumula o texto da linha atual enquanto a árvore é percorrida.
type extrator struct {
	linhas []Linha

	// buf guarda o texto cru da linha: os trechos são emendados sem
	// espaço ("<b>A</b>rt." vira "Art.") e os espaços só são normalizados
	// em fecharLinha.
	buf []byte

	// negrito vale para o primeiro trecho com texto da linha; temTexto
	// diz se esse trecho já apareceu.
	negrito  bool
	temTexto bool
}

func (e *extrator) visitar(n *html.Node, riscado, negrito bool) {
	switch n.Type {
	case html.TextNode:
		if !riscado {
			e.escrever(n.Data, negrito)
		}
		return
	case html.CommentNode, html.DoctypeNode:
		return
	case html.ElementNode:
		if ignorados[n.DataAtom] {
			return
		}
		if n.DataAtom == atom.Br {
			e.fecharLinha() // mesmo riscado: "<strike><br></strike>" no CP
			return
		}
		if n.DataAtom == atom.Sup {
			e.visitarSup(n, riscado, negrito)
			return
		}
		riscado = riscado || ehRiscado(n)
		negrito = pesoDaFonte(n, negrito)
	}

	bloco := n.Type == html.ElementNode && blocos[n.DataAtom]
	if bloco {
		e.fecharLinha()
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e.visitar(c, riscado, negrito)
	}
	if bloco {
		e.fecharLinha()
	}
}

// visitarSup cola o conteúdo do <sup> no texto anterior, sem os espaços
// em volta: "§ 1\n<sup>\no\n</sup>" vira "§ 1o", como aparece na tela.
func (e *extrator) visitarSup(n *html.Node, riscado, negrito bool) {
	riscado = riscado || ehRiscado(n)
	if riscado {
		return
	}
	var sup strings.Builder
	for c := range n.Descendants() {
		if c.Type == html.TextNode {
			sup.WriteString(strings.TrimSpace(c.Data))
		}
	}
	if sup.Len() == 0 {
		return
	}
	e.buf = bytes.TrimRight(e.buf, " \t\r\n ")
	e.escrever(sup.String(), pesoDaFonte(n, negrito))
}

func (e *extrator) escrever(s string, negrito bool) {
	if !e.temTexto && strings.TrimSpace(strings.ReplaceAll(s, " ", " ")) != "" {
		e.negrito = negrito
		e.temTexto = true
	}
	e.buf = append(e.buf, s...)
}

// fecharLinha normaliza os espaços (inclusive &nbsp;) e guarda a linha,
// se ela tiver texto.
func (e *extrator) fecharLinha() {
	texto := strings.Join(strings.Fields(strings.ReplaceAll(string(e.buf), " ", " ")), " ")
	if texto != "" {
		e.linhas = append(e.linhas, Linha{Texto: texto, Negrito: e.negrito})
	}
	e.buf = e.buf[:0]
	e.negrito = false
	e.temTexto = false
}

// ehRiscado informa se o elemento risca o seu conteúdo: <strike>, <s>,
// <del> ou CSS "text-decoration: line-through".
func ehRiscado(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Strike, atom.S, atom.Del:
		return true
	}
	return strings.Contains(estilo(n), "line-through")
}

// pesoDaFonte devolve se o conteúdo de n fica em negrito, partindo do que
// vale para o pai: <b> e <strong> ligam, e "font-weight" no style manda.
func pesoDaFonte(n *html.Node, negrito bool) bool {
	if n.DataAtom == atom.B || n.DataAtom == atom.Strong {
		negrito = true
	}
	s := estilo(n)
	switch {
	case strings.Contains(s, "font-weight:normal"), strings.Contains(s, "font-weight:400"):
		return false
	case strings.Contains(s, "font-weight:bold"), strings.Contains(s, "font-weight:700"):
		return true
	}
	return negrito
}

// estilo devolve o atributo style em minúsculas e sem espaços, para que
// "text-decoration: line-through" e "TEXT-DECORATION:LINE-THROUGH" casem igual.
func estilo(n *html.Node) string {
	for _, a := range n.Attr {
		if a.Key == "style" {
			return strings.Join(strings.Fields(strings.ToLower(a.Val)), "")
		}
	}
	return ""
}
