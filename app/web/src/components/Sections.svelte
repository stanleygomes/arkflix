<script lang="ts">
  // Jogos ou Filmes e séries: as duas metades do arkflix, no topo do Início
  // de cada uma. Trocar é "mudar de canal" (recomeça a navegação).
  import { play } from '../lib/sound'
  import { resetTo } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  let { current }: { current: 'games' | 'watch' } = $props()

  function goTo(s: 'games' | 'watch') {
    if (s === current) return
    play('accept')
    resetTo({ name: s === 'games' ? 'home' : 'watch' })
  }
</script>

<nav class="sections" aria-label="Seção">
  <button class="section" data-nav data-section="games" aria-current={current === 'games' ? 'page' : undefined} onclick={() => goTo('games')}>
    <Icon name="pad" /> Jogos
  </button>
  <button class="section" data-nav data-section="watch" aria-current={current === 'watch' ? 'page' : undefined} onclick={() => goTo('watch')}>
    <Icon name="film" /> Filmes e séries
  </button>
</nav>

<style>
  /* nav sóbria: rótulos em Bevan, a seção atual com o traço mostarda embaixo */
  .sections {
    display: inline-flex;
    gap: 0.4rem;
  }
  .section {
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--muted);
    background: none;
    border: none;
    border-bottom: var(--edge) solid transparent;
    border-radius: 0;
    padding: 0.55rem 0.55rem 0.45rem;
    display: inline-flex;
    align-items: center;
    gap: 0.55rem;
    cursor: pointer;
    white-space: nowrap;
    transition:
      color var(--t) var(--ease),
      border-color var(--t) var(--ease);
  }
  .section:hover {
    color: var(--text);
    border-bottom-color: var(--outline);
  }
  .section[aria-current='page'] {
    color: var(--text);
    border-bottom-color: var(--accent);
  }
</style>
