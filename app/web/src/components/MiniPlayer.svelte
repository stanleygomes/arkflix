<script lang="ts">
  // Barrinha da música (em toda tela enquanto tem fila): capa, faixa, tocar/
  // pausar, próxima e o progresso num traço mostarda em cima. Mora embaixo
  // da tela: o App encolhe a camada das telas pra ela não cobrir a barra de
  // atalhos. Sóbria (nada de desenho animado aqui).
  import { mmss, next, nowTrack, player, toggle } from '../lib/music.svelte'
  import { go } from '../lib/state.svelte'
  import Cover from './Cover.svelte'
  import Icon from './Icon.svelte'

  const t = $derived(nowTrack())
  const duration = $derived(player.duration || t?.duration || 0)
  const pct = $derived(duration ? Math.min(100, (player.time / duration) * 100) : 0)
</script>

{#if t}
  <aside class="mini" aria-label="Música">
    <span class="progress" aria-hidden="true"><span style:width={`${pct}%`}></span></span>
    <button class="now" data-nav data-mini="open" aria-label={`Tocando agora: ${t.title}, ${t.artist}`} onclick={() => go({ name: 'nowplaying' })}>
      <span class="art"><Cover src={t.cover} name={t.album || t.title} /></span>
      <span class="text">
        <span class="song">{t.title}</span>
        <span class="artist">{t.artist}{player.held ? ' · pausado pelo jogo' : ''}</span>
      </span>
    </button>
    <span class="time mono" aria-hidden="true">{mmss(player.time)} / {mmss(duration)}</span>
    <button class="btn cbtn" data-nav data-mini="toggle" aria-label={player.playing ? 'Pausar música' : 'Tocar música'} onclick={toggle}>
      <Icon name={player.playing ? 'pause' : 'play'} />
    </button>
    <button class="btn cbtn" data-nav data-mini="next" aria-label="Próxima música" onclick={next}><Icon name="next" /></button>
  </aside>
{/if}

<style>
  .mini {
    position: fixed;
    inset-inline: var(--safe);
    inset-block-end: var(--safe);
    height: var(--mini-h);
    z-index: 40;
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.4rem 1rem 0.4rem 0.6rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  .progress {
    position: absolute;
    inset-inline: 0;
    inset-block-start: 0;
    height: 4px;
    background: color-mix(in srgb, var(--outline) 18%, transparent);
  }
  .progress span {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .now {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 0.8rem;
    padding: 0.2rem;
    font: inherit;
    color: var(--text);
    text-align: start;
    background: none;
    border: 0;
    border-radius: 4px;
    cursor: pointer;
  }
  .now:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: 2px;
  }
  .art {
    width: calc(var(--mini-h) - 1.4rem);
    flex: none;
  }
  .art :global(.cover) {
    box-shadow: none;
    border-width: 2px;
  }
  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .song {
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .artist {
    font-size: 0.9rem;
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .time {
    color: var(--muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .cbtn {
    min-width: 2.8rem;
    min-height: 2.8rem;
    justify-content: center;
  }
</style>
