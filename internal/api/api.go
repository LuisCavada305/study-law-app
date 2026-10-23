// Package api expõe as leis e o estudo em JSON, para o frontend.
//
//	GET    /api/leis                  leis com a fila de cada uma
//	POST   /api/leis                  importa (ou reimporta) {sigla, nome, url}
//	POST   /api/leis/{id}/atualizar   baixa de novo da URL gravada
//	DELETE /api/leis/{id}             remove a lei, os cards e o histórico
//	GET    /api/leis/{id}/arvore      árvore completa, para a leitura
//	GET    /api/estudo?lei={id}       próximo card (lei opcional) e a fila
//	POST   /api/cards/{id}/resposta   grava a nota {nota: 1..4}
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/banco"
	"github.com/LuisCavada305/study-law-app/internal/cards"
	"github.com/LuisCavada305/study-law-app/internal/estudo"
	"github.com/LuisCavada305/study-law-app/internal/lei"
	"github.com/LuisCavada305/study-law-app/internal/planalto"
)

// Servidor guarda as dependências dos handlers. Os campos de função existem
// para os testes trocarem a rede e o relógio.
type Servidor struct {
	Banco *banco.Banco

	// Baixar busca a página da lei; normalmente planalto.Fetch.
	Baixar func(ctx context.Context, url string) ([]byte, error)
	// Agora é o relógio; normalmente time.Now.
	Agora func() time.Time
	// LimiteNovo é quantos cards novos entram por dia.
	LimiteNovo int
	// HostsPermitidos são os hosts de onde se pode importar. O parser só
	// entende o HTML do Planalto, e a restrição evita que a API seja usada
	// para fazer requisições a outros endereços.
	HostsPermitidos []string
}

// Novo devolve o servidor com os padrões de produção.
func Novo(b *banco.Banco) *Servidor {
	return &Servidor{
		Banco:           b,
		Baixar:          planalto.Fetch,
		Agora:           time.Now,
		LimiteNovo:      20,
		HostsPermitidos: []string{"www.planalto.gov.br", "planalto.gov.br"},
	}
}

// Rotas devolve o handler com todas as rotas da API.
func (s *Servidor) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/leis", s.listarLeis)
	mux.HandleFunc("POST /api/leis", s.importarLei)
	mux.HandleFunc("POST /api/leis/{id}/atualizar", s.atualizarLei)
	mux.HandleFunc("DELETE /api/leis/{id}", s.removerLei)
	mux.HandleFunc("GET /api/leis/{id}/arvore", s.arvoreLei)
	mux.HandleFunc("GET /api/estudo", s.proximoCard)
	mux.HandleFunc("POST /api/cards/{id}/resposta", s.responder)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		escreverErro(w, http.StatusNotFound, "rota não encontrada")
	})
	return mux
}

func (s *Servidor) listarLeis(w http.ResponseWriter, r *http.Request) {
	leis, err := s.Banco.Leis(r.Context(), s.Agora())
	if err != nil {
		erroInterno(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, leis)
}

// pedidoImportacao é o corpo de POST /api/leis.
type pedidoImportacao struct {
	Sigla string `json:"sigla"`
	Nome  string `json:"nome"`
	URL   string `json:"url"`
}

func (s *Servidor) importarLei(w http.ResponseWriter, r *http.Request) {
	var p pedidoImportacao
	if !lerJSON(w, r, &p) {
		return
	}
	p.Sigla = strings.ToUpper(strings.TrimSpace(p.Sigla))
	p.Nome = strings.TrimSpace(p.Nome)
	p.URL = strings.TrimSpace(p.URL)

	switch {
	case p.Sigla == "" || len(p.Sigla) > 20:
		escreverErro(w, http.StatusBadRequest, "sigla é obrigatória (até 20 caracteres)")
		return
	case p.Nome == "":
		escreverErro(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	if err := s.validarURL(p.URL); err != nil {
		escreverErro(w, http.StatusBadRequest, err.Error())
		return
	}
	s.importar(w, r, banco.Lei{Sigla: p.Sigla, Nome: p.Nome, URL: p.URL})
}

func (s *Servidor) atualizarLei(w http.ResponseWriter, r *http.Request) {
	id, ok := lerID(w, r)
	if !ok {
		return
	}
	l, _, err := s.Banco.BuscarLei(r.Context(), id)
	if err != nil {
		erroDoBanco(w, err)
		return
	}
	s.importar(w, r, l)
}

// importar baixa, interpreta e grava a lei.
func (s *Servidor) importar(w http.ResponseWriter, r *http.Request, l banco.Lei) {
	pagina, err := s.Baixar(r.Context(), l.URL)
	if err != nil {
		escreverErro(w, http.StatusBadGateway, "não foi possível baixar a lei: "+err.Error())
		return
	}
	arvore, err := lei.Parse(bytes.NewReader(pagina))
	if err != nil {
		erroInterno(w, err)
		return
	}
	cs := cards.Gerar(arvore)
	if len(cs) == 0 {
		escreverErro(w, http.StatusUnprocessableEntity, "nenhum dispositivo encontrado: a página é uma lei do Planalto?")
		return
	}

	l.ImportadaEm = s.Agora()
	l, resumo, err := s.Banco.SalvarLei(r.Context(), l, arvore, cs)
	if err != nil {
		erroInterno(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, map[string]any{"lei": l, "resumo": resumo})
}

func (s *Servidor) validarURL(bruta string) error {
	u, err := url.Parse(bruta)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("url inválida: use o endereço completo da lei, com https://")
	}
	if !slices.Contains(s.HostsPermitidos, u.Host) {
		return fmt.Errorf("só é possível importar leis de %s", strings.Join(s.HostsPermitidos, " ou "))
	}
	return nil
}

func (s *Servidor) removerLei(w http.ResponseWriter, r *http.Request) {
	id, ok := lerID(w, r)
	if !ok {
		return
	}
	if err := s.Banco.RemoverLei(r.Context(), id); err != nil {
		erroDoBanco(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Servidor) arvoreLei(w http.ResponseWriter, r *http.Request) {
	id, ok := lerID(w, r)
	if !ok {
		return
	}
	l, arvore, err := s.Banco.BuscarLei(r.Context(), id)
	if err != nil {
		erroDoBanco(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, map[string]any{"lei": l, "arvore": arvore})
}

// respostaEstudo é o corpo de GET /api/estudo. Card é null quando acabou.
type respostaEstudo struct {
	Card *banco.CardEstudo `json:"card"`
	Fila banco.Fila        `json:"fila"`
	// Previsoes diz, em segundos, quando o card volta com cada nota ("1" a
	// "4"), para os botões mostrarem "Bom · 6 d".
	Previsoes map[string]int64 `json:"previsoes,omitempty"`
}

func (s *Servidor) proximoCard(w http.ResponseWriter, r *http.Request) {
	var leiID int64
	if v := r.URL.Query().Get("lei"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			escreverErro(w, http.StatusBadRequest, "lei inválida")
			return
		}
		leiID = id
	}

	agora := s.Agora()
	ano, mes, dia := agora.Date()
	filtro := banco.Filtro{
		LeiID:      leiID,
		LimiteNovo: s.LimiteNovo,
		InicioDia:  time.Date(ano, mes, dia, 0, 0, 0, 0, agora.Location()),
	}
	c, fila, err := s.Banco.ProximoCard(r.Context(), filtro, agora)
	if err != nil {
		erroInterno(w, err)
		return
	}

	resp := respostaEstudo{Card: c, Fila: fila}
	if c != nil {
		resp.Previsoes = map[string]int64{}
		for n, d := range estudo.Previsoes(c.Estado, agora) {
			resp.Previsoes[strconv.Itoa(int(n))] = int64(d / time.Second)
		}
	}
	escreverJSON(w, http.StatusOK, resp)
}

func (s *Servidor) responder(w http.ResponseWriter, r *http.Request) {
	id, ok := lerID(w, r)
	if !ok {
		return
	}
	var p struct {
		Nota estudo.Nota `json:"nota"`
	}
	if !lerJSON(w, r, &p) {
		return
	}
	if !p.Nota.Valida() {
		escreverErro(w, http.StatusBadRequest, "nota deve ser de 1 (Errei) a 4 (Fácil)")
		return
	}

	e, err := s.Banco.Responder(r.Context(), id, p.Nota, s.Agora())
	if err != nil {
		erroDoBanco(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, map[string]any{"vencimento": e.Vencimento, "intervalo": e.Intervalo})
}

// --- auxiliares ---

func lerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		escreverErro(w, http.StatusBadRequest, "id inválido")
		return 0, false
	}
	return id, true
}

// lerJSON decodifica o corpo em v; em erro, já responde 400.
func lerJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		escreverErro(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return false
	}
	return true
}

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("escrevendo resposta: %v", err)
	}
}

func escreverErro(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, map[string]string{"erro": msg})
}

func erroDoBanco(w http.ResponseWriter, err error) {
	if errors.Is(err, banco.ErrNaoEncontrado) {
		escreverErro(w, http.StatusNotFound, "não encontrado")
		return
	}
	erroInterno(w, err)
}

// erroInterno registra o erro no log e responde sem expor detalhes.
func erroInterno(w http.ResponseWriter, err error) {
	log.Printf("erro interno: %v", err)
	escreverErro(w, http.StatusInternalServerError, "erro interno")
}
