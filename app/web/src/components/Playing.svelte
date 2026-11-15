<script lang="ts">
  // Jogo abrindo/aberto: "PLAYER 1 START" num ingresso de cinema sobre a capa
  // enquanto o emulador sobe (no desktop a janela some logo depois; no
  // navegador fica).
  import { panelIn } from '../lib/motion'
  import { app, game, glyphFamily } from '../lib/state.svelte'
  import Glyph from './Glyph.svelte'

  const p = $derived(app.playing)
  const family = $derived(glyphFamily())
  const cover = $derived(p ? game(p.id)?.cover : '')
</script>

{#if p?.state === 'running'}
  <div class="playing" role="dialog" aria-modal="true" aria-labelledby="playing-title">
    {#if cover}<div class="cover" style:background-image={`url("${cover}")`} aria-hidden="true"></div>{/if}
    <div class="ticket sketched" in:panelIn>
      <p class="kicker meta" aria-hidden="true">Em cartaz · sessão agora</p>
      <p class="start" aria-hidden="true">PLAYER 1 START</p>
      <h2 id="playing-title">Jogando {p.name}</h2>
      <hr aria-hidden="true" />
      <p class="muted">Feche o jogo pra voltar. Atalhos dentro do jogo:</p>
      <ul class="tips">
        <li><Glyph button="select" {family} /> + <Glyph button="north" {family} /> menu do emulador · Esc</li>
        <li><Glyph button="select" {family} /> + <Glyph button="start" {family} /> sair do jogo · F10</li>
        <li><Glyph button="select" {family} /> + <Glyph button="r1" {family} /> salvar · <Glyph button="select" {family} /> + <Glyph button="l1" {family} /> carregar (menos PSP e PS2)</li>
      </ul>
    </div>
  </div>
{/if}

<style>
  /* Fundo de cinema escuro com a capa apagada em sépia atrás; no meio, um
     ingresso (🎬 contorno à mão, recortes dos lados, picote) com o
     letreiro PLAYER 1 START. */
  .playing {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: var(--bg);
    display: grid;
    place-items: center;
    padding: var(--safe);
    overflow: hidden;
  }
  .cover {
    position: absolute;
    inset: 0;
    background-size: cover;
    background-position: center;
    filter: sepia(0.7) brightness(0.45);
    opacity: 0.35;
  }
  .ticket {
    position: relative;
    max-width: min(46rem, 100%);
    text-align: center;
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    align-items: center;
    padding: 1.8rem 3rem 2rem;
    background: var(--surface);
    border: var(--edge) solid transparent;
  }
  /* recortes do ingresso: meio-círculo de cada lado, na cor do fundo */
  .ticket::after {
    content: '';
    position: absolute;
    inset: 50% calc(var(--edge) * -1) auto;
    height: 2.4rem;
    margin-top: -1.2rem;
    background:
      radial-gradient(circle at 0 50%, var(--bg) 0.95rem, var(--outline) 0.95rem 1.15rem, transparent 1.15rem) left / 1.2rem 100% no-repeat,
      radial-gradient(circle at 100% 50%, var(--bg) 0.95rem, var(--outline) 0.95rem 1.15rem, transparent 1.15rem) right / 1.2rem 100% no-repeat;
    pointer-events: none;
  }
  .kicker {
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--muted);
  }
  .start {
    font-family: var(--font-title);
    font-size: clamp(2.4rem, 6vw, 4.4rem);
    line-height: 1;
    color: var(--accent);
    -webkit-text-stroke: 1.5px var(--on-accent);
    paint-order: stroke fill;
    text-shadow: 4px 4px 0 var(--outline);
    letter-spacing: 0.04em;
    animation: marquee 0.6s steps(3) both;
  }
  /* letreiro acende em 3 quadros, uma vez */
  @keyframes marquee {
    0% {
      opacity: 0;
    }
    34% {
      opacity: 0.5;
    }
    100% {
      opacity: 1;
    }
  }
  h2 {
    font-size: clamp(1.3rem, 2.6vw, 1.8rem);
  }
  hr {
    align-self: stretch;
    border: 0;
    border-block-start: 3px dashed color-mix(in srgb, var(--outline) 45%, transparent);
    margin: 0.3rem -1rem;
  }
  .tips {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.6rem;
  }
  .tips li {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    align-items: center;
    justify-content: center;
  }
</style>
