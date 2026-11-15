<script lang="ts">
  // Menu de energia (só no app desktop): minimizar, tela cheia/janela, fechar.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, confirm, toast } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  let fullscreen = $state(true)
  let previous: HTMLElement | null = null

  function close() {
    app.power = false
    previous?.focus()
  }

  $effect(() => {
    if (!app.power) return
    previous = document.activeElement as HTMLElement | null
    api.isFullscreen().then((v) => (fullscreen = v)).catch(() => {})
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.confirm) return false
      if (a === 'back') {
        play('back')
        close()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  async function run(fn: () => Promise<unknown>) {
    try {
      await fn()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function minimize() {
    close()
    await run(api.minimize)
  }

  async function toggle() {
    close()
    await run(api.toggleFullscreen)
  }

  async function quit() {
    if (!(await confirm('Fechar o arkflix?', 'Fechar'))) return
    close()
    await run(api.quit)
  }
</script>

{#if app.power}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="dialog" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="power-title" data-nav-scope={app.confirm ? undefined : ''}>
      <h2 id="power-title">Energia</h2>
      <div class="options">
        <button class="btn" data-nav data-nav-default onclick={minimize}><Icon name="back" /> Minimizar</button>
        <button class="btn" data-nav onclick={toggle}><Icon name="sliders" /> {fullscreen ? 'Sair da tela cheia' : 'Tela cheia'}</button>
        <button class="btn danger" data-nav onclick={quit}><Icon name="power" /> Fechar o app</button>
      </div>
      <div class="actions">
        <button class="btn" data-nav onclick={close}>Cancelar</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Diálogo sóbrio (noemi-ui): uma coluna de botões grandes, o perigoso
     por último; Cancelar no rodapé. */
  .backdrop {
    position: fixed;
    inset: 0;
    /* tinta (#24170f, o contorno do tema claro) a 55%, sem blur: o creme
       do contorno escuro lavaria a tela */
    background: color-mix(in srgb, var(--on-accent) 55%, transparent);
    display: grid;
    place-items: center;
    padding: var(--safe);
    z-index: 85;
  }
  .dialog {
    width: min(24rem, 100%);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 8px 8px 0 var(--outline);
  }
  h2 {
    font-size: 1.2rem;
    padding: 1.1rem 1.25rem 0;
  }
  .options {
    display: grid;
    gap: 0.9rem;
    padding: 1rem 1.25rem 1.4rem;
  }
  .options .btn {
    justify-content: flex-start;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    padding: 1rem 1.25rem;
    border-block-start: var(--edge) solid var(--outline);
  }
</style>
