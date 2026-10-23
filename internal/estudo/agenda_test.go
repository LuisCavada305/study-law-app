package estudo

import (
	"testing"
	"time"
)

var agora = time.Date(2026, 10, 22, 21, 0, 0, 0, time.UTC)

// responder aplica as notas em sequência, cada uma no vencimento da anterior,
// e devolve os intervalos em dias depois de cada resposta.
func responder(notas ...Nota) ([]int, Estado) {
	var e Estado
	quando := agora
	var intervalos []int
	for _, n := range notas {
		e = Agendar(e, n, quando)
		intervalos = append(intervalos, e.Intervalo)
		quando = e.Vencimento
	}
	return intervalos, e
}

func TestAgendarSequencias(t *testing.T) {
	testes := []struct {
		nome       string
		notas      []Nota
		intervalos []int
	}{
		{"sempre Bom", []Nota{Bom, Bom, Bom, Bom}, []int{1, 6, 15, 38}},
		{"sempre Fácil", []Nota{Facil, Facil, Facil}, []int{4, 8, 29}},
		{"sempre Difícil", []Nota{Dificil, Dificil, Dificil, Dificil}, []int{1, 3, 4, 5}},
		// Depois do erro recomeça do 1, mas com facilidade menor (2,3).
		{"erro no meio", []Nota{Bom, Bom, Errei, Bom, Bom, Bom}, []int{1, 6, 0, 1, 6, 14}},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			got, _ := responder(tc.notas...)
			if len(got) != len(tc.intervalos) {
				t.Fatalf("intervalos = %v; esperado %v", got, tc.intervalos)
			}
			for i := range got {
				if got[i] != tc.intervalos[i] {
					t.Fatalf("intervalos = %v; esperado %v", got, tc.intervalos)
				}
			}
		})
	}
}

func TestAgendarErro(t *testing.T) {
	_, e := responder(Bom, Bom, Errei)

	if want := RevisaoDoErro; e.Vencimento.Sub(agora.AddDate(0, 0, 7)) != want {
		t.Errorf("vencimento = %v; esperado 10 min depois da resposta", e.Vencimento)
	}
	if e.Lapsos != 1 || e.Repeticoes != 0 {
		t.Errorf("Lapsos = %d, Repeticoes = %d; esperado 1 e 0", e.Lapsos, e.Repeticoes)
	}
	if e.Facilidade != FacilidadeInicial-0.2 {
		t.Errorf("Facilidade = %v; esperado %v", e.Facilidade, FacilidadeInicial-0.2)
	}

	// Errar um card novo não é lapso: ele ainda não tinha sido aprendido.
	if novo := Agendar(Estado{}, Errei, agora); novo.Lapsos != 0 {
		t.Errorf("Lapsos de card novo errado = %d; esperado 0", novo.Lapsos)
	}
}

func TestAgendarFacilidadeMinima(t *testing.T) {
	var e Estado
	for range 20 {
		e = Agendar(e, Errei, agora)
	}
	if e.Facilidade != FacilidadeMinima {
		t.Errorf("Facilidade = %v; esperado o mínimo %v", e.Facilidade, FacilidadeMinima)
	}
}

func TestPrevisoes(t *testing.T) {
	p := Previsoes(Estado{}, agora)
	esperado := map[Nota]time.Duration{
		Errei:   10 * time.Minute,
		Dificil: 24 * time.Hour,
		Bom:     24 * time.Hour,
		Facil:   4 * 24 * time.Hour,
	}
	for n, d := range esperado {
		if p[n] != d {
			t.Errorf("Previsoes[%v] = %v; esperado %v", n, p[n], d)
		}
	}
}

func TestNotaValida(t *testing.T) {
	for n, ok := range map[Nota]bool{0: false, Errei: true, Facil: true, 5: false} {
		if n.Valida() != ok {
			t.Errorf("Nota(%d).Valida() = %v; esperado %v", n, !ok, ok)
		}
	}
}
