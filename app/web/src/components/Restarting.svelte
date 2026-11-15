<script lang="ts">
  // "Atualizando": cobre tudo enquanto o app instala e reabre. Cartão de
  // "intervalo técnico" parado (nada anima: a espera não tem fim conhecido,
  // então o texto diz o que acontece).
  import { app } from '../lib/state.svelte'
</script>

{#if app.restarting}
  <div class="restarting" role="alertdialog" aria-modal="true" aria-labelledby="restart-title" aria-describedby="restart-text">
    <span class="reel" aria-hidden="true"></span>
    <p class="kicker meta" aria-hidden="true">Intervalo técnico</p>
    <h2 id="restart-title">Atualizando pra {app.restarting}</h2>
    <p id="restart-text" class="muted">O app fecha e abre de novo sozinho em alguns segundos.</p>
  </div>
{/if}

<style>
  .restarting {
    position: fixed;
    inset: 0;
    z-index: 30000;
    background: var(--bg);
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 0.9rem;
    padding: var(--safe);
    text-align: center;
  }
  /* rolo de filme parado: disco claro, 5 furos e o miolo escuros */
  .reel {
    width: 5rem;
    height: 5rem;
    border: var(--edge) solid var(--outline);
    border-radius: 50%;
    box-shadow: var(--hard);
    background:
      radial-gradient(circle at 50% 50%, var(--bg) 0 12%, transparent 13%),
      radial-gradient(circle at 50% 20%, var(--bg) 0 11%, transparent 12%),
      radial-gradient(circle at 78.5% 40.7%, var(--bg) 0 11%, transparent 12%),
      radial-gradient(circle at 67.6% 74.3%, var(--bg) 0 11%, transparent 12%),
      radial-gradient(circle at 32.4% 74.3%, var(--bg) 0 11%, transparent 12%),
      radial-gradient(circle at 21.5% 40.7%, var(--bg) 0 11%, transparent 12%),
      var(--muted);
    margin-block-end: 0.6rem;
  }
  .kicker {
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.16em;
    color: var(--muted);
  }
  h2 {
    font-family: var(--font-title);
    font-size: clamp(2rem, 5vw, 3.4rem);
    line-height: 1.1;
    color: var(--accent);
    -webkit-text-stroke: 1.5px var(--on-accent);
    paint-order: stroke fill;
    text-shadow: 3px 3px 0 var(--outline);
  }
  #restart-text {
    max-width: 30rem;
  }
</style>
