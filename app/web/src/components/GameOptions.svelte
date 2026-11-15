<script lang="ts">
  // Opções do jogo: listas, editar, configurar, ocultar, apagar. Tudo que mexe
  // no console fica aqui dentro, longe dos botões principais. Com quick (Start
  // num jogo do Início), vem antes o básico: jogar/baixar, continuar,
  // favoritar e detalhes.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { bytes } from '../lib/format'
  import { app, back, confirm, game, go, isKids, putGame, system, toast } from '../lib/state.svelte'
  import Icon from './Icon.svelte'
  import { toggleFavorite } from '../lib/actions'
  import { downloadGame } from '../lib/downloads.svelte'
  import { arkEditable, arkWritable } from '../lib/ark'

  let {
    gameId,
    onclose,
    onlists,
    onconfig,
    onserverlists,
    quick = false,
  }: { gameId: string; onclose: () => void; onlists: () => void; onconfig: () => void; onserverlists?: () => void; quick?: boolean } = $props()

  async function launch(state = '') {
    if (!g) return
    onclose()
    try {
      play('start')
      await api.play(g.id, state)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  function details() {
    onclose()
    go({ name: 'game', id: gameId })
  }
  const g = $derived(game(gameId))
  const canResume = $derived(!!g && !g.remote && g.lastPlayed > 0 && system(g.system)?.emulator === 'retroarch')
  const configurable = $derived(!!g && !g.remote && system(g.system)?.emulator === 'retroarch')
  // jogo do RomM do arkflix-server: listas de lá e esconder (admin, gravável)
  const onArk = $derived(!!g?.onServer && !!app.ark?.games)
  const canServerList = $derived(onArk && !!onserverlists && arkEditable())
  const canHide = $derived(onArk && arkWritable())

  async function hideOnServer() {
    if (!g) return
    const ok = await confirm(
      `Esconder "${g.name}" do servidor? O jogo vai pra pasta de escondidos e some do Catálogo pra todo mundo. A cópia deste PC fica. Pra desfazer, só movendo de volta no servidor.`,
      'Esconder do servidor',
      true,
    )
    if (!ok) return
    try {
      await api.arkHide('game', gameId)
      toast(`"${g.name}" escondido do servidor`, 'ok', 3000)
      play('accept')
      onclose()
      if (g.remote) back()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  const inLists = $derived(app.lists.filter((l) => l.games.includes(gameId)).length)

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.confirm || app.prompt) return false
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  function edit() {
    onclose()
    go({ name: 'edit', id: gameId })
  }

  async function toggleHidden() {
    if (!g) return
    try {
      const updated = await api.setHidden(gameId, !g.hidden)
      putGame(updated)
      play('toggle')
      toast(updated.hidden ? 'Jogo oculto. Pra mostrar de novo: configurações › Biblioteca › Jogos ocultos.' : 'Jogo de volta na biblioteca', 'ok', 4000)
      onclose()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function remove() {
    if (!g) return
    try {
      const info = await api.gameSize(gameId)
      const after = info.server ? ' Ele continua no servidor e dá pra baixar de novo.' : ''
      const ok = await confirm(
        `Apagar "${g.name}" do PC? Libera ${bytes(info.size)}. Os saves e pontos salvos não são apagados.${after}`,
        'Apagar do PC',
        true,
      )
      if (!ok) return
      await api.deleteGame(gameId)
      toast(`"${g.name}" apagado do PC`, 'ok', 3000)
      play('accept')
      onclose()
      if (!info.server) {
        app.games = app.games.filter((x) => x.id !== gameId)
        back()
      }
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div
    class="dialog panel"
    in:panelIn
    out:panelOut
    role="dialog"
    aria-modal="true"
    aria-labelledby="options-title"
    data-nav-scope={app.confirm || app.prompt ? undefined : ''}
  >
    <h2 id="options-title">Opções</h2>
    <p class="name">{g?.name}</p>
    <div class="options">
      {#if quick && g}
        {#if g.remote}
          {#if !isKids()}<button class="item primary" data-nav data-nav-default onclick={() => (onclose(), downloadGame(g))}><Icon name="download" /> Baixar do servidor</button>{/if}
        {:else}
          <button class="item primary" data-nav data-nav-default onclick={() => launch()}><Icon name="play" /> Jogar</button>
          {#if canResume}<button class="item" data-nav onclick={() => launch('auto')}><Icon name="resume" /> Continuar de onde parei</button>{/if}
        {/if}
        <button class="item" data-nav onclick={() => (toggleFavorite(gameId), onclose())}><Icon name="star" /> {g.favorite ? 'Tirar dos favoritos' : 'Favoritar'}</button>
        <button class="item" data-nav onclick={details}><Icon name="disc" /> Informações do jogo</button>
        <hr aria-hidden="true" />
      {/if}
      <button class="item" data-nav data-nav-default={quick ? undefined : ''} aria-haspopup="dialog" onclick={onlists}>
        <Icon name="plus" /> Listas{inLists ? ` (${inLists})` : ''}
      </button>
      {#if canServerList}
        <button class="item" data-nav aria-haspopup="dialog" onclick={onserverlists}><Icon name="cloud" /> Listas do servidor</button>
      {/if}
      {#if !isKids()}
        <button class="item" data-nav onclick={edit}><Icon name="pencil" /> Editar informações</button>
        {#if configurable}
          <button class="item" data-nav aria-haspopup="dialog" onclick={onconfig}><Icon name="gear" /> Configurar este jogo</button>
        {/if}
        <button class="item" data-nav onclick={toggleHidden}>
          <Icon name={g?.hidden ? 'check' : 'close'} /> {g?.hidden ? 'Mostrar na biblioteca' : 'Ocultar da biblioteca'}
        </button>
        {#if canHide}
          <button class="item" data-nav onclick={hideOnServer}><Icon name="close" /> Esconder do servidor</button>
        {/if}
        {#if g?.local && !g.remote}
          <hr aria-hidden="true" />
          <button class="item danger" data-nav onclick={remove}><Icon name="trash" /> Apagar do PC</button>
        {/if}
      {/if}
    </div>
    <div class="foot">
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
  /* diálogo sóbrio com a lista de ações no jeito de menu suspenso */
  .dialog {
    display: grid;
    gap: 0.6rem;
    width: min(30rem, 100% - 2rem);
    padding: 1.25rem 1.25rem 1rem;
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
    margin-top: -0.3rem;
  }
  .options {
    display: grid;
    gap: 0.15rem;
    padding: 0.4rem;
    margin-inline: -0.4rem;
  }
  .item {
    display: flex;
    align-items: center;
    gap: 0.8rem;
    min-height: 3.1rem;
    padding: 0.5rem 0.9rem;
    font: inherit;
    font-size: 1.1rem;
    line-height: 1.3;
    text-align: start;
    color: var(--text);
    background: none;
    border: 0;
    border-radius: 4px;
    cursor: pointer;
  }
  .item :global(svg) {
    flex: none;
    width: 1.2rem;
    height: 1.2rem;
  }
  .item.primary {
    font-weight: 700;
    color: var(--accent);
  }
  .item.danger {
    color: var(--danger);
  }
  .item:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  /* item em destaque: o preenchimento mostarda É o foco (sem anel dentro do menu) */
  .item:focus {
    outline: none;
    background: var(--accent);
    color: var(--on-accent);
  }
  .item.danger:focus {
    background: var(--danger);
    color: var(--bg);
  }
  hr {
    width: 100%;
    border: none;
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
    margin: 0.3rem 0;
  }
  .foot {
    display: flex;
    justify-content: flex-end;
    padding-top: 0.9rem;
    border-top: var(--edge) solid var(--outline);
  }
</style>
