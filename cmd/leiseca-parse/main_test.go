package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const fixture = "../../internal/lei/testdata/cp_arts187a191.html"

func TestRunTexto(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{fixture}, &stdout, &stderr); code != 0 {
		t.Fatalf("código = %d; stderr: %s", code, stderr.String())
	}

	esperado := `Capítulo II: DOS CRIMES CONTRA O PRIVILÉGIO DE INVENÇÃO
  Artigo 187 «Violação de privilégio de invenção» [revogado]
  Artigo 188 «Falsa atribuição de privilégio» [revogado]
  Artigo 189 «Usurpação ou indevida exploração de modelo ou desenho privilegiado» [revogado]
  Artigo 190 «Falsa declaração de depósito em modelo ou desenho» [revogado]
  Artigo 191 [revogado]
`
	if got := stdout.String(); got != esperado {
		t.Errorf("saída =\n%s\nesperado:\n%s", got, esperado)
	}
}

func TestRunJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"-json", fixture}, &stdout, &stderr); code != 0 {
		t.Fatalf("código = %d; stderr: %s", code, stderr.String())
	}

	var raiz []struct {
		Tipo   string `json:"tipo"`
		Filhos []struct {
			Rotulo    string   `json:"rotulo"`
			Anotacoes []string `json:"anotacoes"`
		} `json:"filhos"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raiz); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, stdout.String())
	}
	if len(raiz) != 1 || raiz[0].Tipo != "Capítulo" || len(raiz[0].Filhos) != 5 {
		t.Fatalf("raiz = %+v; esperado 1 capítulo com 5 artigos", raiz)
	}
	if got := raiz[0].Filhos[0].Anotacoes; len(got) != 1 || got[0] != "(Revogado pela Lei nº 9.279, de 14.5.1996)" {
		t.Errorf("anotações do art. 187 = %q", got)
	}
}

func TestRunErros(t *testing.T) {
	testes := []struct {
		nome   string
		args   []string
		codigo int
		stderr string
	}{
		{"sem argumento", nil, 2, "uso:"},
		{"argumentos demais", []string{"a", "b"}, 2, "uso:"},
		{"flag desconhecida", []string{"-x", fixture}, 2, "-x"},
		{"arquivo inexistente", []string{"nao-existe.html"}, 1, "nao-existe.html"},
	}

	for _, tc := range testes {
		t.Run(tc.nome, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(context.Background(), tc.args, &stdout, &stderr); code != tc.codigo {
				t.Errorf("código = %d; esperado %d", code, tc.codigo)
			}
			if !strings.Contains(stderr.String(), tc.stderr) {
				t.Errorf("stderr = %q; esperado conter %q", stderr.String(), tc.stderr)
			}
		})
	}
}
