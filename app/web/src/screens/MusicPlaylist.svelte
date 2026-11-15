<script lang="ts">
  // Página da playlist: tocar, aleatório, renomear, apagar, baixar pro PC e
  // as faixas (tirar e mudar de lugar pelas opções da faixa). O perfil
  // infantil só ouve. No arkflix-server, é a mesma lista do servidor que
  // filmes e jogos usam.
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import MusicHead from '../components/MusicHead.svelte'
  import TopBar from '../components/TopBar.svelte'
  import TrackList from '../components/TrackList.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { addToQueue, longDuration, playTracks } from '../lib/music.svelte'
  import { cancelDownload, download, downloading, onDownloaded, removeLocal } from '../lib/musicdl.svelte'
  import { play } from '../lib/sound'
  import { back, confirm, go, isKids, prompt, toast } from '../lib/state.svelte'
  import type { MusicPlaylist } from '../lib/types'

  let { id }: { id: string } = $props()

  let pl = $state<MusicPlaylist | null>(null)
  let failed = $state('')
  let offline = $state(false)

  async function load() {
    try {
      pl = await api.musicPlaylist(id)
      offline = pl.offline
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    void id
    load()
    return onDownloaded('playlist', id, (ev) => {
      if (ev.state === 'done') load()
    })
  })

  const task = $derived(downloading('playlist', id))
  const editable = $derived(!!pl?.mine && !isKids() && !offline)
  const tracks = $derived(pl?.tracks ?? [])

  async function rename() {
    if (!pl) return
    await prompt('Nome da playlist', pl.name, 'Renomear', async (v) => {
      try {
        await api.musicRenamePlaylist(id, v)
        pl!.name = v.trim()
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 60 })
  }

  async function remove() {
    if (!pl) return
    const ok = await confirm(`Apagar a playlist "${pl.name}"? As músicas continuam no servidor.`, 'Apagar playlist', true)
    if (!ok) return
    try {
      await api.musicDeletePlaylist(id)
      toast(`Playlist "${pl.name}" apagada`, 'ok', 2500)
      back()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function removeTrack(i: number) {
    if (!pl) return
    const t = pl.tracks[i]
    try {
      await api.musicRemoveFromPlaylist(id, [i])
      pl.tracks.splice(i, 1)
      pl.songs = pl.tracks.length
      play('back')
      toast(`${t.title} saiu de "${pl.name}"`, 'ok', 2000)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function move(i: number, delta: number) {
    if (!pl) return
    const j = i + delta
    if (j < 0 || j >= pl.tracks.length) return
    const next = [...pl.tracks]
    ;[next[i], next[j]] = [next[j], next[i]]
    try {
      await api.musicReorderPlaylist(id, next.map((t) => t.id))
      pl.tracks = next
      play('move')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function dropLocal() {
    if (pl && (await removeLocal('playlist', id, pl.name))) load()
  }

  $effect(() =>
    pushHandler((a) => {
      if (a === 'random') {
        go({ name: 'nowplaying' })
        return true
      }
      return false
    }),
  )
</script>

<div class="page">
  <TopBar />
  <main class="content" aria-labelledby="music-page-title">
    {#if failed}
      <section class="alert panel" role="alert">
        <span class="chip error">Erro</span>
        <h1 id="music-page-title">Sem sinal</h1>
        <p class="muted">{failed}</p>
        <button class="btn primary" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button>
      </section>
    {:else if !pl}
      <p class="muted" role="status" id="music-page-title">Procurando a playlist…</p>
    {:else}
      <MusicHead
        kind="Playlist"
        title={pl.name}
        cover={pl.cover}
        local={pl.local}
        details={[pl.mine ? '' : `de ${pl.owner}`, `${tracks.length} faixa${tracks.length === 1 ? '' : 's'}`, longDuration(tracks.reduce((t, x) => t + x.duration, 0))].filter(Boolean)}
      >
        <span class="ticket">
          <button data-nav data-nav-default disabled={!tracks.length} onclick={() => playTracks(tracks)}><Icon name="play" /> Tocar</button>
        </span>
        <button class="btn" data-nav disabled={!tracks.length} onclick={() => playTracks(tracks, 0, { shuffle: true })}><Icon name="shuffle" /> Aleatório</button>
        <button class="btn" data-nav disabled={!tracks.length} onclick={() => (addToQueue(tracks), toast(`${pl!.name} na fila`, 'ok', 2000))}><Icon name="queue" /> Pôr na fila</button>
        {#if editable}
          <button class="btn" data-nav onclick={rename}><Icon name="pencil" /> Renomear</button>
          <button class="btn danger" data-nav onclick={remove}><Icon name="trash" /> Apagar playlist</button>
        {/if}
        {#if !isKids()}
          {#if task}
            <span class="chip" role="status">Baixando {Math.round(task.progress * 100)}%</span>
            <button class="btn" data-nav onclick={() => cancelDownload('playlist', id)}><Icon name="close" /> Cancelar download</button>
          {:else if pl.local}
            <button class="btn danger" data-nav onclick={dropLocal}><Icon name="trash" /> Apagar do PC</button>
          {:else if tracks.length}
            <button class="btn" data-nav onclick={() => download('playlist', id, pl!.name)}><Icon name="download" /> Baixar pro PC</button>
          {/if}
        {/if}
      </MusicHead>
      {#if offline}<p class="chip offline" role="status">Sem servidor: tocando a cópia do PC</p>{/if}
      {#if tracks.length}
        <TrackList
          {tracks}
          label={`Faixas de ${pl.name}`}
          numbers="index"
          album
          onplay={(i) => playTracks(tracks, i)}
          onremove={editable ? removeTrack : undefined}
          onmove={editable ? move : undefined}
        />
      {:else}
        <p class="empty">Playlist vazia. Nas opções de uma faixa ou na página de um álbum, use “Adicionar à playlist”.</p>
      {/if}
    {/if}
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Tocar' },
      { button: 'north', label: 'Favoritar faixa' },
      { button: 'west', label: 'Opções da faixa' },
      { button: 'r3', label: 'Tocando agora' },
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .content {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1.6rem;
    padding: 1rem 2.5rem 1.5rem;
  }
  .offline {
    align-self: flex-start;
  }
  .empty {
    padding: 1.2rem;
    color: var(--muted);
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
    max-width: 40rem;
  }
  .alert {
    margin: auto;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
    max-width: 38rem;
    border-color: var(--danger);
    box-shadow: 4px 4px 0 var(--danger);
  }
  .alert h1 {
    font-family: var(--font-label);
    font-size: 1.25rem;
    -webkit-text-stroke: 0;
    text-shadow: none;
  }
</style>
