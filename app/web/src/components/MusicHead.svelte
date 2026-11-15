<script lang="ts">
  // Cabeçalho das páginas de álbum, artista e playlist: capa grande, tipo,
  // nome (Limelight), uma linha de detalhes e as ações. Favorito ganha o
  // 🎬 carimbo (o elemento barulhento da tela).
  import type { Snippet } from 'svelte'
  import Cover from './Cover.svelte'

  let {
    kind,
    title,
    cover = '',
    round = false,
    local = false,
    starred = false,
    details = [],
    children,
  }: { kind: string; title: string; cover?: string; round?: boolean; local?: boolean; starred?: boolean; details?: string[]; children?: Snippet } = $props()
</script>

<header class="head">
  <div class="art">
    <Cover src={cover} name={title} {round} {local} />
    {#if starred}<span class="stamp new" aria-hidden="true">Favorito</span>{/if}
  </div>
  <div class="text">
    <p class="kind"><span class="chip">{kind}</span></p>
    <h1 id="music-page-title">{title}</h1>
    {#if details.length}
      <p class="details">
        {#each details as d (d)}<span>{d}</span>{/each}
      </p>
    {/if}
    <div class="actions">{@render children?.()}</div>
  </div>
</header>

<style>
  .head {
    display: grid;
    grid-template-columns: clamp(9rem, 22vh, 15rem) 1fr;
    gap: 2rem;
    align-items: end;
  }
  .art {
    position: relative;
  }
  .art .stamp {
    position: absolute;
    inset-block-start: -0.6rem;
    inset-inline-end: -1rem;
    background: var(--bg);
    animation: stamp 220ms steps(3) 1;
  }
  @keyframes stamp {
    from {
      transform: rotate(-4deg) scale(1.6);
      opacity: 0;
    }
  }
  :global(.reduce-motion) .art .stamp {
    animation: none;
  }
  .text {
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
    min-width: 0;
  }
  h1 {
    font-size: clamp(1.8rem, min(4vw, 6vh), 3.4rem);
    overflow-wrap: anywhere;
  }
  .details {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem 1.2rem;
    color: var(--muted);
    font-family: var(--font-hud);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.9rem;
    margin-top: 0.4rem;
  }
  /* 🎬 bilhete de cinema (a ação da tela): entalhe de cada lado; a sombra
     e o anel de foco moram no invólucro, que a máscara não corta */
  .actions :global(.ticket) {
    display: inline-flex;
    border-radius: var(--radii-sm);
    filter: drop-shadow(3px 4px 0 var(--outline));
    transition:
      transform var(--t) var(--ease),
      filter var(--t) var(--ease);
  }
  .actions :global(.ticket:hover),
  .actions :global(.ticket:focus-within) {
    transform: translate(-2px, -2px);
    filter: drop-shadow(5px 6px 0 var(--outline));
  }
  .actions :global(.ticket:focus-within) {
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .actions :global(.ticket button) {
    --notch: radial-gradient(circle 8px at 0 50%, #0000 97%, #000);
    --notch-end: radial-gradient(circle 8px at 100% 50%, #0000 97%, #000);
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    min-height: 3rem;
    padding: 0.7rem 1.6rem;
    font-family: var(--font-label);
    font-size: 0.85rem;
    line-height: 1;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--on-accent);
    background: var(--accent);
    border: none;
    border-radius: var(--radii-sm);
    cursor: pointer;
    -webkit-mask:
      var(--notch) left / 51% 100% no-repeat,
      var(--notch-end) right / 51% 100% no-repeat;
    mask:
      var(--notch) left / 51% 100% no-repeat,
      var(--notch-end) right / 51% 100% no-repeat;
  }
  .actions :global(.ticket button:focus) {
    outline: none;
  }
  .actions :global(.ticket button:disabled) {
    opacity: 0.6;
    cursor: default;
  }
</style>
