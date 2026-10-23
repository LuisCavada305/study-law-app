// Comando leiseca-parse baixa uma lei do Planalto e imprime a árvore de dispositivos.
//
// Uso:
//
//	leiseca-parse [-json] [-timeout 60s] <url ou arquivo>
//
// Com uma URL http(s), a página é baixada; qualquer outro argumento é lido
// como arquivo local (útil com as amostras de internal/lei/testdata).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/LuisCavada305/study-law-app/internal/lei"
	"github.com/LuisCavada305/study-law-app/internal/planalto"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// run executa o comando e devolve o código de saída: 0 em sucesso, 1 em
// erro e 2 em uso incorreto (a convenção do pacote flag).
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("leiseca-parse", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "uso: leiseca-parse [-json] [-timeout 60s] <url ou arquivo>")
		fs.PrintDefaults()
	}
	comoJSON := fs.Bool("json", false, "imprime a árvore em JSON, com anotações")
	timeout := fs.Duration("timeout", 60*time.Second, "tempo máximo para baixar a página")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	pagina, err := carregar(ctx, fs.Arg(0), *timeout)
	if err != nil {
		fmt.Fprintln(stderr, "leiseca-parse:", err)
		return 1
	}

	raiz, err := lei.Parse(bytes.NewReader(pagina))
	if err != nil {
		fmt.Fprintln(stderr, "leiseca-parse:", err)
		return 1
	}

	if *comoJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(raiz)
	} else {
		err = imprimir(stdout, raiz, 0)
	}
	if err != nil {
		fmt.Fprintln(stderr, "leiseca-parse:", err)
		return 1
	}
	return 0
}

// carregar baixa a URL ou lê o arquivo, e devolve o HTML em UTF-8.
func carregar(ctx context.Context, origem string, timeout time.Duration) ([]byte, error) {
	if strings.HasPrefix(origem, "http://") || strings.HasPrefix(origem, "https://") {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return planalto.Fetch(ctx, origem)
	}

	b, err := os.ReadFile(origem)
	if err != nil {
		return nil, err
	}
	return planalto.ParaUTF8(b)
}

// imprimir desenha a árvore indentada, um nó por linha: "Tipo Rotulo
// «Rubrica»: Texto [revogado]". As anotações só saem no -json.
func imprimir(w io.Writer, ds []*lei.Dispositivo, nivel int) error {
	for _, d := range ds {
		linha := strings.Repeat("  ", nivel) + d.Tipo.String() + " " + d.Rotulo
		if d.Rubrica != "" {
			linha += " «" + d.Rubrica + "»"
		}
		if d.Texto != "" {
			linha += ": " + d.Texto
		}
		if d.Revogado {
			linha += " [revogado]"
		}
		if d.Vetado {
			linha += " [vetado]"
		}
		if _, err := fmt.Fprintln(w, linha); err != nil {
			return err
		}
		if err := imprimir(w, d.Filhos, nivel+1); err != nil {
			return err
		}
	}
	return nil
}
