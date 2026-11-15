<script lang="ts">
  import { tick } from 'svelte'
  import Cheevos from '../components/Cheevos.svelte'
  import GameArt from '../components/GameArt.svelte'
  import GameConfig from '../components/GameConfig.svelte'
  import GameOptions from '../components/GameOptions.svelte'
  import ListPicker from '../components/ListPicker.svelte'
  import ArkListPicker from '../components/ArkListPicker.svelte'
  import MediaViewer from '../components/MediaViewer.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api, on } from '../lib/api'
  import { ago, bytes, dateTime, playTime } from '../lib/format'
  import { downloadGame, downloadText, downloads } from '../lib/downloads.svelte'
  import { pushHandler } from '../lib/input'
  import { burst } from '../lib/motion'
  import { current, focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { back, confirm, game, isKids, prompt, putGame, system, toast } from '../lib/state.svelte'
  import type { BiosInfo, GameEvent, KidsTimeInfo, Listing, MediaFile, MediaItem, SaveState } from '../lib/types'

  let { id }: { id: string } = $props()
  const g = $derived(game(id))
  const sys = $derived(g ? system(g.system) : undefined)
  let listing = $state<Listing | null>(null)
  let shots = $state<MediaFile[]>([])
  let note = $state('')
  let loadError = $state('')
  let stamp = $state(Date.now())
  let picking = $state(false)
  let serverPicking = $state(false)
  let menu = $state(false)
  let configuring = $state(false)
  let optionsBtn = $state<HTMLButtonElement>()
  let viewing = $state<number | null>(null)
  const media = $derived.by(() => {
    if (!g) return [] as MediaItem[]
    const out: MediaItem[] = []
    if (g.cover) out.push({ src: g.cover, caption: 'Capa', kind: 'image' })
    if (g.snap) out.push({ src: g.snap, caption: 'Tela do jogo', kind: 'image' })
    if (g.titleShot) out.push({ src: g.titleShot, caption: 'Tela-título', kind: 'image' })
    for (const st of listing?.states ?? []) {
      if (st.thumb) out.push({ src: `/media/thumb/${st.id}?v=${stamp}`, caption: `Ponto salvo: ${st.label}`, kind: 'image' })
    }
    for (const m of shots) {
      out.push({ src: `/media/shot/${m.id}`, caption: `${m.kind === 'video' ? 'Gravação' : 'Captura'} de ${dateTime(m.at)}`, kind: m.kind })
    }
    if (g.manual) out.push({ src: g.manual, caption: 'Manual', kind: 'pdf' })
    return out
  })
  // BIOS obrigatória do console faltando: um aviso só, acima dos botões
  let bios = $state<BiosInfo | null>(null)
  const orList = (xs: string[]) => (xs.length > 1 ? `${xs.slice(0, -1).join(', ')} ou ${xs[xs.length - 1]}` : (xs[0] ?? ''))
  $effect(() => {
    const slug = g?.system
    if (!slug) return
    api
      .biosStatus()
      .then((all) => (bios = all.find((b) => b.system === slug && b.required && !b.ok && !b.fallback) ?? null))
      .catch(() => (bios = null))
  })
  // perfil infantil com limite: sem tempo, o Jogar fica desligado com o motivo
  let kidsTime = $state<KidsTimeInfo | null>(null)
  const blocked = $derived(kidsTime?.reason ?? '')
  const loadKidsTime = () => {
    if (!isKids()) return
    api
      .kidsTime()
      .then((t) => {
        kidsTime = t
        if (t.reason) tick().then(focusDefault) // o Jogar desligou: foco vai pro próximo
      })
      .catch(() => {})
  }

  async function load() {
    try {
      listing = await api.saves(id)
      shots = await api.gameMedia(id).catch(() => [])
      note = await api.gameNote(id).catch(() => '')
      loadError = ''
      stamp = Date.now()
    } catch (err) {
      loadError = (err as Error).message
    }
  }

  $effect(() => {
    load()
    loadKidsTime()
    tick().then(focusDefault)
    // depois de jogar, recarrega (state novo, miniatura nova)
    return on<GameEvent>('game', (e) => {
      if (e.state === 'exited' && e.id === id) load()
      if (e.state === 'exited') loadKidsTime()
    })
  })

  const auto = $derived(listing?.states.find((s) => s.slot === -1))

  async function launch(state = '') {
    if (!g) return
    try {
      play('start')
      await api.play(g.id, state)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  async function editNote() {
    await prompt('Anotação deste jogo', note, 'Salvar', async (v) => {
      try {
        await api.setGameNote(id, v)
        note = v.trim()
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 2000, multiline: true })
  }

  async function toggleFav() {
    if (!g) return
    try {
      const from = current()
      const updated = await api.setFavorite(g.id, !g.favorite)
      putGame(updated)
      play('toggle')
      if (updated.favorite) burst(from)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function remove(st: SaveState) {
    const ok = await confirm(`Apagar o ponto salvo "${st.label}" de ${dateTime(st.modified)}? Não dá pra desfazer.`, 'Apagar', true)
    if (!ok) return
    try {
      listing = await api.deleteState(id, st.id)
      toast('Ponto salvo apagado', 'ok', 2500)
      tick().then(focusDefault)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() =>
    pushHandler((a) => {
      if (menu || picking || serverPicking || configuring || viewing !== null) return false
      if (a === 'fav') {
        const stateId = current()?.dataset.state
        const st = isKids() ? undefined : listing?.states.find((s) => s.id === stateId)
        if (st) remove(st)
        else toggleFav()
        return true
      }
      return a === 'details' || a === 'prevTab' || a === 'nextTab'
    }),
  )
</script>

{#if g}
  <div class="page">
    <div class="backdrop" style:background-image={g.cover ? `url("${g.cover}")` : 'none'} aria-hidden="true"></div>
    <TopBar />
    <main class="layout">
      <div class="art">
        <GameArt game={g} system={sys} large />
      </div>
      <div class="info">
        <p class="system">{sys?.name ?? g.system}</p>
        <h1>{g.name}</h1>
        <ul class="facts">
          {#if g.remote}<li class="chip ok">No servidor</li>{/if}
          {#if g.hidden}<li class="chip">Oculto</li>{/if}
          {#if g.rating}<li class="chip rating" aria-label={`Nota ${g.rating} de 5`}>{'★'.repeat(g.rating)}</li>{/if}
          {#if g.completed}<li class="chip ok">Completo</li>{/if}
          {#if g.broken}<li class="chip error">Quebrado</li>{/if}
          {#if g.year}<li class="chip">{g.year}</li>{/if}
          {#if g.developer}<li class="chip">{g.developer}</li>{/if}
          {#if g.players}<li class="chip">{g.players} jogador{g.players === '1' ? '' : 'es'}</li>{/if}
          {#each g.genres as genre (genre)}<li class="chip">{genre}</li>{/each}
          <li class="chip">{g.plays} partida{g.plays === 1 ? '' : 's'}</li>
          <li class="chip">Tempo: {playTime(g.playTime)}</li>
          <li class="chip">Última vez: {ago(g.lastPlayed)}</li>
          <li class="chip">{bytes(g.size)}</li>
        </ul>
        {#if g.summary}<p class="summary">{g.summary}</p>{/if}
        {#if bios}
          <p class="bios-warn" role="note">
            <Icon name="warn" />
            <span>Falta a BIOS do {bios.name} ({orList(bios.missing.map((m) => m.file))}). Suba no seu RomM (página do console › Firmware) que o app baixa sozinho, ou coloque em <span class="path">{bios.dir}</span></span>
          </p>
        {/if}
        <div class="actions">
          {#if g.remote}
            {#if downloads[g.id] !== undefined}
              <div class="downloading" role="status">
                <span><Icon name="download" /> {downloadText(g.id)}</span>
                <span class="energy" aria-hidden="true"><span style:width={`${Math.floor(downloads[g.id] * 20) * 5}%`}></span></span>
              </div>
            {:else if isKids()}
              <p class="muted">Este jogo está no servidor. Peça pra um adulto baixar.</p>
            {:else}
              <button class="btn primary" data-nav data-nav-default onclick={() => downloadGame(g)}>
                <Icon name="download" /> Baixar do servidor ({bytes(g.size)})
              </button>
            {/if}
          {:else}
            <!-- 🎬 bilhete: a ação da tela (contorno à mão só no papel, nunca no texto) -->
            <button class="ticket" data-nav data-nav-default={!blocked || undefined} disabled={!!blocked} aria-describedby={blocked ? 'kids-blocked' : undefined} onclick={() => launch()}>
              <span class="paper" aria-hidden="true"><span></span></span>
              <Icon name="play" /> Jogar do início
            </button>
          {/if}
          {#if auto}
            <button class="btn" data-nav disabled={!!blocked} onclick={() => launch('auto')}><Icon name="resume" /> Continuar de onde parei</button>
          {/if}
          <button class="btn" data-nav data-nav-default={blocked ? true : undefined} aria-pressed={g.favorite} onclick={toggleFav}>
            <Icon name="star" /> {g.favorite ? 'Favorito' : 'Favoritar'}
          </button>
          <button class="btn" data-nav aria-haspopup="dialog" bind:this={optionsBtn} onclick={() => (menu = true)}>
            <Icon name="sliders" /> Opções
          </button>
        </div>
        {#if blocked}<p class="blocked" id="kids-blocked" role="status">{blocked}</p>{/if}

        {#if media.length}
          <section class="media" aria-labelledby="media-title">
            <h2 id="media-title">Mídia</h2>
            <ul class="strip">
              {#each media as m, i (m.src)}
                <li>
                  <button class="thumb" data-nav aria-haspopup="dialog" onclick={() => (viewing = i)}>
                    {#if m.kind === 'pdf'}
                      <span class="nothumb" aria-hidden="true">PDF</span>
                    {:else if m.kind === 'video'}
                      <span class="nothumb" aria-hidden="true">VÍDEO</span>
                    {:else}
                      <img src={m.src} alt="" loading="lazy" />
                    {/if}
                    <span class="thumb-label"><span class="sr-only">Ver </span>{m.caption}</span>
                  </button>
                </li>
              {/each}
            </ul>
          </section>
        {/if}

        {#if !g.remote}<Cheevos {id} />{/if}

        {#if note}
          <section class="note-box" aria-labelledby="note-title">
            <h2 id="note-title">Sua anotação</h2>
            <p class="note-text">{note}</p>
            <button class="btn small" data-nav onclick={editNote}><Icon name="pencil" /> Editar anotação</button>
          </section>
        {/if}

        <section class="saves" aria-labelledby="saves-title">
          <h2 id="saves-title">Pontos salvos</h2>
          {#if loadError}
            <p class="error">{loadError}</p>
          {:else if listing}
            {#if listing.states.length === 0}
              <p class="muted">Nenhum ponto salvo ainda. No jogo, segure Select e aperte R pra salvar.</p>
            {:else}
              <ul class="states">
                {#each listing.states as st (st.id)}
                  <li>
                    <button
                      class="state"
                      data-nav
                      data-state={st.id}
                      title={`Salvo em ${dateTime(st.modified)}`}
                      onclick={() => launch(st.id)}
                    >
                      {#if st.thumb}
                        <img src={`/media/thumb/${st.id}?v=${stamp}`} alt="" loading="lazy" />
                      {:else}
                        <span class="nothumb" aria-hidden="true">{st.slot === -1 ? 'AUTO' : `#${st.slot}`}</span>
                      {/if}
                      <span class="state-label"><span class="sr-only">Carregar </span>{st.label}</span>
                      <span class="state-date">{ago(st.modified)}</span>
                    </button>
                  </li>
                {/each}
              </ul>
            {/if}
            {#if listing.files.length}
              <h3 class="sub">Saves do jogo</h3>
              <ul class="files">
                {#each listing.files as f (f.path)}
                  <li>
                    <Icon name="folder" />
                    <span>{f.name}</span>
                    <span class="meta muted">{bytes(f.size)} · {ago(f.modified)}{f.shared ? ' · compartilhado entre jogos' : ''}</span>
                  </li>
                {/each}
              </ul>
            {/if}
            {#if listing.note}<p class="muted note">{listing.note}</p>{/if}
          {/if}
        </section>
      </div>
    </main>
    <HelpBar
      hints={[
        { button: 'south', label: 'Escolher' },
        { button: 'north', label: isKids() ? 'Favorito' : 'Favorito / apagar save' },
        { button: 'east', label: 'Voltar' },
      ]}
    />
  </div>
  {#if menu}
    <GameOptions
      gameId={id}
      onclose={() => {
        menu = false
        tick().then(() => optionsBtn?.focus())
      }}
      onlists={() => {
        menu = false
        picking = true
      }}
      onconfig={() => {
        menu = false
        configuring = true
      }}
      onserverlists={() => {
        menu = false
        serverPicking = true
      }}
    />
  {/if}
  {#if serverPicking && g}
    <ArkListPicker
      kind="game"
      {id}
      name={g.name}
      onclose={() => {
        serverPicking = false
        tick().then(() => optionsBtn?.focus())
      }}
    />
  {/if}
  {#if configuring}
    <GameConfig
      gameId={id}
      onclose={() => {
        configuring = false
        tick().then(() => optionsBtn?.focus())
      }}
    />
  {/if}
  {#if picking}
    <ListPicker
      gameId={id}
      onclose={() => {
        picking = false
        tick().then(() => optionsBtn?.focus())
      }}
    />
  {/if}
  {#if viewing !== null && media.length}
    <MediaViewer
      items={media}
      start={Math.min(viewing, media.length - 1)}
      gameId={id}
      onclose={() => {
        viewing = null
        tick().then(focusDefault)
      }}
    />
  {/if}
{:else}
  <main class="layout">
    <p>Jogo não encontrado.</p>
    <button class="btn" data-nav data-nav-default onclick={() => back()}>Voltar</button>
  </main>
{/if}

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  /* capa do jogo ao fundo, desbotada em sépia: dá clima sem pintar a tela */
  .backdrop {
    position: fixed;
    inset: 0;
    background-size: cover;
    background-position: center;
    filter: blur(16px) grayscale(1) sepia(0.5);
    opacity: 0.1;
    z-index: -1;
  }
  .layout {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: grid;
    grid-template-columns: minmax(12rem, 22rem) 1fr;
    gap: 3rem;
    padding: 1rem 2.5rem 2rem 3.5rem;
    align-items: start;
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    min-width: 0;
  }
  .system {
    font-family: var(--font-hud);
    font-size: 0.9rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--muted);
  }
  h1 {
    font-size: clamp(2.6rem, 5vw, 4.2rem);
    margin-top: -0.6rem;
    overflow-wrap: anywhere;
  }
  .facts {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .chip.rating {
    color: var(--accent);
  }
  .summary {
    max-width: 48rem;
    font-size: 1.05rem;
  }

  /* aviso (alerta sóbrio): caixa de cartão com barra de informação */
  .bios-warn {
    display: flex;
    gap: 0.75rem;
    align-items: flex-start;
    max-width: 48rem;
    padding: 0.75rem 1rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-inline-start: 8px solid var(--accent-2);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    font-size: 0.95rem;
  }
  .bios-warn :global(svg) {
    color: var(--accent-2);
    flex: none;
    margin-top: 0.2rem;
  }
  .bios-warn .path {
    overflow-wrap: anywhere;
    font-family: var(--font-mono);
  }
  .blocked {
    font-weight: 700;
    color: var(--accent-2);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1rem 1.25rem;
  }
  .downloading {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    font-family: var(--font-hud);
    color: var(--text);
  }

  /* 🎬 Bilhete de cinema: retângulo com um entalhe em cada lado. O papel
     (mostarda) leva a máscara; o contorno à mão e a sombra dura moram no
     pai dele (a máscara cortaria os dois). O texto fica fora do filtro. */
  .ticket {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    min-height: 3.75rem;
    padding: 0 2.1rem;
    font-family: var(--font-label);
    font-size: 1.05rem;
    line-height: 1;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--on-accent);
    background: none;
    border: 0;
    border-radius: var(--radii-sm);
    cursor: pointer;
    transition: transform var(--t) var(--ease);
  }
  .ticket :global(svg) {
    width: 1.25rem;
    height: 1.25rem;
  }
  .paper {
    position: absolute;
    inset: 0;
    z-index: -1;
    filter: drop-shadow(3px 0 0 var(--outline)) drop-shadow(-3px 0 0 var(--outline)) drop-shadow(0 3px 0 var(--outline))
      drop-shadow(0 -3px 0 var(--outline)) url(#sketch) drop-shadow(3px 4px 0 var(--outline));
    transition: filter var(--t) var(--ease);
  }
  .paper > span {
    position: absolute;
    inset: 0;
    background: var(--accent);
    border-radius: var(--radii-sm);
    -webkit-mask:
      radial-gradient(circle at 0 50%, #0000 10px, #000 11px) left / 51% 100% no-repeat,
      radial-gradient(circle at 100% 50%, #0000 10px, #000 11px) right / 51% 100% no-repeat;
    mask:
      radial-gradient(circle at 0 50%, #0000 10px, #000 11px) left / 51% 100% no-repeat,
      radial-gradient(circle at 100% 50%, #0000 10px, #000 11px) right / 51% 100% no-repeat;
  }
  .ticket:hover:not(:disabled),
  .ticket:focus-visible:not(:disabled) {
    transform: translate(-2px, -2px);
  }
  .ticket:hover:not(:disabled) .paper,
  .ticket:focus-visible:not(:disabled) .paper {
    filter: drop-shadow(3px 0 0 var(--outline)) drop-shadow(-3px 0 0 var(--outline)) drop-shadow(0 3px 0 var(--outline))
      drop-shadow(0 -3px 0 var(--outline)) url(#sketch) drop-shadow(5px 6px 0 var(--outline));
  }
  .ticket:active:not(:disabled) {
    transform: translate(3px, 4px);
  }
  .ticket:active:not(:disabled) .paper {
    filter: drop-shadow(3px 0 0 var(--outline)) drop-shadow(-3px 0 0 var(--outline)) drop-shadow(0 3px 0 var(--outline))
      drop-shadow(0 -3px 0 var(--outline)) url(#sketch);
  }
  .ticket:disabled {
    color: var(--muted);
    cursor: not-allowed;
  }
  .ticket:disabled .paper {
    filter: drop-shadow(3px 0 0 var(--muted)) drop-shadow(-3px 0 0 var(--muted)) drop-shadow(0 3px 0 var(--muted))
      drop-shadow(0 -3px 0 var(--muted)) url(#sketch);
  }
  .ticket:disabled .paper > span {
    background: var(--surface);
  }
  :global(body.high-contrast) .paper {
    filter: drop-shadow(3px 0 0 var(--outline)) drop-shadow(-3px 0 0 var(--outline)) drop-shadow(0 3px 0 var(--outline))
      drop-shadow(0 -3px 0 var(--outline));
  }

  /* seções: título em Bevan sobre um fio reto */
  section h2 {
    font-size: 1.15rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    padding-top: 0.9rem;
    border-top: var(--edge) solid color-mix(in srgb, var(--outline) 35%, transparent);
  }
  .sub {
    font-size: 0.95rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-top: 0.5rem;
  }

  /* cartão de imagem (mídia e pontos salvos): imagem no topo, fio, legenda */
  .strip {
    list-style: none;
    padding: 0.4rem 0.6rem 0.8rem 0.4rem;
    margin: 0.6rem 0 0;
    display: flex;
    gap: 1.1rem;
    overflow-x: auto;
  }
  .states {
    list-style: none;
    padding: 0.4rem 0.6rem 0.8rem 0.4rem;
    margin: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
    gap: 1.1rem;
  }
  .strip li {
    display: flex;
  }
  .thumb {
    width: 10.5rem;
  }
  .state {
    width: 100%;
  }
  .thumb,
  .state {
    display: flex;
    flex-direction: column;
    padding: 0;
    font: inherit;
    color: var(--text);
    text-align: start;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    overflow: hidden;
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .thumb:hover,
  .thumb:focus-visible,
  .state:hover,
  .state:focus-visible {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .thumb:active,
  .state:active {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  .thumb img,
  .state img,
  .nothumb {
    display: block;
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    image-rendering: pixelated;
    background: var(--bg);
    border-bottom: var(--edge) solid var(--outline);
  }
  .nothumb {
    display: grid;
    place-items: center;
    font-family: var(--font-label);
    font-size: 1.4rem;
    letter-spacing: 0.06em;
    color: var(--accent);
  }
  .thumb-label,
  .state-label {
    padding: 0.5rem 0.7rem 0;
    font-weight: 700;
    line-height: 1.3;
  }
  .thumb-label {
    padding-bottom: 0.6rem;
  }
  .state-date {
    padding: 0.15rem 0.7rem 0.6rem;
    font-family: var(--font-hud);
    font-size: 0.8rem;
    color: var(--muted);
  }

  .saves {
    margin-top: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }
  .files {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.4rem;
    font-family: var(--font-mono);
    font-size: 0.9rem;
  }
  .files li {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .files .meta {
    font-size: 0.8rem;
  }
  .error {
    color: var(--danger);
  }
  .note {
    font-style: italic;
  }

  .note-box {
    display: grid;
    gap: 0.75rem;
    justify-items: start;
  }
  .note-text {
    white-space: pre-line;
    max-width: 48rem;
    padding: 0.75rem 1rem;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
  }
</style>
