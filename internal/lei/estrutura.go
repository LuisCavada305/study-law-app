package lei

import (
	"regexp"
	"strings"
)

var reArtigo = regexp.MustCompile(`^Art\.?\s+(\d{1,3}(?:\.\d{3})*(?:-[A-Z])?)[º°o]?\s*(?:[.\-–]\s*)?(.*)$`)

// Fase 2: linhas → árvore de dispositivos.

// separarRotulo verifica se a linha inicia com um dispositivo e em caso positivo retorna o rotulo separado do texto.
func separarRotulo(linha string) (Dispositivo, bool) {
	m := reArtigo.FindStringSubmatch(linha)

	if m != nil {
		return Dispositivo{Tipo: Artigo, Rotulo: strings.ReplaceAll(m[1], ".", ""), Texto: m[2]}, true
	}

	return Dispositivo{}, false
}
