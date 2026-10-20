// Package lei transforma o texto de uma lei na árvore de dispositivos
// (artigo, parágrafo, inciso, alínea, item).
package lei

import "fmt"

// Tipo identifica o nível de um dispositivo na hierarquia da lei.
type Tipo int

// Tipos de dispositivo, do mais alto ao mais baixo da hierarquia.
const (
	Artigo Tipo = iota + 1
	Paragrafo
	Inciso
	Alinea
	Item
)

// String retorna o nome do tipo, ou o número se for inválido.
func (t Tipo) String() string {
	switch t {
	case Artigo:
		return "Artigo"
	case Paragrafo:
		return "Paragrafo"
	case Inciso:
		return "Inciso"
	case Alinea:
		return "Alinea"
	case Item:
		return "Item"
	default:
		return fmt.Sprintf("Tipo(%d)", int(t))
	}
}
