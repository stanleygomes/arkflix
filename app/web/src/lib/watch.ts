// Filmes e séries: textos e regras puras (testáveis) da parte do Jellyfin.
import type { WatchItem } from './types'

/** "T1:E2" pra episódio; vazio pro resto. */
export function episodeTag(it: Pick<WatchItem, 'type' | 'season' | 'episode'>): string {
  if (it.type !== 'Episode' || !it.episode) return ''
  return `T${it.season || 1}:E${it.episode}`
}

/** "1 h 52 min", "45 min", "" (sem duração). */
export function runtime(seconds: number): string {
  const min = Math.round(seconds / 60)
  if (min <= 0) return ''
  const h = Math.floor(min / 60)
  const m = min % 60
  if (!h) return `${m} min`
  return m ? `${h} h ${m} min` : `${h} h`
}

/** Relógio do player: "1:02:03" ou "2:03". */
export function clock(seconds: number): string {
  const s = Math.max(0, Math.floor(Number.isFinite(seconds) ? seconds : 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = String(s % 60).padStart(2, '0')
  return h ? `${h}:${String(m).padStart(2, '0')}:${sec}` : `${m}:${sec}`
}

/** Quanto falta: "faltam 38 min". */
export function remaining(it: Pick<WatchItem, 'runtime' | 'position'>): string {
  const left = it.runtime - it.position
  if (it.position <= 0 || left <= 0) return ''
  const r = runtime(left)
  return r ? `faltam ${r}` : ''
}

/** Título pra mostrar: episódio leva a série junto. */
export function displayName(it: WatchItem): string {
  if (it.type === 'Episode' && it.seriesName) return it.seriesName
  return it.name
}

/** Nome acessível do cartão (tudo que o cartão mostra, em texto). */
export function cardLabel(it: WatchItem): string {
  const parts = [displayName(it)]
  if (it.type === 'Episode') parts.push(`${episodeTag(it)} ${it.name}`.trim())
  else if (it.year) parts.push(String(it.year))
  if (it.type === 'Series') parts.push('série')
  if (it.progress > 0 && !it.played) parts.push(`${Math.round(it.progress * 100)}% assistido`)
  if (it.played) parts.push('assistido')
  if (it.favorite) parts.push('favorito')
  return parts.join(', ')
}

/** Abrir um cartão toca direto (episódio, filme começado) ou abre a página. */
export function opensPlayer(it: WatchItem): boolean {
  return it.type === 'Episode' || (it.type === 'Movie' && it.position > 0 && !it.played)
}

/** Texto do botão principal da página. */
export function playLabel(it: WatchItem, next: WatchItem | null): string {
  if (it.type === 'Series') {
    if (!next) return 'Assistir'
    const tag = episodeTag(next)
    return next.position > 0 ? `Continuar ${tag}` : `Assistir ${tag}`
  }
  return it.position > 0 && !it.played ? 'Continuar' : 'Assistir'
}

/** Ponto de partida salvo vale? (muito perto do fim recomeça). */
export function resumeAt(position: number, duration: number): number {
  if (position <= 0) return 0
  if (duration > 0 && duration - position < 30) return 0
  return position
}
