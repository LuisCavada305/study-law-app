// Comando leiseca-desktop é o app de estudo numa janela própria (Wails), com o
// frontend embutido: um executável só, sem servidor nem navegador.
//
// Uso:
//
//	leiseca-desktop [-banco caminho] [-novos 20]
//
// Compile com as tags do Wails (make windows); sem elas o Wails só avisa e sai.
package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/LuisCavada305/study-law-app/frontend"
	"github.com/LuisCavada305/study-law-app/internal/api"
	"github.com/LuisCavada305/study-law-app/internal/banco"
)

func main() {
	caminhoBanco := flag.String("banco", bancoPadrao(), "arquivo SQLite")
	novos := flag.Int("novos", 20, "cards novos por dia")
	flag.Parse()

	b, err := banco.Abrir(*caminhoBanco)
	if err != nil {
		log.Fatalf("abrindo banco: %v", err)
	}
	defer b.Fechar()

	s := api.Novo(b)
	s.LimiteNovo = *novos

	dist, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		log.Fatalf("frontend sem build (rode npm run build): %v", err)
	}

	// Os arquivos do build saem de Assets; o resto cai aqui: a API e as rotas
	// do React Router (/estudo, /leis/3…), que recebem o index.html.
	mux := http.NewServeMux()
	mux.Handle("/api/", s.Rotas())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
	})

	err = wails.Run(&options.App{
		Title:     "Lei Seca",
		Width:     1100,
		Height:    800,
		MinWidth:  480,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets:  dist,
			Handler: mux,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// bancoPadrao guarda o banco na pasta de configuração do usuário
// (%AppData%\LeiSeca no Windows), para o app abrir de qualquer lugar.
func bancoPadrao() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "leiseca.db"
	}
	dir = filepath.Join(dir, "LeiSeca")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "leiseca.db"
	}
	return filepath.Join(dir, "leiseca.db")
}

