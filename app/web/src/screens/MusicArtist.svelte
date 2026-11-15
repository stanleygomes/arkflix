<script lang="ts">
  // Página do artista: as mais tocadas (lista) e os álbuns (fileira).
  import Cover from '../components/Cover.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import MusicHead from '../components/MusicHead.svelte'
  import TopBar from '../components/TopBar.svelte'
  import TrackList from '../components/TrackList.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { playTracks } from '../lib/music.svelte'
  import { play } from '../lib/sound'
  import { go, toast } from '../lib/state.svelte'
  import type { MusicAlbum, MusicArtist, MusicTrack } from '../lib/types'

  let { id }: { id: string } = $props()

  let artist = $state<MusicArtist | null>(null)
  let albums = $state<MusicAlbum[]>([])
  let top = $state<MusicTrack[]>([])
  let failed = $state('')

  async function load() {
    try {
      const r = await api.musicArtist(id)
      artist = r.artist
      albums = r.albums
      top = r.top
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    void id
    load()
  })

  async function toggleStar() {
    if (!artist) return
    try {
      const on = !artist.starred
      await api.musicStar('artist', artist.id, on)
      artist.starred = on
      play('toggle')
      toast(on ? `${artist.name} nos favoritos` : `${artist.name} saiu dos favoritos`, 'ok', 2000)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  // tocar tudo: as faixas dos álbuns, na ordem
  async function playAll(shuffle: boolean) {
    try {
      const all = (await Promise.all(albums.map((a) => api.musicAlbum(a.id)))).flatMap((r) => r.tracks)
      playTracks(all.length ? all : top, 0, { shuffle })
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() =>
    pushHandler((a) => {
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
    {:else if !artist}
      <p class="muted" role="status" id="music-page-title">Procurando o artista…</p>
    {:else}
      <MusicHead kind="Artista" title={artist.name} cover={artist.cover} round starred={artist.starred} details={[`${albums.length} álbu${albums.length === 1 ? 'm' : 'ns'}`]}>
        <span class="ticket">
          <button data-nav data-nav-default onclick={() => playAll(false)}><Icon name="play" /> Tocar tudo</button>
        </span>
        <button class="btn" data-nav onclick={() => playAll(true)}><Icon name="shuffle" /> Aleatório</button>
        <button class="btn" data-nav aria-pressed={artist.starred} onclick={toggleStar}><Icon name="star" /> Favorito</button>
      </MusicHead>
      {#if top.length}
        <section aria-labelledby="top-title">
          <h2 class="row-title" id="top-title">Mais tocadas</h2>
          <TrackList tracks={top} label={`Mais tocadas de ${artist.name}`} numbers="index" artist={false} album onplay={(i) => playTracks(top, i)} group="top" />
        </section>
      {/if}
      <section aria-labelledby="albums-title">
        <h2 class="row-title" id="albums-title">Álbuns</h2>
        <ul class="row" data-nav-group="artist-albums">
          {#each albums as a (a.id)}
            <li>
              <button class="card" data-nav data-nav-inline="center" data-item={`album:${a.id}`} aria-label={[a.name, a.year ? String(a.year) : '', a.local ? 'no PC' : ''].filter(Boolean).join(', ')} onclick={() => go({ name: 'album', id: a.id })}>
                <Cover showName={false} src={a.cover} name={a.name} local={a.local} />
                <span class="caption"><span class="name">{a.name}</span><span class="muted">{a.year || ' '}</span></span>
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir / tocar' },
      { button: 'north', label: 'Favoritar' },
      { button: 'west', label: 'Opções da faixa' },
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
  .row-title {
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    display: flex;
    align-items: center;
    gap: 0.6rem;
    margin-bottom: 0.6rem;
  }
  .row-title::before {
    content: '';
    width: 0.6rem;
    height: 0.6rem;
    background: var(--accent);
    border: 2px solid var(--outline);
  }
  .row {
    list-style: none;
    margin: 0;
    padding: 0.8rem 0.6rem 1.2rem;
    display: flex;
    gap: 1.5rem;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .card {
    display: flex;
    flex-direction: column;
    gap: 0.55rem;
    width: clamp(8rem, min(12vw, 19vh), 12rem);
    padding: 0;
    font: inherit;
    color: var(--text);
    text-align: start;
    border: none;
    background: none;
    cursor: pointer;
  }
  .card:focus {
    outline: none;
  }
  .card:hover :global(.cover) {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card:focus :global(.cover) {
    transform: translate(-4px, -4px);
    box-shadow: 8px 8px 0 var(--outline);
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .caption {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .caption .name {
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .caption .muted {
    font-family: var(--font-hud);
    font-size: 0.8rem;
  }
  .card:focus .caption .name {
    color: var(--accent);
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
