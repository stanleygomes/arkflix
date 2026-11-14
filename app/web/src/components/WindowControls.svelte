<script lang="ts">
  // Botões da janela (a moldura do sistema sai: a barra de título é a nossa).
  // Só no desktop e fora da tela cheia; no controle, o menu Energia faz o mesmo.
  import { onMount } from 'svelte'
  import { api } from '../lib/api'
  import { app } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  let fullscreen = $state(true)
  let maximised = $state(false)

  async function refresh() {
    try {
      const w = await api.windowState()
      fullscreen = w.fullscreen
      maximised = w.maximised
    } catch {
      // modo web: sem janela
    }
  }

  onMount(() => {
    if (app.mode !== 'desktop') return
    refresh()
    // F11, Energia, arrastar até a borda: a janela muda de tamanho
    let t = 0
    const onResize = () => {
      clearTimeout(t)
      t = window.setTimeout(refresh, 120)
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  })

  async function run(fn: () => Promise<void>) {
    await fn().catch(() => {})
    refresh()
  }
</script>

{#if app.mode === 'desktop' && !fullscreen}
  <div class="win" role="group" aria-label="Janela">
    <button class="wbtn" aria-label="Minimizar" title="Minimizar" onclick={() => run(api.minimize)}>
      <Icon name="minimize" size="0.95em" />
    </button>
    <button class="wbtn" aria-label={maximised ? 'Restaurar' : 'Maximizar'} title={maximised ? 'Restaurar' : 'Maximizar'} onclick={() => run(api.toggleMaximise)}>
      <Icon name={maximised ? 'restore' : 'maximize'} size="0.95em" />
    </button>
    <button class="wbtn close" aria-label="Fechar o arkflix" title="Fechar" onclick={() => run(api.quit)}>
      <Icon name="close" size="0.95em" />
    </button>
  </div>
{/if}

<style>
  .win {
    display: flex;
    gap: 0.4rem;
    --wails-draggable: no-drag;
  }
  /* blocos de console, menores que os da barra: não competem com o conteúdo */
  .wbtn {
    width: 2.2rem;
    height: 2.2rem;
    display: grid;
    place-items: center;
    color: var(--text);
    background: color-mix(in srgb, var(--surface) 70%, transparent);
    border: 2px solid color-mix(in srgb, var(--text) 18%, transparent);
    box-shadow: 2px 2px 0 #000;
    cursor: pointer;
  }
  .wbtn:hover,
  .wbtn:focus-visible {
    border-color: var(--accent);
    transform: translate(-1px, -1px);
    box-shadow: 3px 3px 0 #000;
  }
  .wbtn:active {
    transform: translate(1px, 1px);
    box-shadow: none;
  }
  .close:hover,
  .close:focus-visible {
    color: #fff;
    background: #c4253f;
    border-color: #ff6b81;
  }
  :global(.reduce-motion) .wbtn {
    transform: none !important;
  }
</style>
