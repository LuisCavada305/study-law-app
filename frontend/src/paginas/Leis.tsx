import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api, type LeiComFila, type Resumo } from '../api'

// Leis testadas com o parser. Outras do Planalto podem funcionar pela URL.
const SUGESTOES = [
  {
    sigla: 'CF',
    nome: 'Constituição Federal de 1988',
    url: 'https://www.planalto.gov.br/ccivil_03/constituicao/constituicao.htm',
  },
  {
    sigla: 'CP',
    nome: 'Código Penal',
    url: 'https://www.planalto.gov.br/ccivil_03/decreto-lei/del2848compilado.htm',
  },
]

function descreverResumo(r: Resumo): string {
  const partes = [`${r.novos} novos`]
  if (r.atualizados) partes.push(`${r.atualizados} com texto alterado`)
  if (r.removidos) partes.push(`${r.removidos} removidos`)
  return partes.join(', ')
}

export default function Leis() {
  const [leis, setLeis] = useState<LeiComFila[] | null>(null)
  const [erro, setErro] = useState('')
  const [aviso, setAviso] = useState('')
  const [ocupado, setOcupado] = useState<string | null>(null)

  const [sigla, setSigla] = useState('')
  const [nome, setNome] = useState('')
  const [url, setUrl] = useState('')

  const carregar = useCallback(() => {
    api.leis().then(setLeis, (e: Error) => setErro(e.message))
  }, [])

  useEffect(carregar, [carregar])

  async function executar(chave: string, acao: () => Promise<string>) {
    setOcupado(chave)
    setErro('')
    setAviso('')
    try {
      setAviso(await acao())
      carregar()
    } catch (e) {
      setErro((e as Error).message)
    } finally {
      setOcupado(null)
    }
  }

  function importar(e: FormEvent) {
    e.preventDefault()
    executar('importar', async () => {
      const { lei, resumo } = await api.importar(sigla, nome, url)
      setSigla('')
      setNome('')
      setUrl('')
      return `${lei.sigla} importada: ${descreverResumo(resumo)}.`
    })
  }

  // Só depois de carregar: antes disso não dá para saber o que já existe.
  const disponiveis = leis ? SUGESTOES.filter((s) => !leis.some((l) => l.sigla === s.sigla)) : []
  const totalVencidos = leis?.reduce((n, l) => n + l.vencidos, 0) ?? 0
  const totalNovos = leis?.reduce((n, l) => n + l.novos, 0) ?? 0

  return (
    <div className="pagina">
      <div className="cabecalho">
        <h1>Minhas leis</h1>
        {leis && leis.length > 0 && (
          <Link to="/estudo" className="botao primario">
            Estudar tudo
          </Link>
        )}
      </div>

      {erro && <p className="alerta erro">{erro}</p>}
      {aviso && <p className="alerta ok">{aviso}</p>}

      {leis === null && !erro && <p className="discreto">Carregando…</p>}

      {leis && leis.length === 0 && (
        <p className="vazio">
          Nenhuma lei ainda. Importe a Constituição ou o Código Penal abaixo para começar.
        </p>
      )}

      {leis && leis.length > 0 && (
        <>
          <p className="discreto">
            {totalVencidos} {totalVencidos === 1 ? 'revisão pendente' : 'revisões pendentes'} ·{' '}
            {totalNovos} cards ainda não estudados
          </p>
          <ul className="lista-leis">
            {leis.map((l) => (
              <li key={l.id} className="lei">
                <div className="lei-info">
                  <span className="sigla">{l.sigla}</span>
                  <div>
                    <h2>{l.nome}</h2>
                    <p className="discreto">
                      {l.total} cards · {l.novos} novos ·{' '}
                      <strong className={l.vencidos ? 'destaque' : ''}>{l.vencidos} para revisar</strong>
                      <br />
                      Importada em {new Date(l.importadaEm).toLocaleDateString('pt-BR')}
                    </p>
                  </div>
                </div>
                <div className="lei-acoes">
                  <Link to={`/estudo/${l.id}`} className="botao primario">
                    Estudar
                  </Link>
                  <Link to={`/leis/${l.id}`} className="botao">
                    Ler
                  </Link>
                  <button
                    className="botao"
                    disabled={ocupado !== null}
                    onClick={() =>
                      executar(`atualizar-${l.id}`, async () => {
                        const { resumo } = await api.atualizar(l.id)
                        return `${l.sigla} atualizada: ${descreverResumo(resumo)}. O progresso foi mantido.`
                      })
                    }
                  >
                    {ocupado === `atualizar-${l.id}` ? 'Baixando…' : 'Atualizar'}
                  </button>
                  <button
                    className="botao perigo"
                    disabled={ocupado !== null}
                    onClick={() => {
                      if (confirm(`Remover ${l.nome}? O progresso de estudo dela será apagado.`)) {
                        executar(`remover-${l.id}`, async () => {
                          await api.remover(l.id)
                          return `${l.sigla} removida.`
                        })
                      }
                    }}
                  >
                    Remover
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}

      <section className="importar">
        <h2>Importar uma lei do Planalto</h2>
        {disponiveis.length > 0 && (
          <div className="sugestoes">
            {disponiveis.map((s) => (
              <button
                key={s.sigla}
                className="botao"
                disabled={ocupado !== null}
                onClick={() =>
                  executar('importar', async () => {
                    const { resumo } = await api.importar(s.sigla, s.nome, s.url)
                    return `${s.sigla} importada: ${descreverResumo(resumo)}.`
                  })
                }
              >
                + {s.nome}
              </button>
            ))}
          </div>
        )}
        <form onSubmit={importar} className="formulario">
          <label>
            Sigla
            <input value={sigla} onChange={(e) => setSigla(e.target.value)} placeholder="CPP" maxLength={20} required />
          </label>
          <label>
            Nome
            <input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Código de Processo Penal" required />
          </label>
          <label className="largo">
            Endereço no planalto.gov.br
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://www.planalto.gov.br/ccivil_03/decreto-lei/del3689compilado.htm"
              required
            />
          </label>
          <button className="botao primario" disabled={ocupado !== null}>
            {ocupado === 'importar' ? 'Baixando e importando…' : 'Importar'}
          </button>
        </form>
        <p className="discreto">
          Use a versão compilada da lei (com as alterações). Importar de novo uma sigla que já existe atualiza
          o texto e mantém o seu progresso.
        </p>
      </section>
    </div>
  )
}
