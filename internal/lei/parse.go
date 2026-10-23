package lei

import "io"

// Parse lê o HTML de uma lei (em UTF-8) e devolve a árvore de dispositivos:
// agrupadores na raiz (ou artigos, se a lei não tiver agrupadores).
func Parse(r io.Reader) ([]*Dispositivo, error) {
	linhas, err := extrairLinhas(r)
	if err != nil {
		return nil, err
	}
	return montarArvore(linhas), nil
}
