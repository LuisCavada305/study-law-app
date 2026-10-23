// Package lei transforma o texto de uma lei na árvore de dispositivos:
// agrupadores (parte, livro, título, capítulo, seção, subseção) e
// dispositivos (artigo, parágrafo, inciso, alínea, item).
package lei

import "fmt"

// Tipo identifica o nível de um dispositivo na hierarquia da lei.
type Tipo int

// Tipos de dispositivo, do mais alto ao mais baixo da hierarquia.
// A ordem numérica é a ordem hierárquica: um tipo menor contém os maiores.
const (
	Parte Tipo = iota + 1
	Livro
	Titulo
	Capitulo
	Secao
	Subsecao
	Artigo
	Paragrafo
	Inciso
	Alinea
	Item
)

// String retorna o nome do tipo, ou o número se for inválido.
func (t Tipo) String() string {
	switch t {
	case Parte:
		return "Parte"
	case Livro:
		return "Livro"
	case Titulo:
		return "Título"
	case Capitulo:
		return "Capítulo"
	case Secao:
		return "Seção"
	case Subsecao:
		return "Subseção"
	case Artigo:
		return "Artigo"
	case Paragrafo:
		return "Parágrafo"
	case Inciso:
		return "Inciso"
	case Alinea:
		return "Alínea"
	case Item:
		return "Item"
	default:
		return fmt.Sprintf("Tipo(%d)", int(t))
	}
}

// MarshalText faz o tipo aparecer pelo nome no JSON ("Artigo", não 7).
func (t Tipo) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// Agrupador informa se t é um agrupador (de Parte a Subsecao), isto é,
// um título de organização sem texto normativo, que não gera cards.
func (t Tipo) Agrupador() bool {
	return Parte <= t && t < Artigo
}

// Dispositivo é um nó da árvore da lei: um agrupador ou um dispositivo
// normativo. Texto riscado no original nunca chega até aqui.
type Dispositivo struct {
	Tipo Tipo `json:"tipo"`

	// Rotulo é o identificador normalizado, sem "Art.", "§" nem "º":
	// "5", "121-A", "unico" (parágrafo único), "II", "a", "especial".
	Rotulo string `json:"rotulo"`

	// Texto é o conteúdo sem o rótulo e sem as anotações. Para agrupadores,
	// é o nome do bloco ("Dos crimes contra a pessoa").
	Texto string `json:"texto,omitempty"`

	// Rubrica é o título em negrito que o CP põe antes de alguns
	// dispositivos ("Homicídio qualificado" antes do § 2º do art. 121),
	// sem anotações. Vazia quando não há.
	Rubrica string `json:"rubrica,omitempty"`

	// Revogado marca o dispositivo revogado que aparece sem riscado,
	// como "Art. 187. (Revogado pela Lei nº 9.279, de 1996)".
	Revogado bool `json:"revogado,omitempty"`

	// Vetado marca o dispositivo vetado, que nunca entrou em vigor:
	// "Art. 359-O. (VETADO)". Como o revogado, não gera card.
	Vetado bool `json:"vetado,omitempty"`

	// Anotacoes guarda as notas do original, como "(Redação dada pela
	// Lei nº …)", na ordem em que aparecem. Elas não fazem parte de Texto.
	Anotacoes []string `json:"anotacoes,omitempty"`

	// Filhos são os dispositivos contidos neste, na ordem do texto.
	Filhos []*Dispositivo `json:"filhos,omitempty"`
}
