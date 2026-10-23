package planalto

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParaUTF8(t *testing.T) {
	testes := []struct {
		nome     string
		entrada  []byte
		esperado string
	}{
		{"ascii", []byte("Art. 1"), "Art. 1"},
		// "º" = 0xBA, "–" = 0x96 e "’" = 0x92 em windows-1252.
		{"windows-1252", []byte("Art. 5\xba I \x96 o Estado\x92s"), "Art. 5º I – o Estado’s"},
		{"já em utf-8", []byte("Art. 5º Todos são iguais"), "Art. 5º Todos são iguais"},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			got, err := ParaUTF8(tc.entrada)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.esperado {
				t.Errorf("ParaUTF8(%q) = %q; esperado %q", tc.entrada, got, tc.esperado)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cp.htm":
			w.Write([]byte("<p>Art. 121. Matar alguem: Pena \x96 reclus\xe3o</p>"))
		case "/grande.htm":
			w.Write(make([]byte, TamanhoMaximo+1))
		case "/lento.htm":
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Run("ok, convertido para UTF-8", func(t *testing.T) {
		got, err := Fetch(context.Background(), srv.URL+"/cp.htm")
		if err != nil {
			t.Fatal(err)
		}
		if esperado := "<p>Art. 121. Matar alguem: Pena – reclusão</p>"; string(got) != esperado {
			t.Errorf("Fetch() = %q; esperado %q", got, esperado)
		}
	})

	t.Run("status 404", func(t *testing.T) {
		_, err := Fetch(context.Background(), srv.URL+"/nao-existe.htm")
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Errorf("err = %v; esperado erro com 404", err)
		}
	})

	t.Run("página grande demais", func(t *testing.T) {
		_, err := Fetch(context.Background(), srv.URL+"/grande.htm")
		if err == nil || !strings.Contains(err.Error(), "maior que") {
			t.Errorf("err = %v; esperado erro de tamanho", err)
		}
	})

	t.Run("context cancelado", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := Fetch(ctx, srv.URL+"/lento.htm")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("err = %v; esperado context.DeadlineExceeded", err)
		}
	})

	t.Run("url inválida", func(t *testing.T) {
		if _, err := Fetch(context.Background(), "://sem-esquema"); err == nil {
			t.Error("esperado erro para url inválida")
		}
	})
}
