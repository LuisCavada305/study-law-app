package banco

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/estudo"
)

// AntecedenciaAprendendo deixa estudar antes da hora um card errado há pouco
// (que volta em 10 minutos) quando não sobrou mais nada na fila, em vez de
// encerrar a sessão com ele pendente.
const AntecedenciaAprendendo = 20 * time.Minute

// CardEstudo é o card com o que a tela de estudo precisa.
type CardEstudo struct {
	ID         int64  `json:"id"`
	LeiID      int64  `json:"leiId"`
	Sigla      string `json:"sigla"`
	Tipo       string `json:"tipo"`
	Referencia string `json:"referencia"`
	Caminho    string `json:"caminho"`
	Rubrica    string `json:"rubrica,omitempty"`
	Contexto   string `json:"contexto,omitempty"`
	Texto      string `json:"texto"`
	Novo       bool   `json:"novo"`

	Estado estudo.Estado `json:"-"`
}

// Fila conta o que falta estudar hoje.
type Fila struct {
	Novos    int `json:"novos"`    // novos que ainda cabem no limite do dia
	Vencidos int `json:"vencidos"` // revisões vencidas
}

// Filtro escolhe de onde vêm os cards e quantos novos por dia.
type Filtro struct {
	LeiID      int64 // 0: todas as leis
	LimiteNovo int   // novos por dia
	InicioDia  time.Time
}

// ProximoCard escolhe o próximo card: primeiro as revisões vencidas (as mais
// atrasadas antes), depois os novos na ordem da lei, enquanto não passar do
// limite do dia, e por último os errados há pouco. Devolve nil quando acabou.
func (b *Banco) ProximoCard(ctx context.Context, f Filtro, agora time.Time) (*CardEstudo, Fila, error) {
	var fila Fila
	filtroLei, args := "", []any{}
	if f.LeiID != 0 {
		filtroLei, args = " AND lei_id = ?", []any{f.LeiID}
	}

	var novosHoje, novosRestantes int
	err := b.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE vencimento <= ?),
			COUNT(*) FILTER (WHERE estudado_em >= ?),
			COUNT(*) FILTER (WHERE vencimento IS NULL)
		FROM cards WHERE ativo = 1`+filtroLei,
		append([]any{agora.Unix(), f.InicioDia.Unix()}, args...)...,
	).Scan(&fila.Vencidos, &novosHoje, &novosRestantes)
	if err != nil {
		return nil, fila, err
	}
	fila.Novos = min(max(f.LimiteNovo-novosHoje, 0), novosRestantes)

	tentativas := []struct {
		onde  string
		args  []any
		ordem string
	}{
		{"vencimento <= ?", []any{agora.Unix()}, "vencimento"},
		{"vencimento IS NULL", nil, "lei_id, ordem"},
		{"vencimento <= ?", []any{agora.Add(AntecedenciaAprendendo).Unix()}, "vencimento"},
	}
	for i, t := range tentativas {
		if i == 1 && fila.Novos == 0 {
			continue
		}
		c, err := b.buscarCard(ctx, "ativo = 1 AND "+t.onde+filtroLei+" ORDER BY "+t.ordem, append(t.args, args...)...)
		if errors.Is(err, ErrNaoEncontrado) {
			continue
		}
		return c, fila, err
	}
	return nil, fila, nil
}

// Responder grava a nota do card e devolve o novo estado.
func (b *Banco) Responder(ctx context.Context, id int64, nota estudo.Nota, agora time.Time) (estudo.Estado, error) {
	if !nota.Valida() {
		return estudo.Estado{}, fmt.Errorf("nota inválida: %d", nota)
	}

	var novo estudo.Estado
	err := b.transacao(ctx, func(tx *sql.Tx) error {
		c, err := b.buscarCardTx(ctx, tx, "id = ?", id)
		if err != nil {
			return err
		}
		novo = estudo.Agendar(c.Estado, nota, agora)

		_, err = tx.ExecContext(ctx, `
			UPDATE cards SET repeticoes = ?, intervalo = ?, facilidade = ?, lapsos = ?,
				vencimento = ?, estudado_em = COALESCE(estudado_em, ?)
			WHERE id = ?`,
			novo.Repeticoes, novo.Intervalo, novo.Facilidade, novo.Lapsos,
			novo.Vencimento.Unix(), agora.Unix(), id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO revisoes (card_id, nota, em) VALUES (?, ?, ?)`,
			id, int(nota), agora.Unix())
		return err
	})
	return novo, err
}

// BuscarCard devolve um card pelo id.
func (b *Banco) BuscarCard(ctx context.Context, id int64) (*CardEstudo, error) {
	return b.buscarCard(ctx, "id = ?", id)
}

const colunasCard = `c.id, c.lei_id, l.sigla, c.tipo, c.referencia, c.caminho, c.rubrica, c.contexto,
	c.texto, c.repeticoes, c.intervalo, c.facilidade, c.lapsos, c.vencimento`

// consulta é *sql.DB ou *sql.Tx.
type consulta interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (b *Banco) buscarCard(ctx context.Context, onde string, args ...any) (*CardEstudo, error) {
	return b.buscarCardTx(ctx, b.db, onde, args...)
}

// buscarCardTx busca o primeiro card que satisfaz onde (uma condição SQL
// sobre a tabela cards, que pode terminar em ORDER BY).
func (b *Banco) buscarCardTx(ctx context.Context, q consulta, onde string, args ...any) (*CardEstudo, error) {
	var c CardEstudo
	var vencimento sql.NullInt64
	err := q.QueryRowContext(ctx,
		`SELECT `+colunasCard+` FROM cards c JOIN leis l ON l.id = c.lei_id
		WHERE c.id IN (SELECT id FROM cards WHERE `+onde+` LIMIT 1)`, args...,
	).Scan(&c.ID, &c.LeiID, &c.Sigla, &c.Tipo, &c.Referencia, &c.Caminho, &c.Rubrica, &c.Contexto,
		&c.Texto, &c.Estado.Repeticoes, &c.Estado.Intervalo, &c.Estado.Facilidade, &c.Estado.Lapsos, &vencimento)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if vencimento.Valid {
		c.Estado.Vencimento = time.Unix(vencimento.Int64, 0)
	}
	c.Novo = c.Estado.Novo()
	return &c, nil
}
