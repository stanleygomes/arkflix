<script lang="ts">
  // Player de filmes e episódios. O Go escolhe o caminho (arquivo direto ou HLS
  // transcodificado pelo Jellyfin) e serve pelo proxy local; aqui é só tocar,
  // mostrar os controles e avisar o servidor do progresso (start, a cada 10 s,
  // ao pausar e ao sair), que é o que faz o "continuar de onde parou".
  //
  // Controle: A = play/pause (ou o botão em foco), esquerda/direita = 10 s,
  // L1/R1 = 30 s, X/Quadrado = áudio e legendas, B = sair. Qualquer entrada
  // mostra os controles; parado, eles somem em 3,5 s.
  import Hls from 'hls.js'
  import { onDestroy, onMount, tick } from 'svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { pauseMusic, play } from '../lib/sound'
  import { app, back } from '../lib/state.svelte'
  import { tubeMoment } from '../lib/tube.svelte'
  import type { WatchStream } from '../lib/types'
  import { clock, displayName, episodeTag, resumeAt } from '../lib/watch'

  let { id, start = -1 }: { id: string; start?: number } = $props()

  let video = $state<HTMLVideoElement | null>(null)
  let stream = $state<WatchStream | null>(null)
  let failed = $state('')
  let paused = $state(true)
  let buffering = $state(true)
  let time = $state(0)
  let length = $state(0)
  let shown = $state(true)
  let panel = $state(false)
  let subtitle = $state(-1) // posição em stream.subtitles; -1 = desligada
  let countdown = $state(0)

  let hls: Hls | null = null
  let reported = false // o servidor já sabe que começou (esta sessão)
  let lit = false // o tubo já ligou neste filme (só na abertura, não ao trocar o áudio)
  let hideTimer = 0
  let tickTimer = 0
  let countTimer = 0
  let pendingSeek = -1

  const duration = $derived(length > 0 ? length : (stream?.duration ?? 0))
  const item = $derived(stream?.item)
  const heading = $derived(item ? displayName(item) : 'Carregando')
  const sub = $derived(item?.type === 'Episode' ? `${episodeTag(item)} · ${item.name}` : item?.year ? String(item.year) : '')

  function report(event: 'start' | 'progress' | 'stop', pos = video?.currentTime ?? 0) {
    if (!stream) return
    api.watchReport(event, id, stream.session, stream.source, pos, paused).catch(() => {})
  }

  function attach(s: WatchStream, at: number) {
    if (!video) return
    hls?.destroy()
    hls = null
    reported = false
    buffering = true
    pendingSeek = at
    if (s.hls && Hls.isSupported()) {
      hls = new Hls({ startPosition: at })
      pendingSeek = -1
      hls.on(Hls.Events.ERROR, (_e, data) => {
        if (data.fatal) failed = 'O vídeo parou de chegar do servidor.'
      })
      hls.loadSource(s.url)
      hls.attachMedia(video)
    } else {
      video.src = s.url
    }
    video.play().catch(() => (paused = true))
  }

  async function load(audio: number, at: number) {
    try {
      if (stream && reported) report('stop')
      const s = await api.watchPlay(id, audio, at)
      stream = s
      subtitle = -1
      if (!lit) {
        lit = true
        // começa o filme: o momento curto do tubo ligando, com o nome
        tubeMoment('on', displayName(s.item))
      }
      await tick()
      const from = at >= 0 ? at : resumeAt(s.start, s.duration)
      attach(s, from)
    } catch (err) {
      failed = (err as Error).message
    }
  }

  onMount(() => {
    app.watching = true
    pauseMusic(true)
    load(-1, start)
    tickTimer = window.setInterval(() => {
      if (reported && !paused) report('progress')
    }, 10_000)
    poke()
  })

  onDestroy(() => {
    clearInterval(tickTimer)
    clearTimeout(hideTimer)
    clearInterval(countTimer)
    if (reported) report('stop')
    hls?.destroy()
    if (video) {
      video.pause()
      video.removeAttribute('src')
      video.load()
    }
    app.watching = false
    pauseMusic(false)
  })

  /** Mostra os controles e agenda o sumiço (tocando e sem painel). */
  function poke() {
    shown = true
    clearTimeout(hideTimer)
    hideTimer = window.setTimeout(() => {
      if (!paused && !panel && !countdown) shown = false
    }, 3500)
  }

  function togglePlay() {
    if (!video) return
    if (video.paused) video.play().catch(() => {})
    else video.pause()
    poke()
  }

  function seek(delta: number) {
    if (!video || !duration) return
    video.currentTime = Math.min(Math.max(0, video.currentTime + delta), Math.max(0, duration - 1))
    time = video.currentTime
    play('move')
    poke()
  }

  function onMeta() {
    if (!video) return
    if (Number.isFinite(video.duration)) length = video.duration
    if (pendingSeek > 0 && pendingSeek < (length || Infinity)) video.currentTime = pendingSeek
    pendingSeek = -1
  }

  function onError() {
    const e = video?.error
    // código 4 = formato que o sistema não abre; o resto, rede/servidor
    failed = e?.code === 4 ? 'Este vídeo não tocou aqui (formato que o sistema não abre).' : `O vídeo parou de chegar do servidor (${e?.message || `erro ${e?.code ?? '?'}`}).`
  }

  function onPlaying() {
    buffering = false
    paused = false
    if (!reported) {
      reported = true
      report('start')
    }
    poke()
  }

  function onPause() {
    paused = true
    if (reported) report('progress')
    poke()
  }

  function onEnded() {
    paused = true
    if (reported) {
      report('stop', duration || video?.currentTime || 0)
      reported = false
    }
    if (stream?.next) {
      countdown = 10
      shown = true
      clearInterval(countTimer)
      countTimer = window.setInterval(() => {
        countdown -= 1
        if (countdown <= 0) playNext()
      }, 1000)
      tick().then(() => focusEl(document.querySelector<HTMLElement>('[data-next]')))
    } else {
      back()
    }
  }

  function playNext() {
    clearInterval(countTimer)
    countdown = 0
    const next = stream?.next
    if (!next) return
    // troca a tela do player pela do próximo (voltar leva pra onde estava)
    app.stack[app.stack.length - 1] = { name: 'player', id: next.id, start: 0 }
  }

  function cancelNext() {
    clearInterval(countTimer)
    countdown = 0
    poke()
  }

  function setSubtitle(i: number) {
    subtitle = i
    if (video) {
      for (let t = 0; t < video.textTracks.length; t++) video.textTracks[t].mode = t === i ? 'showing' : 'disabled'
    }
    play('toggle')
  }

  async function setAudio(index: number) {
    if (!stream || index === stream.audioIndex) return
    const at = video?.currentTime ?? 0
    play('toggle')
    await load(index, at)
    openPanel(false)
  }

  function openPanel(open: boolean) {
    panel = open
    poke()
    tick().then(() => focusEl(document.querySelector<HTMLElement>(open ? '.tracks [aria-checked="true"]' : '[data-play]')))
  }

  $effect(() =>
    pushHandler((a) => {
      poke()
      if (panel) {
        if (a === 'back' || a === 'details') {
          openPanel(false)
          return true
        }
        return false
      }
      if (countdown && a === 'back') {
        cancelNext()
        return true
      }
      const onBar = !!current()?.closest('.bar')
      switch (a) {
        case 'accept':
          if (!shown || !current()) {
            togglePlay()
            return true
          }
          return false
        case 'left':
        case 'right':
          if (!shown || onBar) {
            seek(a === 'left' ? -10 : 10)
            return true
          }
          return false
        case 'up':
        case 'down':
          return !shown // primeiro toque só mostra
        case 'prevTab':
          seek(-30)
          return true
        case 'nextTab':
          seek(30)
          return true
        case 'menu':
          togglePlay()
          return true
        case 'details':
          openPanel(true)
          return true
        case 'fav':
          return true
      }
      return false
    }),
  )

  const pct = $derived(duration ? Math.min(100, (time / duration) * 100) : 0)
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<main class="player" class:hide={!shown} aria-labelledby="player-title" onmousemove={poke}>
  <!-- svelte-ignore a11y_media_has_caption (as legendas do servidor entram como <track> quando existem) -->
  <video
    bind:this={video}
    crossorigin="anonymous"
    playsinline
    aria-label={`${heading}${sub ? `, ${sub}` : ''}`}
    onloadedmetadata={onMeta}
    ondurationchange={onMeta}
    ontimeupdate={() => (time = video?.currentTime ?? 0)}
    onplaying={onPlaying}
    onpause={onPause}
    onwaiting={() => (buffering = true)}
    oncanplay={() => (buffering = false)}
    onended={onEnded}
    onerror={onError}
    onclick={togglePlay}
  >
    {#each stream?.subtitles ?? [] as t, i (t.index)}
      <track kind="subtitles" src={t.url} label={t.label} srclang={t.language || undefined} default={i === subtitle} />
    {/each}
  </video>

  {#if buffering && !failed}
    <p class="spin" role="status"><span class="ring" aria-hidden="true"></span> <span class="meta">Carregando…</span></p>
  {/if}

  {#if failed}
    <section class="error panel" role="alert">
      <span class="chip error">Erro</span>
      <h2>Sem sinal</h2>
      <p>{failed}</p>
      <button class="btn primary" data-nav data-nav-default onclick={() => back()}><Icon name="back" /> Voltar</button>
    </section>
  {/if}

  <header class="head">
    <button class="btn" data-nav aria-label="Sair do vídeo" onclick={() => back()}><Icon name="back" /> Sair</button>
    <div class="names">
      <h1 id="player-title">{heading}</h1>
      {#if sub}<p class="meta">{sub}</p>{/if}
    </div>
  </header>

  {#if countdown}
    <section class="up-next panel" aria-live="polite">
      <p>Próximo episódio em <strong class="mono">{countdown}</strong> s: {stream?.next ? `${episodeTag(stream.next)} · ${stream.next.name}` : ''}</p>
      <div class="row">
        <button class="btn primary" data-nav data-next onclick={playNext}><Icon name="next" /> Assistir agora</button>
        <button class="btn" data-nav onclick={cancelNext}>Ficar aqui</button>
      </div>
    </section>
  {/if}

  <footer class="controls" role="group" aria-label="Controles do vídeo">
    <div class="timeline">
      <span class="time mono">{clock(time)}</span>
      <!-- 🎬 rolo de filme: trilho com furos de engrenagem, mostarda o que já passou -->
      <div
        class="bar"
        data-nav
        role="slider"
        tabindex="0"
        aria-label="Posição no vídeo"
        aria-valuemin={0}
        aria-valuemax={Math.round(duration)}
        aria-valuenow={Math.round(time)}
        aria-valuetext={`${clock(time)} de ${clock(duration)}`}
        onkeydown={(e) => {
          if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') e.preventDefault()
        }}
        onclick={(e) => {
          const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
          if (video && duration) video.currentTime = ((e.clientX - r.left) / r.width) * duration
          poke()
        }}
      >
        <span class="fill" style:width={`${pct}%`}></span>
        <span class="holes" aria-hidden="true"></span>
        <span class="knob" style:left={`${pct}%`}></span>
      </div>
      <span class="time mono">{clock(duration)}</span>
    </div>
    <div class="buttons">
      <button class="btn cbtn" data-nav aria-label="Voltar 10 segundos" onclick={() => seek(-10)}><Icon name="rewind" /></button>
      <button class="btn primary cbtn big" data-nav data-play data-nav-default aria-label={paused ? 'Tocar' : 'Pausar'} onclick={togglePlay}>
        <Icon name={paused ? 'play' : 'pause'} size="1.5em" />
      </button>
      <button class="btn cbtn" data-nav aria-label="Avançar 10 segundos" onclick={() => seek(10)}><Icon name="forward" /></button>
      <span class="gap"></span>
      <button class="btn cbtn wide" data-nav aria-label="Áudio e legendas" aria-expanded={panel} onclick={() => openPanel(!panel)}>
        <Icon name="cc" /> <span>Áudio e legendas</span>
      </button>
      {#if stream?.next}
        <button class="btn cbtn wide" data-nav onclick={playNext}><Icon name="next" /> <span>Próximo episódio</span></button>
      {/if}
    </div>
  </footer>

  {#if panel && stream}
    <div class="tracks panel" data-nav-scope role="dialog" aria-modal="true" aria-label="Áudio e legendas">
      <div class="col">
        <h2><Icon name="audio" /> Áudio</h2>
        <div role="radiogroup" aria-label="Áudio">
          {#each stream.audio as t (t.index)}
            <button class="opt" data-nav role="radio" aria-checked={t.index === stream.audioIndex} onclick={() => setAudio(t.index)}>{t.label || t.language || `Faixa ${t.index}`}</button>
          {:else}
            <p class="muted">Uma faixa só.</p>
          {/each}
        </div>
      </div>
      <div class="col">
        <h2><Icon name="cc" /> Legendas</h2>
        <div role="radiogroup" aria-label="Legendas">
          <button class="opt" data-nav role="radio" aria-checked={subtitle === -1} onclick={() => setSubtitle(-1)}>Desligadas</button>
          {#each stream.subtitles as t, i (t.index)}
            <button class="opt" data-nav role="radio" aria-checked={subtitle === i} onclick={() => setSubtitle(i)}>{t.label || t.language}</button>
          {/each}
        </div>
      </div>
      <button class="btn" data-nav onclick={() => openPanel(false)}><Icon name="close" /> Fechar</button>
    </div>
  {/if}

  <div class="help">
    <HelpBar
      hints={[
        { button: 'south', label: 'Tocar / pausar' },
        { button: 'left', label: '10 s' },
        { button: 'l1', label: '30 s' },
        { button: 'west', label: 'Áudio e legendas' },
        { button: 'east', label: 'Sair' },
      ]}
    />
  </div>
</main>

<style>
  /* Sem tubo e sem linhas aqui: o CRT só aparece no momento de abrir
     (tubeMoment). O filme e os controles ficam limpos. */
  .player {
    position: fixed;
    inset: 0;
    z-index: 5;
    background: #000;
    color: var(--text);
    overflow: hidden;
  }
  .player.hide {
    cursor: none;
  }
  video {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: contain;
    background: #000;
  }
  video::cue {
    font-family: var(--font-body);
    font-size: 1.6rem;
    background: rgb(0 0 0 / 0.75);
  }
  .head,
  .controls,
  .help {
    position: absolute;
    inset-inline: 0;
    transition: opacity 200ms steps(4);
  }
  .hide .head,
  .hide .controls,
  .hide .help {
    opacity: 0;
    pointer-events: none;
  }
  .head {
    top: 0;
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 1.5rem 2.5rem 3rem;
    background: linear-gradient(to bottom, color-mix(in srgb, var(--bg) 92%, transparent), transparent);
  }
  .names {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .names h1 {
    font-size: 1.9rem;
    -webkit-text-stroke-width: 1px;
    text-shadow: 2px 2px 0 var(--outline);
  }
  .names .meta {
    color: var(--text);
    font-size: 1rem;
  }
  .controls {
    bottom: 3.2rem;
    padding: 3rem 2.5rem 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    background: linear-gradient(to top, color-mix(in srgb, var(--bg) 95%, transparent) 55%, transparent);
  }
  .help {
    bottom: 0;
    background: var(--bg);
    border-top: var(--edge) solid var(--outline);
  }
  .timeline {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  .time {
    font-variant-numeric: tabular-nums;
    font-weight: 700;
    font-size: 1.05rem;
    min-width: 4.5rem;
    text-align: center;
  }
  /* 🎬 rolo de filme: trilho escuro de película (cor do texto, como no
     componente), furos de engrenagem em cima e embaixo, mostarda o que já
     passou; a cabeça de leitura é um retângulo com sombra dura */
  .bar {
    position: relative;
    flex: 1;
    height: 1.6rem;
    background: var(--text);
    border: var(--edge) solid var(--outline);
    border-radius: 4px;
    box-shadow: var(--hard-sm);
    cursor: pointer;
  }
  .bar:focus,
  .bar:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: 4px;
  }
  .fill {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    background: var(--accent);
  }
  .holes {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background:
      linear-gradient(var(--bg), var(--bg)) 0 3px / 100% 6px repeat-x,
      linear-gradient(var(--bg), var(--bg)) 0 calc(100% - 3px) / 100% 6px repeat-x;
    -webkit-mask: repeating-linear-gradient(90deg, #000 0 6px, transparent 6px 16px);
    mask: repeating-linear-gradient(90deg, #000 0 6px, transparent 6px 16px);
  }
  .knob {
    position: absolute;
    top: 50%;
    width: 0.9rem;
    height: 2.4rem;
    transform: translate(-50%, -50%);
    background: var(--bg);
    border: var(--edge) solid var(--outline);
    border-radius: 3px;
    box-shadow: var(--hard-sm);
  }
  .bar:focus .knob {
    border-color: var(--focus);
  }
  .buttons {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  .gap {
    flex: 1;
  }
  .cbtn {
    min-width: 3.4rem;
    min-height: 3.4rem;
    justify-content: center;
    padding: 0 0.9rem;
  }
  .cbtn.big {
    min-width: 4.4rem;
    min-height: 4.4rem;
  }
  .cbtn:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  /* painel aberto: o botão fica afundado */
  .cbtn[aria-expanded='true'] {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  /* carregando: anel sóbrio parado + texto (nada gira em loop) */
  .spin {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 1rem;
    background: var(--bg);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  .ring {
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    border: 3px solid color-mix(in srgb, var(--outline) 25%, transparent);
    border-top-color: var(--outline);
  }
  .error,
  .up-next {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
    max-width: 34rem;
    z-index: 2;
  }
  .error {
    border-color: var(--danger);
    box-shadow: 4px 4px 0 var(--danger);
  }
  .error h2 {
    font-size: 1.25rem;
  }
  .up-next {
    top: auto;
    left: auto;
    inset-inline-end: 2.5rem;
    bottom: 13rem;
    transform: none;
  }
  .up-next strong {
    color: var(--accent);
    font-size: 1.15em;
  }
  .row {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }
  /* áudio e legendas: painel sóbrio sobre o filme, sombra dura maior */
  .tracks {
    position: absolute;
    inset-inline-end: 2.5rem;
    bottom: 13rem;
    z-index: 3;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.25rem 2rem;
    min-width: 32rem;
    box-shadow: 8px 8px 0 var(--outline);
  }
  .tracks .btn {
    grid-column: 1 / -1;
    justify-self: start;
  }
  .col {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .col h2 {
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    display: flex;
    gap: 0.5rem;
    align-items: center;
    padding-bottom: 0.5rem;
    border-bottom: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  [role='radiogroup'] {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  /* opção: bolinha que enche de mostarda quando escolhida */
  .opt {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    min-height: 2.8rem;
    text-align: start;
    padding: 0.4rem 0.8rem;
    font: inherit;
    color: var(--text);
    background: none;
    border: 2px solid transparent;
    border-radius: var(--radii-sm);
    cursor: pointer;
  }
  .opt::before {
    content: '';
    flex: none;
    width: 1.1rem;
    height: 1.1rem;
    border-radius: 50%;
    border: var(--edge) solid var(--outline);
    background: var(--bg);
  }
  .opt[aria-checked='true'] {
    font-weight: 700;
    background: color-mix(in srgb, var(--accent) 18%, transparent);
  }
  .opt[aria-checked='true']::before {
    background: var(--accent);
    box-shadow: inset 0 0 0 2px var(--bg);
  }
  .opt:hover {
    border-color: var(--outline);
  }
  .opt:focus {
    outline-offset: 0;
  }
  :global(.reduce-motion) .head,
  :global(.reduce-motion) .controls,
  :global(.reduce-motion) .help {
    transition: none;
  }
</style>
