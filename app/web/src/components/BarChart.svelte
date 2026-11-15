<script lang="ts">
  // Gráfico de barras de uma série só (verde-azulado; a maior em mostarda
  // carrega o recado), começa no zero, com tabela como alternativa.
  // horizontal = ranking (valor escrito em cada linha); vertical = tempo.
  type Item = { label: string; value: number; display: string; detail?: string }
  let {
    title,
    items,
    vertical = false,
    every = 1,
  }: { title: string; items: Item[]; vertical?: boolean; every?: number } = $props()
  let table = $state(false)
  const max = $derived(Math.max(1, ...items.map((i) => i.value)))
  const top = $derived(items.findIndex((i) => i.value > 0 && i.value === max))
  const id = `chart-${Math.random().toString(36).slice(2, 8)}`
</script>

<section class="chart" aria-labelledby={id}>
  <header>
    <h2 id={id}>{title}</h2>
    <button class="btn small" data-nav aria-pressed={table} onclick={() => (table = !table)}>{table ? 'Ver gráfico' : 'Ver tabela'}</button>
  </header>
  {#if table}
    <div class="scroll">
    <table>
      <caption class="sr-only">{title}</caption>
      <tbody>
        {#each items as it, i (i)}
          <tr><th scope="row">{it.label}</th><td>{it.display}</td>{#if it.detail}<td>{it.detail}</td>{/if}</tr>
        {/each}
      </tbody>
    </table>
    </div>
  {:else if vertical}
    <ol class="columns" aria-label={title}>
      {#each items as it, i (i)}
        <li title={`${it.label}: ${it.display}${it.detail ? ` · ${it.detail}` : ''}`}>
          <span class="col" class:top={i === top} style:height={`${(it.value / max) * 100}%`}></span>
          <span class="tick" class:hidden={i % every !== 0 && i !== items.length - 1}>{it.label}</span>
          <span class="sr-only">{it.label}: {it.display}</span>
        </li>
      {/each}
    </ol>
  {:else}
    <ol class="rows" aria-label={title}>
      {#each items as it, i (i)}
        <li title={it.detail ? `${it.label}: ${it.detail}` : undefined}>
          <span class="name">{it.label}</span>
          <span class="track"><span class="bar" class:top={i === top} style:width={`${Math.max(2, (it.value / max) * 100)}%`}></span></span>
          <span class="val">{it.display}</span>
        </li>
      {/each}
    </ol>
  {/if}
</section>

<style>
  /* moldura de cartão sóbrio; marcas chapadas com fio de contorno */
  .chart {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-width: 0;
    padding: 1rem 1.25rem 1.25rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }
  h2 {
    font-size: 1rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.55rem;
  }
  .rows li {
    display: grid;
    grid-template-columns: minmax(7rem, 12rem) minmax(8rem, 1fr) 6.5rem;
    gap: 0.8rem;
    align-items: center;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  /* trilho com a linha de base (zero) à esquerda */
  .track {
    height: 1.1rem;
    border-inline-start: 2px solid var(--outline);
  }
  .bar {
    display: block;
    height: 100%;
    background: var(--accent-2);
    border: 2px solid var(--outline);
    border-inline-start: 0;
    border-radius: 0 3px 3px 0;
    transform-origin: left;
  }
  .val {
    text-align: end;
    font-family: var(--font-mono);
    font-size: 0.85rem;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  /* colunas: até 4 linhas de grade horizontais bem fracas, base de 2px */
  .columns {
    list-style: none;
    margin: 0;
    padding: 0;
    height: 9.5rem;
    display: flex;
    align-items: flex-end;
    gap: 3px;
    border-bottom: 2px solid var(--outline);
    margin-bottom: 1.5rem;
    background: repeating-linear-gradient(
      to top,
      transparent 0 calc(25% - 1px),
      color-mix(in srgb, var(--outline) 18%, transparent) calc(25% - 1px) 25%
    );
  }
  .columns li {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    position: relative;
  }
  .col {
    display: block;
    background: var(--accent-2);
    border: 2px solid var(--outline);
    border-bottom: 0;
    border-radius: 3px 3px 0 0;
    transform-origin: bottom;
    width: min(100%, 3.5rem);
    align-self: center;
  }
  .col[style*='height: 0%'] {
    border: 0;
  }
  .bar.top,
  .col.top {
    background: var(--accent);
  }
  .tick {
    position: absolute;
    bottom: -1.5rem;
    left: 50%;
    transform: translateX(-50%);
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--muted);
    white-space: nowrap;
  }
  .tick.hidden {
    visibility: hidden;
  }
  /* aparece uma vez: barras crescem da base em degraus */
  @media (prefers-reduced-motion: no-preference) {
    .bar {
      animation: grow-x 320ms steps(4) both;
    }
    .col {
      animation: grow-y 320ms steps(4) both;
    }
  }
  @keyframes grow-x {
    from {
      transform: scaleX(0);
    }
  }
  @keyframes grow-y {
    from {
      transform: scaleY(0);
    }
  }
  .scroll {
    overflow-x: auto;
  }
  table {
    border-collapse: collapse;
    width: 100%;
    border: var(--edge) solid var(--outline);
  }
  th,
  td {
    text-align: start;
    padding: 0.6rem 0.75rem;
    border-bottom: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  th {
    font-weight: 400;
  }
  td:nth-child(2) {
    text-align: end;
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  td:nth-child(3) {
    color: var(--muted);
  }
</style>
