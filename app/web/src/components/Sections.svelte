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
  .sections {
    display: inline-flex;
    gap: 0.3rem;
    padding: 0.25rem;
    background: color-mix(in srgb, var(--surface) 75%, transparent);
    border: 2px solid color-mix(in srgb, var(--text) 18%, transparent);
    box-shadow: 3px 3px 0 #000;
  }
  /* fonte de texto: nome curto precisa ser lido de longe */
  .section {
    font-family: var(--font-body);
    font-weight: 700;
    font-size: 1.05rem;
    letter-spacing: 0.03em;
    color: #d8cfe6;
    background: none;
    border: 2px solid transparent;
    padding: 0.35rem 0.8rem;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    white-space: nowrap;
  }
  .section:hover {
    border-color: var(--accent);
  }
  /* a seção atual vira a "fita" acesa */
  .section[aria-current='page'] {
    color: var(--on-accent);
    background: var(--accent);
  }
</style>
