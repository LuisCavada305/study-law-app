package lei

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// parseAmostra roda Parse numa amostra inteira de testdata (windows-1252).
func parseAmostra(t *testing.T, nome string) []*Dispositivo {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", nome))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	raiz, err := Parse(charmap.Windows1252.NewDecoder().Reader(f))
	if err != nil {
		t.Fatalf("Parse(%s): %v", nome, err)
	}
	return raiz
}

// percorrer chama f para cada nó da árvore, em ordem.
func percorrer(ds []*Dispositivo, f func(*Dispositivo)) {
	for _, d := range ds {
		f(d)
		percorrer(d.Filhos, f)
	}
}

// buscarArtigo devolve o artigo com o rótulo dado, ou nil.
func buscarArtigo(raiz []*Dispositivo, rotulo string) *Dispositivo {
	var achado *Dispositivo
	percorrer(raiz, func(d *Dispositivo) {
		if achado == nil && d.Tipo == Artigo && d.Rotulo == rotulo {
			achado = d
		}
	})
	return achado
}

// Testes de integração nas leis inteiras: pegam regressões que as tabelas
// pequenas não pegam (um padrão novo que engole metade da lei, por exemplo).
func TestParseCF(t *testing.T) {
	raiz := parseAmostra(t, "cf88.html")

	if len(raiz) != 9 {
		t.Errorf("len(raiz) = %d; esperado 9 títulos", len(raiz))
	}
	for _, d := range raiz {
		if d.Tipo != Titulo {
			t.Errorf("raiz tem %v %s; esperado só títulos", d.Tipo, d.Rotulo)
		}
	}

	var ultimo *Dispositivo
	percorrer(raiz, func(d *Dispositivo) {
		if d.Tipo == Artigo {
			ultimo = d
		}
	})
	if ultimo == nil || ultimo.Rotulo != "250" {
		t.Errorf("último artigo = %v; esperado 250 (o ADCT fica de fora)", ultimo)
	}

	art5 := buscarArtigo(raiz, "5")
	if art5 == nil {
		t.Fatal("art. 5º não encontrado")
	}
	if got := len(art5.Filhos); got < 79 {
		t.Errorf("art. 5º tem %d filhos; esperado ao menos 79 incisos e parágrafos", got)
	}
	if !strings.HasPrefix(art5.Texto, "Todos são iguais perante a lei") {
		t.Errorf("art. 5º Texto = %.40q…", art5.Texto)
	}
}

func TestParseCP(t *testing.T) {
	raiz := parseAmostra(t, "cp.html")

	if len(raiz) != 2 || raiz[0].Rotulo != "geral" || raiz[1].Rotulo != "especial" {
		t.Errorf("raiz = %d nós; esperado Parte geral e Parte especial", len(raiz))
	}

	art121 := buscarArtigo(raiz, "121")
	if art121 == nil {
		t.Fatal("art. 121 não encontrado")
	}
	if art121.Rubrica != "Homicídio simples" {
		t.Errorf("art. 121 Rubrica = %q", art121.Rubrica)
	}
	if art121.Texto != "Matar alguem: Pena - reclusão, de seis a vinte anos." {
		t.Errorf("art. 121 Texto = %q", art121.Texto)
	}

	art187 := buscarArtigo(raiz, "187")
	if art187 == nil || !art187.Revogado {
		t.Errorf("art. 187 = %+v; esperado revogado", art187)
	}

	if art := buscarArtigo(raiz, "361"); art == nil {
		t.Error("art. 361 (o último) não encontrado")
	}
}
