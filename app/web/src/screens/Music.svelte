<script lang="ts">
  // Músicas (Navidrome/Subsonic da conta do perfil). Abas no topo: Início
  // (fileiras de álbuns), Artistas (índice por letra), Álbuns (grade com
  // ordem), Playlists, Favoritas e No PC (o que foi baixado: toca sem
  // internet). Mesmo jeito de filmes e séries: L1/R1 trocam de aba.
  import { tick } from 'svelte'
  import Cover from '../components/Cover.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import Sections from '../components/Sections.svelte'
  import TopBar from '../components/TopBar.svelte'
  import TrackList from '../components/TrackList.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { active, playTracks, resumeSaved, toggle, nowTrack, player } from '../lib/music.svelte'
  import { play } from '../lib/sound'
  import { app, go, isKids, prompt, toast } from '../lib/state.svelte'
  import type { MusicAlbum, MusicArtist, MusicFound, MusicHome, MusicIndex, MusicLocalSet, MusicPlaylist } from '../lib/types'

  type Tab = typeof app.musicTab
  const TABS: { id: Tab; label: string }[] = [
    { id: 'home', label: 'Início' },
    { id: 'artists', label: 'Artistas' },
    { id: 'albums', label: 'Álbuns' },
    { id: 'playlists', label: 'Playlists' },
    { id: 'starred', label: 'Favoritas' },
    { id: 'local', label: 'No PC' },
  ]
  const SORTS = [
    { id: 'newest', label: 'Recentes' },
    { id: 'name', label: 'A-Z' },
    { id: 'artist', label: 'Artista' },
    { id: 'year', label: 'Ano' },
    { id: 'frequent', label: 'Mais tocados' },
    { id: 'random', label: 'Aleatório' },
  ]
  const PAGE = 60

  const connected = $derived(!!app.music?.connected)
  let home = $state<MusicHome | null>(null)
  let artists = $state<MusicIndex[] | null>(null)
  let albums = $state<MusicAlbum[]>([])
  let more = $state(false)
  let sort = $state('newest')
  let playlists = $state<MusicPlaylist[] | null>(null)
  let starred = $state<MusicFound | null>(null)
  let local = $state<MusicLocalSet[] | null>(null)
  let loading = $state(false)
  let failed = $state('')
  let resumable = $state(false)

  async function load(tab: Tab, order: string) {
    if (!connected) return
    loading = true
    failed = ''
    try {
      if (tab === 'home') {
        home = await api.musicHome()
        if (!active()) resumable = await resumeSaved().catch(() => false)
      } else if (tab === 'artists') artists = await api.musicArtists()
      else if (tab === 'albums') {
        const page = await api.musicAlbums(order, 0, PAGE)
        albums = page.albums
        more = page.more
      } else if (tab === 'playlists') playlists = await api.musicPlaylists()
      else if (tab === 'starred') starred = await api.musicStarred()
      else local = await api.musicLocal()
    } catch (err) {
      failed = (err as Error).message
    } finally {
      loading = false
    }
  }

  $effect(() => {
    load(app.musicTab, sort)
  })

  async function loadMore() {
    try {
      const first = albums.length
      const page = await api.musicAlbums(sort, first, PAGE)
      albums = [...albums, ...page.albums]
      more = page.more
      await tick()
      focusEl(document.querySelectorAll<HTMLElement>('.grid [data-item]')[first])
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  type Row = { id: string; title: string; items: MusicAlbum[] }
  const rows = $derived.by((): Row[] => {
    if (!home) return []
    return [
      { id: 'recent', title: 'Tocados há pouco', items: home.recent },
      { id: 'newest', title: 'Adicionados', items: home.newest },
      { id: 'frequent', title: 'Mais tocados', items: home.frequent },
      { id: 'starred', title: 'Álbuns favoritos', items: home.starred },
      { id: 'random', title: 'Pra descobrir', items: home.random },
    ].filter((r) => r.items.length > 0)
  })

  const empty = $derived.by(() => {
    if (loading || failed) return false
    switch (app.musicTab) {
      case 'home':
        return home !== null && rows.length === 0
      case 'artists':
        return artists !== null && artists.length === 0
      case 'albums':
        return albums.length === 0
      case 'playlists':
        return false // tem o cartão "Nova playlist"
      case 'starred':
        return starred !== null && starred.albums.length + starred.artists.length + starred.tracks.length === 0
      default:
        return local !== null && local.length === 0
    }
  })
  const ready = $derived(
    {
      home: home !== null,
      artists: artists !== null,
      albums: !loading || albums.length > 0,
      playlists: playlists !== null,
      starred: starred !== null,
      local: local !== null,
    }[app.musicTab],
  )

  function selectTab(t: Tab) {
    if (app.musicTab === t) return
    app.musicTab = t
  }

  function cycleTab(delta: number) {
    const i = TABS.findIndex((t) => t.id === app.musicTab)
    const next = TABS[(i + delta + TABS.length) % TABS.length]
    selectTab(next.id)
    play('move')
    tick().then(() => focusEl(document.querySelector<HTMLElement>(`[data-music-tab="${next.id}"]`)))
  }

  function albumLabel(a: MusicAlbum): string {
    return [a.name, a.artist, a.year ? String(a.year) : '', a.starred ? 'favorito' : '', a.local ? 'no PC' : ''].filter(Boolean).join(', ')
  }

  async function starFocused() {
    const id = current()?.dataset.item ?? ''
    const [kind, key] = id.split(':')
    const pool: (MusicAlbum | MusicArtist)[] =
      kind === 'album' ? [...rows.flatMap((r) => r.items), ...albums, ...(starred?.albums ?? [])] : kind === 'artist' ? [...(artists?.flatMap((i) => i.artists) ?? []), ...(starred?.artists ?? [])] : []
    const it = pool.find((x) => x.id === key)
    if (!it) return false
    try {
      const on = !it.starred
      await api.musicStar(kind as 'album' | 'artist', key, on)
      for (const x of pool) if (x.id === key) x.starred = on
      play('toggle')
      toast(on ? `${it.name} nas favoritas` : `${it.name} saiu das favoritas`, 'ok', 2000)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
    return true
  }

  $effect(() =>
    pushHandler((a) => {
      switch (a) {
        case 'fav': {
          const id = current()?.dataset.item ?? ''
          if (id.startsWith('album:') || id.startsWith('artist:')) {
            starFocused()
            return true
          }
          return false
        }
        case 'prevTab':
          cycleTab(-1)
          return true
        case 'nextTab':
          cycleTab(1)
          return true
        case 'menu':
          go({ name: 'you' })
          return true
        case 'random':
          if (active()) {
            go({ name: 'nowplaying' })
            return true
          }
          return false
      }
      return false
    }),
  )

  async function newPlaylist() {
    let made: MusicPlaylist | null = null
    await prompt('Nova playlist', '', 'Criar playlist', async (v) => {
      try {
        made = await api.musicCreatePlaylist(v, [])
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 60 })
    const got = made as MusicPlaylist | null
    if (got) {
      toast(`Playlist “${got.name}” criada`, 'ok', 2500)
      go({ name: 'playlist', id: got.id })
    }
  }

  function jumpTo(letter: string) {
    focusEl(document.querySelector<HTMLElement>(`li[data-letter="${CSS.escape(letter)}"] [data-item]`))
  }

  function localLine(s: MusicLocalSet): string {
    return `${s.kind === 'album' ? s.artist || 'Álbum' : 'Playlist'} · ${s.songs} faixa${s.songs === 1 ? '' : 's'}`
  }

  function openLocal(s: MusicLocalSet) {
    go(s.kind === 'album' ? { name: 'album', id: s.id } : { name: 'playlist', id: s.id })
  }

  function continueSaved() {
    resumable = false
    toggle()
    go({ name: 'nowplaying' })
  }
</script>

{#snippet albumCard(a: MusicAlbum)}
  <button class="card" data-nav data-nav-inline="center" data-item={`album:${a.id}`} aria-label={albumLabel(a)} onclick={() => go({ name: 'album', id: a.id })}>
    <Cover showName={false} src={a.cover} name={a.name} local={a.local} />
    <span class="caption">
      <span class="name">{a.name}</span>
      <span class="muted">{a.artist}{a.year ? ` · ${a.year}` : ''}</span>
    </span>
  </button>
{/snippet}

{#snippet artistCard(a: MusicArtist)}
  <button class="card" data-nav data-item={`artist:${a.id}`} aria-label={`${a.name}, ${a.albums} álbu${a.albums === 1 ? 'm' : 'ns'}${a.starred ? ', favorito' : ''}`} onclick={() => go({ name: 'artist', id: a.id })}>
    <Cover showName={false} src={a.cover} name={a.name} round />
    <span class="caption center">
      <span class="name">{a.name}</span>
      <span class="muted">{a.albums} álbu{a.albums === 1 ? 'm' : 'ns'}</span>
    </span>
  </button>
{/snippet}

<div class="music">
  <TopBar>
    <div class="top-nav">
      <Sections current="music" />
      {#if connected}
        <nav class="tabs" aria-label="Músicas">
          {#each TABS as t (t.id)}
            <button class="tab" data-nav data-music-tab={t.id} aria-current={app.musicTab === t.id ? 'page' : undefined} onclick={() => selectTab(t.id)}>{t.label}</button>
          {/each}
        </nav>
      {/if}
    </div>
  </TopBar>

  <main class="content" aria-labelledby="music-title">
    {#if !connected}
      <section class="empty">
        <Icon name="music" size="3rem" />
        <h1 id="music-title">Músicas</h1>
        {#if isKids()}
          <p class="muted">Este perfil ainda não tem conta de músicas. Peça pra um adulto vincular uma em Perfis da casa.</p>
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'home' })}><Icon name="pad" /> Ir pros jogos</button>
          </div>
        {:else}
          <p class="muted">
            Vincule a sua conta do Navidrome (ou de outro servidor Subsonic, como o arkflix-server) a este perfil pra ouvir as
            suas músicas, playlists e favoritas. Cada perfil usa a própria conta.
          </p>
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'you', item: 'profile' })}><Icon name="music" /> Vincular conta de músicas</button>
          </div>
        {/if}
      </section>
    {:else if failed}
      <section class="alert panel" role="alert">
        <span class="chip error">Erro</span>
        <h1 id="music-title">Sem sinal</h1>
        <p class="muted">{failed}</p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => load(app.musicTab, sort)}><Icon name="resume" /> Tentar de novo</button>
          <button class="btn" data-nav onclick={() => selectTab('local')}><Icon name="download" /> Ouvir o que está no PC</button>
        </div>
      </section>
    {:else if empty}
      <section class="empty">
        <Icon name={app.musicTab === 'starred' ? 'star' : app.musicTab === 'local' ? 'download' : 'music'} size="3rem" />
        <h1 id="music-title">{app.musicTab === 'starred' ? 'Nenhuma favorita' : app.musicTab === 'local' ? 'Nada no PC' : 'Nada por aqui'}</h1>
        <p class="muted">
          {app.musicTab === 'starred'
            ? 'Aperte o botão de favorito numa faixa, num álbum ou num artista pra guardar aqui.'
            : app.musicTab === 'local'
              ? 'Na página de um álbum ou playlist, use “Baixar pro PC” pra ouvir sem internet.'
              : 'O servidor ainda não tem músicas pra esta conta.'}
        </p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => selectTab('home')}><Icon name="home" /> Voltar ao início</button>
        </div>
      </section>
    {:else if !ready}
      <div class="loading" role="status" aria-busy="true">
        <!-- 🎬 disco de vinil: dá uma volta ao aparecer e para -->
        <span class="vinyl" aria-hidden="true"></span>
        <p class="meta" id="music-title">Pondo o disco na vitrola…</p>
      </div>
    {:else}
      <h1 class="sr-only" id="music-title">Músicas · {TABS.find((t) => t.id === app.musicTab)?.label}</h1>
      {#if app.musicTab === 'home'}
        {#if resumable && nowTrack()}
          <section class="resume panel" aria-label="Continuar de onde parou">
            <Cover src={nowTrack()!.cover} name={nowTrack()!.album} />
            <div class="resume-text">
              <p class="meta">Continuar de onde parou</p>
              <p class="resume-title">{nowTrack()!.title}</p>
              <p class="muted">{nowTrack()!.artist} · {player.queue.length} na fila</p>
            </div>
            <button class="btn primary" data-nav data-nav-default onclick={continueSaved}><Icon name="play" /> Continuar</button>
          </section>
        {/if}
        <div class="shelves">
          {#each rows as row (row.id)}
            <section class="shelf" aria-label={row.title}>
              <h2 class="row-title">{row.title}</h2>
              <ul class="row" data-nav-group={`music-${row.id}`}>
                {#each row.items as a (a.id)}
                  <li>{@render albumCard(a)}</li>
                {/each}
              </ul>
            </section>
          {/each}
        </div>
      {:else if app.musicTab === 'artists' && artists}
        <nav class="letters" aria-label="Índice de artistas">
          {#each artists as idx (idx.name)}
            <button class="letter" data-nav onclick={() => jumpTo(idx.name)} aria-label={`Artistas com ${idx.name}`}>{idx.name}</button>
          {/each}
        </nav>
        <ul class="grid" data-nav-group="music-artists" aria-label="Artistas">
          {#each artists as idx (idx.name)}
            {#each idx.artists as a, i (a.id)}
              <li data-letter={i === 0 ? idx.name : undefined}>
                {#if i === 0}<span class="initial" aria-hidden="true">{idx.name}</span>{/if}
                {@render artistCard(a)}
              </li>
            {/each}
          {/each}
        </ul>
      {:else if app.musicTab === 'albums'}
        <div class="toolbar">
          <p class="count meta">{albums.length}{more ? '+' : ''} álbu{albums.length === 1 ? 'm' : 'ns'}</p>
          <div class="segmented" role="group" aria-label="Ordem">
            {#each SORTS as s (s.id)}
              <button class="btn small" data-nav aria-pressed={sort === s.id} onclick={() => (sort = s.id)}>{s.label}</button>
            {/each}
          </div>
        </div>
        <ul class="grid" data-nav-group="music-grid" aria-label="Álbuns">
          {#each albums as a (a.id)}
            <li>{@render albumCard(a)}</li>
          {/each}
          {#if more}
            <li><button class="card more" data-nav onclick={loadMore}><Icon name="plus" size="1.6em" /> Mais</button></li>
          {/if}
        </ul>
      {:else if app.musicTab === 'playlists' && playlists}
        <ul class="grid" data-nav-group="music-playlists" aria-label="Playlists">
          {#each playlists as p (p.id)}
            <li>
              <button class="card" data-nav data-item={`playlist:${p.id}`} aria-label={`${p.name}, ${p.songs} faixa${p.songs === 1 ? '' : 's'}${p.local ? ', no PC' : ''}`} onclick={() => go({ name: 'playlist', id: p.id })}>
                <Cover showName={false} src={p.cover} name={p.name} local={p.local} />
                <span class="caption">
                  <span class="name">{p.name}</span>
                  <span class="muted">{p.songs} faixa{p.songs === 1 ? '' : 's'}{p.mine ? '' : ` · de ${p.owner}`}</span>
                </span>
              </button>
            </li>
          {/each}
          {#if !isKids()}
            <li><button class="card more" data-nav onclick={newPlaylist}><Icon name="plus" size="1.6em" /> Nova playlist</button></li>
          {/if}
        </ul>
      {:else if app.musicTab === 'starred' && starred}
        {#if starred.tracks.length}
          <section aria-labelledby="fav-tracks">
            <div class="section-head">
              <h2 class="row-title" id="fav-tracks">Faixas</h2>
              <button class="btn small" data-nav onclick={() => playTracks(starred!.tracks, 0, { shuffle: true })}><Icon name="shuffle" /> Aleatório</button>
            </div>
            <TrackList tracks={starred.tracks} label="Faixas favoritas" album onplay={(i) => playTracks(starred!.tracks, i)} group="fav-tracks" />
          </section>
        {/if}
        {#if starred.albums.length}
          <section class="shelf" aria-label="Álbuns favoritos">
            <h2 class="row-title">Álbuns</h2>
            <ul class="row" data-nav-group="fav-albums">
              {#each starred.albums as a (a.id)}
                <li>{@render albumCard(a)}</li>
              {/each}
            </ul>
          </section>
        {/if}
        {#if starred.artists.length}
          <section class="shelf" aria-label="Artistas favoritos">
            <h2 class="row-title">Artistas</h2>
            <ul class="row" data-nav-group="fav-artists">
              {#each starred.artists as a (a.id)}
                <li>{@render artistCard(a)}</li>
              {/each}
            </ul>
          </section>
        {/if}
      {:else if app.musicTab === 'local' && local}
        <p class="count meta">No PC, em ~/Music/arkflix: toca mesmo sem internet.</p>
        <ul class="grid" data-nav-group="music-local" aria-label="No PC">
          {#each local as s (s.kind + s.id)}
            <li>
              <button class="card" data-nav data-item={`${s.kind}:${s.id}`} aria-label={`${s.name}, ${localLine(s)}, ${s.kind === 'album' ? 'álbum' : 'playlist'} no PC`} onclick={() => openLocal(s)}>
                <Cover showName={false} src={s.cover} name={s.name} local />
                <span class="caption">
                  <span class="name">{s.name}</span>
                  <span class="muted">{localLine(s)}</span>
                </span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
  </main>

  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'north', label: 'Favoritar' },
      ...(app.musicTab === 'starred' ? [{ button: 'west', label: 'Opções da faixa' }] : []),
      { button: 'l1', label: 'Aba anterior' },
      { button: 'r1', label: 'Próxima aba' },
      ...(active() ? [{ button: 'r3', label: 'Tocando agora' }] : []),
      { button: 'start', label: 'Você' },
    ]}
  />
</div>

<style>
  .music {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .top-nav {
    display: flex;
    align-items: center;
    gap: 1.4rem;
    flex-wrap: wrap;
  }
  .tabs {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }
  .tab {
    font-family: var(--font-label);
    font-size: 0.8rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--text);
    background: none;
    border: none;
    border-bottom: var(--edge) solid transparent;
    padding: 0.55rem 0.7rem 0.4rem;
    min-height: 2.6rem;
    cursor: pointer;
  }
  .tab:hover {
    border-bottom-color: var(--outline);
  }
  .tab[aria-current='page'] {
    border-bottom-color: var(--accent);
    color: var(--accent);
  }
  .content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
    padding: 1rem 2.5rem 1rem;
    overflow-y: auto;
  }
  .loading {
    margin: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }
  .loading .meta {
    color: var(--muted);
    font-size: 1rem;
  }
  /* 🎬 vinil: sulcos em anéis, selo mostarda no meio; uma volta em degraus */
  .vinyl {
    width: 4.5rem;
    height: 4.5rem;
    border-radius: 50%;
    border: var(--edge) solid var(--outline);
    box-shadow: var(--hard-sm);
    background:
      radial-gradient(circle, var(--bg) 0 0.18rem, var(--accent) 0.2rem 0.85rem, var(--outline) 0.88rem 0.95rem, transparent 1rem),
      repeating-radial-gradient(circle, color-mix(in srgb, var(--outline) 30%, var(--bg)) 0 1px, var(--bg) 1px 4px);
    animation: spin 1.2s steps(12) 1;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  :global(.reduce-motion) .vinyl {
    animation: none;
  }
  .resume {
    display: grid;
    grid-template-columns: 6rem 1fr auto;
    align-items: center;
    gap: 1.2rem;
    max-width: 46rem;
  }
  .resume-text {
    min-width: 0;
  }
  .resume-title {
    font-family: var(--font-label);
    font-size: 1.2rem;
  }
  .shelves {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }
  .row-title {
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text);
    display: flex;
    align-items: center;
    gap: 0.6rem;
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
    scroll-snap-type: x proximity;
    scrollbar-width: none;
  }
  .card {
    position: relative;
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
    scroll-snap-align: center;
  }
  .card:focus,
  .card:focus-visible {
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
  .card:active :global(.cover) {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  .caption {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    min-width: 0;
  }
  .caption.center {
    align-items: center;
    text-align: center;
  }
  .caption .name {
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
  }
  .caption .muted {
    font-family: var(--font-hud);
    font-size: 0.8rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
  }
  .card:focus .caption .name {
    color: var(--accent);
  }
  .grid {
    list-style: none;
    margin: 0;
    padding: 0.8rem 0.6rem 1.5rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1.8rem 1.5rem;
  }
  .grid .card {
    width: 100%;
  }
  .more {
    aspect-ratio: 1;
    align-items: center;
    justify-content: center;
    font-family: var(--font-label);
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
  }
  .more:focus {
    outline: 3px solid var(--focus) !important;
    outline-offset: 3px;
    border-style: solid;
    border-color: var(--outline);
  }
  /* primeira letra de cada grupo: etiqueta datilografada em cima do cartão */
  .grid li {
    position: relative;
  }
  .initial {
    position: absolute;
    inset-block-start: -0.7rem;
    inset-inline-start: -0.3rem;
    z-index: 1;
    font-family: var(--font-label);
    font-size: 0.8rem;
    color: var(--on-accent);
    background: var(--accent);
    border: 2px solid var(--outline);
    border-radius: 4px;
    padding: 0.05rem 0.45rem;
  }
  .letters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .letter {
    min-width: 2.4rem;
    min-height: 2.4rem;
    font-family: var(--font-label);
    font-size: 0.9rem;
    color: var(--text);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    cursor: pointer;
  }
  .letter:hover,
  .letter:focus-visible {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard);
  }
  .toolbar,
  .section-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .section-head {
    margin-bottom: 0.6rem;
  }
  .count {
    color: var(--muted);
  }
  .segmented {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
  }
  .empty {
    margin: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 1rem;
    max-width: 34rem;
    padding: 2rem 1.25rem;
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
    color: var(--muted);
  }
  .empty h1,
  .alert h1 {
    font-family: var(--font-label);
    font-size: 1.25rem;
    color: var(--text);
    -webkit-text-stroke: 0;
    text-shadow: none;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1.2rem;
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
</style>
