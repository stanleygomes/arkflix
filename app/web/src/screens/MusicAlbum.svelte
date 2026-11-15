<script lang="ts">
  // Página do álbum: capa, tocar (🎬 bilhete), aleatório, favorito, playlist,
  // baixar pro PC e as faixas. Sem servidor, abre a cópia baixada.
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import MusicHead from '../components/MusicHead.svelte'
  import PlaylistPicker from '../components/PlaylistPicker.svelte'
  import TopBar from '../components/TopBar.svelte'
  import TrackList from '../components/TrackList.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { addToQueue, longDuration, playNext, playTracks } from '../lib/music.svelte'
  import { cancelDownload, download, downloading, onDownloaded, removeLocal } from '../lib/musicdl.svelte'
  import { play } from '../lib/sound'
  import { go, isKids, toast } from '../lib/state.svelte'
  import type { MusicAlbum, MusicTrack } from '../lib/types'

  let { id }: { id: string } = $props()

  let album = $state<MusicAlbum | null>(null)
  let tracks = $state<MusicTrack[]>([])
  let offline = $state(false)
  let failed = $state('')
  let picking = $state(false)

  async function load() {
    try {
      const r = await api.musicAlbum(id)
      album = r.album
      tracks = r.tracks
      offline = r.offline
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    void id
    load()
    return onDownloaded('album', id, (ev) => {
      if (ev.state === 'done') load()
    })
  })

  const task = $derived(downloading('album', id))
  const kids = $derived(isKids())

  async function toggleStar() {
    if (!album) return
    try {
      const on = !album.starred
      await api.musicStar('album', album.id, on)
      album.starred = on
      play('toggle')
      toast(on ? `${album.name} nos favoritos` : `${album.name} saiu dos favoritos`, 'ok', 2000)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function dropLocal() {
    if (album && (await removeLocal('album', album.id, album.name))) load()
  }

  $effect(() =>
    pushHandler((a) => {
      if (picking) return false
      if (a === 'fav' && !document.activeElement?.closest('.tracks')) {
        toggleStar()
        return true
      }
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
    {:else if !album}
      <p class="muted" role="status" id="music-page-title">Procurando o disco…</p>
    {:else}
      <MusicHead
        kind="Álbum"
        title={album.name}
        cover={album.cover}
        local={album.local}
        starred={album.starred}
        details={[album.artist, album.year ? String(album.year) : '', album.genre, `${tracks.length} faixa${tracks.length === 1 ? '' : 's'}`, longDuration(tracks.reduce((t, x) => t + x.duration, 0))].filter(Boolean)}
      >
        <span class="ticket">
          <button data-nav data-nav-default disabled={!tracks.length} onclick={() => playTracks(tracks)}><Icon name="play" /> Tocar</button>
        </span>
        <button class="btn" data-nav disabled={!tracks.length} onclick={() => playTracks(tracks, 0, { shuffle: true })}><Icon name="shuffle" /> Aleatório</button>
        <button class="btn" data-nav aria-pressed={album.starred} onclick={toggleStar}><Icon name="star" /> Favorito</button>
        <button class="btn" data-nav onclick={() => (playNext(tracks), toast(`${album!.name} toca a seguir`, 'ok', 2000))}><Icon name="next" /> Tocar a seguir</button>
        <button class="btn" data-nav onclick={() => (addToQueue(tracks), toast(`${album!.name} na fila`, 'ok', 2000))}><Icon name="queue" /> Pôr na fila</button>
        {#if !kids && !offline}
          <button class="btn" data-nav onclick={() => (picking = true)}><Icon name="plus" /> Adicionar à playlist</button>
        {/if}
        {#if album.artistId}
          <button class="btn" data-nav onclick={() => go({ name: 'artist', id: album!.artistId })}><Icon name="user" /> {album.artist}</button>
        {/if}
        {#if !kids}
          {#if task}
            <span class="chip" role="status">Baixando {Math.round(task.progress * 100)}%</span>
            <button class="btn" data-nav onclick={() => cancelDownload('album', id)}><Icon name="close" /> Cancelar download</button>
          {:else if album.local}
            <button class="btn danger" data-nav onclick={dropLocal}><Icon name="trash" /> Apagar do PC</button>
          {:else if !offline}
            <button class="btn" data-nav onclick={() => download('album', id, album!.name)}><Icon name="download" /> Baixar pro PC</button>
          {/if}
        {/if}
      </MusicHead>
      {#if offline}<p class="chip offline" role="status">Sem servidor: tocando a cópia do PC</p>{/if}
      <TrackList {tracks} label={`Faixas de ${album.name}`} artist={false} onplay={(i) => playTracks(tracks, i)} />
    {/if}
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Tocar' },
      { button: 'north', label: 'Favoritar' },
      { button: 'west', label: 'Opções da faixa' },
      { button: 'r3', label: 'Tocando agora' },
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

{#if picking && album}
  <PlaylistPicker name={album.name} album={album.id} onclose={() => (picking = false)} />
{/if}

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
