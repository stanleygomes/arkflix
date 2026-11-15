<script lang="ts">
  // 🎬 Textura de filme antigo por cima do app (noemi-ui, film texture):
  // grão pintado UMA vez num canvas pequeno e esticado, mais a vinheta.
  // Nada anima (app de uso o dia todo). Sem clique, fora da árvore de
  // acessibilidade; o alto contraste desliga.
  let canvas = $state<HTMLCanvasElement | null>(null)

  $effect(() => {
    const ctx = canvas?.getContext('2d')
    if (!canvas || !ctx) return
    const { width: w, height: h } = canvas
    ctx.clearRect(0, 0, w, h)
    // pontinhos claros e escuros, poucos e fracos
    for (let i = 0; i < 2600; i++) {
      const light = Math.random() < 0.5
      ctx.fillStyle = light ? `rgb(241 228 198 / ${0.04 + Math.random() * 0.08})` : `rgb(0 0 0 / ${0.06 + Math.random() * 0.1})`
      ctx.fillRect(Math.random() * w, Math.random() * h, 1, 1)
    }
  })
</script>

<div class="film" aria-hidden="true">
  <canvas bind:this={canvas} width="320" height="200"></canvas>
  <div class="vignette"></div>
</div>

<style>
  .film {
    position: fixed;
    inset: 0;
    z-index: 9400;
    pointer-events: none;
  }
  canvas {
    width: 100%;
    height: 100%;
    opacity: 0.4;
    image-rendering: auto;
  }
  .vignette {
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse at center, transparent 68%, rgb(0 0 0 / 0.5));
  }
  :global(body.high-contrast) .film {
    display: none;
  }
</style>
