<script lang="ts">
  // só o filtro: sem lógica
</script>

<!-- 🎬 Filtro do contorno à mão (noemi-ui): parado, semente fixa. Usado só
     no ::before dos componentes de desenho (.sketched), nunca em texto. -->
<svg class="defs" aria-hidden="true" width="0" height="0" style="position: absolute">
  <filter id="sketch" x="-10%" y="-10%" width="120%" height="120%">
    <feTurbulence type="fractalNoise" baseFrequency="0.035" numOctaves="2" seed="7" result="noise" />
    <feDisplacementMap in="SourceGraphic" in2="noise" scale="1.2" xChannelSelector="R" yChannelSelector="G" />
  </filter>
</svg>
