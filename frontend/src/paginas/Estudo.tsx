import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, type Estudo as EstudoAPI, type Nota } from '../api'
import { formatarIntervalo } from '../formatar'

const NOTAS: { nota: Nota; nome: string; classe: string }[] = [
  { nota: 1, nome: 'Errei', classe: 'nota-errei' },
  { nota: 2, nome: 'Difícil', classe: 'nota-dificil' },
  { nota: 3, nome: 'Bom', classe: 'nota-bom' },
  { nota: 4, nome: 'Fácil', classe: 'nota-facil' },
]

export default function Estudo() {
  const { leiId } = useParams()
  const id = leiId ? Number(leiId) : undefined

  const [estudo, setEstudo] = useState<EstudoAPI | null>(null)
  const [revelado, setRevelado] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState('')
  const [feitos, setFeitos] = useState(0)

  const carregar = useCallback(() => {
    return api.estudo(id).then(
      (e) => {
        setErro('')
        setEstudo(e)
        setRevelado(false)
      },
      (e: Error) => setErro(e.message),
    )
  }, [id])

  useEffect(() => {
    carregar()
  }, [carregar])

  const responder = useCallback(
    async (nota: Nota) => {
      const card = estudo?.card
      if (!card || enviando) return
      setEnviando(true)
      try {
        await api.responder(card.id, nota)
        setFeitos((n) => n + 1)
        await carregar()
      } catch (e) {
        setErro((e as Error).message)
      } finally {
        setEnviando(false)
      }
    },
    [estudo, enviando, carregar],
  )

  // Espaço ou Enter revelam; 1 a 4 respondem.
  useEffect(() => {
    function tecla(e: KeyboardEvent) {
      if (!estudo?.card || e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
      if (!revelado && (e.key === ' ' || e.key === 'Enter')) {
        e.preventDefault()
        setRevelado(true)
      } else if (revelado && ['1', '2', '3', '4'].includes(e.key)) {
        e.preventDefault()
        responder(Number(e.key) as Nota)
      }
    }
    window.addEventListener('keydown', tecla)
    return () => window.removeEventListener('keydown', tecla)
  }, [estudo, revelado, responder])

  if (erro) {
    return (
      <div className="pagina">
        <p className="alerta erro">{erro}</p>
        <button className="botao" onClick={carregar}>
          Tentar de novo
        </button>
      </div>
    )
  }
  if (!estudo) return <p className="pagina discreto">Carregando…</p>

  const { card, fila, previsoes } = estudo

  if (!card) {
    return (
      <div className="pagina fim-estudo">
        <p className="fim-simbolo">✓</p>
        <h1>Tudo em dia</h1>
        <p className="discreto">
          {feitos > 0 ? `${feitos} ${feitos === 1 ? 'card estudado' : 'cards estudados'} nesta sessão. ` : ''}
          Não há revisões pendentes nem cards novos para hoje{id ? ' nesta lei' : ''}. Volte amanhã.
        </p>
        <Link to="/" className="botao">
          Voltar às leis
        </Link>
      </div>
    )
  }

  return (
    <div className="pagina estudo">
      <div className="placar">
        <span>
          <strong>{fila.vencidos}</strong> para revisar
        </span>
        <span>
          <strong>{fila.novos}</strong> novos hoje
        </span>
        {feitos > 0 && (
          <span>
            <strong>{feitos}</strong> feitos
          </span>
        )}
      </div>

      <article className="card">
        <p className="card-caminho">
          <span className="sigla pequena">{card.sigla}</span> {card.caminho}
        </p>
        <h1 className="card-referencia">
          {card.referencia}
          {card.novo && <span className="selo">novo</span>}
        </h1>
        {card.rubrica && <p className="card-rubrica">{card.rubrica}</p>}
        {card.contexto && <blockquote className="card-contexto">{card.contexto}</blockquote>}

        {revelado ? (
          <p className="card-texto">{card.texto}</p>
        ) : (
          <div className="card-oculto">
            <p className="discreto">Recite o texto de memória e depois confira.</p>
            <button className="botao primario grande" onClick={() => setRevelado(true)}>
              Mostrar texto <kbd>espaço</kbd>
            </button>
          </div>
        )}
      </article>

      {revelado && (
        <div className="notas">
          {NOTAS.map(({ nota, nome, classe }) => (
            <button key={nota} className={`botao nota ${classe}`} disabled={enviando} onClick={() => responder(nota)}>
              <span>{nome}</span>
              <small>
                {previsoes?.[nota] !== undefined && formatarIntervalo(previsoes[nota])} <kbd>{nota}</kbd>
              </small>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
