// Package planalto busca páginas de leis em planalto.gov.br e as entrega em UTF-8.
package planalto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// TamanhoMaximo limita o corpo da resposta. A CF, a maior lei usada até
// aqui, tem menos de 2 MB.
const TamanhoMaximo = 20 << 20

// Cliente é o cliente HTTP usado por Fetch. Ele não tem Timeout próprio:
// o prazo vem do context (veja PrazoPadrao).
var Cliente = &http.Client{}

// PrazoPadrao é o tempo máximo de Fetch, download do corpo incluído,
// quando o context não tem prazo.
const PrazoPadrao = 60 * time.Second

// UserAgent identifica o programa. O firewall do Planalto derruba a conexão
// de User-Agents que não começam com "Mozilla/5.0" (inclusive o do curl),
// então ele segue o formato "compatible" que os robôs de busca usam.
const UserAgent = "Mozilla/5.0 (compatible; leiseca-parse/0.1)"

// Fetch baixa a página em url e devolve o HTML em UTF-8.
// Resposta com status diferente de 200 é erro. Se ctx não tiver prazo,
// vale PrazoPadrao.
func Fetch(ctx context.Context, url string) ([]byte, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, PrazoPadrao)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := Cliente.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, TamanhoMaximo+1))
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	if len(corpo) > TamanhoMaximo {
		return nil, fmt.Errorf("GET %s: página maior que %d bytes", url, TamanhoMaximo)
	}
	return ParaUTF8(corpo)
}

// ParaUTF8 converte o HTML do Planalto para UTF-8. As páginas não declaram
// charset (nem no HTML nem no cabeçalho HTTP) e vêm em windows-1252, mas
// se o conteúdo já for UTF-8 válido ele volta como está: um texto em
// windows-1252 com acentos quase nunca é UTF-8 válido por acaso.
func ParaUTF8(b []byte) ([]byte, error) {
	if utf8.Valid(b) {
		return b, nil
	}
	return charmap.Windows1252.NewDecoder().Bytes(b)
}
