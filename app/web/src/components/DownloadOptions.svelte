<script lang="ts">
  // Opções de download (Central): velocidade máxima e "só de madrugada".
  // Cada escolha já vale; prende o foco ([data-nav-scope]); voltar fecha.
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, toast } from '../lib/state.svelte'
  import type { DownloadOptions } from '../lib/types'

  let { onclose }: { onclose: () => void } = $props()

  const SPEEDS = [
    { kbps: 0, label: 'Sem limite' },
    { kbps: 1024, label: '1 MB/s' },
    { kbps: 5120, label: '5 MB/s' },
    { kbps: 10240, label: '10 MB/s' },
  ]
  const opts = $derived<DownloadOptions>(app.settings?.downloads ?? { limitKBps: 0, nightOnly: false })

  async function save(next: DownloadOptions) {
    try {
      app.settings = await api.setDownloadOptions(next)
      play('accept')
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

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
  <div class="menu panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="dl-options-title" data-nav-scope>
    <h2 id="dl-options-title">Opções de download</h2>
    <fieldset>
      <legend>Velocidade</legend>
      <div class="options" role="radiogroup" aria-label="Velocidade">
        {#each SPEEDS as s (s.kbps)}
          <button
            class="choice"
            role="radio"
            aria-checked={opts.limitKBps === s.kbps}
            data-nav
            data-nav-default={opts.limitKBps === s.kbps ? '' : undefined}
            onclick={() => save({ ...opts, limitKBps: s.kbps })}
          >
            <span class="mark radio" aria-hidden="true"></span>{s.label}
          </button>
        {/each}
      </div>
    </fieldset>
    <fieldset>
      <legend>Horário</legend>
      <button class="choice" role="switch" aria-checked={opts.nightOnly} aria-describedby="dl-night-hint" data-nav onclick={() => save({ ...opts, nightOnly: !opts.nightOnly })}>
        <span class="track" aria-hidden="true"></span>Só de madrugada
      </button>
      <p id="dl-night-hint" class="hint muted">Os downloads esperam na fila e começam entre 0h e 6h.</p>
    </fieldset>
    <div class="actions">
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
  fieldset {
    border: none;
    margin: 0;
    padding: 0.9rem 0 0;
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
    gap: 0.15rem;
    margin-top: 0.4rem;
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
  .track {
    flex: none;
    position: relative;
    width: 44px;
    height: 24px;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 999px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .track::after {
    content: '';
    position: absolute;
    top: 2px;
    inset-inline-start: 2px;
    width: 14px;
    height: 14px;
    background: var(--outline);
    border-radius: var(--radii-sm);
    transition: inset-inline-start 150ms steps(3);
  }
  [aria-checked='true'] > .track {
    background: var(--accent-2);
  }
  [aria-checked='true'] > .track::after {
    inset-inline-start: 22px;
  }
  .hint {
    margin: 0.2rem 0 0;
    padding-inline-start: 0.8rem;
    font-size: 0.95rem;
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
