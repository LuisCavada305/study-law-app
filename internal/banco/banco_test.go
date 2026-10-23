package banco

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/cards"
	"github.com/LuisCavada305/study-law-app/internal/estudo"
	"github.com/LuisCavada305/study-law-app/internal/lei"
)

var (
	ctx   = context.Background()
	agora = time.Date(2026, 10, 22, 21, 0, 0, 0, time.UTC)
	hoje  = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
)

func abrir(t *testing.T) *Banco {
	t.Helper()
	b, err := Abrir(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Fechar() })
	return b
}

// artigos monta uma lei só com artigos, um por texto.
func artigos(textos ...string) ([]*lei.Dispositivo, []cards.Card) {
	var raiz []*lei.Dispositivo
	for i, texto := range textos {
		raiz = append(raiz, &lei.Dispositivo{Tipo: lei.Artigo, Rotulo: string(rune('1' + i)), Texto: texto})
	}
	return raiz, cards.Gerar(raiz)
}

func salvar(t *testing.T, b *Banco, sigla string, textos ...string) (Lei, Resumo) {
	t.Helper()
	raiz, cs := artigos(textos...)
	l, r, err := b.SalvarLei(ctx, Lei{Sigla: sigla, Nome: "Lei " + sigla, URL: "http://x/" + sigla, ImportadaEm: agora}, raiz, cs)
	if err != nil {
		t.Fatal(err)
	}
	return l, r
}

func TestSalvarLeiReimportacao(t *testing.T) {
	b := abrir(t)

	l, r := salvar(t, b, "CP", "um", "dois", "três")
	if r != (Resumo{Novos: 3}) {
		t.Errorf("primeira importação: %+v", r)
	}

	// Estuda o art. 1º, para ver o progresso sobreviver à reimportação.
	c, _, err := b.ProximoCard(ctx, Filtro{LimiteNovo: 20, InicioDia: hoje}, agora)
	if err != nil || c == nil || c.Referencia != "Art. 1º" {
		t.Fatalf("ProximoCard() = %+v, %v", c, err)
	}
	if _, err := b.Responder(ctx, c.ID, estudo.Bom, agora); err != nil {
		t.Fatal(err)
	}

	// Art. 2º muda de texto, art. 3º some e entra um art. 4º.
	raiz := []*lei.Dispositivo{
		{Tipo: lei.Artigo, Rotulo: "1", Texto: "um"},
		{Tipo: lei.Artigo, Rotulo: "2", Texto: "dois, alterado"},
		{Tipo: lei.Artigo, Rotulo: "4", Texto: "quatro"},
	}
	l2, r, err := b.SalvarLei(ctx, Lei{Sigla: "CP", Nome: "Código Penal", URL: "http://x/CP", ImportadaEm: agora}, raiz, cards.Gerar(raiz))
	if err != nil {
		t.Fatal(err)
	}
	if l2.ID != l.ID {
		t.Errorf("reimportação criou outra lei: id %d → %d", l.ID, l2.ID)
	}
	if r != (Resumo{Novos: 1, Atualizados: 1, Removidos: 1}) {
		t.Errorf("reimportação: %+v", r)
	}

	art1, err := b.BuscarCard(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if art1.Novo || art1.Estado.Repeticoes != 1 {
		t.Errorf("art. 1º perdeu o progresso: %+v", art1.Estado)
	}

	leis, err := b.Leis(ctx, agora)
	if err != nil {
		t.Fatal(err)
	}
	if len(leis) != 1 || leis[0].Nome != "Código Penal" || leis[0].Total != 3 || leis[0].Novos != 2 {
		t.Errorf("Leis() = %+v; esperado Código Penal com 3 cards, 2 novos", leis)
	}
}

func TestProximoCardOrdemELimite(t *testing.T) {
	b := abrir(t)
	salvar(t, b, "CF", "a", "b", "c")
	filtro := Filtro{LimiteNovo: 2, InicioDia: hoje}

	// Os novos vêm na ordem da lei, até o limite do dia.
	var vistos []string
	for {
		c, _, err := b.ProximoCard(ctx, filtro, agora)
		if err != nil {
			t.Fatal(err)
		}
		if c == nil {
			break
		}
		vistos = append(vistos, c.Referencia)
		if _, err := b.Responder(ctx, c.ID, estudo.Bom, agora); err != nil {
			t.Fatal(err)
		}
	}
	if len(vistos) != 2 || vistos[0] != "Art. 1º" || vistos[1] != "Art. 2º" {
		t.Errorf("estudados = %v; esperado só Art. 1º e Art. 2º (limite 2)", vistos)
	}

	// No dia seguinte: os dois voltam vencidos antes do novo que sobrou.
	amanha := agora.Add(24 * time.Hour)
	c, fila, err := b.ProximoCard(ctx, Filtro{LimiteNovo: 2, InicioDia: hoje.Add(24 * time.Hour)}, amanha)
	if err != nil {
		t.Fatal(err)
	}
	if fila != (Fila{Novos: 1, Vencidos: 2}) {
		t.Errorf("fila = %+v; esperado 1 novo e 2 vencidos", fila)
	}
	if c == nil || c.Novo {
		t.Errorf("primeiro card = %+v; esperado uma revisão", c)
	}
}

func TestProximoCardErroVoltaNaSessao(t *testing.T) {
	b := abrir(t)
	salvar(t, b, "CF", "a")
	filtro := Filtro{LimiteNovo: 20, InicioDia: hoje}

	c, _, _ := b.ProximoCard(ctx, filtro, agora)
	if _, err := b.Responder(ctx, c.ID, estudo.Errei, agora); err != nil {
		t.Fatal(err)
	}

	// Vence em 10 min, mas como não sobrou nada, já pode ser estudado.
	deNovo, fila, err := b.ProximoCard(ctx, filtro, agora.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if deNovo == nil || deNovo.ID != c.ID {
		t.Errorf("ProximoCard() = %+v; esperado o card errado de novo", deNovo)
	}
	if fila != (Fila{}) {
		t.Errorf("fila = %+v; esperado vazia (ele ainda não venceu)", fila)
	}
}

func TestProximoCardFiltroPorLei(t *testing.T) {
	b := abrir(t)
	salvar(t, b, "CF", "da CF")
	cp, _ := salvar(t, b, "CP", "do CP")

	c, _, err := b.ProximoCard(ctx, Filtro{LeiID: cp.ID, LimiteNovo: 20, InicioDia: hoje}, agora)
	if err != nil || c == nil || c.Sigla != "CP" || c.Texto != "do CP" {
		t.Errorf("ProximoCard(CP) = %+v, %v", c, err)
	}
}

func TestResponderErros(t *testing.T) {
	b := abrir(t)
	if _, err := b.Responder(ctx, 999, estudo.Bom, agora); !errors.Is(err, ErrNaoEncontrado) {
		t.Errorf("card inexistente: err = %v", err)
	}
	if _, err := b.Responder(ctx, 1, estudo.Nota(9), agora); err == nil {
		t.Error("nota inválida: esperado erro")
	}
}

func TestBuscarERemoverLei(t *testing.T) {
	b := abrir(t)
	l, _ := salvar(t, b, "CP", "texto")

	got, arvore, err := b.BuscarLei(ctx, l.ID)
	if err != nil || got.Sigla != "CP" || len(arvore) == 0 {
		t.Fatalf("BuscarLei() = %+v, %s, %v", got, arvore, err)
	}

	if err := b.RemoverLei(ctx, l.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.BuscarLei(ctx, l.ID); !errors.Is(err, ErrNaoEncontrado) {
		t.Errorf("depois de remover: err = %v", err)
	}
	if err := b.RemoverLei(ctx, l.ID); !errors.Is(err, ErrNaoEncontrado) {
		t.Errorf("remover de novo: err = %v", err)
	}
	// Os cards vão junto (ON DELETE CASCADE).
	if c, _, _ := b.ProximoCard(ctx, Filtro{LimiteNovo: 20, InicioDia: hoje}, agora); c != nil {
		t.Errorf("sobrou card da lei removida: %+v", c)
	}
}
