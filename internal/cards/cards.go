// Package cards transforma a árvore de uma lei nos cards de estudo: um card
// por dispositivo normativo em vigor, com a referência e o contexto para
// reconhecê-lo sem ver o texto.
package cards

import (
	"strconv"
	"strings"

	"github.com/LuisCavada305/study-law-app/internal/lei"
)

// Card é um dispositivo pronto para estudo. A frente mostra Referencia,
// Caminho, Rubrica e Contexto; o verso mostra Texto.
type Card struct {
	// Chave identifica o dispositivo entre importações da mesma lei:
	// "art121/par2/incI". É o caminho a partir do artigo, nunca os
	// agrupadores, que mudam de numeração com mais frequência.
	Chave string
	// Ordem é a posição na lei, para estudar os novos na ordem do texto.
	Ordem int

	Tipo   lei.Tipo
	Rotulo string

	// Referencia é como o dispositivo é citado: "Art. 121, § 2º, inciso I".
	Referencia string
	// Caminho são os agrupadores acima dele: "Parte Especial › Título I — DOS
	// CRIMES CONTRA A PESSOA".
	Caminho string
	// Rubrica é a do próprio dispositivo ou a do ancestral mais próximo
	// ("Homicídio qualificado" vale para os incisos do § 2º).
	Rubrica string
	// Contexto é o texto do dispositivo pai, sem o qual um inciso não se
	// entende ("Se o homicídio é cometido:"). Vazio para artigos.
	Contexto string

	Texto string
}

// Gerar percorre a árvore e devolve os cards na ordem do texto. Agrupadores,
// revogados, vetados e dispositivos sem texto não viram card, mas os filhos
// de um artigo sem texto próprio, sim.
func Gerar(raiz []*lei.Dispositivo) []Card {
	g := gerador{vistas: map[string]int{}}
	g.visitar(raiz, nil, nil)
	return g.cards
}

type gerador struct {
	cards  []Card
	vistas map[string]int // chaves já usadas, para desempatar repetidas
}

// visitar desce pela árvore levando os agrupadores (caminho) e os
// dispositivos normativos (linhagem) acima de cada nó.
func (g *gerador) visitar(ds []*lei.Dispositivo, caminho, linhagem []*lei.Dispositivo) {
	for _, d := range ds {
		if d.Tipo.Agrupador() {
			g.visitar(d.Filhos, append(caminho[:len(caminho):len(caminho)], d), nil)
			continue
		}

		linha := append(linhagem[:len(linhagem):len(linhagem)], d)
		if !d.Revogado && !d.Vetado && d.Texto != "" {
			g.cards = append(g.cards, g.card(caminho, linha))
		}
		g.visitar(d.Filhos, caminho, linha)
	}
}

func (g *gerador) card(caminho, linha []*lei.Dispositivo) Card {
	d := linha[len(linha)-1]

	partesChave := make([]string, len(linha))
	partesRef := make([]string, len(linha))
	for i, x := range linha {
		partesChave[i] = prefixoChave[x.Tipo] + x.Rotulo
		partesRef[i] = Citar(x.Tipo, x.Rotulo)
	}

	// Rótulo repetido na mesma lei (erro do original) ganha "#2", "#3"…
	// pela ordem, o que se mantém entre importações.
	chave := strings.Join(partesChave, "/")
	g.vistas[chave]++
	if n := g.vistas[chave]; n > 1 {
		chave += "#" + strconv.Itoa(n)
	}

	var rubrica string
	for i := len(linha) - 1; i >= 0 && rubrica == ""; i-- {
		rubrica = linha[i].Rubrica
	}

	var contexto string
	if len(linha) > 1 {
		contexto = linha[len(linha)-2].Texto
	}

	nomes := make([]string, len(caminho))
	for i, a := range caminho {
		nomes[i] = Citar(a.Tipo, a.Rotulo)
		if a.Texto != "" {
			nomes[i] += " — " + a.Texto
		}
	}

	return Card{
		Chave:      chave,
		Ordem:      len(g.cards),
		Tipo:       d.Tipo,
		Rotulo:     d.Rotulo,
		Referencia: strings.Join(partesRef, ", "),
		Caminho:    strings.Join(nomes, " › "),
		Rubrica:    rubrica,
		Contexto:   contexto,
		Texto:      d.Texto,
	}
}

var prefixoChave = map[lei.Tipo]string{
	lei.Artigo:    "art",
	lei.Paragrafo: "par",
	lei.Inciso:    "inc",
	lei.Alinea:    "ali",
	lei.Item:      "ite",
}

// Citar escreve o rótulo como ele é citado: "Art. 5º", "Art. 121-A",
// "§ 1º", "Parágrafo único", "inciso IV", "alínea a", "Título II",
// "Parte Especial".
func Citar(t lei.Tipo, rotulo string) string {
	switch t {
	case lei.Artigo:
		return "Art. " + ordinal(rotulo)
	case lei.Paragrafo:
		if rotulo == "unico" {
			return "Parágrafo único"
		}
		return "§ " + ordinal(rotulo)
	case lei.Inciso:
		return "inciso " + rotulo
	case lei.Alinea:
		return "alínea " + rotulo
	case lei.Item:
		return "item " + rotulo
	case lei.Parte:
		return "Parte " + strings.ToUpper(rotulo[:1]) + rotulo[1:]
	default:
		return t.String() + " " + rotulo
	}
}

// ordinal põe "º" nos números de 1 a 9, como na lei: "5" → "5º",
// "5-A" → "5º-A", "10" → "10", "1000" → "1.000".
func ordinal(rotulo string) string {
	num, sufixo, _ := strings.Cut(rotulo, "-")
	n, err := strconv.Atoi(num)
	if err != nil {
		return rotulo
	}
	if n < 10 {
		num += "º"
	} else if n >= 1000 {
		num = strconv.Itoa(n/1000) + "." + num[len(num)-3:]
	}
	if sufixo != "" {
		return num + "-" + sufixo
	}
	return num
}
