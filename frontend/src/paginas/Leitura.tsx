import { useEffect, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, type Dispositivo, type Lei } from '../api'
import { ehAgrupador, rotuloNaLei } from '../formatar'

function Agrupador({ d, nivel, notas }: { d: Dispositivo; nivel: number; notas: boolean }) {
  const Titulo = (`h${Math.min(nivel + 2, 6)}`) as 'h2'
  return (
    <section className={`agrupador nivel-${nivel}`}>
      <Titulo>
        <span className="agrupador-rotulo">{rotuloNaLei(d.tipo, d.rotulo)}</span>
        {d.texto && <span className="agrupador-nome">{d.texto}</span>}
        {d.vetado && <span className="marca-estado">vetado</span>}
      </Titulo>
      {notas && <Anotacoes d={d} />}
      <Nos ds={d.filhos} nivel={nivel + 1} notas={notas} />
    </section>
  )
}

function Normativo({ d, nivel, notas }: { d: Dispositivo; nivel: number; notas: boolean }) {
  const inativo = d.revogado || d.vetado
  return (
    <div
      className={`dispositivo tipo-${d.tipo === 'Artigo' ? 'artigo' : 'filho'}${inativo ? ' inativo' : ''}`}
      id={d.tipo === 'Artigo' ? `art-${d.rotulo}` : undefined}
    >
      {d.rubrica && <p className="rubrica">{d.rubrica}</p>}
      <p>
        <strong className="rotulo">{rotuloNaLei(d.tipo, d.rotulo)}</strong> {d.texto}
        {d.revogado && <span className="marca-estado">revogado</span>}
        {d.vetado && <span className="marca-estado">vetado</span>}
      </p>
      {notas && <Anotacoes d={d} />}
      <Nos ds={d.filhos} nivel={nivel} notas={notas} />
    </div>
  )
}

function Anotacoes({ d }: { d: Dispositivo }) {
  if (!d.anotacoes?.length) return null
  return <p className="anotacoes">{d.anotacoes.join(' ')}</p>
}

function Nos({ ds, nivel, notas }: { ds?: Dispositivo[]; nivel: number; notas: boolean }) {
  if (!ds?.length) return null
  return (
    <>
      {ds.map((d, i) =>
        ehAgrupador(d.tipo) ? (
          <Agrupador key={i} d={d} nivel={nivel} notas={notas} />
        ) : (
          <Normativo key={i} d={d} nivel={nivel} notas={notas} />
        ),
      )}
    </>
  )
}

export default function Leitura() {
  const { leiId } = useParams()
  const [dados, setDados] = useState<{ lei: Lei; arvore: Dispositivo[] } | null>(null)
  const [erro, setErro] = useState('')
  const [notas, setNotas] = useState(false)
  const [artigo, setArtigo] = useState('')
  const [naoAchou, setNaoAchou] = useState('')

  useEffect(() => {
    api.arvore(Number(leiId)).then(setDados, (e: Error) => setErro(e.message))
  }, [leiId])

  function irPara(e: FormEvent) {
    e.preventDefault()
    // Aceita "121", "art. 5º", "5-a": vira o rótulo normalizado do parser.
    const rotulo = artigo
      .replace(/^\s*art\.?\s*/i, '')
      .replace(/[º°.\s]/g, '')
      .toUpperCase()
    const alvo = document.getElementById(`art-${rotulo}`)
    if (alvo) {
      setNaoAchou('')
      alvo.scrollIntoView({ behavior: 'smooth', block: 'start' })
      alvo.classList.remove('realce')
      void alvo.offsetWidth // reinicia a animação
      alvo.classList.add('realce')
    } else {
      setNaoAchou(`Art. ${artigo} não encontrado.`)
    }
  }

  if (erro) return <p className="pagina alerta erro">{erro}</p>
  if (!dados) return <p className="pagina discreto">Carregando…</p>

  return (
    <div className="pagina leitura">
      <div className="cabecalho">
        <h1>
          <span className="sigla">{dados.lei.sigla}</span> {dados.lei.nome}
        </h1>
        <Link to={`/estudo/${dados.lei.id}`} className="botao primario">
          Estudar
        </Link>
      </div>

      <div className="barra-leitura">
        <form onSubmit={irPara}>
          <input
            value={artigo}
            onChange={(e) => setArtigo(e.target.value)}
            placeholder="Ir para o art. …"
            aria-label="Número do artigo"
          />
          <button className="botao">Ir</button>
        </form>
        <label className="alternar">
          <input type="checkbox" checked={notas} onChange={(e) => setNotas(e.target.checked)} />
          Mostrar anotações
        </label>
        {naoAchou && <span className="discreto">{naoAchou}</span>}
      </div>

      <div className="texto-lei">
        <Nos ds={dados.arvore} nivel={0} notas={notas} />
      </div>
      <p className="discreto rodape-fonte">
        Fonte:{' '}
        <a href={dados.lei.url} target="_blank" rel="noreferrer">
          {dados.lei.url}
        </a>
      </p>
    </div>
  )
}
