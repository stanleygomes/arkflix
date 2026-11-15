<script lang="ts">
  import GameGrid from '../components/GameGrid.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Keyboard from '../components/Keyboard.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { focusedGameId, toggleFavorite } from '../lib/actions'
  import { pushHandler } from '../lib/input'
  import { search } from '../lib/library'
  import { focusEl } from '../lib/nav'
  import { app, go, isKids, shown } from '../lib/state.svelte'
  import { shortcuts, type Shortcut } from '../lib/shortcuts'
  import Icon from '../components/Icon.svelte'
  import Poster from '../components/Poster.svelte'
  import { api } from '../lib/api'
  import type { WatchItem } from '../lib/types'
  import { cardLabel } from '../lib/watch'

  let query = $state('')
  const results = $derived(search(shown(), query))
  const extra = $derived(
    shortcuts(query, {
      systems: app.systems,
      withGames: new Set(shown().map((g) => g.system)),
      lists: app.lists,
      kids: isKids(),
      desktop: app.mode === 'desktop',
    }),
  )
  const icons = { console: 'pad', list: 'star', setting: 'gear' } as const

  // filmes e séries do Jellyfin do perfil (busca no servidor, com atraso)
  let videos = $state<WatchItem[]>([])
  $effect(() => {
    const q = query.trim()
    if (!app.watch?.connected || q.length < 2) {
      videos = []
      return
    }
    const t = setTimeout(() => {
      api.watchSearch(q).then((r) => query.trim() === q && (videos = r)).catch(() => (videos = []))
    }, 300)
    return () => clearTimeout(t)
  })

  function openShortcut(s: Shortcut) {
    if (s.power) app.power = true
    else if (s.go) go(s.go)
  }
  // quem usa teclado físico começa no campo; quem usa controle, no teclado na tela
  const typing = app.family === 'keyboard'

  // F/Y favorita o resultado em foco (nos campos de texto o F digita)
  $effect(() =>
    pushHandler((a) => {
      if (a === 'fav' && focusedGameId()) {
        toggleFavorite(focusedGameId())
        return true
      }
      return false
    }),
  )

  function toResults() {
    // jogo primeiro (é o que mais se busca); sem jogo, o primeiro atalho
    focusEl(document.querySelector<HTMLElement>('.results [data-game]') ?? document.querySelector<HTMLElement>('.results [data-nav]'))
  }
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="layout">
    <section class="query" aria-labelledby="search-title">
      <h1 id="search-title">Buscar</h1>
      <label class="sr-only" for="search-input">Buscar jogo, console ou configuração</label>
      <input
        id="search-input"
        type="text"
        data-nav
        data-nav-default={typing ? '' : undefined}
        bind:value={query}
        placeholder={app.watch?.connected ? "Jogo, filme, série ou configuração" : "Jogo, console ou configuração"}
        autocomplete="off"
        spellcheck="false"
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            // sem isso o mesmo Enter "clica" no resultado que acabou de receber o foco
            e.preventDefault()
            toResults()
          }
        }}
      />
      <Keyboard bind:value={query} onsubmit={toResults} submitLabel="Ver resultados" autofocus={!typing} />
    </section>
    <section class="results" aria-live="polite" aria-labelledby="results-count">
      <p id="results-count" class="count">
        {#if !query.trim()}
          Jogo, console, lista ou configuração ("bluetooth", "tela cheia"). Acento e maiúscula não importam.
        {:else if results.length === 0 && extra.length === 0 && videos.length === 0}
          Nada encontrado pra "{query}".
        {:else}
          {results.length} jogo{results.length === 1 ? '' : 's'}{videos.length ? `, ${videos.length} filme${videos.length === 1 ? '' : 's'} ou série${videos.length === 1 ? '' : 's'}` : ''}{extra.length ? ` e ${extra.length} atalho${extra.length === 1 ? '' : 's'}` : ''}
        {/if}
      </p>
      {#if extra.length}
        <ul class="shortcuts" aria-label="Consoles, listas e configurações">
          {#each extra as s (s.id)}
            <li>
              <button class="shortcut" data-nav data-shortcut={s.kind} onclick={() => openShortcut(s)}>
                <Icon name={icons[s.kind]} />
                <span class="label">{s.label}</span>
                <span class="hint muted">{s.hint}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
      {#if results.length}
        <GameGrid games={results} label="Resultados da busca" />
      {/if}
      {#if videos.length}
        <h2 class="sub-title">Filmes e séries</h2>
        <ul class="videos" aria-label="Filmes e séries encontrados">
          {#each videos as v (v.id)}
            <li>
              <button class="video" data-nav data-item={v.id} aria-label={cardLabel(v)} onclick={() => go({ name: 'title', id: v.id })}>
                <Poster item={v} />
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Digitar / abrir' }, { button: 'east', label: 'Voltar' }]} />
</div>

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 2.5rem;
    padding: 0.5rem 2.5rem 1rem 3.5rem;
  }
  .query {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  h1 {
    font-size: 2.6rem;
  }
  .results {
    overflow-y: auto;
    min-height: 0;
    padding-top: 0.4rem;
  }
  .shortcuts {
    list-style: none;
    margin: 0 0 1.2rem;
    padding: 0.4rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
  }
  /* atalho: item sóbrio clicável (ícone, nome, dica datilografada) */
  .shortcut {
    display: grid;
    grid-template-columns: auto auto;
    gap: 0.2rem 0.7rem;
    align-items: center;
    padding: 0.6rem 1rem 0.55rem 0.8rem;
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
  .shortcut:hover,
  .shortcut:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .shortcut :global(svg) {
    grid-row: span 2;
    width: 1.5em;
    height: 1.5em;
    color: var(--accent);
  }
  .shortcut .label {
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.04em;
  }
  .shortcut .hint {
    font-family: var(--font-hud);
    font-size: 0.8rem;
  }
  .sub-title {
    font-size: 1.1rem;
    margin: 1.4rem 0 0.4rem;
  }
  .videos {
    list-style: none;
    margin: 0;
    padding: 0.5rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(8rem, 1fr));
    gap: 1.2rem;
  }
  .video {
    width: 100%;
    padding: 0;
    border: none;
    background: none;
    cursor: pointer;
    border-radius: 6px;
  }
  .count {
    font-family: var(--font-hud);
    letter-spacing: 0.03em;
    color: var(--muted);
    margin-bottom: 0.8rem;
  }
</style>
