package banco

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/cards"
	"github.com/LuisCavada305/study-law-app/internal/lei"
)

// Lei é uma lei importada.
type Lei struct {
	ID          int64     `json:"id"`
	Sigla       string    `json:"sigla"`
	Nome        string    `json:"nome"`
	URL         string    `json:"url"`
	ImportadaEm time.Time `json:"importadaEm"`
}

// LeiComFila é a lei com a contagem de cards para a tela inicial.
type LeiComFila struct {
	Lei
	Total    int `json:"total"`    // cards ativos
	Novos    int `json:"novos"`    // nunca estudados
	Vencidos int `json:"vencidos"` // para revisar agora
}

// Resumo conta o que mudou numa importação.
type Resumo struct {
	Novos       int `json:"novos"`
	Atualizados int `json:"atualizados"` // texto mudou; o progresso é mantido
	Removidos   int `json:"removidos"`   // sumiram da lei; ficam inativos
}

// SalvarLei grava a lei e os seus cards. Se a sigla já existe, atualiza:
// cards com a mesma chave mantêm o progresso, os novos entram e os que
// sumiram ficam inativos (sem apagar o histórico).
func (b *Banco) SalvarLei(ctx context.Context, l Lei, arvore []*lei.Dispositivo, cs []cards.Card) (Lei, Resumo, error) {
	var resumo Resumo
	arvoreJSON, err := json.Marshal(arvore)
	if err != nil {
		return l, resumo, err
	}

	err = b.transacao(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			INSERT INTO leis (sigla, nome, url, arvore, importada_em) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (sigla) DO UPDATE SET
				nome = excluded.nome, url = excluded.url,
				arvore = excluded.arvore, importada_em = excluded.importada_em
			RETURNING id`,
			l.Sigla, l.Nome, l.URL, string(arvoreJSON), l.ImportadaEm.Unix(),
		).Scan(&l.ID)
		if err != nil {
			return err
		}

		// Texto atual de cada chave ativa, para contar o que mudou.
		antes := map[string]string{}
		linhas, err := tx.QueryContext(ctx, `SELECT chave, texto FROM cards WHERE lei_id = ? AND ativo = 1`, l.ID)
		if err != nil {
			return err
		}
		for linhas.Next() {
			var chave, texto string
			if err := linhas.Scan(&chave, &texto); err != nil {
				linhas.Close()
				return err
			}
			antes[chave] = texto
		}
		if err := linhas.Close(); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `UPDATE cards SET ativo = 0 WHERE lei_id = ?`, l.ID); err != nil {
			return err
		}

		inserir, err := tx.PrepareContext(ctx, `
			INSERT INTO cards (lei_id, chave, ordem, tipo, rotulo, referencia, caminho, rubrica, contexto, texto)
			VALUES (`+placeholders(10)+`)
			ON CONFLICT (lei_id, chave) DO UPDATE SET
				ordem = excluded.ordem, tipo = excluded.tipo, rotulo = excluded.rotulo,
				referencia = excluded.referencia, caminho = excluded.caminho,
				rubrica = excluded.rubrica, contexto = excluded.contexto,
				texto = excluded.texto, ativo = 1`)
		if err != nil {
			return err
		}
		defer inserir.Close()

		for _, c := range cs {
			_, err := inserir.ExecContext(ctx, l.ID, c.Chave, c.Ordem, c.Tipo.String(), c.Rotulo,
				c.Referencia, c.Caminho, c.Rubrica, c.Contexto, c.Texto)
			if err != nil {
				return err
			}
			texto, existia := antes[c.Chave]
			switch {
			case !existia:
				resumo.Novos++
			case texto != c.Texto:
				resumo.Atualizados++
			}
			delete(antes, c.Chave)
		}
		resumo.Removidos = len(antes)
		return nil
	})
	return l, resumo, err
}

// Leis lista as leis com a fila de cada uma em agora.
func (b *Banco) Leis(ctx context.Context, agora time.Time) ([]LeiComFila, error) {
	linhas, err := b.db.QueryContext(ctx, `
		SELECT l.id, l.sigla, l.nome, l.url, l.importada_em,
			COUNT(c.id),
			COUNT(c.id) FILTER (WHERE c.vencimento IS NULL),
			COUNT(c.id) FILTER (WHERE c.vencimento <= ?)
		FROM leis l
		LEFT JOIN cards c ON c.lei_id = l.id AND c.ativo = 1
		GROUP BY l.id
		ORDER BY l.sigla`, agora.Unix())
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	leis := []LeiComFila{}
	for linhas.Next() {
		var l LeiComFila
		var importada int64
		if err := linhas.Scan(&l.ID, &l.Sigla, &l.Nome, &l.URL, &importada, &l.Total, &l.Novos, &l.Vencidos); err != nil {
			return nil, err
		}
		l.ImportadaEm = time.Unix(importada, 0)
		leis = append(leis, l)
	}
	return leis, linhas.Err()
}

// BuscarLei devolve a lei e a sua árvore em JSON, como foi gravada.
func (b *Banco) BuscarLei(ctx context.Context, id int64) (Lei, json.RawMessage, error) {
	var l Lei
	var importada int64
	var arvore string
	err := b.db.QueryRowContext(ctx,
		`SELECT id, sigla, nome, url, importada_em, arvore FROM leis WHERE id = ?`, id,
	).Scan(&l.ID, &l.Sigla, &l.Nome, &l.URL, &importada, &arvore)
	if errors.Is(err, sql.ErrNoRows) {
		return l, nil, ErrNaoEncontrado
	}
	if err != nil {
		return l, nil, err
	}
	l.ImportadaEm = time.Unix(importada, 0)
	return l, json.RawMessage(arvore), nil
}

// RemoverLei apaga a lei, os seus cards e o histórico deles.
func (b *Banco) RemoverLei(ctx context.Context, id int64) error {
	res, err := b.db.ExecContext(ctx, `DELETE FROM leis WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNaoEncontrado
	}
	return nil
}
