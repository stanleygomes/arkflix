<script lang="ts">
  // Pôster de filme/série (ou miniatura de episódio), como cartão de imagem da
  // noemi-ui: moldura reta de 3px, cantos irregulares e sombra dura. Imagem do
  // Jellyfin pelo proxy; sem imagem (ou falhou), o nome num cartão hachurado.
  // Quem usa decide o foco (sobe + sombra maior + anel verde-azulado).
  import { displayName, episodeTag } from '../lib/watch'
  import type { WatchItem } from '../lib/types'
  import Icon from './Icon.svelte'

  let { item, wide = false }: { item: WatchItem; wide?: boolean } = $props()
  const src = $derived(wide ? item.thumb || item.backdrop || item.poster : item.poster)
  let failed = $state(false)
  let ready = $state(false)
  $effect(() => {
    void src
    failed = false
    ready = false
  })
</script>

<span class="poster" class:wide>
  {#if src && !failed}
    <img {src} alt="" loading="lazy" class:ready onload={() => (ready = true)} onerror={() => (failed = true)} />
  {:else}
    <span class="blank" aria-hidden="true">
      <span class="name">{displayName(item)}</span>
      {#if episodeTag(item)}<span class="tag">{episodeTag(item)}</span>{/if}
    </span>
  {/if}
  {#if item.played}
    <span class="badge ok" aria-hidden="true"><Icon name="check" size="0.95em" /> Assistido</span>
  {:else if item.type === 'Series' && item.unplayed > 0}
    <span class="badge count" aria-hidden="true">{item.unplayed}</span>
  {/if}
  {#if item.progress > 0 && !item.played}
    <span class="bar" aria-hidden="true"><span style:width={`${Math.max(4, item.progress * 100)}%`}></span></span>
  {/if}
</span>

<style>
  .poster {
    position: relative;
    display: block;
    aspect-ratio: 2 / 3;
    overflow: hidden;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .poster.wide {
    aspect-ratio: 16 / 9;
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  /* imagem chegou: aparece em dois degraus, uma vez */
  img.ready {
    animation: arrive 120ms steps(2);
  }
  @keyframes arrive {
    from {
      opacity: 0;
    }
  }
  /* sem imagem: cartão hachurado com o nome no meio */
  .blank {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 0.5rem;
    padding: 0.8rem;
    text-align: center;
    background: repeating-linear-gradient(
      135deg,
      transparent 0 10px,
      color-mix(in srgb, var(--outline) 12%, transparent) 10px 16px
    );
  }
  .blank .name {
    font-family: var(--font-label);
    font-size: 0.85rem;
    line-height: 1.25;
    color: var(--text);
    background: var(--surface);
    padding: 0.2rem 0.4rem;
  }
  .blank .tag {
    font-family: var(--font-hud);
    font-size: 0.75rem;
    letter-spacing: 0.08em;
    color: var(--text);
    background: var(--surface);
    padding: 0.1rem 0.4rem;
  }
  /* etiqueta datilografada no canto de cima, 8px pra dentro */
  .badge {
    position: absolute;
    top: 0.45rem;
    inset-inline-start: 0.45rem;
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    min-width: 1.4rem;
    min-height: 1.4rem;
    justify-content: center;
    padding: 0.2rem 0.4rem 0.05rem;
    font-family: var(--font-hud);
    font-size: 0.72rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    line-height: 1;
    border: 2px solid var(--n-on-mustard);
    border-radius: 4px;
  }
  .badge.ok {
    background: var(--accent-2);
    color: var(--bg);
  }
  .badge.count {
    background: var(--text);
    color: var(--bg);
    font-variant-numeric: tabular-nums;
  }
  /* quanto já foi visto: trilho de cartão, preenchimento verde-azulado */
  .bar {
    position: absolute;
    inset-inline: 0;
    bottom: 0;
    height: 0.6rem;
    background: var(--surface);
    border-top: var(--edge) solid var(--outline);
  }
  .bar span {
    display: block;
    height: 100%;
    background: var(--accent-2);
  }
</style>
