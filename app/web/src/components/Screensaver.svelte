<script lang="ts">
  // Protetor de tela: liga depois de X minutos parado (nunca com jogo aberto).
  // Cartaz de "Intervalo" de cinema dos anos 30. "Cobrinha": a cobrinha joga
  // sozinha num quadro, um passo por segundo. "Vitrine": uma capa da coleção
  // em cartaz, trocada a cada 8 s. Nada anima em loop: só o relógio do JS
  // muda o quadro, e ele para quando o protetor fecha. Qualquer entrada sai.
  import { idle, idleFor, sleep, watchPointer } from '../lib/idle.svelte'
  import { choose, newGame, rng, step, stuck, type Snake } from '../lib/snake'
  import { app, shown as visible, system } from '../lib/state.svelte'
  import type { Game } from '../lib/types'

  const W = 16
  const H = 9
  const TICK = 1000 // um passo por segundo: CPU parada entre um quadro e outro
  const CELL = 48

  const cfg = $derived(app.settings?.screensaver)
  const reduce = $derived(
    !!app.settings?.a11y.reduceMotion ||
      (typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches),
  )

  // relógio da inatividade
  $effect(() => {
    watchPointer()
    const iv = setInterval(() => {
      if (!cfg?.enabled || idle.active || app.playing || app.watching || !app.ready) return
      if (idleFor() >= cfg.minutes * 60_000) sleep()
    }, 1000)
    return () => clearInterval(iv)
  })

  // com jogo aberto o protetor sai
  $effect(() => {
    if (app.playing) idle.active = false
  })

  // descanso: depois de X min de protetor, tela preta e nada animando (poupa
  // a CPU e a tela da TV). Qualquer entrada acorda do mesmo jeito.
  let resting = $state(false)
  $effect(() => {
    resting = false
    if (!idle.active || !cfg?.rest) return
    const t = setTimeout(() => (resting = true), cfg.rest * 60_000)
    return () => clearTimeout(t)
  })

  // ---- cobrinha
  let canvas = $state<HTMLCanvasElement | null>(null)
  let snake = $state<Snake | null>(null)
  let hi = $state(0)

  // cores dos tokens (alto contraste troca os tokens, o quadro acompanha)
  function ink(name: string, fallback: string) {
    return (canvas && getComputedStyle(canvas).getPropertyValue(name).trim()) || fallback
  }

  function box(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
    ctx.beginPath()
    // WebKit antigo não tem roundRect: cai no retângulo
    const round = (ctx as Partial<Pick<CanvasRenderingContext2D, 'roundRect'>>).roundRect
    if (round) round.call(ctx, x, y, w, h, r)
    else ctx.rect(x, y, w, h)
    ctx.fill()
    ctx.stroke()
  }

  function draw(s: Snake) {
    const ctx = canvas?.getContext('2d')
    if (!ctx || !canvas) return
    const card = ink('--surface', '#241a13')
    const line = ink('--outline', '#e9daba')
    const paper = ink('--text', '#f1e4c6')
    const dark = ink('--bg', '#17110c')
    const food = ink('--accent', '#e8b34a')
    const c = CELL
    ctx.fillStyle = card
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    // marcas de giz bem fracas no quadro
    ctx.globalAlpha = 0.12
    ctx.fillStyle = line
    for (let i = 0; i < W * H; i++) ctx.fillRect((i % W) * c + c / 2 - 2, Math.floor(i / W) * c + c / 2 - 2, 4, 4)
    ctx.globalAlpha = 1
    ctx.lineWidth = 3
    ctx.strokeStyle = dark
    if (s.food >= 0) {
      ctx.fillStyle = food
      ctx.beginPath()
      ctx.arc((s.food % W) * c + c / 2, Math.floor(s.food / W) * c + c / 2, c * 0.3, 0, Math.PI * 2)
      ctx.fill()
      ctx.stroke()
    }
    // corpo de borracha: gomos claros com contorno de tinta, cabeça por cima
    for (let i = s.body.length - 1; i >= 0; i--) {
      const at = s.body[i]
      ctx.fillStyle = paper
      box(ctx, (at % W) * c + 4, Math.floor(at / W) * c + 4, c - 8, c - 8, i === 0 ? 14 : 10)
    }
    const head = s.body[0]
    if (head !== undefined) {
      ctx.fillStyle = dark
      for (const dx of [0.36, 0.64]) {
        ctx.beginPath()
        ctx.arc((head % W) * c + c * dx, Math.floor(head / W) * c + c * 0.42, 4, 0, Math.PI * 2)
        ctx.fill()
      }
    }
  }

  $effect(() => {
    if (!idle.active || resting || cfg?.mode !== 'snake' || !canvas) return
    const rnd = rng(Date.now())
    let s = newGame(W, H, rnd)
    if (reduce) {
      // versão parada: um quadro de uma partida já em andamento
      for (let i = 0; i < 400 && s.alive; i++) s = step(s, rnd)
      hi = Math.max(hi, s.score)
      snake = s
      draw(s)
      return
    }
    let dead = 0
    snake = s
    draw(s)
    const iv = setInterval(() => {
      if (!s.alive || stuck(s) || choose(s) === null) {
        if (++dead > 4) {
          s = newGame(W, H, rnd)
          dead = 0
        }
      } else {
        s = step(s, rnd)
        hi = Math.max(hi, s.score)
      }
      snake = s
      draw(s)
    }, TICK)
    return () => clearInterval(iv)
  })

  // ---- vitrine
  const covers = $derived(visible().filter((g) => g.cover))
  let shown = $state<Game | null>(null)
  $effect(() => {
    if (!idle.active || resting || cfg?.mode !== 'showcase') return
    const pool = [...covers]
    if (!pool.length) return
    const pick = () => (shown = pool[Math.floor(Math.random() * pool.length)])
    pick()
    if (reduce) return
    const iv = setInterval(pick, 8000)
    return () => clearInterval(iv)
  })
</script>

{#if idle.active && cfg && resting}
  <div class="saver rest" role="dialog" aria-modal="true" aria-label="Descanso" data-mode="rest">
    <p class="sr-only">Descanso. Aperte qualquer botão ou tecla para voltar.</p>
  </div>
{:else if idle.active && cfg}
  <div class="saver" class:reduce role="dialog" aria-modal="true" aria-label="Protetor de tela" data-mode={cfg.mode}>
    <p class="kicker meta" aria-hidden="true">arkflix apresenta</p>
    <p class="title intermission" aria-hidden="true">Intervalo</p>
    {#if cfg.mode === 'snake'}
      <div class="board">
        <canvas bind:this={canvas} width={W * CELL} height={H * CELL} aria-hidden="true"></canvas>
        <p class="score" aria-hidden="true">
          <span>PLACAR {String(snake?.score ?? 0).padStart(4, '0')}</span>
          <span>RECORDE {String(hi).padStart(4, '0')}</span>
        </p>
      </div>
    {:else if shown}
      {@const sys = system(shown.system)}
      <div class="showcase">
        {#key shown.id}
          <img src={shown.cover} alt="" />
        {/key}
        <p class="label now">Em cartaz</p>
        <p class="name">{shown.name}</p>
        <p class="chip">{sys?.name ?? shown.system}</p>
      </div>
    {/if}
    <p class="coin meta">Aperte qualquer botão pra voltar à sessão</p>
    <p class="sr-only">Protetor de tela. Aperte qualquer botão ou tecla para voltar.</p>
  </div>
{/if}

<style>
  /* Cartaz de intervalo: fundo de cinema escuro, "Intervalo" em letreiro,
     um quadro sóbrio no meio e o recado datilografado embaixo. */
  .saver {
    position: fixed;
    inset: 0;
    z-index: 10000; /* acima das camadas de tubo */
    background:
      radial-gradient(ellipse at 0% 0%, rgb(232 179 74 / 0.1), transparent 55%),
      radial-gradient(ellipse at 100% 100%, rgb(95 168 161 / 0.1), transparent 55%),
      var(--bg);
    color: var(--text);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    padding: var(--safe);
    cursor: none;
    overflow: hidden;
  }
  .rest {
    background: #000;
  }
  .kicker {
    font-size: 0.9rem;
    text-transform: uppercase;
    letter-spacing: 0.2em;
    color: var(--muted);
  }
  .intermission {
    font-size: clamp(3rem, 9vh, 6rem);
    margin-block-end: 0.4rem;
  }
  .board {
    display: grid;
    gap: 0.6rem;
  }
  canvas {
    display: block;
    height: min(54vh, 48vw);
    aspect-ratio: 16 / 9;
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-up);
  }
  .score {
    display: flex;
    justify-content: space-between;
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.05rem;
    color: var(--muted);
  }
  .showcase {
    display: grid;
    justify-items: center;
    gap: 0.5rem;
  }
  .showcase img {
    height: 46vh;
    max-width: 70vw;
    object-fit: contain;
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-up);
    margin-block-end: 0.8rem;
    animation: cut 0.36s steps(3) both;
  }
  .now {
    color: var(--accent);
  }
  .name {
    font-family: var(--font-label);
    font-size: clamp(1.3rem, 3vh, 2rem);
    text-align: center;
    max-width: 80vw;
  }
  .coin {
    font-size: 1rem;
    letter-spacing: 0.08em;
    color: var(--muted);
    margin-block-start: 0.6rem;
  }
  .reduce img {
    animation: none;
  }
  /* troca de capa: corte em 3 quadros, uma vez por troca */
  @keyframes cut {
    from {
      opacity: 0;
    }
  }
</style>
