package lei

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTipoString(t *testing.T) {
	testes := []struct {
		tipo     Tipo
		esperado string
	}{
		{Parte, "Parte"},
		{Livro, "Livro"},
		{Titulo, "Título"},
		{Subtitulo, "Subtítulo"},
		{Capitulo, "Capítulo"},
		{Secao, "Seção"},
		{Subsecao, "Subseção"},
		{Artigo, "Artigo"},
		{Paragrafo, "Parágrafo"},
		{Inciso, "Inciso"},
		{Alinea, "Alínea"},
		{Item, "Item"},
		{Item + 1, fmt.Sprintf("Tipo(%d)", Item+1)},
		{0, "Tipo(0)"},
	}

	for _, tc := range testes {
		t.Run(tc.esperado, func(t *testing.T) {
			if got := tc.tipo.String(); got != tc.esperado {
				t.Errorf("Tipo(%d).String() = %q; esperado %q", tc.tipo, got, tc.esperado)
			}
		})
	}
}

// Os casos cobrem as bordas da faixa [Parte, Subsecao] e os inválidos
// dos dois lados, onde um < trocado por <= apareceria.
func TestTipoAgrupador(t *testing.T) {
	testes := []struct {
		tipo     Tipo
		esperado bool
	}{
		{0, false},
		{Parte, true},
		{Titulo, true},
		{Subsecao, true},
		{Artigo, false},
		{Item, false},
		{Item + 1, false},
	}

	for _, tc := range testes {
		t.Run(tc.tipo.String(), func(t *testing.T) {
			if got := tc.tipo.Agrupador(); got != tc.esperado {
				t.Errorf("Tipo(%d).Agrupador() = %v; esperado %v", tc.tipo, got, tc.esperado)
			}
		})
	}
}

// art14 monta, à mão, um trecho da CF/88: art. 14, § 1º, I.
func art14() *Dispositivo {
	return &Dispositivo{
		Tipo:   Artigo,
		Rotulo: "14",
		Texto:  "A soberania popular será exercida pelo sufrágio universal e pelo voto direto e secreto, com valor igual para todos, e, nos termos da lei, mediante:",
		Filhos: []*Dispositivo{
			{
				Tipo:   Paragrafo,
				Rotulo: "1",
				Texto:  "O alistamento eleitoral e o voto são:",
				Filhos: []*Dispositivo{
					{
						Tipo:   Inciso,
						Rotulo: "I",
						Texto:  "obrigatórios para os maiores de dezoito anos;",
					},
				},
			},
		},
	}
}

func TestDispositivoNavegacao(t *testing.T) {
	inciso := art14().Filhos[0].Filhos[0]

	if inciso.Tipo != Inciso {
		t.Errorf("Tipo = %v; esperado %v", inciso.Tipo, Inciso)
	}
	if inciso.Rotulo != "I" {
		t.Errorf("Rotulo = %q; esperado %q", inciso.Rotulo, "I")
	}
	if len(inciso.Filhos) != 0 {
		t.Errorf("len(Filhos) = %d; esperado 0", len(inciso.Filhos))
	}
}

// Filhos guarda ponteiros: copiar um elemento copia o ponteiro, não o
// Dispositivo, então a alteração aparece também pela árvore.
func TestDispositivoFilhosCompartilhados(t *testing.T) {
	artigo := art14()
	paragrafo := artigo.Filhos[0]

	paragrafo.Texto = "alterado"

	if got := artigo.Filhos[0].Texto; got != "alterado" {
		t.Errorf("artigo.Filhos[0].Texto = %q; esperado %q", got, "alterado")
	}
}

// No JSON o tipo sai pelo nome e os campos vazios somem.
func TestDispositivoJSON(t *testing.T) {
	d := &Dispositivo{
		Tipo:      Artigo,
		Rotulo:    "187",
		Revogado:  true,
		Anotacoes: []string{"(Revogado pela Lei nº 9.279, de 14.5.1996)"},
	}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"tipo":"Artigo","rotulo":"187","revogado":true,"anotacoes":["(Revogado pela Lei nº 9.279, de 14.5.1996)"]}`
	if string(got) != esperado {
		t.Errorf("json.Marshal() =\n%s\nesperado:\n%s", got, esperado)
	}
}
