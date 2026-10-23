package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/banco"
	"github.com/LuisCavada305/study-law-app/internal/planalto"
)

const urlCP = "https://planalto.test/cp.htm"

// novoServidor sobe a API com banco em memória, o relógio parado em agora
// e a "rede" servindo a fixture do art. 121 do CP.
func novoServidor(t *testing.T, agora *time.Time) *httptest.Server {
	t.Helper()
	b, err := banco.Abrir(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Fechar() })

	bruto, err := os.ReadFile("../lei/testdata/cp_art121.html")
	if err != nil {
		t.Fatal(err)
	}
	pagina, err := planalto.ParaUTF8(bruto)
	if err != nil {
		t.Fatal(err)
	}

	s := Novo(b)
	s.Agora = func() time.Time { return *agora }
	s.HostsPermitidos = []string{"planalto.test"}
	s.Baixar = func(ctx context.Context, url string) ([]byte, error) {
		switch url {
		case urlCP:
			return pagina, nil
		case "https://planalto.test/vazia.htm":
			return []byte("<p>Página sem lei</p>"), nil
		}
		return nil, errors.New("404 Not Found")
	}

	srv := httptest.NewServer(s.Rotas())
	t.Cleanup(srv.Close)
	return srv
}

// chamar faz a requisição e decodifica a resposta JSON em v (se não for nil).
func chamar(t *testing.T, srv *httptest.Server, metodo, caminho, corpo string, v any) int {
	t.Helper()
	req, err := http.NewRequest(metodo, srv.URL+caminho, strings.NewReader(corpo))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if v != nil {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		if err := json.Unmarshal(buf.Bytes(), v); err != nil {
			t.Fatalf("%s %s: resposta não é JSON: %v\n%s", metodo, caminho, err, buf.String())
		}
	}
	return resp.StatusCode
}

type respostaImportacao struct {
	Lei    banco.Lei    `json:"lei"`
	Resumo banco.Resumo `json:"resumo"`
	Erro   string       `json:"erro"`
}

func TestFluxoCompleto(t *testing.T) {
	agora := time.Date(2026, 10, 22, 21, 0, 0, 0, time.UTC)
	srv := novoServidor(t, &agora)

	// Importa.
	var imp respostaImportacao
	if st := chamar(t, srv, "POST", "/api/leis", `{"sigla":"cp","nome":"Código Penal","url":"`+urlCP+`"}`, &imp); st != 200 {
		t.Fatalf("importar: status %d, %+v", st, imp)
	}
	if imp.Lei.Sigla != "CP" || imp.Resumo.Novos == 0 {
		t.Fatalf("importar = %+v", imp)
	}

	// Lista.
	var leis []banco.LeiComFila
	chamar(t, srv, "GET", "/api/leis", "", &leis)
	if len(leis) != 1 || leis[0].Total != imp.Resumo.Novos || leis[0].Novos != leis[0].Total {
		t.Fatalf("leis = %+v", leis)
	}

	// Primeiro card: o caput do art. 121, com rubrica e previsões.
	var est respostaEstudo
	chamar(t, srv, "GET", "/api/estudo", "", &est)
	if est.Card == nil || est.Card.Referencia != "Art. 121" || est.Card.Rubrica != "Homicídio simples" {
		t.Fatalf("estudo = %+v", est.Card)
	}
	if est.Previsoes["3"] != 86400 || est.Previsoes["1"] != 600 {
		t.Errorf("previsoes = %v; esperado Bom = 1 dia e Errei = 10 min", est.Previsoes)
	}

	// Responde Bom: o card vai para amanhã e o próximo é outro.
	var resp map[string]any
	if st := chamar(t, srv, "POST", "/api/cards/"+itoa(est.Card.ID)+"/resposta", `{"nota":3}`, &resp); st != 200 {
		t.Fatalf("responder: status %d, %v", st, resp)
	}
	if resp["intervalo"] != float64(1) {
		t.Errorf("intervalo = %v; esperado 1", resp["intervalo"])
	}
	var est2 respostaEstudo
	chamar(t, srv, "GET", "/api/estudo?lei="+itoa(imp.Lei.ID), "", &est2)
	if est2.Card == nil || est2.Card.ID == est.Card.ID {
		t.Errorf("segundo card = %+v; esperado outro card", est2.Card)
	}

	// Árvore para leitura.
	var arv struct {
		Lei    banco.Lei         `json:"lei"`
		Arvore []json.RawMessage `json:"arvore"`
	}
	if st := chamar(t, srv, "GET", "/api/leis/"+itoa(imp.Lei.ID)+"/arvore", "", &arv); st != 200 || len(arv.Arvore) == 0 {
		t.Errorf("arvore: status %d, %d nós", st, len(arv.Arvore))
	}

	// Reimporta pela URL gravada: nada muda e o progresso fica.
	var reimp respostaImportacao
	chamar(t, srv, "POST", "/api/leis/"+itoa(imp.Lei.ID)+"/atualizar", "", &reimp)
	if reimp.Resumo != (banco.Resumo{}) {
		t.Errorf("reimportação = %+v; esperado sem mudanças", reimp.Resumo)
	}

	// Remove.
	if st := chamar(t, srv, "DELETE", "/api/leis/"+itoa(imp.Lei.ID), "", nil); st != 204 {
		t.Errorf("remover: status %d", st)
	}
	chamar(t, srv, "GET", "/api/leis", "", &leis)
	if len(leis) != 0 {
		t.Errorf("leis depois de remover = %+v", leis)
	}
}

func TestErros(t *testing.T) {
	agora := time.Date(2026, 10, 22, 21, 0, 0, 0, time.UTC)
	srv := novoServidor(t, &agora)

	testes := []struct {
		nome    string
		metodo  string
		caminho string
		corpo   string
		status  int
		erro    string
	}{
		{"sem sigla", "POST", "/api/leis", `{"nome":"X","url":"` + urlCP + `"}`, 400, "sigla"},
		{"sem nome", "POST", "/api/leis", `{"sigla":"X","url":"` + urlCP + `"}`, 400, "nome"},
		{"url relativa", "POST", "/api/leis", `{"sigla":"X","nome":"X","url":"cp.htm"}`, 400, "url inválida"},
		{"host não permitido", "POST", "/api/leis", `{"sigla":"X","nome":"X","url":"http://localhost:6379/"}`, 400, "planalto.test"},
		{"campo desconhecido", "POST", "/api/leis", `{"sigla":"X","nome":"X","url":"` + urlCP + `","extra":1}`, 400, "JSON inválido"},
		{"falha ao baixar", "POST", "/api/leis", `{"sigla":"X","nome":"X","url":"https://planalto.test/nao.htm"}`, 502, "404"},
		{"página sem lei", "POST", "/api/leis", `{"sigla":"X","nome":"X","url":"https://planalto.test/vazia.htm"}`, 422, "nenhum dispositivo"},
		{"lei inexistente", "GET", "/api/leis/99/arvore", "", 404, "não encontrado"},
		{"id inválido", "DELETE", "/api/leis/abc", "", 400, "id inválido"},
		{"card inexistente", "POST", "/api/cards/99/resposta", `{"nota":3}`, 404, "não encontrado"},
		{"nota inválida", "POST", "/api/cards/1/resposta", `{"nota":7}`, 400, "nota"},
		{"filtro de lei inválido", "GET", "/api/estudo?lei=x", "", 400, "lei inválida"},
		{"rota inexistente", "GET", "/api/nada", "", 404, "rota"},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			var resp struct {
				Erro string `json:"erro"`
			}
			st := chamar(t, srv, tc.metodo, tc.caminho, tc.corpo, &resp)
			if st != tc.status || !strings.Contains(resp.Erro, tc.erro) {
				t.Errorf("status %d, erro %q; esperado %d com %q", st, resp.Erro, tc.status, tc.erro)
			}
		})
	}
}

// Sem cards para estudar, card vem null (e não um erro).
func TestEstudoVazio(t *testing.T) {
	agora := time.Date(2026, 10, 22, 21, 0, 0, 0, time.UTC)
	srv := novoServidor(t, &agora)

	var corpo map[string]any
	if st := chamar(t, srv, "GET", "/api/estudo", "", &corpo); st != 200 {
		t.Fatalf("status %d", st)
	}
	if card, ok := corpo["card"]; !ok || card != nil {
		t.Errorf("card = %v; esperado null", corpo["card"])
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
