<script lang="ts">
  // "Adicionar à playlist": escolhe uma playlist da conta (ou cria uma) e
  // põe as faixas (ou o álbum inteiro) no fim. No arkflix-server, são as
  // mesmas listas do servidor que filmes e jogos usam.
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, prompt, toast } from '../lib/state.svelte'
  import type { MusicPlaylist } from '../lib/types'
  import Icon from './Icon.svelte'

  let { name, tracks = [], album = '', onclose }: { name: string; tracks?: string[]; album?: string; onclose: () => void } = $props()
  let lists = $state<MusicPlaylist[] | null>(null)
  let busy = $state(false)

  async function load() {
    try {
      lists = (await api.musicPlaylists()).filter((p) => p.mine)
    } catch (err) {
      toast((err as Error).message, 'error')
      onclose()
    }
    tick().then(focusDefault)
  }

  $effect(() => {
    load()
    return pushHandler((a) => {
      if (app.prompt) return false
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })

  async function put(p: MusicPlaylist) {
    if (busy) return
    busy = true
    try {
      if (album) await api.musicAddAlbumToPlaylist(p.id, album)
      else await api.musicAddToPlaylist(p.id, tracks)
      play('accept')
      toast(`${name} em “${p.name}”`, 'ok', 2500)
      onclose()
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function create() {
    let made: MusicPlaylist | null = null
    await prompt('Nova playlist', '', 'Criar playlist', async (v) => {
      try {
        made = await api.musicCreatePlaylist(v, album ? [] : tracks)
        if (album) await api.musicAddAlbumToPlaylist(made.id, album)
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 60 })
    const got = made as MusicPlaylist | null
    if (got) {
      toast(`Playlist “${got.name}” criada com ${name}`, 'ok', 2500)
      onclose()
      return
    }
    tick().then(focusDefault)
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="pl-pick-title" data-nav-scope={app.prompt ? undefined : ''}>
    <h2 id="pl-pick-title">Adicionar à playlist</h2>
    <p class="name">{name}</p>
    {#if lists === null}
      <p class="empty" role="status">Buscando as playlists…</p>
    {:else if lists.length}
      <div class="choices" role="group" aria-label="Playlists">
        {#each lists as p, i (p.id)}
          <button class="choice" data-nav data-nav-default={i === 0 ? '' : undefined} disabled={busy} onclick={() => put(p)}>
            <Icon name="queue" />{p.name}
            <span class="count">{p.songs}</span>
          </button>
        {/each}
      </div>
    {:else}
      <p class="empty">Nenhuma playlist ainda. Crie a primeira com “Nova playlist”.</p>
    {/if}
    <div class="actions">
      <button class="btn" data-nav data-nav-default={lists?.length ? undefined : ''} onclick={create}><Icon name="plus" /> Nova playlist</button>
      <button class="btn" data-nav onclick={onclose}>Cancelar</button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: color-mix(in srgb, var(--bg) 78%, transparent);
    display: grid;
    place-items: center;
    z-index: 80;
  }
  .dialog {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: min(30rem, 100% - 2rem);
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 8px 8px 0 var(--outline);
  }
  h2 {
    font-size: 1.25rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .name {
    font-family: var(--font-hud);
    color: var(--muted);
    margin-top: -0.6rem;
  }
  .choices {
    display: grid;
    gap: 0.15rem;
  }
  .choice {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-height: 3rem;
    padding: 0.4rem 0.8rem;
    font: inherit;
    font-size: 1.05rem;
    text-align: start;
    color: var(--text);
    background: none;
    border: 0;
    border-radius: 4px;
    cursor: pointer;
  }
  .choice:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .count {
    margin-inline-start: auto;
    font-family: var(--font-mono);
    font-size: 0.9rem;
    color: var(--muted);
  }
  .empty {
    padding: 1rem;
    color: var(--muted);
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    justify-content: flex-end;
    padding-top: 1rem;
    border-top: var(--edge) solid var(--outline);
  }
</style>
