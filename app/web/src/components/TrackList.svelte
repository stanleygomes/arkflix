<script lang="ts">
  // Lista de faixas (tabela sóbria de programa de cinema): número, título,
  // artista, duração. A linha toca dali em diante; X/Quadrado (ou "⋯") abre as
  // opções e Y/Triângulo favorita. A que está tocando ganha o traço mostarda.
  import { pushHandler } from '../lib/input'
  import { current } from '../lib/nav'
  import { mmss, nowTrack, player, starTrack } from '../lib/music.svelte'
  import { play } from '../lib/sound'
  import { toast } from '../lib/state.svelte'
  import type { MusicTrack } from '../lib/types'
  import Icon from './Icon.svelte'
  import TrackMenu from './TrackMenu.svelte'

  let {
    tracks,
    label,
    onplay,
    numbers = 'track',
    artist = true,
    album = false,
    onremove,
    onmove,
    removeLabel,
    group = 'tracks',
  }: {
    tracks: MusicTrack[]
    label: string
    onplay: (i: number) => void
    numbers?: 'track' | 'index'
    artist?: boolean
    album?: boolean
    onremove?: (i: number) => void
    onmove?: (i: number, delta: number) => void
    removeLabel?: string
    group?: string
  } = $props()

  let menu = $state(-1)
  const playing = $derived(nowTrack()?.id)

  function rowLabel(t: MusicTrack, i: number): string {
    // na ordem do que aparece na linha (o nome acessível contém o visível)
    const parts = [`${numbers === 'track' && t.track ? t.track : i + 1}. ${t.title}`]
    if (artist) parts.push(t.artist)
    if (album) parts.push(t.album)
    parts.push(mmss(t.duration))
    if (t.starred) parts.push('favorita')
    if (t.local) parts.push('No PC')
    if (t.id === playing) parts.push(player.playing ? 'tocando' : 'pausada')
    return parts.join(', ')
  }

  function focusedIndex(): number {
    const el = current()
    if (!el || el.dataset.list !== group) return -1
    return Number(el.dataset.track ?? -1)
  }

  $effect(() =>
    pushHandler((a) => {
      if (menu >= 0) return false
      const i = focusedIndex()
      if (i < 0) return false
      if (a === 'details') {
        play('accept')
        menu = i
        return true
      }
      if (a === 'fav') {
        const t = tracks[i]
        starTrack(t)
          .then(() => {
            play('toggle')
            toast(t.starred ? `${t.title} nas favoritas` : `${t.title} saiu das favoritas`, 'ok', 2000)
          })
          .catch((err) => toast((err as Error).message, 'error'))
        return true
      }
      return false
    }),
  )
</script>

<ol class="tracks" aria-label={label} data-nav-group={group}>
  {#each tracks as t, i (`${t.id}-${i}`)}
    <li class="row" class:now={t.id === playing}>
      <button class="main" data-nav data-list={group} data-track={i} aria-label={rowLabel(t, i)} onclick={() => onplay(i)}>
        <span class="num mono" aria-hidden="true">
          {#if t.id === playing}<Icon name={player.playing ? 'audio' : 'pause'} />{:else}{numbers === 'track' && t.track ? t.track : i + 1}{/if}
        </span>
        <span class="cell">
          <span class="name">{t.title}</span>
          {#if artist || album}<span class="by">{[artist ? t.artist : '', album ? t.album : ''].filter(Boolean).join(' · ')}</span>{/if}
        </span>
        <span class="flags" aria-hidden="true">
          {#if t.starred}<Icon name="star" />{/if}
          {#if t.local}<span class="pc"></span>{/if}
        </span>
        <span class="time mono" aria-hidden="true">{mmss(t.duration)}</span>
      </button>
      <button class="more" data-nav aria-label={`Opções de ${t.title}`} onclick={() => (menu = i)}>⋯</button>
    </li>
  {/each}
</ol>

{#if menu >= 0 && tracks[menu]}
  <TrackMenu
    track={tracks[menu]}
    onclose={() => (menu = -1)}
    onplay={() => onplay(menu)}
    onremove={onremove ? () => onremove(menu) : undefined}
    onmove={onmove ? (d) => onmove(menu, d) : undefined}
    {removeLabel}
  />
{/if}

<style>
  .tracks {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    border-top: var(--edge) solid var(--outline);
  }
  .row {
    display: flex;
    align-items: stretch;
    gap: 0.4rem;
    border-bottom: 1px solid color-mix(in srgb, var(--outline) 22%, transparent);
  }
  .main {
    flex: 1;
    min-width: 0;
    display: grid;
    grid-template-columns: 2.4rem 1fr auto auto;
    align-items: center;
    gap: 0.9rem;
    min-height: 3.2rem;
    padding: 0.35rem 0.7rem;
    font: inherit;
    color: var(--text);
    text-align: start;
    background: none;
    border: 0;
    border-inline-start: 4px solid transparent;
    cursor: pointer;
  }
  .main:hover {
    background: color-mix(in srgb, var(--outline) 8%, transparent);
  }
  .main:focus-visible,
  .more:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: -3px;
  }
  .now .main {
    border-inline-start-color: var(--accent);
  }
  .now .name {
    color: var(--accent);
  }
  .num {
    color: var(--muted);
    text-align: end;
    display: inline-flex;
    justify-content: flex-end;
  }
  .cell {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .name {
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .by {
    font-size: 0.9rem;
    color: var(--muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .flags {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--accent);
  }
  .pc::after {
    content: 'No PC';
  }
  .pc {
    font-family: var(--font-hud);
    font-size: 0.75rem;
    color: var(--accent-2);
    border: 2px solid currentColor;
    border-radius: 4px;
    padding: 0 0.35rem;
  }
  .time {
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .more {
    width: 2.8rem;
    font: inherit;
    font-size: 1.3rem;
    line-height: 1;
    color: var(--muted);
    background: none;
    border: 0;
    cursor: pointer;
  }
  .more:hover {
    color: var(--text);
  }
</style>
