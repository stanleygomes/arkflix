<script lang="ts">
  // Teclado na tela, pra digitar só com o controle. O campo de texto de verdade
  // fica acima (dá pra digitar no teclado físico também).
  import { play } from '../lib/sound'

  let {
    value = $bindable(''),
    onsubmit,
    submitLabel = 'OK',
    max = 60,
    autofocus = false,
  }: { value?: string; onsubmit?: () => void; submitLabel?: string; max?: number; autofocus?: boolean } = $props()

  let upper = $state(false)
  const rows = ['1234567890', 'qwertyuiop', 'asdfghjkl-', "zxcvbnm.'&"]

  function type(ch: string) {
    if (value.length >= max) return play('error')
    value += upper ? ch.toUpperCase() : ch
    play('move')
  }
</script>

<div class="osk" role="group" aria-label="Teclado na tela">
  {#each rows as row, r (r)}
    <div class="row">
      {#each row.split('') as ch, c (c)}
        <button
          type="button"
          class="key"
          data-nav
          data-key={ch}
          data-sfx="none"
          data-nav-default={autofocus && r === 1 && c === 0 ? '' : undefined}
          onclick={() => type(ch)}>{upper ? ch.toUpperCase() : ch}</button
        >
      {/each}
    </div>
  {/each}
  <div class="row">
    <button type="button" class="key wide" data-nav data-sfx="none" aria-pressed={upper} onclick={() => (upper = !upper)}>Maiúsculas</button>
    <button type="button" class="key space" data-nav data-sfx="none" onclick={() => type(' ')}>Espaço</button>
    <button type="button" class="key wide" data-nav data-sfx="none" onclick={() => ((value = value.slice(0, -1)), play('back'))}>Apagar</button>
    <button type="button" class="key wide" data-nav data-sfx="none" onclick={() => ((value = ''), play('back'))}>Limpar</button>
    {#if onsubmit}
      <button type="button" class="key wide ok" data-nav onclick={onsubmit}>{submitLabel}</button>
    {/if}
  </div>
</div>

<style>
  /* teclado na tela: teclas grandes (kbd lg da noemi-ui) num painel sóbrio */
  .osk {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.9rem;
    width: fit-content;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
  }
  .row {
    display: flex;
    gap: 0.5rem;
  }
  .key {
    min-width: 2.7rem;
    height: 2.7rem;
    padding: 0 0.4rem;
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.05rem;
    text-transform: uppercase;
    color: var(--text);
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 4px 5px 4px 5px;
    box-shadow: var(--hard-sm);
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  /* letra minúscula aparece minúscula: só as teclas de letra seguem o Shift */
  .key[data-key] {
    text-transform: none;
  }
  .key:hover,
  .key:focus-visible {
    transform: translate(-2px, -2px);
    box-shadow: 5px 5px 0 var(--outline);
  }
  .key:active {
    transform: translate(3px, 3px);
    box-shadow: none;
  }
  .key.wide,
  .key.space {
    font-family: var(--font-label);
    font-weight: 400;
    font-size: 0.72rem;
    letter-spacing: 0.06em;
    padding: 0 0.7rem;
  }
  .key.space {
    flex: 1;
  }
  .key.ok {
    background: var(--accent);
    color: var(--on-accent);
  }
  .key[aria-pressed='true'] {
    transform: translate(3px, 3px);
    box-shadow: none;
    background: var(--accent-2);
    color: var(--bg);
  }
</style>
