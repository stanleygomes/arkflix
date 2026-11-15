<script lang="ts">
  // Filmes e séries (Jellyfin da conta do perfil). Mesmo jeito do Início dos
  // jogos: destaque grande em cima (muda com o foco) e fileiras embaixo. Abas
  // no topo: Início, Filmes, Séries e Minha lista (estas em grade, com ordem)
  // e, com um arkflix-server, Listas (as do usuário de lá, em fileiras).
  import { tick } from 'svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import Poster from '../components/Poster.svelte'
  import Sections from '../components/Sections.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { play } from '../lib/sound'
  import { shade } from '../lib/motion'
  import { app, go, isKids, toast } from '../lib/state.svelte'
  import type { WatchHome, WatchItem } from '../lib/types'
  import { loadArkLists } from '../lib/ark'
  import { cardLabel, displayName, episodeTag, opensPlayer, playLabel, remaining, runtime } from '../lib/watch'

  type Tab = 'home' | 'movies' | 'series' | 'favorites' | 'lists'
  const TABS = $derived<{ id: Tab; label: string }[]>([
    { id: 'home', label: 'Início' },
    { id: 'movies', label: 'Filmes' },
    { id: 'series', label: 'Séries' },
    { id: 'favorites', label: 'Minha lista' },
    ...(app.ark?.lists && app.ark.videos ? [{ id: 'lists' as const, label: 'Listas' }] : []),
  ])
  // fileiras: Início e Listas; o resto é grade
  const shelved = $derived(app.watchTab === 'home' || app.watchTab === 'lists')
  const SORTS = [
    { id: 'added', label: 'Recentes' },
    { id: 'name', label: 'A-Z' },
    { id: 'year', label: 'Ano' },
    { id: 'score', label: 'Nota' },
  ]
  const PAGE = 60

  const connected = $derived(!!app.watch?.connected)
  let home = $state<WatchHome | null>(null)
  let list = $state<WatchItem[]>([])
  let total = $state(0)
  let sort = $state('added')
  let loading = $state(false)
  let failed = $state('')
  let listsLoaded = $state(false)

  async function load(tab: Tab, order: string) {
    if (!connected) return
    loading = true
    failed = ''
    try {
      if (tab === 'home') {
        home = await api.watchHome()
      } else if (tab === 'lists') {
        await loadArkLists()
        listsLoaded = true
      } else {
        const page = await api.watchList(tab, order, '', 0, PAGE)
        list = page.items
        total = page.total
      }
    } catch (err) {
      failed = (err as Error).message
    } finally {
      loading = false
    }
  }

  $effect(() => {
    load(app.watchTab, sort)
  })

  async function more() {
    try {
      const page = await api.watchList(app.watchTab, sort, '', list.length, PAGE)
      const first = list.length
      list = [...list, ...page.items]
      total = page.total
      await tick()
      focusEl(document.querySelectorAll<HTMLElement>('.grid [data-item]')[first])
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  type Row = { id: string; title: string; items: WatchItem[]; wide?: boolean; list?: string }
  const rows = $derived.by((): Row[] => {
    if (app.watchTab === 'lists') {
      return app.arkLists.filter((l) => l.videos.length > 0).map((l) => ({ id: `ark-${l.id}`, title: l.name, items: l.videos, list: l.id }))
    }
    if (!home) return []
    return [
      { id: 'resume', title: 'Continuar assistindo', items: home.resume, wide: true },
      { id: 'next', title: 'Próximos episódios', items: home.nextUp, wide: true },
      { id: 'favorites', title: 'Minha lista', items: home.favorites },
      { id: 'movies', title: 'Filmes adicionados', items: home.movies },
      { id: 'series', title: 'Séries adicionadas', items: home.series },
    ].filter((r) => r.items.length > 0)
  })

  const pool = $derived(shelved ? rows.flatMap((r) => r.items) : list)
  const selected = $derived(pool.find((it) => it.id === app.watchSelected) ?? pool[0])
  const hero = $derived(selected?.backdrop || selected?.poster || '')
  const empty = $derived(
    !loading &&
      !failed &&
      (app.watchTab === 'home' ? home !== null && rows.length === 0 : app.watchTab === 'lists' ? listsLoaded && rows.length === 0 : list.length === 0),
  )

  // cartão largo mostra nome e uma linha embaixo: o nome acessível começa
  // pelo mesmo texto visível e acrescenta o resto (progresso, lista)
  function caption(it: WatchItem): string {
    return it.type === 'Episode' ? `${episodeTag(it)} · ${it.name}` : remaining(it) || runtime(it.runtime)
  }
  function wideLabel(it: WatchItem): string {
    const extra = []
    if (it.progress > 0 && !it.played) extra.push(`${Math.round(it.progress * 100)}% assistido`)
    if (it.favorite) extra.push('na Minha lista')
    return [`${displayName(it)} ${caption(it)}`.trim(), ...extra].join(', ')
  }

  function selectTab(t: Tab) {
    if (app.watchTab === t) return
    app.watchTab = t
    app.watchSelected = ''
  }

  function cycleTab(delta: number) {
    const i = TABS.findIndex((t) => t.id === app.watchTab)
    const next = TABS[(i + delta + TABS.length) % TABS.length]
    selectTab(next.id)
    play('move')
    tick().then(() => focusEl(document.querySelector<HTMLElement>(`[data-watch-tab="${next.id}"]`)))
  }

  /** Abre o cartão: episódio e filme começado tocam; o resto abre a página. */
  function open(it: WatchItem) {
    if (opensPlayer(it)) go({ name: 'player', id: it.id })
    else go({ name: 'title', id: it.id })
  }

  function details(it: WatchItem | undefined) {
    if (!it) return
    go({ name: 'title', id: it.type === 'Episode' && it.seriesId ? it.seriesId : it.id })
  }

  function put(it: WatchItem) {
    const swap = (arr: WatchItem[]) => arr.map((x) => (x.id === it.id ? { ...x, favorite: it.favorite, played: it.played } : x))
    if (home) home = { resume: swap(home.resume), nextUp: swap(home.nextUp), movies: swap(home.movies), series: swap(home.series), favorites: swap(home.favorites) }
    list = swap(list)
  }

  async function toggleFav(it: WatchItem | undefined) {
    if (!it) return
    try {
      const next = await api.watchFavorite(it.id, !it.favorite)
      put(next)
      play('toggle')
      toast(next.favorite ? `${displayName(next)} na Minha lista` : `${displayName(next)} saiu da Minha lista`, 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function focused(): WatchItem | undefined {
    const id = current()?.dataset.item
    return id ? pool.find((it) => it.id === id) : selected
  }

  $effect(() =>
    pushHandler((a) => {
      switch (a) {
        case 'fav':
          toggleFav(focused())
          return true
        case 'details':
          details(focused())
          return true
        case 'prevTab':
          cycleTab(-1)
          return true
        case 'nextTab':
          cycleTab(1)
          return true
        case 'menu':
          go({ name: 'you' })
          return true
      }
      return false
    }),
  )

  function playSelected() {
    if (!selected) return
    if (selected.type === 'Series') go({ name: 'title', id: selected.id })
    else go({ name: 'player', id: selected.id })
  }
</script>

<div class="watch">
  {#key hero}
    <div class="backdrop" in:shade style:background-image={hero ? `url("${hero}")` : 'none'} aria-hidden="true"></div>
    <div class="scrim" aria-hidden="true"></div>
  {/key}
  <TopBar>
    <div class="top-nav">
      <Sections current="watch" />
      {#if connected}
        <nav class="tabs" aria-label="Filmes e séries">
          {#each TABS as t (t.id)}
            <button class="tab" data-nav data-watch-tab={t.id} aria-current={app.watchTab === t.id ? 'page' : undefined} onclick={() => selectTab(t.id)}>{t.label}</button>
          {/each}
        </nav>
      {/if}
    </div>
  </TopBar>

  <main class="content" aria-labelledby="watch-title">
    {#if !connected}
      <section class="empty">
        <Icon name="film" size="3rem" />
        <h1 id="watch-title">Filmes e séries</h1>
        {#if isKids()}
          <p class="muted">Este perfil ainda não tem conta do Jellyfin. Peça pra um adulto vincular uma em Perfis da casa.</p>
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'home' })}><Icon name="pad" /> Ir pros jogos</button>
          </div>
        {:else}
          <p class="muted">
            Vincule a sua conta do Jellyfin a este perfil pra ver os filmes e as séries do seu servidor, continuar de onde
            parou e ter a sua lista. Cada perfil usa a própria conta.
          </p>
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'you', item: 'profile' })}><Icon name="film" /> Vincular conta do Jellyfin</button>
          </div>
        {/if}
      </section>
    {:else if failed}
      <section class="alert panel" role="alert">
        <span class="chip error">Erro</span>
        <h1 id="watch-title">Sem sinal</h1>
        <p class="muted">{failed}</p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => load(app.watchTab, sort)}><Icon name="resume" /> Tentar de novo</button>
          <button class="btn" data-nav onclick={() => go({ name: 'you', item: 'profile' })}><Icon name="gear" /> Conta do perfil</button>
        </div>
      </section>
    {:else if empty}
      <section class="empty">
        <Icon name={app.watchTab === 'favorites' ? 'star' : 'film'} size="3rem" />
        <h1 id="watch-title">{app.watchTab === 'favorites' ? 'Lista vazia' : app.watchTab === 'lists' ? 'Nenhuma lista com filmes' : 'Nada por aqui'}</h1>
        <p class="muted">
          {app.watchTab === 'favorites'
            ? 'Aperte o botão de favorito num filme ou série pra guardar aqui.'
            : app.watchTab === 'lists'
              ? 'Na página de um filme ou série, use “Adicionar à lista”. As listas ficam no servidor, em todo aparelho da sua conta.'
              : 'O servidor não tem nada desse tipo pra esta conta.'}
        </p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => selectTab('home')}><Icon name="home" /> Voltar ao início</button>
        </div>
      </section>
    {:else if !selected}
      <div class="loading" role="status" aria-busy="true">
        <span class="reel sketched" aria-hidden="true"></span>
        <p class="meta" id="watch-title">Rodando o projetor…</p>
      </div>
    {:else}
      <section class="hero" class:compact={app.watchTab !== 'home'}>
        <p class="kind">
          <span class="chip">{selected.type === 'Series' || selected.type === 'Episode' ? 'Série' : 'Filme'}</span>
          {#if selected.type === 'Episode'}<span class="chip">{episodeTag(selected)}</span>{/if}
        </p>
        {#key selected.id}
          {#if selected.logo}
            <h1 class="logo" id="watch-title"><img src={selected.logo} alt={displayName(selected)} /></h1>
          {:else}
            <h1 class="title" id="watch-title">{displayName(selected)}</h1>
          {/if}
        {/key}
        <p class="meta">
          {#if selected.type === 'Episode'}<span>{selected.name}</span>{/if}
          {#if selected.year}<span>{selected.year}</span>{/if}
          {#if selected.rating}<span class="chip rating">{selected.rating}</span>{/if}
          {#if runtime(selected.runtime) && selected.type !== 'Series'}<span>{runtime(selected.runtime)}</span>{/if}
          {#if remaining(selected)}<span class="left">{remaining(selected)}</span>{/if}
          {#each selected.genres.slice(0, 2) as g (g)}<span>{g}</span>{/each}
          {#if selected.favorite}<span class="fav"><Icon name="star" /> Na lista</span>{/if}
        </p>
        {#if selected.overview && app.watchTab === 'home'}<p class="summary">{selected.overview}</p>{/if}
        {#if app.watchTab === 'home'}
          <div class="actions">
            <!-- 🎬 bilhete: a ação da tela (o anel de foco fica no invólucro) -->
            <span class="ticket">
              <button data-nav data-nav-group="actions" onclick={playSelected}>
                <Icon name={selected.position > 0 && !selected.played ? 'resume' : 'play'} />
                {selected.type === 'Episode' ? (selected.position > 0 ? 'Continuar' : 'Assistir') : playLabel(selected, null)}
              </button>
            </span>
            <button class="btn" data-nav data-nav-group="actions" onclick={() => details(selected)}><Icon name="info" /> Detalhes</button>
            <button class="btn" data-nav data-nav-group="actions" aria-pressed={selected.favorite} onclick={() => toggleFav(selected)}>
              <Icon name={selected.favorite ? 'check' : 'plus'} /> Minha lista
            </button>
          </div>
        {/if}
      </section>

      {#if shelved}
        <div class="shelves">
          {#each rows as row (row.id)}
            <section class="shelf" aria-label={row.title}>
              <h2 class="row-title">{row.title}</h2>
              <ul class="row" data-nav-group={`watch-${row.id}`}>
                {#each row.items as it (it.id)}
                  <li>
                    <button
                      class="card"
                      class:wide={row.wide}
                      data-nav
                      data-nav-inline="center"
                      data-item={it.id}
                      aria-label={row.wide ? wideLabel(it) : cardLabel(it)}
                      onfocus={() => (app.watchSelected = it.id)}
                      onclick={() => open(it)}
                    >
                      <Poster item={it} wide={row.wide} />
                      {#if row.wide}
                        <span class="caption">
                          <span class="name">{displayName(it)}</span>
                          <span class="muted">{caption(it)}</span>
                        </span>
                      {/if}
                    </button>
                  </li>
                {/each}
                {#if row.list}
                  <li>
                    <button class="card more" data-nav data-nav-inline="center" aria-label={`Abrir lista ${row.title}`} onclick={() => go({ name: 'arklist', id: row.list! })}>
                      <Icon name="plus" size="1.6em" /> Abrir lista
                    </button>
                  </li>
                {/if}
              </ul>
            </section>
          {/each}
        </div>
      {:else}
        <div class="toolbar">
          <p class="count meta">{total} {app.watchTab === 'movies' ? `filme${total === 1 ? '' : 's'}` : app.watchTab === 'series' ? `série${total === 1 ? '' : 's'}` : `título${total === 1 ? '' : 's'}`}</p>
          <div class="segmented" role="group" aria-label="Ordem">
            {#each SORTS as s (s.id)}
              <button class="btn small" data-nav aria-pressed={sort === s.id} onclick={() => (sort = s.id)}>{s.label}</button>
            {/each}
          </div>
        </div>
        <ul class="grid" data-nav-group="watch-grid" aria-label={TABS.find((t) => t.id === app.watchTab)?.label}>
          {#each list as it (it.id)}
            <li>
              <button class="card" data-nav data-item={it.id} aria-label={cardLabel(it)} onfocus={() => (app.watchSelected = it.id)} onclick={() => open(it)}>
                <Poster item={it} />
              </button>
            </li>
          {/each}
          {#if list.length < total}
            <li>
              <button class="card more" data-nav onclick={more}><Icon name="plus" size="1.6em" /> Mais</button>
            </li>
          {/if}
        </ul>
      {/if}
    {/if}
  </main>

  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'north', label: 'Minha lista' },
      { button: 'west', label: 'Detalhes' },
      { button: 'l1', label: 'Aba anterior' },
      { button: 'r1', label: 'Próxima aba' },
      { button: 'start', label: 'Você' },
    ]}
  />
</div>

<style>
  .watch {
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
  }
  /* a arte do item em foco fica no fundo, apagada, e some no fundo da sala */
  .backdrop {
    position: fixed;
    inset: 0;
    background-size: cover;
    background-position: center 25%;
    opacity: 0.4;
    z-index: -1;
    mask-image: linear-gradient(to bottom, #000 35%, transparent 90%);
  }
  .scrim {
    position: fixed;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    background:
      linear-gradient(to top, var(--bg) 0%, color-mix(in srgb, var(--bg) 90%, transparent) 35%, transparent 60%),
      linear-gradient(
        90deg,
        var(--bg) 0%,
        color-mix(in srgb, var(--bg) 92%, transparent) 30%,
        color-mix(in srgb, var(--bg) 55%, transparent) 65%,
        color-mix(in srgb, var(--bg) 25%, transparent) 100%
      );
  }
  .top-nav {
    display: flex;
    align-items: center;
    gap: 1.4rem;
    flex-wrap: wrap;
  }
  /* abas: navegação sóbria, sublinhado mostarda no item atual */
  .tabs {
    display: flex;
    gap: 0.4rem;
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
    padding: 1rem 2.5rem 0;
    overflow-y: auto;
  }
  /* 🎬 rolo de filme parado (espera grande da tela): dá uma volta ao
     aparecer e para */
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
  .reel {
    width: 4rem;
    height: 4rem;
    border-radius: 50%;
    box-shadow: var(--hard-sm);
    background:
      radial-gradient(circle at 50% 50%, var(--outline) 0 0.45rem, transparent 0.5rem),
      radial-gradient(circle at 50% 18%, var(--bg) 0 0.42rem, transparent 0.45rem),
      radial-gradient(circle at 80% 40%, var(--bg) 0 0.42rem, transparent 0.45rem),
      radial-gradient(circle at 69% 77%, var(--bg) 0 0.42rem, transparent 0.45rem),
      radial-gradient(circle at 31% 77%, var(--bg) 0 0.42rem, transparent 0.45rem),
      radial-gradient(circle at 20% 40%, var(--bg) 0 0.42rem, transparent 0.45rem),
      var(--surface);
    animation: reel 1.2s steps(12) 1;
  }
  .reel::before {
    border-radius: 50%;
  }
  @keyframes reel {
    to {
      transform: rotate(360deg);
    }
  }
  .hero {
    max-width: 62rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-height: 40vh;
    justify-content: flex-end;
  }
  .hero.compact {
    min-height: 0;
    gap: 0.6rem;
  }
  .kind {
    display: flex;
    gap: 0.5rem;
  }
  .title {
    font-size: clamp(2.2rem, min(5vw, 7vh), 4.6rem);
    max-width: 22ch;
  }
  .compact .title {
    font-size: clamp(1.6rem, 3vw, 2.6rem);
  }
  .logo {
    margin: 0;
  }
  .logo img {
    max-width: min(32rem, 60vw);
    max-height: 9rem;
    object-fit: contain;
    object-position: left bottom;
    filter: drop-shadow(4px 4px 0 var(--n-on-mustard));
    display: block;
  }
  .compact .logo img {
    max-height: 5rem;
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1.2rem;
    color: var(--text);
    font-size: 1rem;
  }
  .rating {
    letter-spacing: 0.04em;
  }
  .left {
    color: var(--accent-2);
  }
  .fav {
    color: var(--accent);
    display: inline-flex;
    gap: 0.3rem;
    align-items: center;
  }
  .summary {
    max-width: 44rem;
    font-size: 1.08rem;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1.2rem;
  }

  /* 🎬 bilhete de cinema: retângulo com um entalhe de cada lado. A máscara
     corta a sombra e o anel de foco, então os dois moram no invólucro. */
  .ticket {
    display: inline-flex;
    border-radius: var(--radii-sm);
    filter: drop-shadow(3px 4px 0 var(--outline));
    transition:
      transform var(--t) var(--ease),
      filter var(--t) var(--ease);
  }
  .ticket:hover,
  .ticket:focus-within {
    transform: translate(-2px, -2px);
    filter: drop-shadow(5px 6px 0 var(--outline));
  }
  .ticket:focus-within {
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .ticket:active {
    transform: translate(3px, 4px);
    filter: none;
  }
  .ticket button {
    --notch: radial-gradient(circle 8px at 0 50%, #0000 97%, #000);
    --notch-end: radial-gradient(circle 8px at 100% 50%, #0000 97%, #000);
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    min-height: 3.2rem;
    padding: 0.75rem 1.7rem;
    font-family: var(--font-label);
    font-size: 0.85rem;
    line-height: 1;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--on-accent);
    background: var(--accent);
    border: none;
    border-radius: var(--radii-sm);
    cursor: pointer;
    -webkit-mask:
      var(--notch) left / 51% 100% no-repeat,
      var(--notch-end) right / 51% 100% no-repeat;
    mask:
      var(--notch) left / 51% 100% no-repeat,
      var(--notch-end) right / 51% 100% no-repeat;
  }
  .ticket button:focus {
    outline: none;
  }
  /* contorno à mão, seguindo os entalhes */
  .ticket button::before {
    content: '';
    position: absolute;
    inset: 0;
    border: var(--edge) solid var(--n-on-mustard);
    border-radius: inherit;
    background:
      radial-gradient(circle 11px at 0 50%, transparent 8px, var(--n-on-mustard) 8.5px 11px, transparent 11.5px),
      radial-gradient(circle 11px at 100% 50%, transparent 8px, var(--n-on-mustard) 8.5px 11px, transparent 11.5px);
    filter: url(#sketch);
    pointer-events: none;
  }

  .shelves {
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
    padding-bottom: 0.5rem;
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
  /* cartão: o pôster é a moldura; no foco ele sobe, a sombra cresce e ganha
     o anel verde-azulado (dá pra ver do sofá) */
  .card {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 0.55rem;
    width: clamp(7.5rem, min(11vw, 17vh), 11rem);
    padding: 0;
    font: inherit;
    color: var(--text);
    text-align: start;
    border: none;
    background: none;
    cursor: pointer;
    scroll-snap-align: center;
  }
  .card.wide {
    width: clamp(13rem, min(20vw, 30vh), 19rem);
  }
  .card:focus,
  .card:focus-visible {
    outline: none;
  }
  .card:hover :global(.poster) {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card:focus :global(.poster) {
    transform: translate(-4px, -4px);
    box-shadow: 8px 8px 0 var(--outline);
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .card:active :global(.poster) {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  .caption {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
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
    letter-spacing: 0.04em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card:focus .caption .name {
    color: var(--accent);
  }
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .count {
    color: var(--muted);
  }
  .segmented {
    display: flex;
    gap: 0.6rem;
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
    aspect-ratio: 2 / 3;
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
  /* vazio: caixa tracejada, sem fundo, sem sombra; diz o porquê e o passo */
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
  .empty p {
    max-width: 44ch;
  }
  /* erro: painel com borda e sombra vermelhas, nunca a tela toda */
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
  :global(.reduce-motion) .reel {
    animation: none;
  }
</style>
