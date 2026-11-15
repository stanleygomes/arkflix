<script module lang="ts">
  // Consoles em silhueta de traço, de uma cor só (currentColor): desenho
  // próprio e simplificado, nunca o logotipo (marca registrada) nem a cor
  // do console. 24x24, mesmo traço dos ícones; "*" no começo = preenchido.
  const dot = (x: number, y: number, r = 1.1) => `*M${x - r} ${y}a${r} ${r} 0 1 0 ${2 * r} 0a${r} ${r} 0 1 0 ${-2 * r} 0z`
  const ring = (x: number, y: number, r: number) => `M${x - r} ${y}a${r} ${r} 0 1 0 ${2 * r} 0a${r} ${r} 0 1 0 ${-2 * r} 0`

  const ART: Record<string, string[]> = {
    // Início: a casa
    home: ['M3.5 11.5 12 4l8.5 7.5', 'M6 9.5V20h12V9.5', 'M10 20v-5h4v5'],
    // caixa baixa e reta, tampa na frente
    nes: ['M2.5 8.5h19v9h-19z', 'M2.5 11.5h19', 'M5 14.5h3', dot(18, 14.5)],
    // caixa de cantos curvos com o cartucho em cima
    snes: ['M3 11c0-1.5 1-2.5 2.5-2.5h13c1.5 0 2.5 1 2.5 2.5v4.5c0 1.5-1 2.5-2.5 2.5h-13C4 18 3 17 3 15.5z', 'M8.5 8.5V6h7v2.5', dot(7, 14), dot(17, 14)],
    // corpo largo com o círculo no meio
    genesis: ['M2.5 10.5h19v7h-19z', ring(12, 14, 2.6), 'M5 13h2.5'],
    // cartucho saindo de cima, corpo arredondado
    n64: ['M3 13c0-2 1.5-3 3-3h12c1.5 0 3 1 3 3v3c0 1.5-1 2.5-2.5 2.5h-13C4 18.5 3 17.5 3 16z', 'M9 10V5.5h6V10', dot(7, 15)],
    // quadrado com a tampa redonda do disco
    psx: ['M3 6.5h18v12H3z', ring(10, 12, 3.5), 'M16 16.5h2.5', dot(18, 9.5)],
    // torre fina, em pé
    ps2: ['M8.5 3h7v18h-7z', 'M8.5 7h7', 'M10.5 17.5h3'],
    // portátil largo, tela no meio
    psp: ['M2.5 9c0-1.5 1-2.5 2.5-2.5h14c1.5 0 2.5 1 2.5 2.5v6c0 1.5-1 2.5-2.5 2.5H5c-1.5 0-2.5-1-2.5-2.5z', 'M7.5 9h9v6h-9z', dot(5, 12, 0.9), dot(19, 12, 0.9)],
    // portátil em pé, tela em cima, cruz e botões embaixo
    gbc: ['M6 2.5h12v19H6z', 'M8.5 5h7v6.5h-7z', 'M9 16v3M7.5 17.5h3', dot(15.5, 16.2, 0.9), dot(14, 18.4, 0.9)],
    // duas telas, dobrável
    nds: ['M5 2.5h14v8.5H5z', 'M5 13h14v8.5H5z', 'M8 5h8v4H8z', 'M8 15h8v4H8z'],
    // gabinete de fliperama
    arcade: ['M6 2.5h12v19H6z', 'M6 6h12', 'M8 8h8v5H8z', 'M5.5 15.5h13', 'M9 18v-2.5', dot(14, 17.5, 0.9)],
  }
  ART.gb = ART.gbc
  ART.gba = ART.gbc
</script>

<script lang="ts">
  let { slug, size = '3rem' }: { slug: string; size?: string } = $props()
  const paths = $derived((ART[slug] ?? ART.arcade).map((d) => (d.startsWith('*') ? { d: d.slice(1), fill: true } : { d, fill: false })))
</script>

<svg
  viewBox="0 0 24 24"
  width={size}
  height={size}
  fill="none"
  stroke="currentColor"
  stroke-width="1.8"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#each paths as p, i (i)}<path d={p.d} fill={p.fill ? 'currentColor' : undefined} />{/each}
</svg>
