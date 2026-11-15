<script lang="ts">
  // só o filtro: sem lógica
</script>

<!-- 🎬 Filtros da noemi-ui: contorno à mão (parado, semente fixa; só no
     ::before dos componentes .sketched) e a aberração cromática (vermelho
     pra direita, azul pra esquerda, somados em tela): leve nos títulos
     grandes, forte nos momentos de CRT. Nunca no texto corrido. -->
<svg class="defs" aria-hidden="true" width="0" height="0" style="position: absolute">
  <filter id="sketch" x="-10%" y="-10%" width="120%" height="120%">
    <feTurbulence type="fractalNoise" baseFrequency="0.035" numOctaves="2" seed="7" result="noise" />
    <feDisplacementMap in="SourceGraphic" in2="noise" scale="1.2" xChannelSelector="R" yChannelSelector="G" />
  </filter>
  <filter id="chroma" x="-4%" y="-4%" width="108%" height="108%" color-interpolation-filters="sRGB">
    <feColorMatrix in="SourceGraphic" type="matrix" values="1 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1 0" result="r" />
    <feOffset in="r" dx="0.9" result="r2" />
    <feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 1 0 0 0  0 0 0 0 0  0 0 0 1 0" result="g" />
    <feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 1 0 0  0 0 0 1 0" result="b" />
    <feOffset in="b" dx="-0.9" result="b2" />
    <feBlend in="r2" in2="g" mode="screen" result="rg" />
    <feBlend in="rg" in2="b2" mode="screen" />
  </filter>
  <filter id="chroma-strong" x="-2%" y="-2%" width="104%" height="104%" color-interpolation-filters="sRGB">
    <feColorMatrix in="SourceGraphic" type="matrix" values="1 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1 0" result="r" />
    <feOffset in="r" dx="3" result="r2" />
    <feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 1 0 0 0  0 0 0 0 0  0 0 0 1 0" result="g" />
    <feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 1 0 0  0 0 0 1 0" result="b" />
    <feOffset in="b" dx="-3" result="b2" />
    <feBlend in="r2" in2="g" mode="screen" result="rg" />
    <feBlend in="rg" in2="b2" mode="screen" />
  </filter>
</svg>
