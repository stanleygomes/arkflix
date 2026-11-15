<script lang="ts">
  // Avisos curtos (toast da noemi-ui): caixa clara de papel com sombra dura,
  // um quadradinho com a cor do sentido (verde-azulado ok/info, vermelho
  // erro). Sobe 16px em degraus ao chegar e desce ao sair.
  import { reduced, stepped } from '../lib/motion'
  import { app } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  const kicker = { ok: 'Pronto', error: 'Erro', info: 'Aviso' } as const

  function rise(_n: Element) {
    if (reduced()) return { duration: 0 }
    return { duration: 200, css: (t: number) => `transform: translateY(${(1 - stepped(t, 4)) * 16}px); opacity: ${stepped(t, 4)};` }
  }
  function sink(_n: Element) {
    if (reduced()) return { duration: 0 }
    return { duration: 150, css: (t: number) => `transform: translateY(${(1 - stepped(t, 3)) * 16}px); opacity: ${stepped(t, 3)};` }
  }
</script>

<div class="toasts" role="status" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="toast {t.kind}" in:rise out:sink>
      <span class="mark" aria-hidden="true"><Icon name={t.kind === 'error' ? 'warn' : t.kind === 'ok' ? 'check' : 'info'} /></span>
      <span class="copy">
        <span class="kicker meta">{kicker[t.kind] ?? 'Aviso'}</span>
        <span class="text">{t.text}</span>
      </span>
    </div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    inset-inline-end: calc(1rem + var(--safe));
    bottom: calc(4.5rem + var(--safe));
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    z-index: 97; /* acima do guia */
    max-width: min(30rem, 100vw - 2rem);
    pointer-events: none;
  }
  .toast {
    display: flex;
    gap: 0.75rem;
    align-items: flex-start;
    padding: 0.7rem 1rem 0.75rem 0.75rem;
    background: var(--text);
    color: var(--bg);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 4px 4px 0 var(--outline);
    white-space: pre-line;
  }
  .mark {
    flex: none;
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border: 2px solid var(--bg);
    border-radius: 4px 5px 4px 5px;
    background: var(--accent-2);
    color: var(--bg);
  }
  .error .mark {
    background: var(--danger);
  }
  .copy {
    display: grid;
    gap: 0.1rem;
  }
  .kicker {
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .text {
    font-weight: 700;
    font-size: 1rem;
    line-height: 1.35;
  }
</style>
