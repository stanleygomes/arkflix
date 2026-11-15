<script lang="ts">
  // Menu de opções da lista: ordenação (e, depois, filtros). Prende o foco
  // ([data-nav-scope]); voltar fecha.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { SORTS, type FilterGroup } from '../lib/library'
  import { play } from '../lib/sound'

  let {
    sort,
    onsort,
    onclose,
    groups = [],
    filters = [],
    onfilter,
    onclear,
    extra = [],
  }: {
    sort: string
    onsort: (key: string) => void
    onclose: () => void
    groups?: FilterGroup[]
    filters?: string[]
    onfilter?: (id: string) => void
    onclear?: () => void
    extra?: { label: string; danger?: boolean; onclick: () => void }[]
  } = $props()

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back' || a === 'menu') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'search'
    })
  })
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="menu panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="options-title" data-nav-scope>
    <h2 id="options-title">Opções da lista</h2>
    <fieldset>
      <legend>Ordenar por</legend>
      <div class="options" role="radiogroup" aria-label="Ordenar por">
        {#each SORTS as s (s.key)}
          <button
            class="choice"
            role="radio"
            aria-checked={sort === s.key}
            data-nav
            data-nav-default={sort === s.key ? '' : undefined}
            onclick={() => onsort(s.key)}
          >
            <span class="mark radio" aria-hidden="true"></span>{s.label}
          </button>
        {/each}
      </div>
    </fieldset>
    {#each groups as group (group.id)}
      <fieldset>
        <legend>{group.label}</legend>
        <div class="checks">
          {#each group.options as opt (opt.id)}
            <button class="choice" role="checkbox" aria-checked={filters.includes(opt.id)} data-nav onclick={() => onfilter?.(opt.id)}>
              <span class="mark box" aria-hidden="true"></span>{opt.label}
            </button>
          {/each}
        </div>
      </fieldset>
    {/each}
    {#if extra.length}
      <div class="extra">
        {#each extra as x (x.label)}
          <button class="btn" class:danger={x.danger} data-nav onclick={x.onclick}>{x.label}</button>
        {/each}
      </div>
    {/if}
    <div class="actions">
      {#if filters.length}
        <button class="btn" data-nav onclick={() => onclear?.()}>Limpar filtros ({filters.length})</button>
      {/if}
      <button class="btn primary" data-nav onclick={onclose}>Fechar</button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: color-mix(in srgb, var(--bg) 78%, transparent);
    display: grid;
    place-items: center;
    z-index: 80;
  }
  .menu {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    width: min(40rem, 100% - 2rem);
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 8px 8px 0 var(--outline);
  }
  h2 {
    font-size: 1.25rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  fieldset {
    border: none;
    margin: 0;
    padding: 0.6rem 0 0;
    border-top: var(--edge) solid color-mix(in srgb, var(--outline) 35%, transparent);
  }
  legend {
    font-family: var(--font-label);
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    padding: 0 0.5rem 0 0;
  }
  .options {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr));
    gap: 0.15rem 0.5rem;
    margin-top: 0.4rem;
  }
  .checks {
    display: flex;
    flex-wrap: wrap;
    gap: 0.15rem 0.5rem;
    margin-top: 0.4rem;
  }
  .choice {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-height: 2.75rem;
    padding: 0.4rem 0.8rem;
    font: inherit;
    font-size: 1.05rem;
    line-height: 1.3;
    text-align: start;
    color: var(--text);
    background: none;
    border: 0;
    border-radius: 4px;
    cursor: pointer;
  }
  .choice:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .mark {
    flex: none;
    position: relative;
    width: 22px;
    height: 22px;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    box-shadow: 2px 2px 0 var(--outline);
  }
  .mark.radio {
    border-radius: 50%;
  }
  .mark.radio::after {
    content: '';
    position: absolute;
    inset: 3px;
    border-radius: 50%;
  }
  [aria-checked='true'] > .mark.radio::after {
    background: var(--outline);
  }
  .mark.box {
    border-radius: 4px;
  }
  [aria-checked='true'] > .mark.box {
    background: var(--accent);
  }
  [aria-checked='true'] > .mark.box::after {
    content: '';
    position: absolute;
    left: 4px;
    top: 0;
    width: 6px;
    height: 11px;
    border: solid var(--on-accent);
    border-width: 0 3px 3px 0;
    transform: rotate(45deg);
  }
  .extra {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding-top: 1rem;
    border-top: var(--edge) solid color-mix(in srgb, var(--outline) 35%, transparent);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    justify-content: flex-end;
    padding-top: 1rem;
    border-top: var(--edge) solid var(--outline);
  }
</style>
