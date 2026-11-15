<script lang="ts">
  // Uma lista do servidor arkflix: grade com os filmes e séries (Poster) e,
  // se tiver jogos, o atalho pra Biblioteca com eles. A lista é do usuário de
  // lá: renomear, apagar e tirar item só no perfil adulto.
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import Poster from '../components/Poster.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { arkEditable, loadArkLists } from '../lib/ark'
  import { pushHandler } from '../lib/input'
  import { current, focusDefault, focusEl } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, back, confirm, go, prompt, toast } from '../lib/state.svelte'
  import type { ArkList, WatchItem } from '../lib/types'
  import { cardLabel, displayName } from '../lib/watch'
  import { tick } from 'svelte'

  let { id }: { id: string } = $props()
  let list = $state<ArkList | null>(null)
  let failed = $state('')
  const editable = $derived(arkEditable())

  async function load() {
    try {
      list = await api.arkList(id)
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
    tick().then(focusDefault)
  }

  $effect(() => {
    void id
    load()
  })

  function open(it: WatchItem) {
    if (it.type === 'Episode') go({ name: 'player', id: it.id })
    else go({ name: 'title', id: it.id })
  }

  async function removeItem(it: WatchItem | undefined) {
    if (!it || !list || !editable) return
    try {
      await api.arkSetInList(list.id, 'watch', it.id, false)
      const i = list.videos.findIndex((v) => v.id === it.id)
      list.videos = list.videos.filter((v) => v.id !== it.id)
      list.count = Math.max(0, list.count - 1)
      play('toggle')
      toast(`${displayName(it)} saiu da lista`, 'ok', 2500)
      await tick()
      const cards = document.querySelectorAll<HTMLElement>('.grid [data-item]')
      focusEl(cards[Math.min(i, cards.length - 1)] ?? null)
      if (!cards.length) focusDefault()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function rename() {
    if (!list) return
    await prompt('Renomear lista', list.name, 'Salvar', async (name) => {
      try {
        const next = await api.arkRenameList(id, name)
        if (list) list.name = next.name
        loadArkLists().catch(() => {})
        return null
      } catch (err) {
        return (err as Error).message
      }
    })
  }

  async function remove() {
    if (!list) return
    if (!(await confirm(`Apagar a lista "${list.name}" do servidor? Os filmes, séries e jogos continuam lá.`, 'Apagar lista', true))) return
    try {
      await api.arkDeleteList(id)
      app.arkLists = app.arkLists.filter((l) => l.id !== id)
      toast('Lista apagada', 'ok', 2500)
      back()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function focused(): WatchItem | undefined {
    const item = current()?.dataset.item
    return list?.videos.find((v) => v.id === item)
  }

  $effect(() =>
    pushHandler((a) => {
      if (app.confirm || app.prompt) return false
      if (a === 'details') {
        removeItem(focused())
        return true
      }
      return a === 'fav' || a === 'prevTab' || a === 'nextTab'
    }),
  )
</script>

<div class="page">
  <TopBar />
  <main class="content" aria-labelledby="list-title">
    {#if failed}
      <section class="alert panel" role="alert">
        <span class="chip error">Erro</span>
        <h1 id="list-title">Sem sinal</h1>
        <p class="muted">{failed}</p>
        <button class="btn primary" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button>
      </section>
    {:else if !list}
      <div class="loading" role="status" aria-busy="true">
        <p class="meta" id="list-title">Buscando a lista…</p>
      </div>
    {:else}
      <header class="head">
        <p class="kind"><span class="chip"><Icon name="cloud" size="1em" /> Lista do servidor</span></p>
        <h1 id="list-title">{list.name}</h1>
        <p class="meta count">{list.count} ite{list.count === 1 ? 'm' : 'ns'}{list.games.length ? ` · ${list.games.length} jogo${list.games.length === 1 ? '' : 's'}` : ''}</p>
        <div class="actions">
          {#if list.games.length}
            <button class="btn" data-nav data-nav-default={list.videos.length ? undefined : ''} onclick={() => go({ name: 'library', source: `arklist:${id}` })}>
              <Icon name="pad" /> Ver os jogos ({list.games.length})
            </button>
          {/if}
          {#if editable}
            <button class="btn" data-nav onclick={rename}><Icon name="pencil" /> Renomear</button>
            <button class="btn danger" data-nav onclick={remove}><Icon name="trash" /> Apagar lista</button>
          {/if}
        </div>
      </header>
      {#if list.videos.length}
        <ul class="grid" data-nav-group="ark-grid" aria-label={`Filmes e séries de ${list.name}`}>
          {#each list.videos as it, i (it.id)}
            <li>
              <button class="card" data-nav data-item={it.id} data-nav-default={i === 0 ? '' : undefined} aria-label={cardLabel(it)} onclick={() => open(it)}>
                <Poster item={it} />
                <span class="name">{displayName(it)}</span>
              </button>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty muted">Nenhum filme ou série nesta lista.</p>
      {/if}
    {/if}
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      ...(editable && list?.videos.length ? [{ button: 'west' as const, label: 'Tirar da lista' }] : []),
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .content {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.5rem 2.5rem 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }
  .head {
    display: grid;
    gap: 0.6rem;
    justify-items: start;
  }
  .kind {
    margin: 0;
  }
  h1 {
    font-size: clamp(2rem, 4vw, 3rem);
    margin: 0;
    overflow-wrap: anywhere;
  }
  .count {
    margin: 0;
    color: var(--muted);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
    margin-top: 0.4rem;
  }
  .grid {
    list-style: none;
    margin: 0;
    padding: 0.8rem 0.6rem 1.5rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1.8rem 1.5rem;
  }
  /* cartão: o pôster é a moldura; no foco sobe e ganha o anel */
  .card {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 0.55rem;
    width: 100%;
    padding: 0;
    font: inherit;
    color: var(--text);
    text-align: start;
    border: none;
    background: none;
    cursor: pointer;
  }
  .card:focus,
  .card:focus-visible {
    outline: none;
  }
  .card:hover :global(.poster) {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card:focus :global(.poster) {
    transform: translate(-4px, -4px);
    box-shadow: 8px 8px 0 var(--outline);
    outline: 3px solid var(--focus);
    outline-offset: 3px;
  }
  .card:active :global(.poster) {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  .name {
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card:focus .name {
    color: var(--accent);
  }
  .empty {
    padding: 1rem;
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
    max-width: 36rem;
  }
  .alert {
    max-width: 36rem;
    display: grid;
    gap: 0.8rem;
    justify-items: start;
  }
  .loading {
    margin: auto;
  }
</style>
