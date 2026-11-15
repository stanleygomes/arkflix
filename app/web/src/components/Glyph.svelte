<script lang="ts">
  // Símbolo do botão no estilo do controle em uso (A/B do Xbox, ✕/○ do PS...).
  import { glyph } from '../lib/keys'
  import type { Family } from '../lib/types'

  let { button, family }: { button: string; family: Family } = $props()
  const text = $derived(glyph(button, family))
</script>

<kbd class="glyph {family} {button}" class:round={['south', 'east', 'west', 'north'].includes(button) && family !== 'keyboard'}>{text}</kbd>

<style>
  /* Tecla/botão (kbd da noemi-ui): papel claro, borda de 2px, sombrinha
     dura, letra datilografada. Botões de face do controle ficam redondos.
     Sem cor por controle: o símbolo já diz qual é. */
  .glyph {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 0.82rem;
    line-height: 1;
    min-width: 1.8em;
    height: 1.8em;
    padding: 0 0.4em;
    display: inline-grid;
    place-items: center;
    background: var(--balloon);
    color: var(--text);
    border: 2px solid var(--outline);
    border-radius: 4px 5px 4px 5px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .round {
    border-radius: 50%;
    padding: 0;
  }
</style>
