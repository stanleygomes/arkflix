<script lang="ts">
  // Armazenamento, como nos consoles: o disco dos jogos dividido em jogos,
  // saves, app e capas, outros e livre; depois jogos por console e saves.
  import { tick, untrack } from 'svelte'
  import Icon from '../../components/Icon.svelte'
  import { activity } from '../../lib/activity.svelte'
  import { api } from '../../lib/api'
  import { bytes } from '../../lib/format'
  import { focusEl } from '../../lib/nav'
  import { app } from '../../lib/state.svelte'
  import type { StorageInfo } from '../../lib/types'
  import MoveDialog from './MoveDialog.svelte'

  let info = $state<StorageInfo | null>(null)
  let error = $state('')
  let busy = $state(false)
  let loadedAt = 0
  // mover jogos entre pastas só faz sentido com 2+ pastas na biblioteca
  const canMove = $derived((app.settings?.libraryDirs.length ?? 0) >= 2)
  let moving = $state<StorageInfo['byConsole'][number] | null>(null)

  function closeMove() {
    const id = moving ? `move-${moving.system}` : ''
    moving = null
    tick().then(() => focusEl(document.getElementById(id)))
  }

  // terminou de mover (ou cancelou): mede de novo
  $effect(() => {
    const done = activity.tasks.find((t) => t.id.startsWith('move:') && t.state !== 'running')
    if (done && done.at > untrack(() => loadedAt)) untrack(load)
  })

  async function load() {
    loadedAt = Date.now()
    busy = true
    try {
      info = await api.storage()
      error = ''
    } catch (err) {
      error = (err as Error).message
    } finally {
      busy = false
    }
  }

  $effect(() => {
    load()
  })

  type Part = { id: string; label: string; value: number }
  const parts = $derived.by((): Part[] => {
    if (!info || info.total <= 0) return []
    const ours = info.games + info.saves + info.app + info.cache
    const others = Math.max(0, info.total - info.free - ours)
    return [
      { id: 'games', label: 'Jogos', value: info.games },
      { id: 'saves', label: 'Saves', value: info.saves },
      { id: 'app', label: 'App e capas', value: info.app + info.cache },
      { id: 'others', label: 'Outros programas e arquivos', value: others },
      { id: 'free', label: 'Livre', value: info.free },
    ]
  })
  const pct = (v: number) => (info && info.total > 0 ? (v / info.total) * 100 : 0)
  const fmtPct = (v: number) => {
    const p = pct(v)
    return p > 0 && p < 1 ? '<1%' : `${Math.round(p)}%`
  }
  const summary = $derived(
    info && info.total > 0
      ? `${bytes(info.free)} livres de ${bytes(info.total)}. ` + parts.slice(0, 4).map((p) => `${p.label}: ${bytes(p.value)}`).join(', ')
      : '',
  )
  const maxConsole = $derived(Math.max(1, ...(info?.byConsole.map((c) => c.bytes) ?? [1])))
</script>

<div class="panel-body">
  <header class="head">
    <h2>Armazenamento</h2>
    <button class="btn small" data-nav data-nav-default onclick={load} disabled={busy}><Icon name="resume" /> {busy ? 'Medindo…' : 'Atualizar'}</button>
  </header>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {:else if !info}
    <p class="muted" aria-busy="true">Medindo…</p>
  {:else}
    {#if info.total > 0}
      <section aria-labelledby="disk-title">
        <h3 id="disk-title" class="disk">
          <span class="big">{bytes(info.free)}</span> livres de {bytes(info.total)}
        </h3>
        <p class="muted small path">Disco da pasta {info.disk}</p>
        <div class="bar" role="img" aria-label={summary}>
          {#each parts as p (p.id)}
            {#if p.value > 0}<span class={`seg ${p.id}`} style:width={`${pct(p.value)}%`}></span>{/if}
          {/each}
        </div>
        <ul class="legend">
          {#each parts as p (p.id)}
            <li><span class={`dot ${p.id}`} aria-hidden="true"></span><span class="label">{p.label}</span><span class="value hud">{bytes(p.value)}</span><span class="pct muted">{fmtPct(p.value)}</span></li>
          {/each}
        </ul>
      </section>
    {:else}
      <p class="muted">Não deu pra medir o disco de {info.disk}.</p>
    {/if}

    <section aria-labelledby="consoles-title">
      <h3 id="consoles-title">Jogos por console · {info.gameCount} jogo{info.gameCount === 1 ? '' : 's'}, {bytes(info.games)}</h3>
      {#if info.byConsole.length}
        <ul class="rows" class:movable={canMove}>
          {#each info.byConsole as c (c.system)}
            <li>
              <span class="label">{c.name}</span>
              <span class="track" aria-hidden="true"><span class="fill" style:width={`${(c.bytes / maxConsole) * 100}%`}></span></span>
              <span class="value hud">{bytes(c.bytes)}</span>
              <span class="pct muted">{c.count} jogo{c.count === 1 ? '' : 's'}</span>
              {#if canMove}
                <button id={`move-${c.system}`} class="btn small" data-nav aria-label={`Mover jogos de ${c.name}`} onclick={() => (moving = c)}>Mover</button>
              {/if}
            </li>
          {/each}
        </ul>
      {:else}
        <p class="muted">Nenhum jogo neste computador ainda.</p>
      {/if}
      {#if info.remoteCount}
        <p class="muted remote"><Icon name="cloud" /> Mais {info.remoteCount} jogo{info.remoteCount === 1 ? '' : 's'} no servidor ({bytes(info.remote)}), ainda não baixado{info.remoteCount === 1 ? '' : 's'}.</p>
      {/if}
    </section>

    <section aria-labelledby="saves-title">
      <h3 id="saves-title">Saves · {bytes(info.saves)}</h3>
      <ul class="rows plain">
        {#each info.saveDirs as d (d.path)}
          <li>
            <span class="label">{d.label}<span class="muted small path">{d.path}</span></span>
            <span class="value hud">{bytes(d.bytes)}</span>
          </li>
        {/each}
        <li>
          <span class="label">App (configurações, perfis, listas, histórico)</span>
          <span class="value hud">{bytes(info.app)}</span>
        </li>
        <li>
          <span class="label">Capas e catálogo baixados</span>
          <span class="value hud">{bytes(info.cache)}</span>
        </li>
      </ul>
    </section>
  {/if}
</div>

{#if moving}
  <MoveDialog system={moving.system} name={moving.name} roots={moving.roots} onclose={closeMove} />
{/if}

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.75rem;
  }
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  h3 {
    font-family: var(--font-label);
    font-size: 0.9rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  h3 {
    margin-bottom: 0.8rem;
  }
  section + section {
    padding-top: 1.25rem;
    border-top: var(--edge) solid var(--outline);
  }
  .disk {
    font-family: var(--font-body);
    font-size: 1.05rem;
    text-transform: none;
    letter-spacing: 0;
  }
  /* número grande do disco livre: Bevan, o único "alto" da tela */
  .big {
    font-family: var(--font-label);
    font-size: clamp(2rem, 4vw, 2.8rem);
    line-height: 1;
    color: var(--text);
  }
  .path {
    font-family: var(--font-mono);
    overflow-wrap: anywhere;
  }
  .small {
    font-size: 0.85rem;
  }
  /* barra do disco: trilho de cartão com borda de 3px, blocos retos */
  .bar {
    display: flex;
    height: 1.8rem;
    margin: 0.9rem 0 1.1rem;
    background: var(--bg);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    overflow: hidden;
  }
  .seg {
    min-width: 3px;
  }
  .seg + .seg {
    border-inline-start: 2px solid var(--bg);
  }
  /* cores por tipo (a legenda escreve nome e tamanho: a cor não é a única pista) */
  .games {
    background: var(--accent);
  }
  .saves {
    background: var(--accent-2);
  }
  .app {
    background: var(--text);
  }
  .others {
    background: var(--muted);
  }
  .free {
    background: var(--bg);
  }
  .legend {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
    gap: 0.5rem 1.5rem;
  }
  .legend li,
  .rows li {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    align-items: center;
    gap: 0.75rem;
  }
  .dot {
    width: 1rem;
    height: 1rem;
    border: 2px solid var(--outline);
    border-radius: 3px;
  }
  .value {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.05rem;
    font-variant-numeric: tabular-nums;
    letter-spacing: 0;
    color: var(--text);
    text-align: end;
  }
  .pct {
    min-width: 4.5rem;
    text-align: end;
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
  }
  .rows li {
    grid-template-columns: minmax(9rem, 14rem) 1fr auto auto;
  }
  .rows.movable li {
    grid-template-columns: minmax(9rem, 14rem) 1fr auto auto auto;
  }
  .rows.plain li {
    grid-template-columns: 1fr auto;
    padding-bottom: 0.6rem;
    border-bottom: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  .rows .label {
    display: flex;
    flex-direction: column;
    font-family: var(--font-body);
    font-size: 1rem;
    text-transform: none;
    letter-spacing: 0;
  }
  .legend .label {
    font-family: var(--font-body);
    font-size: 1rem;
    text-transform: none;
    letter-spacing: 0;
  }
  .track {
    height: 1rem;
    background: var(--bg);
    border: 2px solid var(--outline);
    border-radius: 4px 5px 4px 5px;
    overflow: hidden;
  }
  .fill {
    display: block;
    height: 100%;
    min-width: 2px;
    background: var(--accent);
  }
  .remote {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    margin-top: 0.9rem;
  }
  .error {
    color: var(--danger);
  }
</style>
