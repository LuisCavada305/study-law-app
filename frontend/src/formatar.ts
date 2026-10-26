import type { Tipo } from './api'

/** Intervalo curto para os botões: "10 min", "1 d", "3 meses", "1,2 ano". */
export function formatarIntervalo(segundos: number): string {
  const minutos = Math.round(segundos / 60)
  if (minutos < 60) return `${minutos} min`
  const horas = Math.round(minutos / 60)
  if (horas < 24) return `${horas} h`
  const dias = Math.round(horas / 24)
  if (dias < 30) return `${dias} d`
  const meses = Math.round(dias / 30)
  if (meses < 12) return meses === 1 ? '1 mês' : `${meses} meses`
  const anos = (dias / 365).toFixed(1).replace('.', ',').replace(',0', '')
  return `${anos} ano${anos === '1' ? '' : 's'}`
}

/** Põe "º" nos números de 1 a 9: "5" → "5º", "5-A" → "5º-A", "1000" → "1.000". */
function ordinal(rotulo: string): string {
  const [num, sufixo] = rotulo.split('-', 2)
  const n = Number(num)
  if (!Number.isInteger(n)) return rotulo
  let texto = num
  if (n < 10) texto += 'º'
  else if (n >= 1000) texto = n.toLocaleString('pt-BR')
  return sufixo ? `${texto}-${sufixo}` : texto
}

/**
 * Escreve o rótulo como ele aparece na lei. Igual a cards.Citar, no Go, mas
 * com inciso e alínea na forma do texto ("IV –", "a)"), não da citação.
 */
export function rotuloNaLei(tipo: Tipo, rotulo: string): string {
  switch (tipo) {
    case 'Artigo':
      return `Art. ${ordinal(rotulo)}`
    case 'Parágrafo':
      return rotulo === 'unico' ? 'Parágrafo único.' : `§ ${ordinal(rotulo)}`
    case 'Inciso':
      return `${rotulo} –`
    case 'Alínea':
      return `${rotulo})`
    case 'Item':
      return `${rotulo}.`
    case 'Parte':
      return `Parte ${rotulo.charAt(0).toUpperCase()}${rotulo.slice(1)}`
  }
  switch (rotulo) {
    case 'unico':
      return tipo === 'Seção' || tipo === 'Subseção' ? `${tipo} Única` : `${tipo} Único`
    case 'complementar':
      return `${tipo} Complementar`
    default:
      return `${tipo} ${rotulo}`
  }
}

/** Agrupadores organizam a lei e não têm texto normativo. */
export function ehAgrupador(tipo: Tipo): boolean {
  return ['Parte', 'Livro', 'Título', 'Subtítulo', 'Capítulo', 'Seção', 'Subseção'].includes(tipo)
}
