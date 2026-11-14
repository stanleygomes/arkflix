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

  {#if buffering && !failed}<p class="spin pixel" role="status">Carregando…</p>{/if}

  {#if failed}
    <section class="error panel" role="alert">
      <h2 class="pixel">Sem sinal</h2>
      <p>{failed}</p>
      <button class="btn primary" data-nav data-nav-default onclick={() => back()}><Icon name="back" /> Voltar</button>
    </section>
  {/if}

  <header class="head">
    <button class="btn" data-nav aria-label="Sair do vídeo" onclick={() => back()}><Icon name="back" /> Sair</button>
    <div class="names">
      <h1 class="pixel" id="player-title">{heading}</h1>
      {#if sub}<p class="muted">{sub}</p>{/if}
    </div>
  </header>

  {#if countdown}
    <section class="up-next panel" aria-live="polite">
      <p>Próximo episódio em <strong>{countdown}</strong> s: {stream?.next ? `${episodeTag(stream.next)} · ${stream.next.name}` : ''}</p>
      <div class="row">
        <button class="btn primary" data-nav data-next onclick={playNext}><Icon name="next" /> Assistir agora</button>
        <button class="btn" data-nav onclick={cancelNext}>Ficar aqui</button>
      </div>
    </section>
  {/if}

  <footer class="controls" role="group" aria-label="Controles do vídeo">
    <div class="timeline">
      <span class="time">{clock(time)}</span>
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
        <span class="knob" style:left={`${pct}%`}></span>
      </div>
      <span class="time">{clock(duration)}</span>
    </div>
    <div class="buttons">
      <button class="cbtn" data-nav aria-label="Voltar 10 segundos" onclick={() => seek(-10)}><Icon name="rewind" /></button>
      <button class="cbtn big" data-nav data-play data-nav-default aria-label={paused ? 'Tocar' : 'Pausar'} onclick={togglePlay}>
        <Icon name={paused ? 'play' : 'pause'} size="1.5em" />
      </button>
      <button class="cbtn" data-nav aria-label="Avançar 10 segundos" onclick={() => seek(10)}><Icon name="forward" /></button>
      <span class="gap"></span>
      <button class="cbtn wide" data-nav aria-label="Áudio e legendas" aria-expanded={panel} onclick={() => openPanel(!panel)}>
        <Icon name="cc" /> <span>Áudio e legendas</span>
      </button>
      {#if stream?.next}
        <button class="cbtn wide" data-nav onclick={playNext}><Icon name="next" /> <span>Próximo episódio</span></button>
      {/if}
    </div>
  </footer>

  {#if panel && stream}
    <div class="tracks panel" data-nav-scope role="dialog" aria-modal="true" aria-label="Áudio e legendas">
      <div class="col">
        <h2 class="pixel"><Icon name="audio" /> Áudio</h2>
        <div role="radiogroup" aria-label="Áudio">
          {#each stream.audio as t (t.index)}
            <button class="opt" data-nav role="radio" aria-checked={t.index === stream.audioIndex} onclick={() => setAudio(t.index)}>{t.label || t.language || `Faixa ${t.index}`}</button>
          {:else}
            <p class="muted">Uma faixa só.</p>
          {/each}
        </div>
      </div>
      <div class="col">
        <h2 class="pixel"><Icon name="cc" /> Legendas</h2>
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
  .player {
    position: fixed;
    inset: 0;
    z-index: 5;
    background: #000;
    color: #fff;
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
    left: 0;
    right: 0;
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
    background: linear-gradient(to bottom, rgb(0 0 0 / 0.85), transparent);
  }
  .names h1 {
    font-size: 1.4rem;
    line-height: 1.3;
  }
  .names .muted {
    color: #ddd;
  }
  .controls {
    bottom: 3.2rem;
    padding: 3rem 2.5rem 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    background: linear-gradient(to top, rgb(0 0 0 / 0.9), transparent);
  }
  .help {
    bottom: 0;
    background: rgb(0 0 0 / 0.9);
  }
  .timeline {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  .time {
    font-variant-numeric: tabular-nums;
    font-size: 1.05rem;
    min-width: 4.5rem;
    text-align: center;
  }
  /* fita: barra grossa de pixel, o foco acende a borda */
  .bar {
    position: relative;
    flex: 1;
    height: 0.9rem;
    background: rgb(255 255 255 / 0.22);
    border: 2px solid #000;
    box-shadow: 3px 3px 0 #000;
    cursor: pointer;
  }
  .bar:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .fill {
    position: absolute;
    inset: 0 auto 0 0;
    background: var(--accent);
  }
  .knob {
    position: absolute;
    top: 50%;
    width: 1.3rem;
    height: 1.3rem;
    transform: translate(-50%, -50%);
    background: #fff;
    border: 3px solid #000;
  }
  .buttons {
    display: flex;
    align-items: center;
    gap: 0.9rem;
  }
  .gap {
    flex: 1;
  }
  .cbtn {
    min-width: 3.2rem;
    height: 3.2rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0 0.7rem;
    font: inherit;
    font-weight: 700;
    color: #fff;
    background: rgb(20 18 30 / 0.85);
    border: 2px solid rgb(255 255 255 / 0.35);
    box-shadow: 3px 3px 0 #000;
    cursor: pointer;
  }
  .cbtn.big {
    min-width: 4.2rem;
    height: 4.2rem;
    color: var(--on-accent);
    background: var(--accent);
    border-color: #000;
  }
  .cbtn:hover {
    border-color: var(--accent);
  }
  .spin {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    font-size: 1.3rem;
    animation: blink 1s steps(2) infinite;
    text-shadow: 3px 3px 0 #000;
  }
  .error,
  .up-next {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 34rem;
    z-index: 2;
  }
  .up-next {
    top: auto;
    left: auto;
    right: 2.5rem;
    bottom: 12rem;
    transform: none;
  }
  .row {
    display: flex;
    gap: 0.8rem;
    flex-wrap: wrap;
  }
  .tracks {
    position: absolute;
    right: 2.5rem;
    bottom: 12rem;
    z-index: 3;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem 2rem;
    min-width: 30rem;
  }
  .tracks .btn {
    grid-column: 1 / -1;
    justify-self: start;
  }
  .col {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }
  .col h2 {
    font-size: 1rem;
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  [role='radiogroup'] {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .opt {
    text-align: left;
    padding: 0.5rem 0.8rem;
    font: inherit;
    color: var(--text);
    background: none;
    border: 2px solid transparent;
    cursor: pointer;
  }
  .opt[aria-checked='true'] {
    color: var(--on-accent);
    background: var(--accent);
  }
  .opt:hover {
    border-color: var(--accent);
  }
  :global(.reduce-motion) .head,
  :global(.reduce-motion) .controls,
  :global(.reduce-motion) .help {
    transition: none;
  }
</style>
