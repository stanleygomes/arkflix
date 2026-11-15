<script lang="ts">
  // Tocando agora: capa, linha do tempo, controles e a fila. Controle igual
  // ao do player de filmes: A toca/pausa (ou o botão em foco), Start também;
  // esquerda/direita na linha do tempo pulam 10 s; LB/RB, faixa anterior e
  // próxima; Y favorita; X abre as opções da faixa da fila; B volta.
  import Cover from '../components/Cover.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import PlaylistPicker from '../components/PlaylistPicker.svelte'
  import TopBar from '../components/TopBar.svelte'
  import TrackList from '../components/TrackList.svelte'
  import { pushHandler } from '../lib/input'
  import { current } from '../lib/nav'
  import {
    active,
    clearQueue,
    cycleRepeat,
    jump,
    mmss,
    moveInQueue,
    next,
    nowTrack,
    player,
    prev,
    removeAt,
    seek,
    seekBy,
    starTrack,
    toggle,
    toggleShuffle,
  } from '../lib/music.svelte'
  import { play } from '../lib/sound'
  import { back, confirm, go, isKids, toast } from '../lib/state.svelte'

  const t = $derived(nowTrack())
  const duration = $derived(player.duration || t?.duration || 0)
  const pct = $derived(duration ? Math.min(100, (player.time / duration) * 100) : 0)
  let picking = $state(false)

  // fila acabou de esvaziar: não tem o que mostrar
  $effect(() => {
    if (!active()) back()
  })

  async function star() {
    if (!t) return
    try {
      await starTrack(t)
      play('toggle')
      toast(t.starred ? `${t.title} nas favoritas` : `${t.title} saiu das favoritas`, 'ok', 2000)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function clear() {
    if (await confirm('Limpar a fila e parar a música?', 'Limpar fila', true)) clearQueue()
  }

  $effect(() =>
    pushHandler((a) => {
      if (picking) return false
      const onBar = !!current()?.closest('.bar')
      const inQueue = !!current()?.closest('.queue')
      switch (a) {
        case 'left':
        case 'right':
          if (onBar) {
            seekBy(a === 'left' ? -10 : 10)
            play('move')
            return true
          }
          return false
        case 'prevTab':
          prev()
          return true
        case 'nextTab':
          next()
          return true
        case 'menu':
          toggle()
          return true
        case 'fav':
          if (inQueue) return false // a lista favorita a faixa em foco
          star()
          return true
      }
      return false
    }),
  )

  const repeatLabel = $derived(player.repeat === 'off' ? 'Repetir: desligado' : player.repeat === 'all' ? 'Repetir: a fila' : 'Repetir: esta faixa')
</script>

<div class="page">
  <TopBar />
  {#if t}
    <main class="np" aria-labelledby="np-title">
      <section class="stage">
        <div class="art">
          <Cover src={t.cover} name={t.album || t.title} local={t.local} />
        </div>
        <div class="info">
          <p class="meta">{player.held ? 'Pausado pelo jogo ou filme' : player.playing ? 'Tocando agora' : 'Pausado'}{player.loading ? ' · carregando…' : ''}</p>
          <h1 id="np-title">{t.title}</h1>
          <p class="who">
            {#if t.artistId}<button class="link" data-nav onclick={() => go({ name: 'artist', id: t.artistId })}>{t.artist}</button>{:else}{t.artist}{/if}
            {#if t.album}
              <span aria-hidden="true">·</span>
              {#if t.albumId}<button class="link" data-nav onclick={() => go({ name: 'album', id: t.albumId })}>{t.album}</button>{:else}{t.album}{/if}
            {/if}
          </p>
          {#if player.error}<p class="err" role="alert">{player.error}</p>{/if}

          <div class="timeline">
            <span class="time mono">{mmss(player.time)}</span>
            <div
              class="bar"
              data-nav
              role="slider"
              tabindex="0"
              aria-label="Posição na faixa"
              aria-valuemin={0}
              aria-valuemax={Math.round(duration)}
              aria-valuenow={Math.round(player.time)}
              aria-valuetext={`${mmss(player.time)} de ${mmss(duration)}`}
              onkeydown={(e) => {
                if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') e.preventDefault()
              }}
              onclick={(e) => {
                const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
                seek(((e.clientX - r.left) / r.width) * duration)
              }}
            >
              <span class="fill" style:width={`${pct}%`}></span>
              <span class="knob" style:inset-inline-start={`${pct}%`}></span>
            </div>
            <span class="time mono">{mmss(duration)}</span>
          </div>

          <div class="controls" role="group" aria-label="Controles da música">
            <button class="btn cbtn" data-nav aria-pressed={player.shuffle} aria-label="Aleatório" onclick={toggleShuffle}><Icon name="shuffle" /></button>
            <button class="btn cbtn" data-nav aria-label="Faixa anterior" onclick={prev}><Icon name="prev" /></button>
            <button class="btn primary cbtn big" data-nav data-nav-default data-play aria-label={player.playing ? 'Pausar' : 'Tocar'} onclick={toggle}>
              <Icon name={player.playing ? 'pause' : 'play'} size="1.5em" />
            </button>
            <button class="btn cbtn" data-nav aria-label="Próxima faixa" onclick={next}><Icon name="next" /></button>
            <button class="btn cbtn" data-nav aria-pressed={player.repeat !== 'off'} aria-label={repeatLabel} onclick={cycleRepeat}>
              <Icon name="repeat" />{#if player.repeat === 'one'}<span class="one" aria-hidden="true">1</span>{/if}
            </button>
            <span class="gap"></span>
            <button class="btn cbtn" data-nav aria-pressed={t.starred} aria-label="Favorita" onclick={star}><Icon name="star" /></button>
            {#if !isKids()}
              <button class="btn cbtn wide" data-nav onclick={() => (picking = true)}><Icon name="plus" /> <span>Playlist</span></button>
            {/if}
          </div>
        </div>
      </section>

      <section class="queue" aria-labelledby="queue-title">
        <div class="queue-head">
          <h2 id="queue-title">Fila <span class="muted">({player.index + 1} de {player.queue.length})</span></h2>
          <button class="btn small" data-nav onclick={clear}><Icon name="trash" /> Limpar fila</button>
        </div>
        <TrackList
          tracks={player.queue}
          label="Fila"
          numbers="index"
          group="queue"
          onplay={jump}
          onremove={removeAt}
          onmove={moveInQueue}
          removeLabel="Tirar da fila"
        />
      </section>
    </main>
  {/if}
  <HelpBar
    hints={[
      { button: 'south', label: 'Tocar / pausar' },
      { button: 'l1', label: 'Anterior' },
      { button: 'r1', label: 'Próxima' },
      { button: 'left', label: '10 s (na linha)' },
      { button: 'north', label: 'Favoritar' },
      { button: 'west', label: 'Opções da faixa' },
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

{#if picking && t}
  <PlaylistPicker name={t.title} tracks={[t.id]} onclose={() => (picking = false)} />
{/if}

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .np {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
    gap: 2.5rem;
    padding: 1rem 2.5rem 1rem;
  }
  .stage {
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
    min-width: 0;
    overflow-y: auto;
  }
  .art {
    width: min(100%, 22rem, 38vh);
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }
  .meta {
    color: var(--muted);
  }
  h1 {
    font-size: clamp(1.8rem, min(4vw, 6vh), 3.2rem);
    overflow-wrap: anywhere;
  }
  .who {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    font-size: 1.1rem;
  }
  .link {
    font: inherit;
    font-weight: 700;
    color: var(--text);
    background: none;
    border: 0;
    border-bottom: 2px solid color-mix(in srgb, var(--outline) 40%, transparent);
    padding: 0.1rem 0;
    cursor: pointer;
  }
  .link:hover {
    border-bottom-color: var(--accent);
  }
  .err {
    color: var(--danger);
  }
  .timeline {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: 1rem;
    margin-top: 0.4rem;
  }
  .time {
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  /* linha do tempo sóbria: trilho com borda, mostarda o que já tocou */
  .bar {
    position: relative;
    height: 1.1rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: 4px;
    cursor: pointer;
  }
  .bar:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .fill {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    background: var(--accent);
  }
  .knob {
    position: absolute;
    top: 50%;
    width: 0.9rem;
    height: 1.7rem;
    translate: -50% -50%;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 3px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .controls {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.8rem;
    margin-top: 0.4rem;
  }
  .cbtn {
    min-width: 3rem;
    min-height: 3rem;
    justify-content: center;
    position: relative;
  }
  .cbtn.big {
    min-width: 4rem;
    min-height: 4rem;
  }
  .cbtn[aria-pressed='true'] {
    color: var(--accent);
  }
  .one {
    position: absolute;
    inset-block-end: 0.15rem;
    inset-inline-end: 0.3rem;
    font-family: var(--font-mono);
    font-size: 0.7rem;
    font-weight: 700;
  }
  .gap {
    width: 0.6rem;
  }
  .queue {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    min-height: 0;
    overflow-y: auto;
  }
  .queue-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
  }
  h2 {
    font-family: var(--font-label);
    font-size: 1rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
</style>
