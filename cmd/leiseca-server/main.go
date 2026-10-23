// Comando leiseca-server serve a API de estudo e, se houver, o build do frontend.
//
// Uso:
//
//	leiseca-server [-addr 127.0.0.1:8080] [-banco leiseca.db] [-static frontend/dist]
//
// Em desenvolvimento, rode o Vite à parte (make dev-web): ele repassa /api
// para este servidor.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/api"
	"github.com/LuisCavada305/study-law-app/internal/banco"
)

func main() {
	// Por padrão só aceita conexões desta máquina: a API não tem login.
	addr := flag.String("addr", "127.0.0.1:8080", "endereço para escutar")
	caminhoBanco := flag.String("banco", "leiseca.db", "arquivo SQLite")
	static := flag.String("static", "frontend/dist", "pasta com o build do frontend (ignorada se não existir)")
	novos := flag.Int("novos", 20, "cards novos por dia")
	flag.Parse()

	b, err := banco.Abrir(*caminhoBanco)
	if err != nil {
		log.Fatalf("abrindo banco: %v", err)
	}
	defer b.Fechar()

	s := api.Novo(b)
	s.LimiteNovo = *novos

	mux := http.NewServeMux()
	mux.Handle("/api/", s.Rotas())
	if info, err := os.Stat(*static); err == nil && info.IsDir() {
		mux.Handle("/", spa(*static))
		log.Printf("servindo o frontend de %s", *static)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           registrar(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		desligar, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(desligar)
	}()

	log.Printf("escutando em http://%s", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// spa serve os arquivos do build e devolve o index.html para as demais
// rotas, que são do React Router (/estudo, /leis/3…).
func spa(dir string) http.Handler {
	arquivos := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(caminho); err == nil && !info.IsDir() {
			arquivos.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

// registrar escreve uma linha de log por requisição da API.
func registrar(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		h.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s (%s)", r.Method, r.URL.RequestURI(), time.Since(inicio).Round(time.Millisecond))
		}
	})
}
