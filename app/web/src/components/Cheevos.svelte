<script lang="ts">
  // Conquistas do jogo (RetroAchievements): placar e os troféus, ganhos em
  // cor e os que faltam em cinza. Só aparece com conta ligada no perfil.
  import { api } from '../lib/api'
  import type { GameCheevos } from '../lib/types'
  import Icon from './Icon.svelte'

  let { id }: { id: string } = $props()
  let data = $state<GameCheevos | null>(null)
  let error = $state('')

  $effect(() => {
    data = null
    api
      .gameCheevos(id)
      .then((d) => (data = d))
      .catch((err) => (error = (err as Error).message))
  })

  const pct = $derived(data && data.total ? Math.round((data.unlocked / data.total) * 100) : 0)
</script>

{#if error || (data?.connected && (data.supported || data.reason))}
  <section class="cheevos" aria-labelledby="cheevos-title">
    <h2 id="cheevos-title"><Icon name="trophy" /> Conquistas</h2>
    {#if error}
      <p class="muted">{error}</p>
    {:else if data && !data.supported}
      <p class="muted">{data.reason}</p>
    {:else if data}
      <p class="score">
        <span class="big">{data.unlocked}<span class="of">/{data.total}</span></span>
        <span class="meta muted">{data.pointsUnlocked} de {data.points} pontos · {pct}%</span>
      </p>
      <div class="energy" role="progressbar" aria-label="Conquistas ganhas" aria-valuemin="0" aria-valuemax="100" aria-valuenow={pct}>
        <span style:width={`${Math.floor(pct / 5) * 5}%`}></span>
      </div>
      <ul class="list">
        {#each data.list ?? [] as c (c.id)}
          <li class:got={c.unlocked}>
            <img src={c.badgeUrl} alt="" loading="lazy" width="48" height="48" />
            <span class="text">
              <span class="title">{c.title}<span class="sr-only">{c.unlocked ? ' (ganha)' : ' (falta)'}</span></span>
              <span class="desc">{c.description}</span>
            </span>
            <span class="pts">{c.points}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}

<style>
  .cheevos {
    display: grid;
    gap: 0.8rem;
  }
  h2 {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    font-size: 1.15rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    padding-top: 0.9rem;
    border-top: var(--edge) solid color-mix(in srgb, var(--outline) 35%, transparent);
  }
  .score {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 0.4rem 1rem;
  }
  .big {
    font-family: var(--font-label);
    font-size: 2rem;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }
  .of {
    color: var(--accent);
  }
  .energy {
    width: min(28rem, 100%);
  }
  /* lista de itens: moldura reta, linhas separadas por fio */
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
    background: var(--surface);
    border: var(--edge) solid var(--outline);
  }
  .list li {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 0.75rem;
    align-items: center;
    min-height: 4rem;
    padding: 0.6rem 1rem;
    border-bottom: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  img {
    image-rendering: pixelated;
    border: var(--edge) solid var(--muted);
    border-radius: var(--radii-sm);
    filter: grayscale(1) brightness(0.7);
  }
  .got img {
    border-color: var(--outline);
    filter: none;
  }
  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .title {
    font-weight: 700;
    color: var(--muted);
  }
  .got .title {
    color: var(--text);
  }
  .desc {
    font-family: var(--font-hud);
    font-size: 0.78rem;
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pts {
    font-family: var(--font-mono);
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--muted);
  }
  .got .pts {
    color: var(--accent-2);
  }
</style>
