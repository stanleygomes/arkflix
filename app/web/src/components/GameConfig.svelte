<script lang="ts">
  // Configurar este jogo (RetroArch): core, imagem e proporção só pra ele.
  // Cada escolha grava na hora e vale na próxima vez que o jogo abrir.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { toast } from '../lib/state.svelte'
  import type { GameConfigInfo, GameSettings } from '../lib/types'

  let { gameId, onclose }: { gameId: string; onclose: () => void } = $props()
  let info = $state<GameConfigInfo | null>(null)
  let error = $state('')

  $effect(() => {
    api
      .gameConfig(gameId)
      .then((v) => {
        info = v
        tick().then(focusDefault)
      })
      .catch((err) => (error = (err as Error).message))
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  type Opt = { value: string; label: string }
  const groups = $derived.by(() => {
    if (!info?.supported) return []
    const def = info.cores.find((c) => c.default)
    const cores: Opt[] = [
      { value: '', label: `Padrão${def ? ` (${def.name})` : ''}` },
      ...info.cores.filter((c) => !c.default).map((c) => ({ value: c.id, label: c.name })),
    ]
    const image: Opt[] = [
      { value: '', label: 'Padrão' },
      { value: 'sharp', label: 'Pixels nítidos' },
      { value: 'smooth', label: 'Suave' },
      ...(info.crt ? [{ value: 'crt', label: 'CRT (TV de tubo)' }] : []),
    ]
    const aspect: Opt[] = [
      { value: '', label: 'Padrão do core' },
      { value: '4:3', label: '4:3' },
      { value: 'full', label: 'Tela toda' },
    ]
    return [
      ...(cores.length > 1 ? [{ key: 'core' as const, label: 'Core', opts: cores, hint: 'Pontos salvos de um core não abrem em outro.' }] : []),
      { key: 'image' as const, label: 'Imagem', opts: image, hint: '' },
      { key: 'aspect' as const, label: 'Proporção', opts: aspect, hint: '' },
    ]
  })

  async function pick(key: keyof GameSettings, value: string) {
    if (!info || info.settings[key] === value) return
    try {
      info = await api.setGameConfig(gameId, { ...info.settings, [key]: value } as GameSettings)
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="config-title" data-nav-scope>
    <h2 id="config-title">Configurar este jogo</h2>
    {#if error}
      <p class="error" role="alert">{error}</p>
    {:else if !info}
      <p class="muted" role="status">Carregando…</p>
    {:else if !info.supported}
      <p class="muted">Este jogo não abre no RetroArch: configure pelo emulador dele.</p>
    {:else}
      <p class="lead muted">Só pra este jogo, neste perfil. Vale na próxima vez que ele abrir.</p>
      {#each groups as gr, gi (gr.key)}
        <fieldset class="group">
          <legend id={`cfg-${gr.key}`}>{gr.label}</legend>
          <div class="opts" role="radiogroup" aria-labelledby={`cfg-${gr.key}`} data-nav-group={`cfg-${gr.key}`}>
            {#each gr.opts as o, oi (o.value)}
              {@const on = (info.settings[gr.key] ?? '') === o.value}
              <button
                class="choice"
                role="radio"
                aria-checked={on}
                data-nav
                data-nav-default={gi === 0 && oi === 0 ? '' : undefined}
                onclick={() => pick(gr.key, o.value)}
              >
                <span class="mark radio" aria-hidden="true"></span>{o.label}
              </button>
            {/each}
          </div>
          {#if gr.hint}<p class="muted hint">{gr.hint}</p>{/if}
        </fieldset>
      {/each}
    {/if}
    <div class="actions">
      <button class="btn primary" data-nav data-nav-default={info?.supported ? undefined : ''} onclick={onclose}>Pronto</button>
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
    width: min(44rem, 100% - 2rem);
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
  .opts {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 0.5rem;
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
  .hint {
    margin: 0.4rem 0 0;
    font-size: 0.95rem;
  }
  .error {
    color: var(--danger);
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
