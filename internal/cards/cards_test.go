package cards

import (
	"testing"

	"github.com/LuisCavada305/study-law-app/internal/lei"
)

func TestCitar(t *testing.T) {
	testes := []struct {
		tipo     lei.Tipo
		rotulo   string
		esperado string
	}{
		{lei.Artigo, "5", "Art. 5º"},
		{lei.Artigo, "5-A", "Art. 5º-A"},
		{lei.Artigo, "10", "Art. 10"},
		{lei.Artigo, "121-A", "Art. 121-A"},
		{lei.Artigo, "1000", "Art. 1.000"},
		{lei.Paragrafo, "1", "§ 1º"},
		{lei.Paragrafo, "2-B", "§ 2º-B"},
		{lei.Paragrafo, "unico", "Parágrafo único"},
		{lei.Inciso, "IV", "inciso IV"},
		{lei.Alinea, "a", "alínea a"},
		{lei.Parte, "especial", "Parte Especial"},
		{lei.Titulo, "II", "Título II"},
	}
	for _, tc := range testes {
		if got := Citar(tc.tipo, tc.rotulo); got != tc.esperado {
			t.Errorf("Citar(%v, %q) = %q; esperado %q", tc.tipo, tc.rotulo, got, tc.esperado)
		}
	}
}

// arvore monta, à mão, um trecho do CP com os casos que importam.
func arvore() []*lei.Dispositivo {
	return []*lei.Dispositivo{{
		Tipo: lei.Parte, Rotulo: "especial",
		Filhos: []*lei.Dispositivo{{
			Tipo: lei.Titulo, Rotulo: "I", Texto: "DOS CRIMES CONTRA A PESSOA",
			Filhos: []*lei.Dispositivo{
				{
					Tipo: lei.Artigo, Rotulo: "121", Rubrica: "Homicídio simples", Texto: "Matar alguem:",
					Filhos: []*lei.Dispositivo{
						{
							Tipo: lei.Paragrafo, Rotulo: "2", Rubrica: "Homicídio qualificado", Texto: "Se o homicídio é cometido:",
							Filhos: []*lei.Dispositivo{
								{Tipo: lei.Inciso, Rotulo: "I", Texto: "mediante paga;"},
								{Tipo: lei.Inciso, Rotulo: "VI", Revogado: true},
							},
						},
					},
				},
				{Tipo: lei.Artigo, Rotulo: "187", Revogado: true},
				{Tipo: lei.Artigo, Rotulo: "359-O", Vetado: true},
				// Rótulo repetido por erro do original.
				{Tipo: lei.Artigo, Rotulo: "200", Texto: "Primeiro."},
				{Tipo: lei.Artigo, Rotulo: "200", Texto: "Segundo."},
			},
		}},
	}}
}

func TestGerar(t *testing.T) {
	got := Gerar(arvore())

	esperado := []Card{
		{
			Chave: "art121", Ordem: 0, Tipo: lei.Artigo, Rotulo: "121",
			Referencia: "Art. 121",
			Caminho:    "Parte Especial › Título I — DOS CRIMES CONTRA A PESSOA",
			Rubrica:    "Homicídio simples",
			Texto:      "Matar alguem:",
		},
		{
			Chave: "art121/par2", Ordem: 1, Tipo: lei.Paragrafo, Rotulo: "2",
			Referencia: "Art. 121, § 2º",
			Caminho:    "Parte Especial › Título I — DOS CRIMES CONTRA A PESSOA",
			Rubrica:    "Homicídio qualificado",
			Contexto:   "Matar alguem:",
			Texto:      "Se o homicídio é cometido:",
		},
		{
			// A rubrica vem do parágrafo; o contexto é o texto dele.
			Chave: "art121/par2/incI", Ordem: 2, Tipo: lei.Inciso, Rotulo: "I",
			Referencia: "Art. 121, § 2º, inciso I",
			Caminho:    "Parte Especial › Título I — DOS CRIMES CONTRA A PESSOA",
			Rubrica:    "Homicídio qualificado",
			Contexto:   "Se o homicídio é cometido:",
			Texto:      "mediante paga;",
		},
		{
			Chave: "art200", Ordem: 3, Tipo: lei.Artigo, Rotulo: "200",
			Referencia: "Art. 200",
			Caminho:    "Parte Especial › Título I — DOS CRIMES CONTRA A PESSOA",
			Texto:      "Primeiro.",
		},
		{
			Chave: "art200#2", Ordem: 4, Tipo: lei.Artigo, Rotulo: "200",
			Referencia: "Art. 200",
			Caminho:    "Parte Especial › Título I — DOS CRIMES CONTRA A PESSOA",
			Texto:      "Segundo.",
		},
	}

	if len(got) != len(esperado) {
		t.Fatalf("len(Gerar()) = %d; esperado %d\n%+v", len(got), len(esperado), got)
	}
	for i := range esperado {
		if got[i] != esperado[i] {
			t.Errorf("card %d =\n%+v\nesperado:\n%+v", i, got[i], esperado[i])
		}
	}
}

// Artigo sem texto próprio não vira card, mas os filhos dele sim.
func TestGerarArtigoSemTexto(t *testing.T) {
	raiz := []*lei.Dispositivo{{
		Tipo: lei.Artigo, Rotulo: "1",
		Filhos: []*lei.Dispositivo{{Tipo: lei.Paragrafo, Rotulo: "unico", Texto: "Texto."}},
	}}
	got := Gerar(raiz)
	if len(got) != 1 || got[0].Chave != "art1/parunico" || got[0].Referencia != "Art. 1º, Parágrafo único" {
		t.Errorf("Gerar() = %+v", got)
	}
}
