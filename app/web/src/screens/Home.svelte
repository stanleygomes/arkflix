<script lang="ts">
  import { tick } from 'svelte'
  import GameArt from '../components/GameArt.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import HeroVideo from '../components/HeroVideo.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import GameMenu from '../components/GameMenu.svelte'
  import ConsoleIcon from '../components/ConsoleIcon.svelte'
  import Sections from '../components/Sections.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { play } from '../lib/sound'
  import { toggleFavorite } from '../lib/actions'
  import { downloadGame, downloadText, downloads } from '../lib/downloads.svelte'
  import { indicator } from '../lib/indicator'
  import { shade } from '../lib/motion'
  import { pickRandom, sortGames } from '../lib/library'
  import { sortOf, sourceGames, sourceTitle } from '../lib/sources'
  import { app, go, isKids, shown, system, toast } from '../lib/state.svelte'
  import type { Game, GameCheevos, SaveState } from '../lib/types'
  import { ago, playTime } from '../lib/format'
  import { loadArkLists } from '../lib/ark'

  // ocultos ficam fora do Início (a página do jogo ainda abre). Duas visões,
  // como o Games/Media do PS5: Biblioteca (o que está neste PC) e Catálogo (o que está
  // no servidor pra baixar). Perfil infantil não vê o Catálogo.
  const store = $derived(app.homeView === 'store' && !isKids())
  const remoteCount = $derived(shown().filter((g) => g.remote).length)
  const visible = $derived(shown().filter((g) => (store ? g.remote : !g.remote)))

  function setView(v: 'games' | 'store') {
    if (app.homeView === v) return
    app.homeView = v
    app.homeTab = 'home'
    app.selected = ''
  }

  // abas: só o Início e os consoles. Coleções e listas viram fileiras do Início.
  type Tab = { slug: string; short: string; name: string; kind: 'home' | 'collection' | 'list' | 'system' }
  const tabs = $derived<Tab[]>([
    { slug: 'home', short: 'Início', name: 'Início', kind: 'home' },
    ...app.systems
      .filter((s) => visible.some((g) => g.system === s.slug))
      .map((s) => ({ slug: s.slug, short: s.short, name: s.name, kind: 'system' as const })),
  ])

  type Row = { id: string; title: string; games: Game[]; source?: string; smart?: boolean; server?: boolean }

  // listas do servidor arkflix com jogos viram fileiras do Início
  $effect(() => {
    if (app.ark?.games) loadArkLists().catch(() => {})
  })
  const rows = $derived.by((): Row[] => {
    if (app.homeTab.startsWith('collection:') || app.homeTab.startsWith('list:')) {
      const games = sortGames(sourceGames(app.homeTab), sortOf(app.homeTab))
      return [{ id: app.homeTab, title: `${sourceTitle(app.homeTab)} · ${games.length} jogo${games.length === 1 ? '' : 's'}`, games, source: app.homeTab }]
    }
    if (app.homeTab !== 'home') {
      const sys = system(app.homeTab)
      const games = sortGames(
        visible.filter((g) => g.system === app.homeTab),
        sortOf(`system:${app.homeTab}`),
      )
      return [{ id: app.homeTab, title: `${sys?.name ?? ''} · ${games.length} jogo${games.length === 1 ? '' : 's'}${store ? ' pra baixar' : ''}`, games, source: `${store ? 'store' : 'system'}:${app.homeTab}` }]
    }
    if (store) {
      // Catálogo: tudo do servidor e uma fileira por console
      const out: Row[] = [{ id: 'store-all', title: `No servidor · ${visible.length} jogo${visible.length === 1 ? '' : 's'}`, games: sortGames(visible, 'name').slice(0, 30), source: 'store:' }]
      for (const s of app.systems) {
        const games = sortGames(visible.filter((g) => g.system === s.slug), 'name')
        if (games.length) out.push({ id: `store-${s.slug}`, title: `${s.name} · ${games.length}`, games: games.slice(0, 20), source: `store:${s.slug}` })
      }
      return out
    }
    const out: Row[] = []
    const recent = visible.filter((g) => g.lastPlayed > 0).sort((a, b) => b.lastPlayed - a.lastPlayed).slice(0, 12)
    if (recent.length) out.push({ id: 'recent', title: 'Continuar jogando', games: recent, source: 'collection:recent' })
    const favs = visible.filter((g) => g.favorite)
    if (favs.length) out.push({ id: 'favorites', title: 'Favoritos', games: favs, source: 'collection:favorites' })
    for (const l of app.lists) {
      const games = sourceGames(`list:${l.id}`)
      if (games.length) out.push({ id: `list-${l.id}`, title: l.name, games, source: `list:${l.id}`, smart: !!l.rule })
    }
    if (app.ark?.games) {
      const here = new Set(visible.map((g) => g.id))
      for (const l of app.arkLists) {
        const games = sourceGames(`arklist:${l.id}`).filter((g) => here.has(g.id))
        if (games.length) out.push({ id: `ark-${l.id}`, title: l.name, games, source: `arklist:${l.id}`, server: true })
      }
    }
    // a biblioteca toda, sempre por último (ou sozinha, no começo)
    const all = sortGames(visible, sortOf('collection:all'))
    out.push({ id: 'all', title: `Sua biblioteca · ${visible.length} jogo${visible.length === 1 ? '' : 's'}`, games: all.slice(0, 30), source: 'collection:all' })
    return out
  })

  const selected = $derived(visible.find((g) => g.id === app.selected) ?? rows[0]?.games[0])
  const selSystem = $derived(selected ? system(selected.system) : undefined)
  const canResume = $derived(!!selected && selected.lastPlayed > 0 && selSystem?.emulator === 'retroarch')

  // cartões de atividade do destaque: o ponto salvo automático (com a
  // miniatura) e as conquistas. Busca com atraso: o destaque muda a cada passo.
  let autoState = $state<SaveState | null>(null)
  let trophies = $state<GameCheevos | null>(null)
  $effect(() => {
    const g = selected
    autoState = null
    trophies = null
    if (!g || g.remote) return
    const t = setTimeout(() => {
      if (g.hasSaves) api.saves(g.id).then((l) => selected?.id === g.id && (autoState = l.states.find((s) => s.slot === -1) ?? null)).catch(() => {})
      api.gameCheevos(g.id).then((c) => selected?.id === g.id && (trophies = c)).catch(() => {})
    }, 450)
    return () => clearTimeout(t)
  })
  const nextTrophy = $derived(trophies?.list?.find((c) => !c.unlocked))

  function selectTab(slug: string) {
    if (app.homeTab === slug) return
    app.homeTab = slug
    app.selected = ''
  }

  async function launch(g: Game | undefined, state = '') {
    if (!g) return
    try {
      play('start')
      await api.play(g.id, state)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  const toggleFav = (g: Game | undefined) => toggleFavorite(g?.id)

  function focusedGame(): Game | undefined {
    const id = current()?.dataset.game
    return id ? visible.find((g) => g.id === id) : selected
  }

  let lastRandom = ''
  let spinning = $state(false)

  // Surpresa: roleta rápida pelas capas e para num jogo da aba atual
  async function surprise() {
    if (spinning) return
    const pool = app.homeTab === 'home' ? visible : rows.flatMap((r) => r.games)
    const pick = pickRandom(pool, lastRandom)
    if (!pick) return
    lastRandom = pick.id
    spinning = true
    if (!app.settings?.a11y.reduceMotion) {
      for (let i = 0; i < 8; i++) {
        app.selected = pickRandom(pool, app.selected)?.id ?? pick.id
        play('move')
        await new Promise((r) => setTimeout(r, 70 + i * 12))
      }
    }
    spinning = false
    play('start')
    toast(`Surpresa: ${pick.name}`, 'info', 3000)
    const card = document.querySelector<HTMLElement>(`.shelf [data-game="${pick.id}"]`)
    if (card) {
      app.selected = pick.id
      focusEl(card)
    } else {
      go({ name: 'game', id: pick.id })
    }
  }

  function cycleTab(delta: number) {
    const i = tabs.findIndex((t) => t.slug === app.homeTab)
    const next = tabs[(i + delta + tabs.length) % tabs.length]
    selectTab(next.slug)
    tick().then(() => focusEl(document.querySelector<HTMLElement>(`[data-tab="${next.slug}"]`)))
    play('move')
  }

  $effect(() =>
    pushHandler((a) => {
      const inTabs = !!current()?.closest('.tabs')
      switch (a) {
        case 'down':
          if (inTabs) {
            // das abas direto pra fileira de jogos (o destaque fica no meio)
            const card = document.querySelector<HTMLElement>('.shelf [data-game]')
            if (card) {
              focusEl(document.querySelector<HTMLElement>(`.shelf [data-game="${app.selected}"]`) ?? card)
              play('move')
              return true
            }
          }
          return false
        case 'up':
          if (current()?.closest('.shelf')) {
            const playBtn = document.querySelector<HTMLElement>('.hero .actions [data-nav]')
            if (playBtn) {
              focusEl(playBtn)
              play('move')
              return true
            }
          }
          return false
        case 'fav':
          toggleFav(focusedGame())
          return true
        case 'details': {
          const g = focusedGame()
          if (g) go({ name: 'game', id: g.id })
          return true
        }
        case 'menu': {
          // Start num jogo = menu do jogo (como o Options do PS5); fora dele, Você
          const id = current()?.dataset.game
          if (id) menuFor = id
          else go({ name: 'you' })
          return true
        }
        case 'random':
          surprise()
          return true
        case 'prevTab':
          cycleTab(-1)
          return true
        case 'nextTab':
          cycleTab(1)
          return true
      }
      return false
    }),
  )

  // menu do jogo em foco (Start/Options)
  let menuFor = $state('')

  const hero = $derived(selected?.cover ?? '')

  // selo "NOVO": jogo que entrou na biblioteca nesta semana (um por tela)
  const WEEK = 7 * 24 * 3600
  const fresh = $derived(!!selected && !selected.remote && selected.added > Date.now() / 1000 - WEEK)
</script>

<div class="home">
  {#key hero}
    <div class="backdrop" in:shade style:background-image={hero ? `url("${hero}")` : 'none'} aria-hidden="true"></div>
    <div class="scrim" aria-hidden="true"></div>
  {/key}
  <HeroVideo game={selected} />
  <TopBar>
    <div class="top-nav">
    <Sections current="games" />
    {#if !isKids()}
      <span class="divider" aria-hidden="true"></span>
      <nav class="views" aria-label="Visão">
        <button class="view" data-nav data-view="games" aria-current={!store ? 'page' : undefined} onclick={() => setView('games')}>Biblioteca</button>
        <button class="view" data-nav data-view="store" aria-current={store ? 'page' : undefined} onclick={() => setView('store')}>
          Catálogo{#if remoteCount}<span class="count">{remoteCount}</span>{/if}
        </button>
      </nav>
    {/if}
    </div>
  </TopBar>
  <!-- consoles: blocos com a silhueta e o nome curto; o escolhido ganha o
       traço mostarda embaixo (lib/indicator.ts), que pula em degraus -->
  <nav class="tabs" aria-label="Consoles" use:indicator={app.homeTab}>
    {#each tabs as t (t.slug)}
      <button
        class="tab"
        class:first-system={t.kind === 'system' && tabs.find((x) => x.kind === 'system') === t}
        data-nav
        data-tab={t.slug}
        data-nav-group="tabs"
        data-nav-default={t.slug === app.homeTab ? '' : undefined}
        aria-current={t.slug === app.homeTab ? 'page' : undefined}
        onfocus={() => selectTab(t.slug)}
        onclick={() => selectTab(t.slug)}
      >
<span class="short">{t.short}</span>{#if t.name !== t.short}<span class="sr-only">{` (${t.name})`}</span>{/if}
        <ConsoleIcon slug={t.slug} size="2.1rem" />
      </button>
    {/each}
  </nav>

  <main class="content" aria-labelledby="hero-title">
    {#if visible.length === 0 && store}
      <section class="empty panel">
        <h1 id="hero-title">Catálogo vazio</h1>
        <p class="muted">O Catálogo mostra os jogos do seu servidor RomM que ainda não estão neste PC. Conecte o servidor (ou já baixou tudo).</p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'you', item: 'server' })}><Icon name="cloud" /> Servidor RomM</button>
          <button class="btn" data-nav onclick={() => setView('games')}><Icon name="pad" /> Ir pra Biblioteca</button>
        </div>
      </section>
    {:else if visible.length === 0}
      <section class="empty panel">
        <h1 id="hero-title">Nenhum jogo ainda</h1>
        <p class="muted">{remoteCount ? `Tem ${remoteCount} jogo${remoteCount === 1 ? '' : 's'} no servidor esperando: baixe pelo Catálogo.` : 'Importe ROMs ou aponte suas pastas de jogos nas configurações.'}</p>
        <div class="actions">
          {#if remoteCount && !isKids()}<button class="btn primary" data-nav data-nav-default onclick={() => setView('store')}><Icon name="download" /> Abrir o Catálogo</button>{/if}
          <button class="btn" class:primary={!remoteCount} data-nav data-nav-default={remoteCount ? undefined : ''} onclick={() => go({ name: 'import' })}><Icon name="plus" /> Importar jogos</button>
          <button class="btn" data-nav onclick={() => go({ name: 'you' })}><Icon name="gear" /> Configurações</button>
        </div>
      </section>
    {:else if selected}
      <div class="stage">
      <section class="hero">
        <p class="system-pill">
          <span class="chip">{selSystem?.name ?? selected.system}</span>{#if fresh}<span class="stamp fresh">Novo</span>{/if}
        </p>
        {#key selected.id}
          <!-- título de cartaz: troca em dois degraus quando o destaque muda -->
          <h1 class="title swap" id="hero-title">{selected.name}</h1>
        {/key}
        <p class="meta">
          {#if selected.year}<span>{selected.year}</span>{/if}
          {#if selected.players}<span>{selected.players} jogador{selected.players === '1' ? '' : 'es'}</span>{/if}
          {#each selected.genres.slice(0, 2) as genre (genre)}<span>{genre}</span>{/each}
          {#if selected.favorite}<span class="fav"><Icon name="star" /> Favorito</span>{/if}
        </p>
        {#if selected.summary}<p class="summary">{selected.summary}</p>{/if}
        <!-- botões e cartões (continuar, conquistas) na mesma linha: altura fixa,
             o destaque troca a cada passo e as fileiras não podem pular -->
        <div class="cta">
        <div class="actions">
          {#if selected.remote}
            <button class="btn primary" data-nav data-nav-group="actions" disabled={downloads[selected.id] !== undefined || isKids()} onclick={() => downloadGame(selected)}>
              <Icon name="download" />
              {downloads[selected.id] !== undefined ? downloadText(selected.id) : 'Baixar do servidor'}
            </button>
          {:else}
            <button class="btn primary" data-nav data-nav-group="actions" onclick={() => launch(selected)}>
              <Icon name="play" /> Jogar
            </button>
          {/if}
          {#if canResume && !selected.remote && !autoState}
            <button class="btn" data-nav data-nav-group="actions" onclick={() => launch(selected, 'auto')}>
              <Icon name="resume" /> Continuar
            </button>
          {/if}
          <button class="btn" data-nav data-nav-group="actions" onclick={() => go({ name: 'game', id: selected.id })}>
            <Icon name="disc" /> Detalhes
          </button>
        </div>
        <div class="cards" data-nav-group="cards">
            {#if autoState && canResume}
              <button class="card-act" data-nav onclick={() => launch(selected, 'auto')}>
                {#if autoState.thumb}<img src={`/media/thumb/${autoState.id}`} alt="" />{:else}<span class="nothumb" aria-hidden="true">AUTO</span>{/if}
                <span class="text">
                  <span class="label">Continuar de onde parou</span>
                  <span class="muted">salvo {ago(autoState.modified)}{selected.playTime ? ` · ${playTime(selected.playTime)} jogados` : ''}</span>
                </span>
              </button>
            {/if}
            {#if trophies?.supported && trophies.total}
              <button class="card-act" data-nav onclick={() => go({ name: 'game', id: selected.id })}>
                <span class="trophy"><Icon name="trophy" size="1.8em" /></span>
                <span class="text">
                  <span class="label">{trophies.unlocked}/{trophies.total} conquistas</span>
                  <span class="muted">{nextTrophy ? `próxima: ${nextTrophy.title}` : 'todas ganhas!'}</span>
                </span>
              </button>
            {/if}
        </div>
        </div>
      </section>
      <div class="hero-art" aria-hidden="true">
        {#key selected.id}<GameArt game={selected} system={selSystem} large />{/key}
      </div>
      </div>

      <div class="shelves">
        {#each rows as row (row.id)}
          <section class="shelf" aria-label={row.title}>
            <h2 class="row-title">
              {row.title}{#if row.smart}<span class="smart" title="Lista automática"><Icon name="auto" size="0.8em" /><span class="sr-only">(lista automática)</span></span>{/if}{#if row.server}<span class="smart" title="Lista do servidor"><Icon name="cloud" size="0.8em" /><span class="sr-only">(lista do servidor)</span></span>{/if}
            </h2>
            <ul class="row" data-nav-group={`row-${row.id}`}>
              {#each row.games as g (g.id)}
                {@const sys = system(g.system)}
                <li>
                  <button
                    class="card"
                    class:selected={g.id === selected.id}
                    data-nav
                    data-nav-inline="center"
                    data-game={g.id}
                    aria-label={`${g.name}, ${sys?.short ?? g.system}${g.year ? `, ${g.year}` : ''}${g.favorite ? ', favorito' : ''}${g.remote ? ', no servidor' : ''}`}
                    onfocus={() => (app.selected = g.id)}
                    onclick={() => go({ name: 'game', id: g.id })}
                  >
                    <GameArt game={g} system={sys} />
                    {#if g.favorite}<span class="badge" aria-hidden="true"><Icon name="star" /></span>{/if}
                    {#if g.remote}<span class="badge cloud" aria-hidden="true"><Icon name="cloud" /></span>{/if}
                  </button>
                </li>
              {/each}
              {#if row.source}
                <li>
                  <button class="card see-all" data-nav data-nav-inline="center" onclick={() => go({ name: 'library', source: row.source! })}>
                    <Icon name="sliders" size="1.6em" /> Ver tudo
                  </button>
                </li>
              {/if}
            </ul>
          </section>
        {/each}
      </div>
    {/if}
  </main>

  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'north', label: 'Favorito' },
      { button: 'west', label: 'Detalhes' },
      { button: 'l1', label: 'Console anterior' },
      { button: 'r1', label: 'Próximo console' },
      { button: 'select', label: 'Buscar' },
      { button: 'r3', label: 'Surpresa' },
      { button: 'start', label: 'Opções do jogo · Você' },
    ]}
  />
</div>
{#if menuFor}<GameMenu gameId={menuFor} onclose={() => (menuFor = '')} />{/if}

<style>
  .home {
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
  }
  /* capa do jogo em destaque, bem apagada no fundo (só clima) */
  .backdrop {
    position: fixed;
    inset: 0;
    background-size: cover;
    background-position: center 30%;
    filter: blur(3px);
    opacity: 0.22;
    transform: scale(1.05);
    z-index: -1;
    mask-image: linear-gradient(to bottom, #000 30%, transparent 85%);
  }
  /* contraste: o fundo apaga onde tem texto (esquerda e fileiras) */
  .scrim {
    position: fixed;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    background:
      linear-gradient(to top, var(--bg) 0%, color-mix(in srgb, var(--bg) 88%, transparent) 30%, transparent 55%),
      linear-gradient(90deg, var(--bg) 0%, color-mix(in srgb, var(--bg) 90%, transparent) 30%, color-mix(in srgb, var(--bg) 55%, transparent) 62%, color-mix(in srgb, var(--bg) 25%, transparent) 100%);
  }

  /* barra de cima: seções | visões, nav sóbria (traço mostarda no atual) */
  /* três seções + as visões não cabem numa linha: quebra como em filmes */
  .top-nav {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0 1.4rem;
  }
  .divider {
    display: none;
    width: var(--edge);
    height: 1.8rem;
    background: color-mix(in srgb, var(--outline) 30%, transparent);
  }
  .views {
    display: flex;
    gap: 0.4rem;
  }
  .view {
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    white-space: nowrap;
    color: var(--muted);
    background: none;
    border: none;
    border-bottom: var(--edge) solid transparent;
    border-radius: 0;
    padding: 0.55rem 0.55rem 0.45rem;
    cursor: pointer;
    display: inline-flex;
    gap: 0.5rem;
    align-items: center;
    transition:
      color var(--t) var(--ease),
      border-color var(--t) var(--ease);
  }
  .view:hover {
    color: var(--text);
    border-bottom-color: var(--outline);
  }
  .view[aria-current='page'] {
    color: var(--text);
    border-bottom-color: var(--accent);
  }
  /* contagem: badge "count" (tinta creme, número em mono) */
  .view .count {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 0.85rem;
    line-height: 1;
    min-width: 1.4rem;
    text-align: center;
    color: var(--bg);
    background: var(--text);
    border-radius: 4px;
    padding: 0.2rem 0.3rem;
  }

  /* consoles: blocos sóbrios, silhueta + nome curto */
  .tabs {
    position: relative;
    display: flex;
    align-items: stretch;
    gap: 0.9rem;
    overflow-x: auto;
    scrollbar-width: none;
    padding: 1rem 2.5rem 1.5rem;
  }
  .tab {
    flex: none;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.3rem;
    min-width: 5.4rem;
    padding: 0.5rem 0.8rem 0.45rem;
    color: var(--muted);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease),
      background var(--t) var(--ease);
  }
  .tab .short {
    order: 2;
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    line-height: 1.1;
    color: var(--text);
    white-space: nowrap;
  }
  /* console escolhido: silhueta mostarda (destaque) e fundo de balão */
  .tab[aria-current='page'] {
    color: var(--accent);
    background: var(--balloon);
  }
  .tab:hover,
  .tab:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .tab.first-system {
    margin-inline-start: 0.8rem;
  }
  /* traço mostarda debaixo do console escolhido (lib/indicator.ts): pula
     pro novo console em dois degraus, uma vez por troca */
  .tabs :global(.tab-light) {
    position: absolute;
    left: 0;
    top: 0.95rem;
    border-bottom: 4px solid var(--accent);
    transition:
      transform var(--t) steps(2),
      width var(--t) steps(2),
      height var(--t) steps(2);
    pointer-events: none;
  }

  .content {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 0;
    padding: 0.6rem 2.5rem 0;
    overflow-y: auto;
  }
  .stage {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 2rem;
    /* nunca encolhe abaixo do destaque: com várias fileiras, a tela rola
       (o foco leva a fileira até a vista) em vez de uma cobrir a outra */
    flex: 1 0 auto;
  }
  .hero {
    max-width: 68rem;
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
  }
  /* console do destaque: etiqueta datilografada; o NOVO é o único carimbo */
  .system-pill {
    display: flex;
    align-items: center;
    gap: 1.4rem;
  }
  .system-pill .chip {
    font-size: 0.9rem;
    min-height: 1.9rem;
    padding: 0.35rem 0.65rem 0.2rem;
  }
  .stamp.fresh {
    color: var(--accent);
    border-color: var(--accent);
    font-size: 0.95rem;
    animation: thunk 240ms steps(3) both;
  }
  @keyframes thunk {
    from {
      transform: scale(1.5) rotate(-8deg);
      opacity: 0;
    }
  }
  /* nome do jogo em letra de cartaz (h1 do app.css), grande */
  .title {
    font-size: clamp(2.2rem, min(5vw, 7vh), 4.6rem);
    line-height: 1.05;
    max-width: 24ch;
    margin: 0.1rem 0 0.2rem;
  }
  .swap {
    animation: title-in 160ms steps(2) both;
  }
  @keyframes title-in {
    from {
      opacity: 0;
    }
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem 1.6rem;
    color: var(--text);
    font-size: 1rem;
  }
  .meta .fav {
    color: var(--accent);
    display: inline-flex;
    gap: 0.35rem;
    align-items: center;
  }
  .summary {
    max-width: 44rem;
    color: var(--text);
    font-size: 1.08rem;
    line-height: 1.6;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 1.2rem;
  }
  .cta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1.2rem 2rem;
    min-height: 4.8rem;
    margin-top: 0.4rem;
  }
  .cards {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
  /* item sóbrio clicável: miniatura + texto */
  .card-act {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.8rem;
    align-items: center;
    max-width: 22rem;
    padding: 0.45rem 0.9rem 0.45rem 0.45rem;
    font: inherit;
    color: var(--text);
    text-align: start;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .card-act:hover,
  .card-act:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card-act img,
  .card-act .nothumb,
  .card-act .trophy {
    width: 5.5rem;
    height: 3.6rem;
    object-fit: cover;
    display: grid;
    place-items: center;
    background: var(--bg);
    border: 2px solid var(--outline);
    border-radius: 4px;
    color: var(--accent);
  }
  .card-act .nothumb {
    font-family: var(--font-hud);
    letter-spacing: 0.1em;
    color: var(--muted);
  }
  .card-act .text {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
  }
  .card-act .label {
    font-size: 0.8rem;
  }
  .card-act .muted {
    font-size: 0.85rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  /* capa do destaque, reta, com a sombra dura de cartaz */
  .hero-art {
    width: clamp(8rem, min(13vw, 22vh), 13rem);
    flex: none;
    margin-top: 1vh;
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-cartoon);
    animation: rise 240ms steps(3) both;
  }
  @keyframes rise {
    from {
      transform: translateY(12px);
      opacity: 0;
    }
  }
  @media (max-height: 820px) {
    .hero .cta {
      min-height: 0;
    }
    .hero-art {
      display: none;
    }
    .tabs {
      padding-block: 0.7rem 1.3rem;
    }
    .tabs :global(.tab-light) {
      top: 0.65rem;
    }
    /* TV 16:9 (1920×1080 ≈ 1440×810): consoles numa linha só (ícone ao
       lado do nome) e destaque mais junto, pra fileira caber inteira */
    .tab {
      flex-direction: row;
      gap: 0.45rem;
      min-width: 0;
      padding: 0.45rem 0.75rem;
    }
    .hero {
      gap: 0.7rem;
    }
    .cta {
      margin-top: 0;
    }
    .shelves {
      padding-top: 0.4rem;
    }
    .row {
      padding-block: 0.7rem 1.2rem;
    }
    /* "continuar de onde parou" na altura dos botões: não empurra a fileira */
    .card-act {
      max-width: none;
      padding-block: 0.3rem;
    }
    .card-act img,
    .card-act .nothumb,
    .card-act .trophy {
      width: 3.8rem;
      height: 2.4rem;
    }
    .card-act .label {
      white-space: nowrap;
    }
  }

  /* fileiras: cartões de imagem sóbrios e clicáveis */
  .shelves {
    display: flex;
    flex-direction: column;
    gap: 1.6rem;
    padding: 1rem 0 0.5rem;
  }
  .row-title {
    font-size: 1.1rem;
    color: var(--text);
    margin-bottom: 0.2rem;
  }
  .row-title .smart {
    margin-inline-start: 0.5rem;
    color: var(--muted);
  }
  .row {
    list-style: none;
    margin: 0;
    padding: 0.9rem 0.6rem 1.5rem;
    display: flex;
    gap: 1.6rem;
    overflow-x: auto;
    scroll-snap-type: x proximity;
    scrollbar-width: none;
  }
  .card {
    /* capa grande (é o que a pessoa procura), mas cabendo na altura da TV */
    width: clamp(7.5rem, min(12vw, 17vh), 11.5rem);
    padding: 0;
    border: none;
    background: none;
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    cursor: pointer;
    scroll-snap-align: center;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .card:hover,
  .card:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card:active {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  /* o jogo do destaque (foco em outro lugar): traço mostarda embaixo */
  .card.selected:not(:focus)::after {
    content: '';
    position: absolute;
    inset-inline: 0;
    bottom: -0.85rem;
    border-bottom: 4px solid var(--accent);
    pointer-events: none;
  }
  .see-all {
    aspect-ratio: 3 / 4;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--text);
    background: transparent;
    border: var(--edge) dashed var(--muted);
    box-shadow: none;
  }
  .see-all:hover,
  .see-all:focus {
    background: var(--surface);
    border-style: solid;
    border-color: var(--outline);
  }
  /* etiquetas no canto da capa: favorito (mostarda) e no servidor (info) */
  .badge {
    position: absolute;
    top: 0.45rem;
    inset-inline-start: 0.45rem;
    color: var(--on-accent);
    background: var(--accent);
    border: 2px solid var(--on-accent);
    border-radius: 4px;
    padding: 0.2rem;
    display: grid;
    place-items: center;
  }
  .badge.cloud {
    inset-inline-start: auto;
    inset-inline-end: 0.45rem;
    color: var(--bg);
    background: var(--accent-2);
    border-color: var(--bg);
  }
  .empty {
    margin: auto;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 36rem;
  }
  .empty h1 {
    font-size: 2.6rem;
  }
</style>
