<script lang="ts" module>
  // Ícones de traço (24x24, currentColor, ~2px), no espírito de gravura de
  // programa de cinema: linha simples, ponta redonda, sem cor própria.
  // Cada ícone é uma lista de caminhos; "*" no começo = preenchido.
  const dot = (x: number, y: number, r = 1.3) => `*M${x - r} ${y}a${r} ${r} 0 1 0 ${2 * r} 0a${r} ${r} 0 1 0 ${-2 * r} 0z`
  const ring = (x: number, y: number, r: number) => `M${x - r} ${y}a${r} ${r} 0 1 0 ${2 * r} 0a${r} ${r} 0 1 0 ${-2 * r} 0`

  const ICONS = {
    play: ['*M7 4.5 19.5 12 7 19.5z'],
    pause: ['*M6.5 5h3.5v14H6.5z', '*M14 5h3.5v14H14z'],
    resume: ['M4.6 13.2A7.5 7.5 0 1 0 6.7 6.7', 'M4.5 3.8v4.6h4.6', '*M10.5 9 15 12l-4.5 3z'],
    star: ['M12 3.5l2.6 5.4 5.9.8-4.3 4.1 1 5.8L12 16.9l-5.2 2.7 1-5.8-4.3-4.1 5.9-.8z'],
    plus: ['M12 5v14', 'M5 12h14'],
    gear: [
      ring(12, 12, 6),
      ring(12, 12, 2.4),
      'M12 2.8v3.2M12 18v3.2M2.8 12H6M18 12h3.2M5.5 5.5l2.2 2.2M16.3 16.3l2.2 2.2M5.5 18.5l2.2-2.2M16.3 7.7l2.2-2.2',
    ],
    back: ['M9 4.5 4 9.5l5 5', 'M4 9.5h10.5a5 5 0 0 1 0 10H10'],
    trash: ['M4 6.5h16', 'M9.5 6.5V4h5v2.5', 'M6 6.5 7 20h10l1-13.5', 'M10 10.5v6M14 10.5v6'],
    pad: [
      'M7 7.5h10c2.8 0 4 2.5 4.5 6s-.3 5-1.8 5c-1.5 0-2.4-1.3-3.4-2.8H7.7c-1 1.5-1.9 2.8-3.4 2.8-1.5 0-2.3-1.5-1.8-5S4.2 7.5 7 7.5z',
      'M7.5 10v4M5.5 12h4',
      dot(15.5, 11, 1.1),
      dot(17.8, 13.4, 1.1),
    ],
    folder: ['M3 5.5h6l2 2.5h10v11.5H3z'],
    check: ['M4.5 12.5l5 5L20 6.5'],
    warn: ['M12 3.5 21.5 20h-19z', 'M12 9.5v5', dot(12, 17.3, 1.1)],
    keyboard: ['M2.5 6.5h19v11h-19z', dot(6.5, 10, 1), dot(10, 10, 1), dot(13.5, 10, 1), dot(17, 10, 1), 'M7.5 14h9'],
    sliders: ['M4 7h5M13 7h7M4 12h11M19 12h1M4 17h2M10 17h10', ring(11, 7, 2), ring(17, 12, 2), ring(8, 17, 2)],
    info: [ring(12, 12, 9), 'M12 11v5.5', dot(12, 7.8, 1.2)],
    search: [ring(10.5, 10.5, 6.5), 'M15.3 15.3 20.5 20.5'],
    dice: ['M4 4h16v16H4z', dot(8.5, 8.5), dot(15.5, 8.5), dot(12, 12), dot(8.5, 15.5), dot(15.5, 15.5)],
    chart: ['M3.5 20.5h17', 'M7 20.5v-6', 'M12 20.5V9', 'M17 20.5V4'],
    power: ['M12 3v8.5', 'M7.2 6.3a7.5 7.5 0 1 0 9.6 0'],
    cloud: ['M7 18.5h10.5a4 4 0 0 0 .4-8 5.5 5.5 0 0 0-10.6-1.2A4.6 4.6 0 0 0 7 18.5z'],
    download: ['M12 3.5v11', 'M7 10l5 5 5-5', 'M4.5 20h15'],
    lock: ['M5.5 10.5h13v10h-13z', 'M8.5 10.5v-3a3.5 3.5 0 0 1 7 0v3', 'M12 14.5V17'],
    pencil: ['M4 20l1-4.5L16 4.5 19.5 8 8.5 19z', 'M13.8 6.7l3.5 3.5'],
    disc: [ring(12, 12, 9), ring(12, 12, 2.4), 'M7.3 10.2a5 5 0 0 1 2.9-2.9'],
    bell: ['M6 16.5V11a6 6 0 0 1 12 0v5.5l1.5 1.5h-15z', 'M10 20.5a2 2 0 0 0 4 0'],
    home: ['M3.5 11.5 12 4l8.5 7.5', 'M6 9.5V20h12V9.5', 'M10 20v-5h4v5'],
    close: ['M6 6l12 12', 'M18 6 6 18'],
    minimize: ['M5 18h14'],
    maximize: ['M5 5h14v14H5z', 'M5 8.5h14'],
    restore: ['M8.5 4.5h11v11', 'M4.5 8.5h11v11h-11z'],
    film: ['M4 3.5h16v17H4z', 'M8 3.5v17M16 3.5v17', 'M4 8h4M4 12h4M4 16h4M16 8h4M16 12h4M16 16h4'],
    tv: ['M3 7.5h18v12H3z', 'M8 3l4 4.5L16 3'],
    cc: ['M3 5.5h18v13H3z', 'M10.5 10.2a2.4 2.4 0 1 0 0 3.6', 'M17.5 10.2a2.4 2.4 0 1 0 0 3.6'],
    audio: ['M4 9.5h3.5L12 5.5v13l-4.5-4H4z', 'M15.5 9a4 4 0 0 1 0 6', 'M18 6.5a7.5 7.5 0 0 1 0 11'],
    next: ['M5 5l9 7-9 7z', 'M18.5 5v14'],
    rewind: ['M11.5 6 4 12l7.5 6z', 'M20 6l-7.5 6 7.5 6z'],
    forward: ['M12.5 6 20 12l-7.5 6z', 'M4 6l7.5 6L4 18z'],
    save: ['M4 4h12.5L20 7.5V20H4z', 'M8 4v5h7V4', 'M7.5 20v-6h9v6'],
    bluetooth: ['M7 7.5l10 9-5 4.5V3l5 4.5-10 9'],
    trophy: ['M7.5 4h9v5a4.5 4.5 0 0 1-9 0z', 'M7.5 6H4.5a3 3 0 0 0 3 4.5', 'M16.5 6h3a3 3 0 0 1-3 4.5', 'M12 13.5v6.5', 'M8 20.5h8'],
    auto: ['M13.5 3 5.5 13.5h6L10 21l8-10.5h-6z'],
    globe: [ring(12, 12, 9), 'M3 12h18', 'M12 3c3 2.5 4 6 4 9s-1 6.5-4 9c-3-2.5-4-6-4-9s1-6.5 4-9z'],
    music: ['M9 17.5V5.5l10-2v12', '*M4.5 17.5a2.5 2 0 1 0 5 0a2.5 2 0 1 0 -5 0z', '*M14.5 15.5a2.5 2 0 1 0 5 0a2.5 2 0 1 0 -5 0z'],
    prev: ['M19 5l-9 7 9 7z', 'M5.5 5v14'],
    shuffle: ['M3.5 7h3.5l9 10h4', 'M3.5 17h3.5l2.5-2.8', 'M13.5 9.8 16 7h4', 'M17.5 4.5 20 7l-2.5 2.5', 'M17.5 14.5 20 17l-2.5 2.5'],
    repeat: ['M4.5 11V9.5a3 3 0 0 1 3-3h12', 'M16.5 3.5l3 3-3 3', 'M19.5 13v1.5a3 3 0 0 1-3 3h-12', 'M7.5 20.5l-3-3 3-3'],
    queue: ['M4 6h11', 'M4 11h11', 'M4 16h7', '*M15 13.5v7l5.5-3.5z'],
    user: [ring(12, 8, 4), 'M4.5 20.5a7.5 7.5 0 0 1 15 0'],
  } satisfies Record<string, string[]>

  export type IconName = keyof typeof ICONS
</script>

<script lang="ts">
  let { name, size = '1.1em', label = '' }: { name: IconName; size?: string; label?: string } = $props()
  const paths = $derived((ICONS[name] ?? []).map((d) => (d.startsWith('*') ? { d: d.slice(1), fill: true } : { d, fill: false })))
</script>

<svg
  class="icon"
  viewBox="0 0 24 24"
  width={size}
  height={size}
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  role={label ? 'img' : undefined}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
>
  {#each paths as p, i (i)}
    <path d={p.d} fill={p.fill ? 'currentColor' : undefined} />
  {/each}
</svg>

<style>
  .icon {
    flex: none;
    overflow: visible;
  }
</style>
