// Player de músicas: um <audio> só, fora das telas (a música continua ao
// trocar de tela), com fila, aleatório e repetir. O Go dá o endereço no
// proxy local (o arquivo original; mp3 do servidor se o sistema não abrir o
// formato; a cópia do PC quando tem). Avisa o servidor (scrobble: "tocando"
// no começo, "tocou" depois de metade ou 4 min) e guarda a fila lá ao parar,
// ao sair e de tempos em tempos (continua de onde parou). Pausa sozinho
// quando um jogo abre ou um filme começa, e volta depois.
import { api } from './api'
import { app, toast } from './state.svelte'
import type { MusicStream, MusicTrack } from './types'

export type Repeat = 'off' | 'all' | 'one'

export const player = $state({
  queue: [] as MusicTrack[],
  index: -1,
  playing: false,
  loading: false,
  time: 0,
  duration: 0,
  shuffle: false,
  repeat: 'off' as Repeat,
  held: false, // pausado pelo jogo ou pelo filme (volta sozinho)
  error: '',
})

let audio: HTMLAudioElement | null = null
let stream: MusicStream | null = null
let usingFallback = false
let nowSent = false
let scrobbled = false
let pendingSeek = 0
let original: MusicTrack[] | null = null // a ordem antes do aleatório
let saveTimer = 0
let loadSeq = 0

/** Faixa tocando (ou parada no ponto). */
export function nowTrack(): MusicTrack | undefined {
  return player.queue[player.index]
}

/** Tem música na fila (a barrinha aparece). */
export function active(): boolean {
  return player.queue.length > 0 && player.index >= 0
}

function el(): HTMLAudioElement {
  if (audio) return audio
  const a = new Audio()
  a.preload = 'auto'
  a.addEventListener('timeupdate', () => {
    player.time = a.currentTime
    checkScrobble()
  })
  a.addEventListener('durationchange', () => {
    if (Number.isFinite(a.duration)) player.duration = a.duration
  })
  a.addEventListener('loadedmetadata', () => {
    if (Number.isFinite(a.duration)) player.duration = a.duration
    if (pendingSeek > 0 && pendingSeek < (player.duration || Infinity)) a.currentTime = pendingSeek
    pendingSeek = 0
  })
  a.addEventListener('playing', () => {
    player.playing = true
    player.loading = false
    const t = nowTrack()
    if (t && !nowSent) {
      nowSent = true
      api.musicScrobble(t.id, false).catch(() => {})
    }
  })
  a.addEventListener('waiting', () => (player.loading = true))
  a.addEventListener('pause', () => {
    player.playing = false
    saveSoon(0)
  })
  a.addEventListener('ended', onEnded)
  a.addEventListener('error', onError)
  audio = a
  mediaSession()
  return a
}

function checkScrobble() {
  const t = nowTrack()
  const d = player.duration || t?.duration || 0
  if (!t || scrobbled || d <= 0) return
  if (player.time >= Math.min(d / 2, 240)) {
    scrobbled = true
    api.musicScrobble(t.id, true).catch(() => {})
  }
}

async function load(i: number, at = 0, autoplay = true) {
  const t = player.queue[i]
  if (!t) return
  const mine = ++loadSeq
  player.index = i
  player.time = at
  player.duration = t.duration
  player.loading = true
  player.error = ''
  nowSent = false
  scrobbled = false
  usingFallback = false
  try {
    const s = await api.musicPlay(t.id)
    if (mine !== loadSeq) return
    stream = s
    const a = el()
    let src = s.url
    // o sistema diz que não toca o formato: já pede o mp3 do servidor
    if (!s.local && t.contentType && a.canPlayType(t.contentType) === '') {
      src = s.fallback
      usingFallback = true
    }
    pendingSeek = at
    a.src = src
    if (autoplay) await a.play().catch(() => (player.playing = false))
    else player.loading = false
  } catch (err) {
    if (mine !== loadSeq) return
    player.loading = false
    player.error = (err as Error).message
    toast(player.error, 'error')
  }
  updateSession()
}

function onError() {
  const a = audio
  if (!a || !stream) return
  // o arquivo original não abriu aqui: tenta o mp3 do servidor no mesmo ponto
  if (!usingFallback && !stream.local) {
    usingFallback = true
    pendingSeek = a.currentTime || player.time
    a.src = stream.fallback
    a.play().catch(() => {})
    return
  }
  player.loading = false
  player.playing = false
  player.error = 'Esta faixa não tocou aqui.'
  toast(`${nowTrack()?.title ?? 'Faixa'}: não tocou`, 'error')
}

function onEnded() {
  if (player.repeat === 'one') {
    load(player.index)
    return
  }
  if (player.index + 1 < player.queue.length) {
    load(player.index + 1)
  } else if (player.repeat === 'all' && player.queue.length) {
    load(0)
  } else {
    player.playing = false
    player.time = 0
    saveSoon(0)
  }
}

/** Troca a fila e toca a partir de start. */
export function playTracks(tracks: MusicTrack[], start = 0, opts: { shuffle?: boolean } = {}) {
  if (!tracks.length) return
  original = null
  player.shuffle = false
  player.queue = [...tracks]
  if (opts.shuffle) {
    shuffleFrom(-1)
    player.shuffle = true
    start = 0
  }
  player.held = false
  load(Math.min(Math.max(0, start), player.queue.length - 1))
}

/** Põe logo depois da que está tocando. */
export function playNext(tracks: MusicTrack[]) {
  if (!tracks.length) return
  if (!active()) return playTracks(tracks)
  player.queue.splice(player.index + 1, 0, ...tracks)
  original?.push(...tracks)
  saveSoon()
}

/** Põe no fim da fila. */
export function addToQueue(tracks: MusicTrack[]) {
  if (!tracks.length) return
  if (!active()) {
    player.queue = [...tracks]
    load(0, 0, false)
    saveSoon()
    return
  }
  player.queue.push(...tracks)
  original?.push(...tracks)
  saveSoon()
}

/** Tira uma faixa da fila (a que toca, passa pra próxima). */
export function removeAt(i: number) {
  if (i < 0 || i >= player.queue.length) return
  const removed = player.queue[i]
  player.queue.splice(i, 1)
  if (original) {
    const j = original.findIndex((t) => t.id === removed.id)
    if (j >= 0) original.splice(j, 1)
  }
  if (!player.queue.length) return clearQueue()
  if (i < player.index) player.index--
  else if (i === player.index) load(Math.min(i, player.queue.length - 1), 0, player.playing)
  saveSoon()
}

/** Muda uma faixa de lugar na fila (a que toca continua tocando). */
export function moveInQueue(i: number, delta: number) {
  const j = i + delta
  if (i < 0 || j < 0 || i >= player.queue.length || j >= player.queue.length) return
  const q = [...player.queue]
  ;[q[i], q[j]] = [q[j], q[i]]
  player.queue = q
  player.index = player.index === i ? j : player.index === j ? i : player.index
  saveSoon()
}

/** Para e esvazia a fila (guarda vazia no servidor). */
export function clearQueue() {
  loadSeq++
  audio?.pause()
  audio?.removeAttribute('src')
  audio?.load()
  player.queue = []
  player.index = -1
  player.playing = false
  player.time = 0
  player.duration = 0
  player.held = false
  original = null
  stream = null
  updateSession()
  saveSoon(0)
}

export function jump(i: number) {
  if (i >= 0 && i < player.queue.length) load(i)
}

export function toggle() {
  const a = audio
  if (!active()) return
  player.held = false
  if (!a || !a.src) {
    load(player.index, player.time)
    return
  }
  if (a.paused) a.play().catch(() => {})
  else a.pause()
}

export function pause() {
  audio?.pause()
}

export function next() {
  if (!active()) return
  if (player.index + 1 < player.queue.length) load(player.index + 1)
  else if (player.repeat === 'all') load(0)
}

export function prev() {
  if (!active()) return
  // passou do começo: volta pro começo da mesma
  if (player.time > 3 || player.index === 0) {
    seek(0)
    return
  }
  load(player.index - 1)
}

export function seek(t: number) {
  const a = audio
  const d = player.duration || nowTrack()?.duration || 0
  const to = Math.min(Math.max(0, t), Math.max(0, d - 0.25))
  player.time = to
  if (a && a.src) a.currentTime = to
}

export function seekBy(delta: number) {
  seek(player.time + delta)
}

// embaralha o que vem depois de from (a que toca fica onde está)
function shuffleFrom(from: number) {
  const head = player.queue.slice(0, from + 1)
  const rest = player.queue.slice(from + 1)
  for (let i = rest.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[rest[i], rest[j]] = [rest[j], rest[i]]
  }
  player.queue = [...head, ...rest]
}

export function toggleShuffle() {
  if (!player.shuffle) {
    original = [...player.queue]
    shuffleFrom(player.index)
    player.shuffle = true
  } else {
    const cur = nowTrack()
    if (original) {
      player.queue = original
      player.index = Math.max(0, original.findIndex((t) => t.id === cur?.id))
    }
    original = null
    player.shuffle = false
  }
  saveSoon()
}

export function cycleRepeat() {
  player.repeat = player.repeat === 'off' ? 'all' : player.repeat === 'all' ? 'one' : 'off'
}

/** Marca a faixa como favorita (em toda a fila). */
export async function starTrack(t: MusicTrack, on = !t.starred) {
  await api.musicStar('track', t.id, on)
  for (const q of player.queue) if (q.id === t.id) q.starred = on
  t.starred = on
}

/** Retoma a fila salva no servidor (sem tocar). */
export async function resumeSaved(): Promise<boolean> {
  if (active() || !app.music?.connected) return false
  const q = await api.musicResume()
  if (!q.tracks.length) return false
  player.queue = q.tracks
  await load(Math.min(q.current, q.tracks.length - 1), q.position, false)
  return true
}

/** Guarda a fila no servidor daqui a pouco (junta mudanças seguidas). */
function saveSoon(ms = 1500) {
  clearTimeout(saveTimer)
  saveTimer = window.setTimeout(save, ms)
}

function save() {
  if (!app.music?.connected) return
  const t = nowTrack()
  api.musicSaveQueue(player.queue.map((x) => x.id), t?.id ?? '', player.time).catch(() => {})
}

// controles de mídia do sistema (teclas de mídia, fones)
function mediaSession() {
  const ms = typeof navigator !== 'undefined' ? navigator.mediaSession : undefined
  if (!ms) return
  const set = (a: MediaSessionAction, fn: () => void) => {
    try {
      ms.setActionHandler(a, fn)
    } catch {
      /* ação que o sistema não tem */
    }
  }
  set('play', toggle)
  set('pause', pause)
  set('nexttrack', next)
  set('previoustrack', prev)
}

function updateSession() {
  const ms = typeof navigator !== 'undefined' ? navigator.mediaSession : undefined
  const t = nowTrack()
  if (!ms || typeof MediaMetadata === 'undefined') return
  ms.metadata = t ? new MediaMetadata({ title: t.title, artist: t.artist, album: t.album, artwork: t.cover ? [{ src: t.cover }] : [] }) : null
}

let started = false
/** Liga os ganchos globais: jogo/filme pausa, trocar de perfil para. */
export function startMusic() {
  if (started) return
  started = true
  $effect.root(() => {
    // jogo aberto ou filme tocando: pausa; acabou: volta
    $effect(() => {
      const busy = app.playing?.state === 'running' || app.watching
      if (busy && player.playing) {
        player.held = true
        audio?.pause()
      } else if (!busy && player.held) {
        player.held = false
        audio?.play().catch(() => {})
      }
    })
    // outro perfil, outra conta: a fila não é dele
    let profile: string | undefined
    $effect(() => {
      const p = app.profile?.current
      if (profile !== undefined && p !== profile) {
        clearTimeout(saveTimer)
        loadSeq++
        audio?.pause()
        player.queue = []
        player.index = -1
        player.playing = false
        player.held = false
      }
      profile = p
    })
  })
  // tocando: guarda a fila de tempos em tempos (fechar o app não perde o ponto)
  window.setInterval(() => {
    if (player.playing) save()
  }, 30_000)
  window.addEventListener('pagehide', () => {
    if (active()) save()
  })
}

let serverSeq = 0
/** Pergunta de novo o que o servidor de músicas libera (admin, arkflix). */
export async function refreshMusicServer(): Promise<void> {
  const mine = ++serverSeq
  if (!app.music?.connected) {
    app.musicServer = null
    return
  }
  try {
    const info = await api.musicServer()
    if (mine === serverSeq) app.musicServer = info
  } catch {
    if (mine === serverSeq) app.musicServer = null
  }
}

/** "3:05" */
export function mmss(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds || 0))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

/** Duração longa: "1 h 05 min" ou "42 min". */
export function longDuration(seconds: number): string {
  if ((seconds || 0) < 60) return mmss(seconds)
  const m = Math.round(seconds / 60)
  if (m < 60) return `${m} min`
  return `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, '0')} min`
}
