<script lang="ts">
  // Dicas de botão no rodapé, com os símbolos do controle em uso, como teclas
  // (kbd da noemi-ui): tecla creme de máquina, sem cor por botão.
  // `coin` ficou por compatibilidade (era o "INSERT COIN" do visual antigo).
  import { glyphFamily } from '../lib/state.svelte'
  import Glyph from './Glyph.svelte'

  let { hints }: { hints: { button: string; label: string }[]; coin?: boolean } = $props()
  const family = $derived(glyphFamily())
</script>

<footer class="help" aria-label="Atalhos">
  {#each hints as h (h.button + h.label)}
    <span class="hint"><Glyph button={h.button} {family} /> {h.label}</span>
  {/each}
</footer>

<style>
  .help {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem 1.6rem;
    justify-content: flex-end;
    align-items: center;
    padding: 0.8rem 2.5rem 1.1rem;
    border-top: var(--edge) solid color-mix(in srgb, var(--outline) 22%, transparent);
    color: var(--muted);
    font-size: 0.95rem;
  }
  .hint {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    white-space: nowrap;
  }
  /* tecla: Courier Prime em negrito, borda e sombra dura pequenas.
     Sobrescreve as cores por botão do Glyph (o símbolo já distingue). */
  .help .hint :global(kbd.glyph) {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 0.82rem;
    line-height: 1;
    min-width: 1.75rem;
    height: 1.75rem;
    padding: 0 0.4rem;
    display: inline-grid;
    place-items: center;
    color: var(--text);
    background: var(--balloon);
    border: 2px solid var(--outline);
    border-radius: 4px 5px 4px 5px;
    box-shadow: 2px 2px 0 var(--outline);
    text-transform: uppercase;
  }
  .help .hint :global(kbd.glyph.round) {
    padding: 0;
    width: 1.75rem;
    border-radius: 50%;
  }
</style>
