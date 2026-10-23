// Package estudo decide quando cada card volta, pelo algoritmo SM-2 (o mesmo
// princípio do Anki): quanto melhor a lembrança, maior o intervalo.
package estudo

import (
	"fmt"
	"math"
	"time"
)

// Nota é a avaliação da lembrança depois de ver a resposta.
type Nota int

const (
	Errei Nota = iota + 1
	Dificil
	Bom
	Facil
)

// Valida informa se n é uma das quatro notas.
func (n Nota) Valida() bool {
	return Errei <= n && n <= Facil
}

const (
	// FacilidadeInicial multiplica o intervalo a cada acerto "Bom".
	FacilidadeInicial = 2.5
	// FacilidadeMinima impede que um card difícil volte todo dia para sempre.
	FacilidadeMinima = 1.3
	// RevisaoDoErro é quando um card errado volta, ainda na mesma sessão.
	RevisaoDoErro = 10 * time.Minute
)

// Estado é o progresso de um card. O valor zero é um card novo.
type Estado struct {
	// Repeticoes conta os acertos seguidos; volta a zero no erro.
	Repeticoes int
	// Intervalo é o último intervalo em dias (0 depois de um erro).
	Intervalo int
	// Facilidade multiplica o intervalo; 0 significa FacilidadeInicial.
	Facilidade float64
	// Lapsos conta quantas vezes o card foi esquecido depois de aprendido.
	Lapsos int
	// Vencimento é quando o card deve ser revisto; zero para card novo.
	Vencimento time.Time
}

// Novo informa se o card nunca foi estudado.
func (e Estado) Novo() bool {
	return e.Vencimento.IsZero()
}

// Agendar devolve o estado depois de responder o card com a nota n em agora.
//
// Errei: o card volta em 10 minutos, recomeça a contagem e fica mais difícil.
// Nos acertos, os dois primeiros intervalos são fixos (1 e 6 dias com "Bom")
// e depois o intervalo é multiplicado pela facilidade; "Difícil" cresce
// devagar e reduz a facilidade, "Fácil" cresce mais e aumenta.
func Agendar(e Estado, n Nota, agora time.Time) Estado {
	if e.Facilidade == 0 {
		e.Facilidade = FacilidadeInicial
	}

	if n == Errei {
		if e.Repeticoes > 0 {
			e.Lapsos++
		}
		e.Repeticoes = 0
		e.Intervalo = 0
		e.Facilidade = math.Max(FacilidadeMinima, e.Facilidade-0.2)
		e.Vencimento = agora.Add(RevisaoDoErro)
		return e
	}

	var dias float64
	switch e.Repeticoes {
	case 0:
		dias = map[Nota]float64{Dificil: 1, Bom: 1, Facil: 4}[n]
	case 1:
		dias = map[Nota]float64{Dificil: 3, Bom: 6, Facil: 8}[n]
	default:
		anterior := float64(e.Intervalo)
		switch n {
		case Dificil:
			dias = anterior * 1.2
		case Bom:
			dias = anterior * e.Facilidade
		case Facil:
			dias = anterior * e.Facilidade * 1.3
		}
		// Acertar nunca encurta o intervalo.
		dias = math.Max(dias, anterior+1)
	}

	switch n {
	case Dificil:
		e.Facilidade = math.Max(FacilidadeMinima, e.Facilidade-0.15)
	case Facil:
		e.Facilidade += 0.15
	}
	e.Repeticoes++
	e.Intervalo = int(math.Round(dias))
	e.Vencimento = agora.AddDate(0, 0, e.Intervalo)
	return e
}

// Previsoes devolve, para cada nota, daqui a quanto tempo o card voltaria.
// A tela mostra isso nos botões ("Bom · 6 d").
func Previsoes(e Estado, agora time.Time) map[Nota]time.Duration {
	p := make(map[Nota]time.Duration, 4)
	for n := Errei; n <= Facil; n++ {
		p[n] = Agendar(e, n, agora).Vencimento.Sub(agora)
	}
	return p
}

// String devolve o nome da nota, para mensagens de erro e logs.
func (n Nota) String() string {
	switch n {
	case Errei:
		return "Errei"
	case Dificil:
		return "Difícil"
	case Bom:
		return "Bom"
	case Facil:
		return "Fácil"
	default:
		return fmt.Sprintf("Nota(%d)", int(n))
	}
}
