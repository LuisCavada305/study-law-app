# Lei Seca

Estudo de lei seca com repetição espaçada. O app baixa a lei do Planalto,
monta a árvore de dispositivos e transforma cada artigo, parágrafo, inciso e
alínea em vigor num card: você vê a referência (e a rubrica e o texto do
dispositivo pai), recita o texto de memória, confere e diz como foi. O
algoritmo SM-2 decide quando o card volta.

## Rodando

Requer Go 1.26+ e Node 22+.

```bash
make run          # compila tudo e abre em http://127.0.0.1:8080
```

App desktop para Windows (janela própria, um `.exe` só, feito com Wails):

```bash
make windows      # gera bin/LeiSeca.exe; o banco fica em %AppData%\LeiSeca
```

O ícone vem de `build/icon.png` (o mesmo § do favicon); a configuração dos
recursos do Windows está em `build/winres.json`.

Em desenvolvimento, com recarga automática do frontend:

```bash
make dev-api      # terminal 1: API em :8080 (banco em ./leiseca.db)
make dev-web      # terminal 2: Vite em http://localhost:5173
make test         # testes do Go e do frontend
```

Na primeira vez, use os botões "+ Constituição Federal" e "+ Código Penal" na
tela inicial. Atalhos no estudo: **espaço** mostra o texto, **1–4** dão a nota
(Errei, Difícil, Bom, Fácil).

O servidor escuta só em `127.0.0.1` porque a API não tem login. Opções:
`leiseca-server -h`.

## Estrutura

| Pasta | O quê |
|---|---|
| `internal/lei` | Parser: HTML do Planalto → linhas (`html.go`) → árvore (`estrutura.go`) |
| `internal/planalto` | Download da página e conversão de windows-1252 para UTF-8 |
| `internal/cards` | Árvore → cards, com chave estável (`art121/par2/incI`) para reimportar sem perder progresso |
| `internal/estudo` | Agendamento SM-2 |
| `internal/banco` | SQLite (Go puro, sem cgo): leis, cards e histórico |
| `internal/api` | API JSON (`net/http`); as rotas estão no comentário do pacote |
| `cmd/leiseca-server` | Servidor da API e do build do frontend |
| `cmd/leiseca-desktop` | App desktop (Wails) com o frontend embutido e a API no mesmo processo |
| `cmd/leiseca-parse` | CLI do parser: `go run ./cmd/leiseca-parse [-json] <url ou arquivo>` |
| `frontend` | React + Vite + TypeScript |

As regras do parser, tiradas das páginas reais do Planalto, estão em
`PROXIMOS_PASSOS.md`.
