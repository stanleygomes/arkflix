<script lang="ts">
  // Capa quadrada de álbum, artista ou playlist (cartão de imagem sóbrio:
  // moldura de 3px, cantos irregulares, sombra dura). Imagem pelo proxy de
  // músicas; sem imagem (ou falhou), o nome num cartão listrado. Selo "No PC"
  // quando tem cópia baixada. Quem usa decide o foco.
  import Icon from './Icon.svelte'

  // showName: o nome no cartão sem capa (desligue quando o nome já está embaixo)
  let { src = '', name, round = false, local = false, showName = true }: { src?: string; name: string; round?: boolean; local?: boolean; showName?: boolean } = $props()
  let failed = $state(false)
  $effect(() => {
    void src
    failed = false
  })
</script>

<span class="cover" class:round>
  {#if src && !failed}
    <img {src} alt="" loading="lazy" onerror={() => (failed = true)} />
  {:else}
    <span class="blank" aria-hidden="true"><Icon name={round ? 'user' : 'music'} size="1.6em" />{#if showName}<span class="name">{name}</span>{/if}</span>
  {/if}
  <!-- o texto vem do CSS: o nome acessível de quem usa já diz "No PC" -->
  {#if local}<span class="badge" aria-hidden="true"><Icon name="download" size="0.9em" /></span>{/if}
</span>

<style>
  .cover {
    position: relative;
    display: block;
    aspect-ratio: 1;
    overflow: hidden;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .cover.round {
    border-radius: 50%;
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  /* sem capa: listras finas de papel velho e o nome em Bevan */
  .blank {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.4rem;
    padding: 0.6rem;
    text-align: center;
    color: var(--muted);
    background: repeating-linear-gradient(-45deg, var(--surface) 0 10px, color-mix(in srgb, var(--outline) 7%, var(--surface)) 10px 12px);
  }
  .name {
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.04em;
    color: var(--text);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .badge {
    position: absolute;
    inset-block-end: 0.4rem;
    inset-inline-start: 0.4rem;
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.15rem 0.45rem;
    font-family: var(--font-hud);
    font-size: 0.72rem;
    color: var(--bg);
    background: var(--accent-2);
    border: 2px solid var(--outline);
    border-radius: 4px;
  }
  .badge::after {
    content: 'No PC';
  }
</style>
