<script lang="ts">
  // Avatar do perfil: personagem (imagem em /avatars) ou desenho 10x10
  // ("#" = claro, "o" = escuro). noemi-ui: sóbrio (quadrado de cantos
  // irregulares, borda reta) nas listas; "cameo" (moldura oval de retrato,
  // contorno à mão) só no retrato único da tela. A cor do perfil fica só
  // como marca pequena no canto (mark), nunca pintando o quadro.
  let {
    avatar,
    color = '',
    size = '6rem',
    variant = 'sober',
    mark = false,
  }: { avatar: string; color?: string; size?: string; variant?: 'sober' | 'cameo'; mark?: boolean } = $props()

  const SPRITES: Record<string, string[]> = {
    knight: ['..######..', '.########.', '.#oooooo#.', '.#o#oo#o#.', '.#oooooo#.', '.########.', '...####...', '.########.', '##.####.##', '.#.#..#.#.'],
    mage: ['....##....', '...####...', '..##o###..', '.########.', '##########', '..#o##o#..', '..######..', '...#oo#...', '..######..', '.##.##.##.'],
    ghost: ['...####...', '.########.', '.########.', '##o##o####', '##o##o####', '##########', '##########', '##########', '##.##.##.#', '#...#...#.'],
    cat: ['#........#', '##......##', '##########', '#o##o#####', '##########', '###oo#####', '.########.', '..######..', '..#.##.#..', '..#.##.#..'],
    robot: ['....#.....', '....#.....', '.########.', '.#o####o#.', '.########.', '.#oooooo#.', '.########.', '##.####.##', '...####...', '...#..#...'],
    mushroom: ['...####...', '.##oo####.', '###oo##o##', '#######oo#', '#o########', '##########', '..#o##o#..', '..######..', '..######..', '...####...'],
    alien: ['..#....#..', '...#..#...', '..######..', '.##o##o##.', '##########', '#.######.#', '#.#....#.#', '...##.##..', '..........', '..........'],
    dragon: ['.......##.', '......####', '......#o##', '.....#####', '.##.####..', '########..', '.#######..', '..######..', '..#..#....', '.##.##....'],
  }

  // sombra dura só a partir de ~56px (spec do avatar)
  const big = $derived(parseFloat(size) * (size.endsWith('rem') ? 17 : 1) >= 56)

  const cells = $derived.by(() => {
    const out: { x: number; y: number; dark: boolean }[] = []
    ;(SPRITES[avatar] ?? []).forEach((row, y) =>
      [...row].forEach((c, x) => {
        if (c !== '.') out.push({ x, y, dark: c === 'o' })
      }),
    )
    return out
  })
</script>

<span class={`avatar ${variant}`} class:sketched={variant === 'cameo'} class:big style:--size={size} aria-hidden="true">
  <span class="pic">
    {#if SPRITES[avatar]}
      <svg viewBox="-1 -1 12 12" shape-rendering="crispEdges">
        {#each cells as c (c.y * 10 + c.x)}
          <rect x={c.x} y={c.y} width="1" height="1" fill={c.dark ? 'var(--bg)' : 'var(--text)'} />
        {/each}
      </svg>
    {:else}
      <img src={`/avatars/${avatar}.png`} alt="" draggable="false" />
    {/if}
  </span>
  {#if mark && color}<span class="mark" style:background={color}></span>{/if}
</span>

<style>
  .avatar {
    position: relative;
    display: inline-grid;
    place-items: center;
    flex: none;
    width: var(--size);
    height: var(--size);
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
  }
  .pic {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    overflow: hidden;
    border-radius: inherit;
  }
  /* sombra dura só nos tamanhos grandes (a partir de ~56px) */
  .sober.big {
    box-shadow: var(--hard-sm);
  }
  svg {
    width: 78%;
    height: 78%;
  }
  img {
    width: 84%;
    height: 84%;
    object-fit: contain;
    image-rendering: pixelated; /* sprite de 64 px: blocos nítidos, sem borrão */
  }

  /* 🎬 cameo: moldura oval de retrato, contorno à mão (.sketched), anel fino
     por dentro como um carimbo, sombra de desenho. A íris abre uma vez. */
  .cameo {
    border-radius: 50% / 46% 54%;
    box-shadow: var(--hard-cartoon);
  }
  .cameo::after {
    content: '';
    position: absolute;
    inset: 6px;
    border: 1px solid var(--outline);
    border-radius: inherit;
    pointer-events: none;
  }
  .cameo .pic {
    animation: iris 250ms steps(5) both;
  }
  .cameo img {
    width: 72%;
    height: 72%;
  }
  @keyframes iris {
    from {
      clip-path: circle(0 at 50% 50%);
    }
    to {
      clip-path: circle(75% at 50% 50%);
    }
  }

  /* marca da cor do perfil: quadradinho no canto, com borda (a cor nunca é
     a única pista: o nome está sempre do lado) */
  .mark {
    position: absolute;
    inset-inline-end: -6px;
    inset-block-end: -6px;
    width: calc(var(--size) * 0.16);
    height: calc(var(--size) * 0.16);
    min-width: 10px;
    min-height: 10px;
    border: 2px solid var(--outline);
    border-radius: 3px;
  }
  .cameo .mark {
    inset-inline-end: 8%;
    inset-block-end: 4%;
  }
</style>
