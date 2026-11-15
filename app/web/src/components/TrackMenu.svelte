<script lang="ts">
  // Opções de uma faixa (menu sóbrio, abre com X/Quadrado ou "⋯"): tocar a
  // seguir, pôr na fila, playlist, favorita, ir pro álbum/artista e, numa
  // playlist da conta, tirar ou mudar de lugar.
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { focusDefault } from '../lib/nav'
  import { addToQueue, playNext, starTrack } from '../lib/music.svelte'
  import { play } from '../lib/sound'
  import { app, go, isKids, toast } from '../lib/state.svelte'
  import type { MusicTrack } from '../lib/types'
  import Icon, { type IconName } from './Icon.svelte'
  import PlaylistPicker from './PlaylistPicker.svelte'

  let {
    track,
    onclose,
    onplay,
    onremove,
    onmove,
    removeLabel = 'Tirar da playlist',
  }: {
    track: MusicTrack
    onclose: () => void
    onplay?: () => void
    onremove?: () => void
    onmove?: (delta: number) => void
    removeLabel?: string
  } = $props()

  let picking = $state(false)

  type Item = { id: string; label: string; icon: IconName; run: () => void | Promise<void> }
  const items = $derived.by((): Item[] => {
    const out: Item[] = []
    if (onplay) out.push({ id: 'play', label: 'Tocar agora', icon: 'play', run: () => (onplay!(), onclose()) })
    out.push(
      { id: 'next', label: 'Tocar a seguir', icon: 'next', run: () => (playNext([track]), toast(`${track.title} toca a seguir`, 'ok', 2000), onclose()) },
      { id: 'queue', label: 'Adicionar à fila', icon: 'queue', run: () => (addToQueue([track]), toast(`${track.title} na fila`, 'ok', 2000), onclose()) },
    )
    if (!isKids()) out.push({ id: 'playlist', label: 'Adicionar à playlist', icon: 'plus', run: () => void (picking = true) })
    out.push({
      id: 'star',
      label: track.starred ? 'Tirar das favoritas' : 'Favoritar',
      icon: 'star',
      run: async () => {
        try {
          await starTrack(track)
          play('toggle')
          toast(track.starred ? `${track.title} nas favoritas` : `${track.title} saiu das favoritas`, 'ok', 2000)
        } catch (err) {
          toast((err as Error).message, 'error')
        }
        onclose()
      },
    })
    if (track.albumId && !(app.stack.at(-1)?.name === 'album' && (app.stack.at(-1) as { id: string }).id === track.albumId))
      out.push({ id: 'album', label: 'Ir pro álbum', icon: 'disc', run: () => (onclose(), go({ name: 'album', id: track.albumId })) })
    if (track.artistId) out.push({ id: 'artist', label: 'Ir pro artista', icon: 'user', run: () => (onclose(), go({ name: 'artist', id: track.artistId })) })
    if (onmove) {
      out.push({ id: 'up', label: 'Subir uma posição', icon: 'back', run: () => (onmove!(-1), onclose()) })
      out.push({ id: 'down', label: 'Descer uma posição', icon: 'next', run: () => (onmove!(1), onclose()) })
    }
    if (onremove) out.push({ id: 'remove', label: removeLabel, icon: 'trash', run: () => (onremove!(), onclose()) })
    return out
  })

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (picking || app.prompt) return false
      if (a === 'back' || a === 'details') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })
</script>

{#if picking}
  <PlaylistPicker name={track.title} tracks={[track.id]} onclose={onclose} />
{:else}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="menu panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="track-menu-title" data-nav-scope>
      <h2 id="track-menu-title">{track.title}</h2>
      <p class="meta">{track.artist}{track.album ? ` · ${track.album}` : ''}</p>
      <div class="items" role="group" aria-label="Opções da faixa">
        {#each items as it, i (it.id)}
          <button class="item" class:danger={it.id === 'remove'} data-nav data-nav-default={i === 0 ? '' : undefined} data-action={it.id} onclick={it.run}>
            <Icon name={it.icon} /> {it.label}
          </button>
        {/each}
      </div>
      <button class="btn" data-nav onclick={onclose}><Icon name="close" /> Fechar</button>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: color-mix(in srgb, var(--bg) 78%, transparent);
    display: grid;
    place-items: center;
    z-index: 80;
  }
  .menu {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    width: min(26rem, 100% - 2rem);
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 8px 8px 0 var(--outline);
  }
  h2 {
    font-family: var(--font-label);
    font-size: 1.15rem;
    -webkit-text-stroke: 0;
    text-shadow: none;
    color: var(--text);
  }
  .meta {
    color: var(--muted);
    margin-top: -0.5rem;
  }
  .items {
    display: grid;
    gap: 0.1rem;
    border-top: var(--edge) solid var(--outline);
    padding-top: 0.5rem;
  }
  .item {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    min-height: 2.9rem;
    padding: 0.35rem 0.7rem;
    font: inherit;
    font-size: 1.05rem;
    text-align: start;
    color: var(--text);
    background: none;
    border: 0;
    border-radius: 4px;
    cursor: pointer;
  }
  .item:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .item.danger {
    color: var(--danger);
  }
</style>
