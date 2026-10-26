// Cliente da API em Go (internal/api). Os tipos espelham o JSON de lá.

export type Tipo =
  | 'Parte'
  | 'Livro'
  | 'Título'
  | 'Subtítulo'
  | 'Capítulo'
  | 'Seção'
  | 'Subseção'
  | 'Artigo'
  | 'Parágrafo'
  | 'Inciso'
  | 'Alínea'
  | 'Item'

export interface Dispositivo {
  tipo: Tipo
  rotulo: string
  texto?: string
  rubrica?: string
  revogado?: boolean
  vetado?: boolean
  anotacoes?: string[]
  filhos?: Dispositivo[]
}

export interface Lei {
  id: number
  sigla: string
  nome: string
  url: string
  importadaEm: string
}

export interface LeiComFila extends Lei {
  total: number
  novos: number
  vencidos: number
}

export interface Resumo {
  novos: number
  atualizados: number
  removidos: number
}

export interface Card {
  id: number
  leiId: number
  sigla: string
  tipo: Tipo
  referencia: string
  caminho: string
  rubrica?: string
  contexto?: string
  texto: string
  novo: boolean
}

export interface Fila {
  novos: number
  vencidos: number
}

/** Nota da lembrança: 1 Errei, 2 Difícil, 3 Bom, 4 Fácil. */
export type Nota = 1 | 2 | 3 | 4

export interface Estudo {
  card: Card | null
  fila: Fila
  /** Segundos até o card voltar, por nota ("1" a "4"). */
  previsoes?: Record<string, number>
}

async function pedir<T>(metodo: string, caminho: string, corpo?: unknown): Promise<T> {
  const resp = await fetch(caminho, {
    method: metodo,
    headers: corpo === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: corpo === undefined ? undefined : JSON.stringify(corpo),
  })
  if (resp.status === 204) {
    return undefined as T
  }
  const dados = await resp.json().catch(() => null)
  if (!resp.ok) {
    throw new Error(dados?.erro ?? `erro ${resp.status}`)
  }
  return dados as T
}

export const api = {
  leis: () => pedir<LeiComFila[]>('GET', '/api/leis'),
  importar: (sigla: string, nome: string, url: string) =>
    pedir<{ lei: Lei; resumo: Resumo }>('POST', '/api/leis', { sigla, nome, url }),
  atualizar: (id: number) => pedir<{ lei: Lei; resumo: Resumo }>('POST', `/api/leis/${id}/atualizar`),
  remover: (id: number) => pedir<void>('DELETE', `/api/leis/${id}`),
  arvore: (id: number) => pedir<{ lei: Lei; arvore: Dispositivo[] }>('GET', `/api/leis/${id}/arvore`),
  estudo: (leiId?: number) => pedir<Estudo>('GET', leiId ? `/api/estudo?lei=${leiId}` : '/api/estudo'),
  responder: (cardId: number, nota: Nota) =>
    pedir<{ vencimento: string; intervalo: number }>('POST', `/api/cards/${cardId}/resposta`, { nota }),
}
