<script lang="ts">
  // Página de um filme ou série: fundo, logo, descrição e ações. Série mostra
  // as temporadas (abas) e os episódios; o botão principal continua o próximo.
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import Poster from '../components/Poster.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, go, toast } from '../lib/state.svelte'
  import type { WatchDetails, WatchItem } from '../lib/types'
  import { episodeTag, playLabel, remaining, runtime } from '../lib/watch'

  let { id }: { id: string } = $props()

  let data = $state<WatchDetails | null>(null)
  let episodes = $state<WatchItem[]>([])
  let season = $state('')
  let failed = $state('')

  $effect(() => {
    if (app.settings?.a11y.highContrast) return applyPalette(highContrastPalette)
    applyPalette(homePalette)
  })

  async function load() {
    try {
      data = await api.watchDetails(id)
      failed = ''
      if (data.item.type === 'Series' && data.seasons.length) {
        const start = data.next?.seasonId ?? ''
        await pickSeason(data.seasons.find((s) => s.id === start)?.id ?? data.seasons[0].id)
      }
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    void id
    load()
  })

  async function pickSeason(sid: string) {
    if (!data) return
    season = sid
    try {
      episodes = await api.watchEpisodes(data.item.id, sid)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  const item = $derived(data?.item)

  function playMain(fromStart = false) {
    if (!item) return
    const target = item.type === 'Series' ? data?.next : item
    if (!target) return
    go({ name: 'player', id: target.id, start: fromStart ? 0 : -1 })
  }

  async function toggleFav() {
    if (!item || !data) return
    try {
      const next = await api.watchFavorite(item.id, !item.favorite)
      data.item = { ...data.item, favorite: next.favorite }
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function togglePlayed(it: WatchItem) {
    try {
      const next = await api.watchPlayed(it.id, !it.played)
      if (data && it.id === data.item.id) data.item = { ...data.item, played: next.played, position: next.position, progress: next.progress }
      episodes = episodes.map((e) => (e.id === it.id ? next : e))
      play('toggle')
      toast(next.played ? `${it.name}: assistido` : `${it.name}: não assistido`, 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() =>
    pushHandler((a) => {
      if (a === 'fav') {
        toggleFav()
        return true
      }
      if (a === 'details') {
        // no episódio em foco: marca como assistido; fora dele, o título
        const ep = episodes.find((e) => e.id === current()?.dataset.episode)
        if (ep) togglePlayed(ep)
        else if (item && item.type === 'Movie') togglePlayed(item)
        return true
      }
      return false
    }),
  )

  const hero = $derived(item?.backdrop || item?.poster || '')
</script>

<div class="page">
  <div class="backdrop" style:background-image={hero ? `url("${hero}")` : 'none'} aria-hidden="true"></div>
  <div class="scrim" aria-hidden="true"></div>
  <TopBar />
  <main class="content" aria-labelledby="title-name">
    {#if failed}
      <section class="empty panel" role="alert">
        <h1 class="pixel" id="title-name">Sem sinal</h1>
        <p class="muted">{failed}</p>
        <button class="btn primary" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button>
      </section>
    {:else if !item}
      <p class="loading pixel" aria-busy="true" id="title-name">Sintonizando…</p>
    {:else}
      <section class="hero">
        <div class="art" aria-hidden="true"><Poster {item} /></div>
        <div class="info">
          <p class="kind pixel">{item.type === 'Series' ? 'Série' : 'Filme'}</p>
          {#if item.logo}
            <h1 class="logo" id="title-name"><img src={item.logo} alt={item.name} /></h1>
          {:else}
            <h1 class="pixel title" id="title-name">{item.name}</h1>
          {/if}
          <p class="meta">
            {#if item.year}<span>{item.year}</span>{/if}
            {#if item.rating}<span class="rating">{item.rating}</span>{/if}
            {#if item.type === 'Series'}<span>{data?.seasons.length} temporada{data?.seasons.length === 1 ? '' : 's'}</span>{/if}
            {#if item.type === 'Movie' && runtime(item.runtime)}<span>{runtime(item.runtime)}</span>{/if}
            {#if item.score}<span>★ {item.score.toFixed(1)}</span>{/if}
            {#if remaining(item)}<span class="left">{remaining(item)}</span>{/if}
            {#if item.played}<span class="seen"><Icon name="check" /> Assistido</span>{/if}
          </p>
          {#if item.genres.length}<p class="genres muted">{item.genres.join(' · ')}</p>{/if}
          {#if item.overview}<p class="summary">{item.overview}</p>{/if}
          {#if item.type === 'Series' && data?.next}
            <p class="next muted">Próximo: {episodeTag(data.next)} · {data.next.name}</p>
          {/if}
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default disabled={item.type === 'Series' && !data?.next} onclick={() => playMain()}>
              <Icon name={playLabel(item, data?.next ?? null).startsWith('Continuar') ? 'resume' : 'play'} />
              {playLabel(item, data?.next ?? null)}
            </button>
            {#if item.type === 'Movie' && item.position > 0 && !item.played}
              <button class="btn" data-nav onclick={() => playMain(true)}><Icon name="play" /> Do início</button>
            {/if}
            <button class="btn" data-nav aria-pressed={item.favorite} onclick={toggleFav}><Icon name={item.favorite ? 'check' : 'plus'} /> Minha lista</button>
            {#if item.type === 'Movie'}
              <button class="btn" data-nav aria-pressed={item.played} onclick={() => togglePlayed(item)}><Icon name="check" /> Assistido</button>
            {/if}
          </div>
        </div>
      </section>

      {#if item.type === 'Series' && data}
        <section class="episodes" aria-labelledby="episodes-title">
          <h2 id="episodes-title" class="sr-only">Episódios</h2>
          {#if data.seasons.length > 1}
            <div class="seasons" role="group" aria-label="Temporadas">
              {#each data.seasons as s (s.id)}
                <button class="btn small" data-nav aria-pressed={season === s.id} onclick={() => pickSeason(s.id)}>{s.name}</button>
              {/each}
            </div>
          {/if}
          <ul class="list" data-nav-group="episodes">
            {#each episodes as ep (ep.id)}
              <li>
                <button
                  class="ep"
                  data-nav
                  data-episode={ep.id}
                  onclick={() => go({ name: 'player', id: ep.id })}
                >
                  <span class="thumb"><Poster item={ep} wide /></span>
                  <span class="text">
                    <span class="name"><span class="tag pixel">{episodeTag(ep)}</span> {ep.name}</span>
                    {#if ep.overview}<span class="muted ov">{ep.overview}</span>{/if}
                    <span class="muted">{remaining(ep) || runtime(ep.runtime)}</span>
                    {#if ep.played}<span class="sr-only">, assistido</span>{:else if ep.progress > 0}<span class="sr-only">, {Math.round(ep.progress * 100)}% assistido</span>{/if}
                  </span>
                </button>
              </li>
            {/each}
          </ul>
        </section>
      {/if}
    {/if}
  </main>
  <HelpBar
    hints={item?.type === 'Series'
      ? [
          { button: 'south', label: 'Assistir' },
          { button: 'north', label: 'Minha lista' },
          { button: 'west', label: 'Episódio assistido' },
          { button: 'east', label: 'Voltar' },
        ]
      : [
          { button: 'south', label: 'Escolher' },
          { button: 'north', label: 'Minha lista' },
          { button: 'west', label: 'Assistido' },
          { button: 'east', label: 'Voltar' },
        ]}
  />
</div>

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
  }
  .backdrop {
    position: fixed;
    inset: 0;
    background-size: cover;
    background-position: center 25%;
    opacity: 0.4;
    z-index: -1;
    mask-image: linear-gradient(to bottom, #000 30%, transparent 85%);
  }
  .scrim {
    position: fixed;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    background:
      linear-gradient(to top, var(--bg) 0%, color-mix(in srgb, var(--bg) 85%, transparent) 40%, transparent 70%),
      linear-gradient(90deg, var(--bg) 0%, color-mix(in srgb, var(--bg) 88%, transparent) 40%, color-mix(in srgb, var(--bg) 30%, transparent) 100%);
  }
  .content {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 1rem 3rem 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }
  .loading {
    margin: auto;
    font-size: 1.4rem;
    animation: blink 1s steps(2) infinite;
  }
  .hero {
    display: grid;
    grid-template-columns: minmax(10rem, 15rem) 1fr;
    gap: 2.5rem;
    align-items: end;
  }
  .art {
    transform: rotate(-2deg);
    filter: drop-shadow(10px 10px 0 var(--shadow));
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    max-width: 60rem;
  }
  .kind {
    color: var(--accent-2);
    font-size: 0.95rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .title {
    font-size: clamp(2rem, 4.2vw, 4rem);
    line-height: 1.05;
    color: #fff;
    text-shadow: 5px 5px 0 color-mix(in srgb, var(--accent) 55%, #000);
  }
  .logo {
    margin: 0;
  }
  .logo img {
    max-width: min(30rem, 55vw);
    max-height: 8rem;
    object-fit: contain;
    object-position: left bottom;
    filter: drop-shadow(4px 4px 0 #000);
    display: block;
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem 1.4rem;
    color: #e6deef;
    font-size: 1.05rem;
  }
  .rating {
    border: 2px solid currentColor;
    padding: 0 0.35rem;
  }
  .left {
    color: var(--focus);
  }
  .seen {
    display: inline-flex;
    gap: 0.3rem;
    align-items: center;
    color: var(--accent);
  }
  .summary {
    color: #fff;
    font-size: 1.08rem;
    line-height: 1.65;
    max-width: 48rem;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 1.1rem;
    margin-top: 0.4rem;
  }
  .episodes {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .seasons {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem;
  }
  .list {
    list-style: none;
    margin: 0;
    padding: 0.3rem;
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
  }
  .ep {
    width: 100%;
    max-width: 62rem;
    display: grid;
    grid-template-columns: clamp(9rem, 16vw, 14rem) 1fr;
    gap: 1.2rem;
    align-items: center;
    padding: 0.6rem;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: color-mix(in srgb, var(--surface) 82%, transparent);
    border: 2px solid color-mix(in srgb, var(--text) 15%, transparent);
    box-shadow: 4px 4px 0 #000;
    cursor: pointer;
  }
  .ep:hover {
    border-color: var(--accent);
  }
  .text {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    min-width: 0;
  }
  .name {
    font-weight: 700;
    font-size: 1.1rem;
  }
  .tag {
    color: var(--focus);
    font-size: 0.8rem;
    margin-right: 0.4rem;
  }
  .ov {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .empty {
    margin: auto;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 36rem;
  }
  @media (max-height: 760px) {
    .hero {
      grid-template-columns: minmax(8rem, 11rem) 1fr;
    }
  }
</style>
