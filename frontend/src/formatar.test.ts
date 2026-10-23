import { describe, expect, it } from 'vitest'
import { formatarIntervalo, rotuloNaLei } from './formatar'

describe('formatarIntervalo', () => {
  it.each([
    [600, '10 min'],
    [3600, '1 h'],
    [86400, '1 d'],
    [6 * 86400, '6 d'],
    [38 * 86400, '1 mês'],
    [90 * 86400, '3 meses'],
    [365 * 86400, '1 ano'],
    [440 * 86400, '1,2 anos'],
  ])('%i s → %s', (segundos, esperado) => {
    expect(formatarIntervalo(segundos)).toBe(esperado)
  })
})

describe('rotuloNaLei', () => {
  it.each([
    ['Artigo', '5', 'Art. 5º'],
    ['Artigo', '5-A', 'Art. 5º-A'],
    ['Artigo', '121', 'Art. 121'],
    ['Artigo', '1000', 'Art. 1.000'],
    ['Parágrafo', '2', '§ 2º'],
    ['Parágrafo', 'unico', 'Parágrafo único.'],
    ['Inciso', 'IV', 'IV –'],
    ['Alínea', 'a', 'a)'],
    ['Parte', 'especial', 'Parte Especial'],
    ['Título', 'II', 'Título II'],
  ] as const)('%s %s → %s', (tipo, rotulo, esperado) => {
    expect(rotuloNaLei(tipo, rotulo)).toBe(esperado)
  })
})
