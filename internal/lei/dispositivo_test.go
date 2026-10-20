package lei

import "testing"

func TestTipoString(t *testing.T) {
	testes := []struct {
		tipo     Tipo
		esperado string
	}{
		{Artigo, "Artigo"},
		{Paragrafo, "Paragrafo"},
		{Inciso, "Inciso"},
		{Alinea, "Alinea"},
		{Item, "Item"},
		{10, "Tipo(10)"},
		{0, "Tipo(0)"},
	}

	for _, tc := range testes {
		t.Run(tc.esperado, func(t *testing.T) {
			got := tc.tipo.String()

			if got != tc.esperado {
				t.Errorf("esperado: %q, resultado: %q", tc.esperado, got)
			}
		})
	}
}
