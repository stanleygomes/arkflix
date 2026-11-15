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
  import { play } from '../lib/sound'
  import { go, toast } from '../lib/state.svelte'
  import type { WatchDetails, WatchItem } from '../lib/types'
  import { episodeTag, playLabel, remaining, runtime } from '../lib/watch'

  let { id }: { id: string } = $props()

  let data = $state<WatchDetails | null>(null)
  let episodes = $state<WatchItem[]>([])
  let season = $state('')
  let failed = $state('')

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
      <section class="alert panel" role="alert">
        <span class="chip error">Erro</span>
        <h1 id="title-name">Sem sinal</h1>
        <p class="muted">{failed}</p>
        <button class="btn primary" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button>
      </section>
    {:else if !item}
      <div class="loading" role="status" aria-busy="true">
        <span class="ring" aria-hidden="true"></span>
        <p class="meta" id="title-name">Buscando o programa…</p>
      </div>
    {:else}
      <section class="hero">
        <!-- 🎬 cartão inclinado: o pôster em destaque da página -->
        <div class="art sketched" aria-hidden="true"><Poster {item} /></div>
        <div class="info">
          <p class="kind"><span class="chip">{item.type === 'Series' ? 'Série' : 'Filme'}</span></p>
          {#if item.logo}
            <h1 class="logo" id="title-name"><img src={item.logo} alt={item.name} /></h1>
          {:else}
            <h1 class="title" id="title-name">{item.name}</h1>
          {/if}
          <p class="meta">
            {#if item.year}<span>{item.year}</span>{/if}
            {#if item.rating}<span class="chip rating">{item.rating}</span>{/if}
            {#if item.type === 'Series'}<span>{data?.seasons.length} temporada{data?.seasons.length === 1 ? '' : 's'}</span>{/if}
            {#if item.type === 'Movie' && runtime(item.runtime)}<span>{runtime(item.runtime)}</span>{/if}
            {#if item.score}<span>★ {item.score.toFixed(1)}</span>{/if}
            {#if remaining(item)}<span class="left">{remaining(item)}</span>{/if}
            {#if item.played}<span class="chip ok"><Icon name="check" size="1em" /> Assistido</span>{/if}
          </p>
          {#if item.genres.length}<p class="genres muted">{item.genres.join(' · ')}</p>{/if}
          {#if item.overview}<p class="summary">{item.overview}</p>{/if}
          {#if item.type === 'Series' && data?.next}
            <p class="next meta">Próximo: {episodeTag(data.next)} · {data.next.name}</p>
          {/if}
          <div class="actions">
            <!-- 🎬 bilhete: a ação da página (o anel de foco fica no invólucro) -->
            <span class="ticket">
              <button data-nav data-nav-default disabled={item.type === 'Series' && !data?.next} onclick={() => playMain()}>
                <Icon name={playLabel(item, data?.next ?? null).startsWith('Continuar') ? 'resume' : 'play'} />
                {playLabel(item, data?.next ?? null)}
              </button>
            </span>
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
                    <span class="name"><span class="tag">{episodeTag(ep)}</span> {ep.name}</span>
                    {#if ep.overview}<span class="ov">{ep.overview}</span>{/if}
                    <span class="dur">{remaining(ep) || runtime(ep.runtime)}</span>
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
    opacity: 0.35;
    z-index: -1;
    mask-image: linear-gradient(to bottom, #000 30%, transparent 85%);
  }
  .scrim {
    position: fixed;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    background:
      linear-gradient(to top, var(--bg) 0%, color-mix(in srgb, var(--bg) 88%, transparent) 40%, transparent 70%),
      linear-gradient(
        90deg,
        var(--bg) 0%,
        color-mix(in srgb, var(--bg) 90%, transparent) 40%,
        color-mix(in srgb, var(--bg) 35%, transparent) 100%
      );
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
  /* espera: anel sóbrio parado com o texto (nada gira em loop) */
  .loading {
    margin: auto;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .loading .meta {
    color: var(--muted);
  }
  .ring {
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    border: 3px solid color-mix(in srgb, var(--outline) 25%, transparent);
    border-top-color: var(--outline);
  }
  .hero {
    display: grid;
    grid-template-columns: minmax(10rem, 15rem) 1fr;
    gap: 2.5rem;
    align-items: end;
  }
  /* 🎬 cartão inclinado: contorno à mão, cantos grandes, sombra de desenho */
  .art {
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii);
    transform: rotate(-1deg);
    margin: 0.5rem 0.5rem 0.5rem 0;
  }
  .art :global(.poster) {
    border: none;
    border-radius: var(--radii);
    box-shadow: none;
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    max-width: 60rem;
  }
  .kind {
    display: flex;
    gap: 0.5rem;
  }
  .title {
    font-size: clamp(2.2rem, 4.4vw, 4.2rem);
  }
  .logo {
    margin: 0;
  }
  .logo img {
    max-width: min(30rem, 55vw);
    max-height: 8rem;
    object-fit: contain;
    object-position: left bottom;
    filter: drop-shadow(4px 4px 0 var(--n-on-mustard));
    display: block;
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1.2rem;
    font-size: 1rem;
  }
  .rating {
    letter-spacing: 0.04em;
  }
  .left {
    color: var(--accent-2);
  }
  .genres {
    font-family: var(--font-hud);
    letter-spacing: 0.04em;
    font-size: 0.9rem;
  }
  .summary {
    font-size: 1.08rem;
    max-width: 48rem;
  }
  .next {
    color: var(--muted);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1.1rem;
    margin-top: 0.4rem;
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
  .ticket:hover:not(:has(:disabled)),
  .ticket:focus-within {
    transform: translate(-2px, -2px);
    filter: drop-shadow(5px 6px 0 var(--outline));
  }
  .ticket:focus-within {
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .ticket:active:not(:has(:disabled)) {
    transform: translate(3px, 4px);
    filter: none;
  }
  .ticket:has(:disabled) {
    filter: none;
  }
  .ticket button {
    --notch: radial-gradient(circle 8px at 0 50%, #0000 97%, #000);
    --notch-end: radial-gradient(circle 8px at 100% 50%, #0000 97%, #000);
    --ink: var(--n-on-mustard);
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
  .ticket button:disabled {
    --ink: var(--muted);
    color: var(--muted);
    background: var(--surface);
    cursor: not-allowed;
  }
  /* contorno à mão, seguindo os entalhes */
  .ticket button::before {
    content: '';
    position: absolute;
    inset: 0;
    border: var(--edge) solid var(--ink);
    border-radius: inherit;
    background:
      radial-gradient(circle 11px at 0 50%, transparent 8px, var(--ink) 8.5px 11px, transparent 11.5px),
      radial-gradient(circle 11px at 100% 50%, transparent 8px, var(--ink) 8.5px 11px, transparent 11.5px);
    filter: url(#sketch);
    pointer-events: none;
  }

  .episodes {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .seasons {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }
  /* lista de episódios: moldura reta, linhas divididas, sem cartão por linha */
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    max-width: 64rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    box-shadow: var(--hard);
  }
  .list li + li {
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  .ep {
    width: 100%;
    display: grid;
    grid-template-columns: clamp(9rem, 16vw, 14rem) 1fr;
    gap: 1.2rem;
    align-items: center;
    padding: 0.9rem 1rem;
    font: inherit;
    color: var(--text);
    text-align: start;
    background: none;
    border: none;
    cursor: pointer;
  }
  .ep:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  /* foco do controle: anel verde-azulado por dentro da linha + barra
     mostarda à esquerda; a miniatura sobe */
  .ep:focus,
  .ep:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: -6px;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    box-shadow: inset 6px 0 0 var(--accent);
  }
  .thumb :global(.poster) {
    box-shadow: var(--hard-sm);
  }
  .ep:focus .thumb :global(.poster) {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
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
    font-family: var(--font-hud);
    font-weight: 400;
    font-size: 0.8rem;
    letter-spacing: 0.08em;
    color: var(--accent-2);
    margin-inline-end: 0.4rem;
  }
  .ov {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .dur {
    font-family: var(--font-hud);
    font-size: 0.85rem;
    letter-spacing: 0.04em;
    color: var(--muted);
  }
  /* erro: painel com borda e sombra vermelhas, nunca a tela toda */
  .alert {
    margin: auto;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
    max-width: 36rem;
    border-color: var(--danger);
    box-shadow: 4px 4px 0 var(--danger);
  }
  .alert h1 {
    font-family: var(--font-label);
    font-size: 1.25rem;
    color: var(--text);
    -webkit-text-stroke: 0;
    text-shadow: none;
  }
  @media (max-height: 760px) {
    .hero {
      grid-template-columns: minmax(8rem, 11rem) 1fr;
    }
  }
</style>
