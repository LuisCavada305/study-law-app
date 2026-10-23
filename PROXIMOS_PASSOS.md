# Próximos passos

Última sessão: 22/10/2026.

## Já feito
- Esqueleto do módulo (`cmd/leiseca-parse`, `internal/lei`, `internal/planalto`)
- `Tipo` com `iota` + `String()`, struct `Dispositivo`, testes table-driven
- Repo privado: https://github.com/LuisCavada305/study-law-app
- Amostras brutas em `internal/lei/testdata/` (`cf88.html`, `cp.html`, baixadas em 20/10/2026, **ainda sem commit**)

## Marco 1 concluído em 22/10/2026 — parser

```bash
go run ./cmd/leiseca-parse https://www.planalto.gov.br/ccivil_03/decreto-lei/del2848compilado.htm
go run ./cmd/leiseca-parse -json internal/lei/testdata/cf88.html > cf88.json
```

- `lei.Parse(io.Reader)` = Fase 1 (`html.go`, `extrairLinhas`) + Fase 2 (`montarArvore`)
- `planalto.Fetch(ctx, url)`: prazo do context (60s se não houver), status != 200 é
  erro, limite de 20 MB, converte windows-1252 → UTF-8 (`ParaUTF8`)
- **User-Agent:** o firewall do Planalto derruba conexões sem "Mozilla/5.0" (até o
  curl). Usamos `Mozilla/5.0 (compatible; leiseca-parse/0.1)`
- CLI: texto indentado ou `-json` (com anotações); aceita URL ou arquivo local
- Testes de integração rodam `Parse` na CF e no CP inteiros (`parse_test.go`)
- Teste com o site real (22/10/2026): CF com 2749 nós até o art. 250; CP com 1455
  nós até o art. 361

## Marco 2 concluído em 22/10/2026 — app de estudo

Decisões (o Luis deixou a meu critério):
- **Card** = dispositivo normativo em vigor com texto (artigo, parágrafo, inciso,
  alínea). Frente: referência ("Art. 121, § 2º, inciso I"), caminho dos agrupadores,
  rubrica (herdada do ancestral mais próximo) e texto do pai como contexto. Verso: texto.
  Método: recitar de memória e conferir
- **SM-2** (`internal/estudo`): Errei volta em 10 min; Bom dá 1 → 6 → ×2,5 dias;
  20 novos por dia (`-novos`). Card errado há pouco aparece antes da hora se a fila acabou
- **SQLite** com `modernc.org/sqlite` (sem cgo). Datas em segundos Unix. Árvore guardada
  em JSON na tabela `leis`, para a tela de leitura
- **Reimportação** pela chave (`art121/par2/incI`): progresso mantido, texto atualizado,
  cards que sumiram ficam inativos. Rótulo repetido na lei ganha `#2`
- **API** só com `net/http`; importação restrita a planalto.gov.br; servidor em 127.0.0.1
- **Frontend** em `frontend/` (como no ROTA): telas Leis, Estudo e Leitura
- Ajuste no parser: linha "Pena - …" volta ao artigo/parágrafo (antes ia para o último inciso)

Testado com o site real: CF = 2613 cards, CP = 1310 cards.

## Ao retomar

Ideias, em ordem de valor:
1. **Modo lacuna**: esconder palavras-chave do texto ("Matar alguem: Pena - ___, de
   ___ a ___ anos") em vez de recitar tudo
2. **Mais leis**: testar CPP, CC, CPC, CLT pelo formulário; cada uma deve trazer
   padrões novos (fazer fixtures como as da CF/CP)
3. **Estatísticas**: a tabela `revisoes` já tem o histórico (acertos por lei, dias seguidos)
4. **Busca** na leitura e "estudar este artigo agora" a partir dela
5. **Deploy** (VPS ou Docker, como no ROTA): precisa de login antes de abrir para a rede

## Fase 1 — regras (tiradas das amostras)
- Negrito da linha = primeiro trecho com texto (89 rubricas do CP vêm seguidas de
  anotação sem negrito; negrito no meio da frase, como *caput*, é comum)
- Trechos emendados sem espaço entre tags (`<b>A</b>rt. 107`), espaços normalizados
  no fim da linha; `<sup>` colado no texto anterior ("§ 1o", 4× na CF)
- `<br>` quebra linha mesmo riscado; riscado = `<strike>`, `<s>`, `<del>` ou
  `line-through` no CSS; `font-weight` no style vale mais que `<b>`
- `html.Parse` corrige tags mal aninhadas como o navegador e converte `&#150;` em "–"

## Fase 2 — feito em 22/10/2026
- Sufixo de letra: `§ 1º-A`, `§ 2o-B`, `Art. 5º-A`, `II-A -` → rótulos `1-A`, `2-B`, `5-A`, `II-A`
- Inciso sem separador: texto em minúscula (`XI nos tribunais…`, 49× na CF) ou
  anotação (`III (Revogado…)`, separador riscado)
- Anotações saem do `Texto` para `Anotacoes` (Redação, Incluído/a, Acrescido, Revogado/a,
  Vide, Vigência, Regulamento, Produção de efeito, (Parágrafo) Renumerado, Transformado).
  Parênteses da norma ficam: `(vinte)`, `(Lei Maria da Penha)`
- `Revogado` = sem texto depois de tirar anotações + anotação de revogação.
  `Art. 88 - Revogado o livramento…` continua texto normal
- Rubricas do CP (negrito, sem rótulo) vão para `Dispositivo.Rubrica` do próximo
  dispositivo; antes de agrupador, são descartadas. Nome de agrupador também é
  negrito: a 1ª linha após o rótulo é sempre nome; as seguintes só se estiverem em
  maiúsculas. Na fumaça: 351 rubricas no CP, 0 na CF
- `(VETADO)` é anotação; sem texto restante → `Dispositivo.Vetado` (não gera card)
- Leitura para na data de assinatura (`Brasília, 5 de outubro de 1988.`) ou no cabeçalho do ADCT
- Teste de fumaça nas amostras inteiras (Fase 1 provisória em Python): CF para no art. 250,
  CP no art. 361. Única anotação que escapa: `((Incluído pela Lei nº 9.677…)` (parêntese
  duplicado no original, CP art. 273 § 1º-B IV). Casos isolados conhecidos:
  "DISPOSIÇÕES FINAIS" (título sem número) vira rubrica do art. 360 do CP; no
  Capítulo V das fraudes em certames (CP), a rubrica não está em negrito e é
  colada ao nome do capítulo


## Achados das amostras (para lembrar)
- Charset não declarado nem no HTML nem no header HTTP; há bytes 0x92 (`’`) → windows-1252
- CF: 777 `<strike>` (todos fechados) + 7 `line-through` em CSS
- CP: rótulos `Art. 121.`, `Art. 81 -`, `Art 188.`, `Art\n187.`, `Art. 121-A.`
- Anotações vêm dentro de `<a href>` para outras leis
- Espaçamento com muitos `&nbsp;` — normalizar antes de reconhecer rótulos
