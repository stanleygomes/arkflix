<script lang="ts">
  // Mídia do jogo em tela cheia: capa, prints, tela-título, miniaturas dos
  // saves e o manual. Esquerda/direita troca, aceitar dá zoom, voltar fecha.
  import { tick, untrack } from 'svelte'
  import { api, isDesktop } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { toast } from '../lib/state.svelte'
  import type { MediaItem } from '../lib/types'
  import Icon from './Icon.svelte'

  let {
    items,
    start = 0,
    gameId,
    onclose,
  }: { items: MediaItem[]; start?: number; gameId: string; onclose: () => void } = $props()

  let index = $state(untrack(() => start))
  let zoom = $state(false)
  const item = $derived(items[index])

  function step(d: number) {
    if (items.length < 2) return
    index = (index + d + items.length) % items.length
    zoom = false
    play('move')
  }

  async function openManual() {
    try {
      await api.openManual(gameId)
      toast('Manual aberto no leitor de PDF', 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      if (a === 'left' || a === 'prevTab') {
        step(-1)
        return true
      }
      if (a === 'right' || a === 'nextTab') {
        step(1)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'search' || a === 'random'
    })
  })
</script>

<div class="viewer" role="dialog" aria-modal="true" aria-labelledby="media-caption" data-nav-scope="">
  <p class="counter" aria-live="polite">{index + 1} / {items.length}</p>
  <div class="stage">
    {#if item.kind === 'pdf'}
      {#if isDesktop()}
        <div class="pdf">
          <Icon name="folder" />
          <p>O manual abre no leitor de PDF do sistema.</p>
          <button class="btn primary" data-nav data-nav-default onclick={openManual}>Abrir manual</button>
        </div>
      {:else}
        <iframe src={item.src} title={item.caption}></iframe>
        <button class="sr-only" data-nav data-nav-default onclick={() => window.open(item.src, '_blank')}>Abrir manual em outra aba</button>
      {/if}
    {:else if item.kind === 'video'}
      <!-- svelte-ignore a11y_media_has_caption -->
      <video class="clip" src={item.src} controls data-nav data-nav-default></video>
    {:else}
      <button class="shot" class:zoom data-nav data-nav-default aria-pressed={zoom} onclick={() => (zoom = !zoom)}>
        <img src={item.src} alt={item.caption} />
        <span class="sr-only">Zoom</span>
      </button>
    {/if}
  </div>
  <p id="media-caption" class="caption">{item.caption}</p>
  <div class="nav">
    <button class="btn" data-nav disabled={items.length < 2} onclick={() => step(-1)}><Icon name="back" /> Anterior</button>
    <button class="btn" data-nav onclick={onclose}>Fechar</button>
    <button class="btn" data-nav disabled={items.length < 2} onclick={() => step(1)}>Próxima <Icon name="play" /></button>
  </div>
</div>

<style>
  /* sala escura: a mídia no meio, numa moldura reta com sombra dura */
  .viewer {
    position: fixed;
    inset: 0;
    z-index: 85;
    background: color-mix(in srgb, var(--bg) 96%, transparent);
    display: grid;
    grid-template-rows: auto 1fr auto auto;
    gap: 1rem;
    padding: 1.2rem 2rem 1.5rem;
    color: var(--text);
  }
  .counter {
    justify-self: end;
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.1rem;
    font-variant-numeric: tabular-nums;
    padding: 0.2rem 0.7rem;
    border: 2px solid var(--outline);
    border-radius: 4px;
  }
  .stage {
    min-height: 0;
    display: grid;
    place-items: center;
    overflow: auto;
    padding: 0.5rem 0.8rem 0.8rem 0.5rem;
  }
  .shot {
    padding: 0;
    border: 0;
    background: none;
    cursor: zoom-in;
    max-width: 100%;
    max-height: 100%;
  }
  .shot img,
  .clip {
    display: block;
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 6px 6px 0 var(--outline);
    background: var(--bg);
  }
  /* print de jogo é pequeno: estica até a altura da sala, sem borrar */
  .shot img {
    height: 66vh;
    width: auto;
    max-width: 100%;
    object-fit: contain;
    image-rendering: pixelated;
  }
  .clip {
    max-width: 100%;
    max-height: 100%;
  }
  .shot.zoom {
    cursor: zoom-out;
  }
  .shot.zoom img {
    max-width: none;
    max-height: none;
    height: auto;
    width: min(200%, 160vw);
  }
  iframe {
    width: min(60rem, 100%);
    height: 100%;
    min-height: 60vh;
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 6px 6px 0 var(--outline);
    background: #fff;
  }
  .pdf {
    display: grid;
    gap: 1rem;
    justify-items: center;
    text-align: center;
    font-size: 1.15rem;
    padding: 2rem 2.5rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
  }
  .pdf :global(svg) {
    width: 3rem;
    height: 3rem;
    color: var(--muted);
  }
  .caption {
    text-align: center;
    font-family: var(--font-label);
    font-size: 1rem;
    letter-spacing: 0.04em;
  }
  .nav {
    display: flex;
    justify-content: center;
    gap: 1rem;
  }
  /* troca de mídia: entra em degraus, uma vez */
  @media (prefers-reduced-motion: no-preference) {
    .shot img {
      animation: pop 0.18s steps(3);
    }
  }
  @keyframes pop {
    from {
      transform: translateY(-12px);
      opacity: 0.4;
    }
  }
</style>
