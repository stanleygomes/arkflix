<script lang="ts">
  // Tubo de TV num momento curto: a tela toda vira o tubo (preto quente,
  // cantos escuros, bojo, reflexo, linhas), com um cartão de teste da época.
  // "on": a linha do meio abre com um clarão; "off": a imagem fecha numa
  // linha e num ponto. Some sozinho em ~1,6 s; qualquer botão pula. Nada
  // fica animando depois. Fora da árvore de acessibilidade.
  import { tube } from '../lib/tube.svelte'

  const MS = 1600
  let shown = $state(0)

  $effect(() => {
    const seq = tube.seq
    if (!seq) return
    shown = seq
    const t = setTimeout(() => {
      if (shown === seq) shown = 0
    }, MS)
    const skip = () => (shown = 0)
    window.addEventListener('keydown', skip, { once: true, capture: true })
    window.addEventListener('pointerdown', skip, { once: true, capture: true })
    return () => {
      clearTimeout(t)
      window.removeEventListener('keydown', skip, { capture: true })
      window.removeEventListener('pointerdown', skip, { capture: true })
    }
  })
</script>

{#if shown}
  {#key shown}
    <div class="tube {tube.kind}" aria-hidden="true">
      <div class="picture">
        <div class="card">
          <span class="ring"></span>
          <p class="name">ARKFLIX</p>
          {#if tube.caption}<p class="caption">{tube.caption}</p>{/if}
        </div>
      </div>
      <div class="lines"></div>
      <div class="bulge"></div>
      <div class="glass"></div>
    </div>
  {/key}
{/if}

<style>
  .tube {
    position: fixed;
    inset: 0;
    z-index: 9800;
    pointer-events: none;
    background: #000;
    animation: fade 1.6s steps(1) forwards;
  }
  @keyframes fade {
    92% {
      opacity: 1;
    }
    100% {
      opacity: 0;
    }
  }
  .picture {
    position: absolute;
    inset: 1.5vmin;
    border-radius: 3vmin;
    overflow: hidden;
    display: grid;
    place-items: center;
    background: radial-gradient(ellipse at center, #2a1a10, #0c0608 70%, #000);
  }
  /* aberração forte: o tubo velho separa as cores na borda das letras */
  .picture .card {
    filter: url(#chroma-strong);
  }
  .on .picture {
    animation: power-on 0.6s steps(5) both;
  }
  .off .picture {
    animation: power-off 0.7s steps(6) 0.5s both;
  }
  @keyframes power-on {
    0% {
      transform: scale(1, 0.006);
      filter: brightness(4);
    }
    40% {
      transform: scale(1, 0.006);
      filter: brightness(3);
    }
    100% {
      transform: none;
      filter: none;
    }
  }
  @keyframes power-off {
    0% {
      transform: none;
      filter: none;
    }
    60% {
      transform: scale(1, 0.006);
      filter: brightness(3);
    }
    100% {
      transform: scale(0.01, 0.006);
      filter: brightness(5);
      opacity: 0;
    }
  }
  /* cartão de teste: anel com mira e o nome, fósforo só no destaque */
  .card {
    display: grid;
    place-items: center;
    gap: 1.2vmin;
    color: #f1e4c6;
  }
  .ring {
    width: 26vmin;
    aspect-ratio: 1;
    border-radius: 50%;
    border: 0.8vmin solid #f1e4c6;
    background:
      linear-gradient(#f1e4c6, #f1e4c6) center / 0.5vmin 100% no-repeat,
      linear-gradient(#f1e4c6, #f1e4c6) center / 100% 0.5vmin no-repeat;
    opacity: 0.6;
  }
  .name {
    font-family: 'Limelight', serif;
    font-size: 9vmin;
    line-height: 1;
    color: #e8b34a;
    text-shadow:
      0 0 1px #ffd98a,
      0 0 8px rgb(232 179 74 / 0.8),
      0 0 22px rgb(232 179 74 / 0.35);
  }
  .caption {
    font-family: 'Special Elite', monospace;
    font-size: 2.6vmin;
    letter-spacing: 0.2em;
    text-transform: uppercase;
  }
  .lines {
    position: absolute;
    inset: 1.5vmin;
    border-radius: 3vmin;
    background: repeating-linear-gradient(transparent 0 2px, rgb(0 0 0 / 0.45) 3px 4px);
    mix-blend-mode: multiply;
    mask-image: radial-gradient(ellipse at center, transparent 30%, #000 85%);
  }
  .bulge {
    position: absolute;
    inset: 1.5vmin;
    border-radius: 3vmin;
    background:
      radial-gradient(circle at 0 0, rgb(0 0 0 / 0.85), transparent 55%),
      radial-gradient(circle at 100% 0, rgb(0 0 0 / 0.85), transparent 55%),
      radial-gradient(circle at 0 100%, rgb(0 0 0 / 0.85), transparent 55%),
      radial-gradient(circle at 100% 100%, rgb(0 0 0 / 0.85), transparent 55%),
      radial-gradient(ellipse at center, rgb(255 240 210 / 0.06), transparent 60%);
  }
  .glass {
    position: absolute;
    inset: 1.5vmin;
    border-radius: 3vmin;
    background: linear-gradient(135deg, rgb(255 255 255 / 0.09), transparent 30%);
  }
</style>
