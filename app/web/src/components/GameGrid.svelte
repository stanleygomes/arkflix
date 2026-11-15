<script lang="ts">
  // Grade de jogos (busca, biblioteca, coleções). Confirmar abre o jogo.
  import { go, system } from '../lib/state.svelte'
  import type { Game } from '../lib/types'
  import GameArt from './GameArt.svelte'
  import Icon from './Icon.svelte'

  let { games, label, showSystem = true }: { games: Game[]; label: string; showSystem?: boolean } = $props()
</script>

<ul class="grid" aria-label={label} data-nav-group="grid">
  {#each games as g (g.id)}
    {@const sys = system(g.system)}
    <li>
      <button class="card" data-nav data-game={g.id} onclick={() => go({ name: 'game', id: g.id })}>
        <GameArt game={g} system={sys} />
        <span class="caption">
          <span class="name">{g.name}</span>
          {#if showSystem}<span class="sys">{sys?.short ?? g.system}{g.year ? ` · ${g.year}` : ''}</span>{/if}
        </span>
        {#if g.favorite}<span class="badge" aria-label="favorito"><Icon name="star" /></span>{/if}
      </button>
    </li>
  {/each}
</ul>

<style>
  .grid {
    list-style: none;
    margin: 0;
    padding: 0.6rem 0.6rem 1.2rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
    gap: 1.8rem 1.6rem;
  }
  /* cartão de imagem sóbrio e clicável: capa em cima, legenda embaixo */
  .card {
    position: relative;
    width: 100%;
    height: 100%;
    padding: 0;
    color: var(--text);
    font: inherit;
    text-align: start;
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    overflow: hidden;
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  /* a capa ocupa a largura toda; a borda do cartão é a dela */
  .card :global(.art) {
    border: none;
    border-bottom: var(--edge) solid var(--outline);
    border-radius: 0;
  }
  .card:hover,
  .card:focus-visible,
  .card:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card:active {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  .caption {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.7rem 0.85rem 0.8rem;
  }
  .name {
    font-family: var(--font-label);
    font-size: 0.88rem;
    line-height: 1.25;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
  }
  .sys {
    font-family: var(--font-hud);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--muted);
  }
  /* favorito: etiqueta mostarda no canto da capa */
  .badge {
    position: absolute;
    top: 0.5rem;
    inset-inline-start: 0.5rem;
    color: var(--on-accent);
    background: var(--accent);
    border: 2px solid var(--on-accent);
    border-radius: 4px;
    padding: 0.2rem;
    display: grid;
  }
</style>
