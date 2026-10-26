package lei

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"github.com/LuisCavada305/study-law-app/internal/planalto"
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

// parseAmostraGz roda Parse numa amostra comprimida de testdata. As páginas
// maiores ficam em .html.gz (9 MB viram 1,5 MB). O encoding varia entre as
// leis, então a conversão é a de planalto.ParaUTF8.
func parseAmostraGz(t *testing.T, nome string) []*Dispositivo {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", nome))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	pagina, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	pagina, err = planalto.ParaUTF8(pagina)
	if err != nil {
		t.Fatal(err)
	}

	raiz, err := Parse(bytes.NewReader(pagina))
	if err != nil {
		t.Fatalf("Parse(%s): %v", nome, err)
	}
	return raiz
}

// buscar devolve o primeiro nó com o tipo e o rótulo dados, ou nil.
func buscar(raiz []*Dispositivo, tipo Tipo, rotulo string) *Dispositivo {
	var achado *Dispositivo
	percorrer(raiz, func(d *Dispositivo) {
		if achado == nil && d.Tipo == tipo && d.Rotulo == rotulo {
			achado = d
		}
	})
	return achado
}

// rotulos lista "Tipo rótulo" dos nós, como "Título I".
func rotulos(ds []*Dispositivo) string {
	var partes []string
	for _, d := range ds {
		partes = append(partes, d.Tipo.String()+" "+d.Rotulo)
	}
	return strings.Join(partes, ", ")
}

// Leis baixadas do Planalto em 26/10/2026. Cada uma trouxe padrões novos;
// os números pegam regressões, e as verificações de cada lei, o defeito
// que ela revelou.
func TestParseLeis(t *testing.T) {
	testes := []struct {
		arquivo  string
		artigos  int
		ultimo   string
		raiz     string // vazio: não confere
		conferir func(t *testing.T, raiz []*Dispositivo)
	}{
		{
			// O decreto-lei que aprova a CLT vem antes dela, com fecho
			// próprio; os quadros do anexo, depois do art. 922.
			arquivo: "clt.html.gz", artigos: 847, ultimo: "922",
			raiz: "Título I, Título II, Título II-A, Título III, Título IV, Título IV-A, Título V, Título VI, " +
				"Título VI-A, Título VII, Título VII-A, Título VIII, Título IX, Título X, Título XI",
			conferir: func(t *testing.T, raiz []*Dispositivo) {
				if a := buscarArtigo(raiz, "1"); a == nil || !strings.HasPrefix(a.Texto, "Esta Consolidação estatui") {
					t.Errorf("art. 1º = %+v; esperado o da CLT, não o do decreto", a)
				}
				if a := buscarArtigo(raiz, "922"); a == nil || len(a.Texto) > 200 {
					t.Errorf("art. 922 = %+v; esperado sem os quadros do anexo", a)
				}
				for _, r := range []string{"189", "194"} { // "Art. . 189", "Art.194"
					if buscarArtigo(raiz, r) == nil {
						t.Errorf("art. %s não encontrado", r)
					}
				}
				if a := buscarArtigo(raiz, "401-A"); a == nil || !a.Vetado { // "Art. 401A."
					t.Errorf("art. 401-A = %+v; esperado vetado", a)
				}
				if a := buscarArtigo(raiz, "625-A"); a == nil || len(a.Filhos) != 1 || a.Filhos[0].Rotulo != "unico" {
					t.Errorf("art. 625-A = %+v; esperado com parágrafo único", a)
				}
			},
		},
		{
			arquivo: "cpp.html.gz", artigos: 846, ultimo: "811",
			raiz: "Livro I, Livro II, Livro III, Livro IV, Livro V, Livro VI",
		},
		{
			// "P A R T E G E R A L", Subtítulo, "Art. 1337" sem ponto e o
			// índice depois da assinatura.
			arquivo: "cc.html.gz", artigos: 2083, ultimo: "2046",
			raiz: "Parte geral, Parte especial",
			conferir: func(t *testing.T, raiz []*Dispositivo) {
				if a := buscarArtigo(raiz, "1337"); a == nil || !strings.HasPrefix(a.Texto, "O condômino, ou possuidor") {
					t.Errorf("art. 1.337 = %+v; no original está \"Art. 1337.\", sem ponto", a)
				}
				if buscar(raiz, Subtitulo, "II") == nil {
					t.Error("Subtítulo II não encontrado")
				}
				if buscar(raiz, Livro, "complementar") == nil {
					t.Error("Livro Complementar não encontrado")
				}
			},
		},
		{
			// Os arts. 1.061 a 1.071 alteram outras leis: o texto citado
			// fica no artigo, sem virar dispositivo do CPC.
			arquivo: "cpc.html.gz", artigos: 1073, ultimo: "1072",
			raiz: "Parte geral, Parte especial",
			conferir: func(t *testing.T, raiz []*Dispositivo) {
				if a := buscarArtigo(raiz, "1071"); a == nil || len(a.Filhos) != 0 || !strings.Contains(a.Texto, "216-A") {
					t.Errorf("art. 1.071 = %.80q…; esperado com o art. 216-A da Lei 6.015 no texto", a.Texto)
				}
			},
		},
		{arquivo: "cdc.html.gz", artigos: 130, ultimo: "119"},
		{
			// "LIVRO PRIMEIRO", por extenso.
			arquivo: "ctn.html.gz", artigos: 231, ultimo: "218",
			raiz: "Artigo 1, Livro I, Livro II",
		},
		{
			// Agrupadores em caixa mista e "§ 2o -A.".
			arquivo: "eca.html.gz", artigos: 327, ultimo: "267",
			raiz: "Título I, Título II, Título III, Parte especial",
			conferir: func(t *testing.T, raiz []*Dispositivo) {
				a := buscarArtigo(raiz, "46")
				if a == nil || !strings.Contains(rotulos(a.Filhos), "Parágrafo 2-A") || !strings.Contains(rotulos(a.Filhos), "Parágrafo 3-A") {
					t.Errorf("art. 46: filhos = %s; esperado com §§ 2º-A e 3º-A", rotulos(a.Filhos))
				}
				if a := buscarArtigo(raiz, "263"); a == nil || len(a.Filhos) != 0 {
					t.Errorf("art. 263 tem os §§ do CP que ele cita como filhos: %s", rotulos(a.Filhos))
				}
			},
		},
		{
			// "Capítulo Único" e "Subseção IV" repetida.
			arquivo: "l8112.html.gz", artigos: 247, ultimo: "253",
			raiz: "Título I, Título II, Título III, Título IV, Título V, Título VI, Título VII, Título VIII, Título IX",
			conferir: func(t *testing.T, raiz []*Dispositivo) {
				if buscar(raiz, Capitulo, "unico") == nil {
					t.Error("Capítulo Único não encontrado")
				}
			},
		},
		{
			// O art. 178 acrescenta ao CP os arts. 337-E a 337-P, dentro
			// de <blockquote>.
			arquivo: "l14133.html.gz", artigos: 196, ultimo: "194",
			raiz: "Título I, Título II, Título III, Título IV, Título V",
			conferir: func(t *testing.T, raiz []*Dispositivo) {
				if buscarArtigo(raiz, "337-E") != nil {
					t.Error("art. 337-E do CP virou artigo da Lei 14.133")
				}
				if a := buscarArtigo(raiz, "178"); a == nil || !strings.Contains(a.Texto, "Art. 337-E") {
					t.Error("art. 178 sem o texto que ele acrescenta ao CP")
				}
			},
		},
	}

	for _, tc := range testes {
		t.Run(tc.arquivo, func(t *testing.T) {
			raiz := parseAmostraGz(t, tc.arquivo)

			var artigos []*Dispositivo
			percorrer(raiz, func(d *Dispositivo) {
				if d.Tipo == Artigo {
					artigos = append(artigos, d)
				}
				if strings.Contains(d.Texto, "Vigência encerrada") {
					t.Errorf("%v %s: selo no texto: %.60q…", d.Tipo, d.Rotulo, d.Texto)
				}
				vistos := map[string]bool{}
				for _, f := range d.Filhos {
					chave := f.Tipo.String() + " " + f.Rotulo
					if vistos[chave] {
						t.Errorf("%v %s tem %s repetido", d.Tipo, d.Rotulo, chave)
					}
					vistos[chave] = true
				}
			})
			if len(artigos) != tc.artigos {
				t.Errorf("%d artigos; esperado %d", len(artigos), tc.artigos)
			}
			if len(artigos) > 0 && (artigos[0].Rotulo != "1" || artigos[len(artigos)-1].Rotulo != tc.ultimo) {
				t.Errorf("artigos de %s a %s; esperado de 1 a %s", artigos[0].Rotulo, artigos[len(artigos)-1].Rotulo, tc.ultimo)
			}
			if tc.raiz != "" && rotulos(raiz) != tc.raiz {
				t.Errorf("raiz = %s\nesperado %s", rotulos(raiz), tc.raiz)
			}
			if tc.conferir != nil {
				tc.conferir(t, raiz)
			}
		})
	}
}
