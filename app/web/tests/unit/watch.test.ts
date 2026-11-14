import { describe, expect, it } from 'vitest'
import type { WatchItem } from '../../src/lib/types'
import { cardLabel, clock, displayName, episodeTag, opensPlayer, playLabel, remaining, resumeAt, runtime } from '../../src/lib/watch'

function item(over: Partial<WatchItem> = {}): WatchItem {
  return {
    id: 'm1', type: 'Movie', name: 'O Grande Teste', overview: '', year: 2024, genres: [], rating: '', score: 0,
    runtime: 7200, position: 0, played: false, favorite: false, unplayed: 0, seriesId: '', seriesName: '', seasonId: '',
    season: 0, episode: 0, poster: '', backdrop: '', logo: '', thumb: '', progress: 0, ...over,
  }
}

const ep = item({ id: 'e2', type: 'Episode', name: 'O Segundo', seriesId: 's1', seriesName: 'Série de Teste', season: 1, episode: 2, runtime: 1500 })

describe('filmes e séries', () => {
  it('etiqueta de episódio', () => {
    expect(episodeTag(ep)).toBe('T1:E2')
    expect(episodeTag(item())).toBe('')
    expect(episodeTag({ type: 'Episode', season: 0, episode: 3 })).toBe('T1:E3')
  })

  it('duração e relógio', () => {
    expect(runtime(0)).toBe('')
    expect(runtime(45 * 60)).toBe('45 min')
    expect(runtime(7200)).toBe('2 h')
    expect(runtime(6720)).toBe('1 h 52 min')
    expect(clock(0)).toBe('0:00')
    expect(clock(125)).toBe('2:05')
    expect(clock(3723)).toBe('1:02:03')
    expect(clock(Number.NaN)).toBe('0:00')
  })

  it('quanto falta', () => {
    expect(remaining(item({ position: 1800 }))).toBe('faltam 1 h 30 min')
    expect(remaining(item())).toBe('')
    expect(remaining(item({ position: 7200 }))).toBe('')
  })

  it('episódio mostra a série; o nome acessível tem tudo', () => {
    expect(displayName(ep)).toBe('Série de Teste')
    expect(cardLabel(ep)).toBe('Série de Teste, T1:E2 O Segundo')
    expect(cardLabel(item({ progress: 0.25, favorite: true }))).toBe('O Grande Teste, 2024, 25% assistido, favorito')
    expect(cardLabel(item({ type: 'Series', played: true }))).toBe('O Grande Teste, 2024, série, assistido')
  })

  it('o que abre o player direto', () => {
    expect(opensPlayer(ep)).toBe(true)
    expect(opensPlayer(item({ position: 10 }))).toBe(true)
    expect(opensPlayer(item({ position: 10, played: true }))).toBe(false)
    expect(opensPlayer(item())).toBe(false)
    expect(opensPlayer(item({ type: 'Series' }))).toBe(false)
  })

  it('texto do botão principal', () => {
    expect(playLabel(item(), null)).toBe('Assistir')
    expect(playLabel(item({ position: 60 }), null)).toBe('Continuar')
    expect(playLabel(item({ type: 'Series' }), ep)).toBe('Assistir T1:E2')
    expect(playLabel(item({ type: 'Series' }), { ...ep, position: 30 })).toBe('Continuar T1:E2')
    expect(playLabel(item({ type: 'Series' }), null)).toBe('Assistir')
  })

  it('perto do fim recomeça', () => {
    expect(resumeAt(0, 100)).toBe(0)
    expect(resumeAt(50, 100)).toBe(50)
    expect(resumeAt(90, 100)).toBe(0)
    expect(resumeAt(50, 0)).toBe(50)
  })
})
