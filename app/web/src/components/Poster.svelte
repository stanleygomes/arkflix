<script lang="ts">
  // Pôster de filme/série (ou miniatura de episódio): imagem do Jellyfin pelo
  // proxy; sem imagem (ou falhou), o nome em pixel numa "fita VHS".
  import { displayName, episodeTag } from '../lib/watch'
  import type { WatchItem } from '../lib/types'
  import Icon from './Icon.svelte'

  let { item, wide = false }: { item: WatchItem; wide?: boolean } = $props()
  const src = $derived(wide ? item.thumb || item.backdrop || item.poster : item.poster)
  let failed = $state(false)
  $effect(() => {
    void src
    failed = false
  })
</script>

<span class="poster" class:wide>
  {#if src && !failed}
    <img {src} alt="" loading="lazy" onerror={() => (failed = true)} />
  {:else}
    <span class="tape pixel" aria-hidden="true">
      <span class="name">{displayName(item)}</span>
      {#if episodeTag(item)}<span class="tag">{episodeTag(item)}</span>{/if}
    </span>
  {/if}
  {#if item.played}
    <span class="seen" aria-hidden="true"><Icon name="check" size="0.9em" /></span>
  {:else if item.type === 'Series' && item.unplayed > 0}
    <span class="seen count" aria-hidden="true">{item.unplayed}</span>
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
    border-radius: 6px;
    background: #0b0a12;
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
  .tape {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.8rem;
    color: #fff;
    text-align: center;
    font-size: 0.8rem;
    line-height: 1.5;
    background:
      repeating-linear-gradient(0deg, transparent 0 6px, rgb(255 255 255 / 0.04) 6px 7px),
      linear-gradient(160deg, color-mix(in srgb, var(--accent) 45%, #111), #111);
  }
  .tape .tag {
    color: var(--focus);
  }
  .seen {
    position: absolute;
    top: 0.4rem;
    right: 0.4rem;
    min-width: 1.6rem;
    height: 1.6rem;
    display: grid;
    place-items: center;
    padding: 0 0.3rem;
    font-weight: 700;
    font-size: 0.85rem;
    color: var(--on-accent);
    background: var(--accent);
    border: 2px solid #05040a;
  }
  /* barra do que já foi visto, como fita rebobinada */
  .bar {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 0.45rem;
    background: rgb(0 0 0 / 0.7);
  }
  .bar span {
    display: block;
    height: 100%;
    background: var(--focus);
  }
</style>
