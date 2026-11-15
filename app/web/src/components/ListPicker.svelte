<script lang="ts">
  // Em quais listas o jogo está: marcar/desmarcar e criar lista nova.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, prompt, putList, toast } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  let { gameId, onclose }: { gameId: string; onclose: () => void } = $props()
  // automáticas não aceitam jogo na mão: só as manuais aparecem aqui
  const manual = $derived(app.lists.filter((l) => !l.rule))

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.prompt) return false
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })

  async function toggle(id: string, inList: boolean) {
    try {
      putList(await api.setInList(id, gameId, inList))
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function create() {
    await prompt('Nova lista', '', 'Criar lista', async (name) => {
      try {
        const l = await api.createList(name)
        putList(await api.setInList(l.id, gameId, true))
        toast(`Lista "${l.name}" criada`, 'ok', 2500)
        return null
      } catch (err) {
        return (err as Error).message
      }
    })
    tick().then(focusDefault)
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="lists-title" data-nav-scope={app.prompt ? undefined : ''}>
    <h2 id="lists-title">Listas</h2>
    {#if manual.length}
      <div class="checks" role="group" aria-label="Listas do jogo">
        {#each manual as l, i (l.id)}
          {@const inList = l.games.includes(gameId)}
          <button class="choice" role="checkbox" aria-checked={inList} data-nav data-nav-default={i === 0 ? '' : undefined} onclick={() => toggle(l.id, !inList)}>
            <span class="mark box" aria-hidden="true"></span>{l.name}
            <span class="count">{l.games.length}</span>
          </button>
        {/each}
      </div>
    {:else}
      <p class="empty">Nenhuma lista ainda. Crie a primeira com “Nova lista”.</p>
    {/if}
    <div class="actions">
      <button class="btn" data-nav data-nav-default={manual.length ? undefined : ''} onclick={create}><Icon name="plus" /> Nova lista</button>
      <button class="btn primary" data-nav onclick={onclose}>Pronto</button>
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
  .dialog {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: min(30rem, 100% - 2rem);
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 8px 8px 0 var(--outline);
  }
  h2 {
    font-size: 1.25rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .checks {
    display: grid;
    gap: 0.15rem;
  }
  .choice {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-height: 3rem;
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
  .count {
    margin-inline-start: auto;
    font-family: var(--font-mono);
    font-size: 0.9rem;
    font-variant-numeric: tabular-nums;
    color: var(--muted);
  }
  .empty {
    padding: 1rem;
    color: var(--muted);
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
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
