<script lang="ts">
  // Guia rápido (botão Home / G): painel lateral por cima de qualquer tela.
  // Com jogo aberto: voltar ao jogo, salvar/carregar estado, fechar o jogo.
  // Sempre: atalhos (início, central, perfil, busca, você, energia) e som.
  import { tick } from 'svelte'
  import { activity, unread } from '../lib/activity.svelte'
  import { api } from '../lib/api'
  import { closeGuide } from '../lib/guide'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, confirm, currentProfile, game, go, isKids, prompt, resetTo, screen, toast } from '../lib/state.svelte'
  import type { GuideInfo } from '../lib/types'
  import { saveSoon } from '../screens/settings/save.svelte'
  import Avatar from './Avatar.svelte'
  import Icon, { type IconName } from './Icon.svelte'

  let info = $state<GuideInfo>({ game: '', commands: false, started: 0, recording: false, note: '' })
  let previous: HTMLElement | null = null
  const me = $derived(currentProfile())
  const playing = $derived(app.playing?.state === 'running' ? app.playing : null)
  const current = $derived(playing ? game(playing.id) : undefined)
  const busy = $derived(activity.tasks.filter((t) => t.state === 'running'))
  const news = $derived(unread())
  let now = $state(new Date())

  $effect(() => {
    if (!app.guide) return
    previous = document.activeElement as HTMLElement | null
    now = new Date()
    api.guide().then((g) => (info = g)).catch(() => {})
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.confirm || app.prompt) return false
      if (a === 'back' || a === 'guide') {
        play('back')
        close()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  function close() {
    closeGuide()
    previous?.focus()
  }

  // atalho: fecha o guia e vai pra tela (sem jogo aberto)
  function open(fn: () => void) {
    app.guide = false
    fn()
  }

  type Cmd = 'save' | 'load' | 'quit' | 'fps' | 'screenshot' | 'record'
  const done: Partial<Record<Cmd, string>> = { save: 'Estado salvo', load: 'Estado carregado', screenshot: 'Captura salva: aparece na página do jogo' }

  async function command(cmd: Cmd) {
    if (cmd === 'quit' && !(await confirm(`Fechar ${playing?.name || 'o jogo'}? O que não foi salvo se perde.`, 'Fechar jogo', true))) return
    try {
      await api.gameCommand(cmd)
      if (done[cmd]) toast(done[cmd]!, 'ok', 2500)
      if (cmd === 'record') toast(info.recording ? 'Gravação parada: aparece na página do jogo' : 'Gravando… pare pelo guia', 'ok', 3000)
      if (cmd === 'screenshot') info = await api.guide()
      else app.guide = false // o resto devolve a tela pro jogo
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  async function editNote() {
    if (!info.game) return
    await prompt('Anotação deste jogo', info.note, 'Salvar', async (v) => {
      try {
        await api.setGameNote(info.game, v)
        info.note = v.trim()
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 2000, multiline: true })
  }

  // retomada rápida: outros jogos recentes, cada um de onde parou
  const recent = $derived(
    playing
      ? app.games
          .filter((g) => g.lastPlayed > 0 && !g.remote && g.id !== playing.id)
          .sort((a, b) => b.lastPlayed - a.lastPlayed)
          .slice(0, 3)
      : [],
  )

  async function switchTo(id: string) {
    try {
      await api.switchGame(id)
      app.guide = false
      toast(`Trocando pra ${game(id)?.name ?? 'outro jogo'}…`, 'info', 2500)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  const minutes = $derived(info.started ? Math.max(0, Math.floor((now.getTime() / 1000 - info.started) / 60)) : 0)

  function toggle(key: 'music' | 'sounds') {
    if (!app.settings) return
    saveSoon({ ...app.settings, a11y: { ...app.settings.a11y, [key]: !app.settings.a11y[key] } })
  }

  type Tile = { id: string; label: string; icon: IconName; badge?: string; run: () => void }
  const tiles = $derived.by((): Tile[] => {
    if (playing) return []
    const out: Tile[] = [
      { id: 'home', label: 'Início', icon: 'home', run: () => open(() => resetTo({ name: 'home' })) },
      { id: 'activity', label: 'Central', icon: 'bell', badge: busy.length ? `${busy.length} baixando` : news ? `${news} novo${news === 1 ? '' : 's'}` : '', run: () => open(() => screen().name !== 'activity' && go({ name: 'activity' })) },
      { id: 'search', label: 'Buscar', icon: 'search', run: () => open(() => screen().name !== 'search' && go({ name: 'search' })) },
      { id: 'you', label: 'Você', icon: 'gear', run: () => open(() => screen().name !== 'you' && go({ name: 'you' })) },
      { id: 'profiles', label: 'Trocar perfil', icon: 'resume', run: () => open(() => go({ name: 'profiles' })) },
    ]
    if (app.mode === 'desktop') out.push({ id: 'power', label: 'Energia', icon: 'power', run: () => open(() => (app.power = true)) })
    return out
  })
</script>

{#if app.guide}
  <div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    <div class="guide" role="dialog" aria-modal="true" aria-labelledby="guide-title" data-nav-scope={app.confirm || app.prompt ? undefined : ''}>
      <header>
        {#if me}<Avatar avatar={me.avatar} color={me.color} size="3rem" />{/if}
        <div>
          <h2 id="guide-title">{me?.name ?? 'Guia'}</h2>
          <p class="muted meta">{now.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })} · {app.pads.length ? app.pads.map((p) => p.name).join(', ') : 'sem controle'}</p>
        </div>
      </header>

      {#if playing}
        <section aria-label="Jogo aberto" class="game">
          {#if current?.cover}<img src={current.cover} alt="" />{/if}
          <p class="now label">Jogando</p>
          <p class="name">{playing.name || current?.name}</p>
          {#if info.started}<p class="muted meta session">Jogando há {minutes < 60 ? `${minutes} min` : `${Math.floor(minutes / 60)} h ${minutes % 60} min`}{info.recording ? ' · ● gravando' : ''}</p>{/if}
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default onclick={close}><Icon name="play" /> Voltar ao jogo</button>
            {#if info.commands}
              <div class="tools">
                <button class="btn small" data-nav onclick={() => command('save')}><Icon name="save" /> Salvar estado</button>
                <button class="btn small" data-nav onclick={() => command('load')}><Icon name="resume" /> Carregar estado</button>
                <button class="btn small" data-nav onclick={() => command('screenshot')}><Icon name="star" /> Capturar tela</button>
                <button class="btn small" data-nav aria-pressed={info.recording} onclick={() => command('record')}><Icon name="disc" /> {info.recording ? 'Parar gravação' : 'Gravar vídeo'}</button>
                <button class="btn small" data-nav onclick={() => command('fps')}><Icon name="chart" /> Mostrar FPS</button>
                <button class="btn small" data-nav onclick={editNote}><Icon name="pencil" /> Anotação</button>
              </div>
            {:else}
              <button class="btn small" data-nav onclick={editNote}><Icon name="pencil" /> Anotação</button>
            {/if}
            {#if info.note}<p class="note">{info.note}</p>{/if}
            <button class="btn danger" data-nav onclick={() => command('quit')}><Icon name="close" /> Fechar jogo</button>
          </div>
        </section>
        {#if recent.length}
          <section aria-labelledby="switch-title" class="switch">
            <h3 id="switch-title" class="label">Trocar de jogo</h3>
            {#each recent as g (g.id)}
              <button class="tile row" data-nav onclick={() => switchTo(g.id)}>
                {#if g.cover}<img src={g.cover} alt="" />{/if}
                <span>{g.name}</span>
              </button>
            {/each}
          </section>
        {/if}
      {:else}
        <nav aria-label="Atalhos" class="tiles">
          {#each tiles as t, i (t.id)}
            <button class="tile" data-nav data-nav-default={i === 0 ? '' : undefined} onclick={t.run}>
              <Icon name={t.icon} size="1.5em" />
              <span>{t.label}</span>
              {#if t.badge}<span class="chip warn">{t.badge}</span>{/if}
            </button>
          {/each}
        </nav>
      {/if}

      {#if busy.length}
        <section aria-label="Em andamento" class="busy">
          {#each busy.slice(0, 2) as t (t.id)}
            <p class="muted meta">{t.label} · {Math.round(t.progress * 100)}%</p>
            <div class="energy" role="progressbar" aria-label={t.label} aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(t.progress * 100)}>
              <span style:width={`${Math.floor(t.progress * 20) * 5}%`}></span>
            </div>
          {/each}
        </section>
      {/if}

      {#if app.settings && !isKids()}
        <section aria-label="Som" class="quick">
          <button class="btn small" data-nav role="switch" aria-checked={app.settings.a11y.music} onclick={() => toggle('music')}>Música</button>
          <button class="btn small" data-nav role="switch" aria-checked={app.settings.a11y.sounds} onclick={() => toggle('sounds')}>Sons do menu</button>
        </section>
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Guia = folha lateral (drawer da noemi-ui) colada na borda esquerda:
     fundo de cartão, linha de 3px só na borda de dentro, sem sombra.
     Entra da esquerda em 4 degraus. */
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 95;
    /* tinta (#24170f, o contorno do tema claro) a 55%, sem blur: o creme
       do contorno escuro lavaria a tela */
    background: color-mix(in srgb, var(--on-accent) 55%, transparent);
  }
  .guide {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    width: min(27rem, 100% - 2rem);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1.3rem;
    padding: 1.4rem 1.4rem 1.6rem;
    background: var(--surface);
    border-inline-end: var(--edge) solid var(--outline);
    animation: guide-in 200ms steps(4) both;
  }
  @keyframes guide-in {
    from {
      transform: translateX(-100%);
    }
  }
  :global(.reduce-motion) .guide {
    animation: none;
  }
  header {
    display: flex;
    gap: 0.9rem;
    align-items: center;
    padding-block-end: 1.1rem;
    border-block-end: var(--edge) solid var(--outline);
  }
  h2 {
    font-size: 1.2rem;
  }
  header p {
    margin: 0.2rem 0 0;
    font-size: 0.85rem;
  }
  .game {
    display: grid;
    gap: 0.6rem;
  }
  .game img {
    width: 100%;
    max-height: 9rem;
    object-fit: cover;
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
  }
  .now {
    color: var(--accent);
  }
  .name {
    font-weight: 700;
    font-size: 1.25rem;
    line-height: 1.25;
  }
  .actions,
  .tiles {
    display: grid;
    gap: 0.9rem;
  }
  .actions .btn {
    justify-content: flex-start;
  }
  .tiles {
    grid-template-columns: 1fr 1fr;
  }
  /* atalho: cartão sóbrio, sobe no foco como os botões */
  .tile {
    display: grid;
    justify-items: start;
    align-content: start;
    gap: 0.45rem;
    padding: 0.9rem;
    font: inherit;
    font-weight: 700;
    color: var(--text);
    text-align: start;
    background: var(--bg);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .tile:hover,
  .tile:focus-visible {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard);
  }
  .tile:active {
    transform: translate(3px, 3px);
    box-shadow: none;
  }
  .tile :global(svg) {
    color: var(--accent);
  }
  .busy p {
    margin: 0.4rem 0 0.3rem;
    font-size: 0.85rem;
  }
  .busy .energy {
    width: 100%;
  }
  .session {
    font-size: 0.85rem;
  }
  .tools {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }
  .tools .btn {
    justify-content: flex-start;
  }
  .note {
    font-size: 0.95rem;
    white-space: pre-line;
    padding: 0.5rem 0.75rem;
    background: var(--balloon);
    border: 2px solid var(--outline);
    border-inline-start-width: 8px;
    border-inline-start-color: var(--accent-2);
    border-radius: var(--radii-sm);
  }
  .switch {
    display: grid;
    gap: 0.6rem;
  }
  .switch h3 {
    color: var(--muted);
  }
  .tile.row {
    grid-template-columns: auto 1fr;
    align-items: center;
    padding: 0.4rem 0.6rem;
  }
  .tile.row img {
    width: 2.6rem;
    height: 2.6rem;
    object-fit: cover;
    border: 2px solid var(--outline);
    border-radius: 4px;
  }
  .quick {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
    margin-top: auto;
    padding-block-start: 1.1rem;
    border-block-start: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
</style>
