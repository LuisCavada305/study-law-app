// Package banco guarda as leis importadas e o progresso dos cards em SQLite.
package banco

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // driver "sqlite", em Go puro (sem cgo)
)

// ErrNaoEncontrado é devolvido quando a lei ou o card pedido não existe.
var ErrNaoEncontrado = errors.New("não encontrado")

// Banco é a conexão com o arquivo SQLite.
type Banco struct {
	db *sql.DB
}

// Abrir abre (ou cria) o banco em caminho e garante o esquema.
// Use ":memory:" para um banco temporário, nos testes.
func Abrir(caminho string) (*Banco, error) {
	dsn := "file:" + caminho + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	if caminho != ":memory:" {
		dsn += "&_pragma=journal_mode(WAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// Uma conexão só: o SQLite aceita um escritor por vez, e com ":memory:"
	// cada conexão nova seria um banco vazio diferente.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, fmt.Errorf("criando esquema: %w", err)
	}
	return &Banco{db: db}, nil
}

// Fechar fecha a conexão.
func (b *Banco) Fechar() error {
	return b.db.Close()
}

// esquema é idempotente: roda a cada Abrir.
//
// Datas são segundos Unix (INTEGER), para comparar e ordenar direto no SQL.
// O estado do SM-2 fica no próprio card (ver estudo.Estado); vencimento NULL
// é card novo.
const esquema = `
CREATE TABLE IF NOT EXISTS leis (
	id           INTEGER PRIMARY KEY,
	sigla        TEXT NOT NULL UNIQUE,
	nome         TEXT NOT NULL,
	url          TEXT NOT NULL,
	arvore       TEXT NOT NULL,  -- JSON da árvore, para a tela de leitura
	importada_em INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS cards (
	id          INTEGER PRIMARY KEY,
	lei_id      INTEGER NOT NULL REFERENCES leis(id) ON DELETE CASCADE,
	chave       TEXT NOT NULL,
	ordem       INTEGER NOT NULL,
	tipo        TEXT NOT NULL,
	rotulo      TEXT NOT NULL,
	referencia  TEXT NOT NULL,
	caminho     TEXT NOT NULL,
	rubrica     TEXT NOT NULL,
	contexto    TEXT NOT NULL,
	texto       TEXT NOT NULL,
	ativo       INTEGER NOT NULL DEFAULT 1, -- 0: sumiu na última importação
	repeticoes  INTEGER NOT NULL DEFAULT 0,
	intervalo   INTEGER NOT NULL DEFAULT 0,
	facilidade  REAL    NOT NULL DEFAULT 0,
	lapsos      INTEGER NOT NULL DEFAULT 0,
	vencimento  INTEGER,                    -- NULL: card novo
	estudado_em INTEGER,                    -- primeira resposta: conta os novos do dia
	UNIQUE (lei_id, chave)
);

CREATE INDEX IF NOT EXISTS cards_fila ON cards (ativo, vencimento);

CREATE TABLE IF NOT EXISTS revisoes (
	id      INTEGER PRIMARY KEY,
	card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
	nota    INTEGER NOT NULL,
	em      INTEGER NOT NULL
);
`

// transacao roda f numa transação: commit se f não devolver erro,
// rollback se devolver.
func (b *Banco) transacao(ctx context.Context, f func(*sql.Tx) error) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// placeholders devolve "?, ?, ?" com n interrogações.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
