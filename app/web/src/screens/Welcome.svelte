<script lang="ts">
  // Primeira vez do app: entrar no RomM (jogos), no Jellyfin (filmes e
  // séries) ou no Navidrome (músicas), ou usar offline. Depois vem a configuração do perfil. A senha só
  // vai pro login (o app guarda um token); o outro servidor liga depois.
  import { tick } from 'svelte'
  import Icon from '../components/Icon.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, prompt, resetTo } from '../lib/state.svelte'

  // o elenco: uma fileira parada de personagens em cima do cartão
  const FACES = ['mario', 'link', 'samus', 'sonic', 'kirby', 'pikachu']

  type Kind = 'romm' | 'jellyfin' | 'navidrome'
  const KINDS: Record<Kind, { label: string; server: string; form: string; host: string; icon: 'cloud' | 'film' | 'music' }> = {
    romm: { label: 'Jogos', server: 'RomM', form: 'Entrar no servidor RomM', host: 'meu-romm.duckdns.org', icon: 'cloud' },
    jellyfin: { label: 'Filmes e séries', server: 'Jellyfin', form: 'Entrar no Jellyfin', host: 'meu-jellyfin.duckdns.org', icon: 'film' },
    navidrome: { label: 'Músicas', server: 'Navidrome', form: 'Entrar no Navidrome', host: 'meu-navidrome.duckdns.org', icon: 'music' },
  }
  let kind = $state<Kind>('romm')
  const k = $derived(KINDS[kind])

  let host = $state('')
  let user = $state('')
  let pass = $state('')
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    tick().then(focusDefault)
  })

  // aqui não tem "voltar": escolhe servidor ou offline
  $effect(() =>
    pushHandler((a) => {
      if (app.prompt) return false
      return a === 'back' || a === 'menu' || a === 'search' || a === 'random' || a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab'
    }),
  )

  async function ask(title: string, value: string, secret = false): Promise<string | null> {
    let got: string | null = null
    await prompt(title, value, 'OK', async (v) => {
      got = secret ? v : v.trim()
      return null
    }, { max: secret ? 200 : 120, secret })
    tick().then(focusDefault)
    return got
  }

  function next(suggest = '') {
    resetTo({ name: 'profile', id: 'main', setup: true, suggest })
  }

  async function login() {
    if (!host || !user || !pass) {
      error = 'Preencha endereço, usuário e senha'
      play('error')
      return
    }
    busy = true
    error = ''
    try {
      let who: string
      if (kind === 'romm') {
        who = (await api.connectRemote(host, user, pass)).user
      } else if (kind === 'navidrome') {
        app.music = await api.connectMusic('main', host, user, pass)
        who = app.music.user
      } else {
        app.watch = await api.connectWatch('main', host, user, pass)
        who = app.watch.user
      }
      pass = ''
      play('accept')
      const name = who.charAt(0).toUpperCase() + who.slice(1)
      next(name)
    } catch (err) {
      error = (err as Error).message
      play('error')
    } finally {
      busy = false
    }
  }
</script>

<div class="welcome">
  <main class="card panel sketched" aria-labelledby="welcome-title">
    <span class="stamp" aria-hidden="true">Estreia</span>
    <p class="logo" aria-hidden="true">Arkflix</p>
    <ul class="cast" aria-hidden="true">
      {#each FACES as f (f)}
        <li><img src={`/avatars/${f}.png`} alt="" draggable="false" /></li>
      {/each}
    </ul>
    <h1 id="welcome-title">Bem-vindo!</h1>
    <p class="lead">Entre no RomM pra ver os jogos e guardar os saves na nuvem, no Jellyfin pra assistir filmes e séries ou no Navidrome pra ouvir música. Ou use só o que está neste computador.</p>

    <div class="kinds" role="radiogroup" aria-label="Entrar em">
      {#each Object.entries(KINDS) as [id, info] (id)}
        <button
          type="button"
          class="kind"
          role="radio"
          data-nav
          aria-checked={kind === id}
          onclick={() => {
            kind = id as Kind
            error = ''
          }}
        >
          <span class="kind-name"><Icon name={info.icon} /> {info.label}</span>
          <span class="kind-server">{info.server}</span>
        </button>
      {/each}
    </div>

    <form
      class="form"
      aria-label={k.form}
      onsubmit={(e) => {
        e.preventDefault()
        login()
      }}
    >
      <button type="button" class="field" data-nav data-nav-default onclick={async () => (host = (await ask('Endereço do servidor', host)) ?? host)}>
        <span class="field-label">Servidor</span>
        <span class="value" class:empty={!host}>{host || k.host}</span>
      </button>
      <button type="button" class="field" data-nav onclick={async () => (user = (await ask(`Usuário do ${k.server}`, user)) ?? user)}>
        <span class="field-label">Usuário</span>
        <span class="value" class:empty={!user}>{user || 'seu usuário'}</span>
      </button>
      <button type="button" class="field" data-nav onclick={async () => (pass = (await ask(`Senha do ${k.server}`, '', true)) ?? pass)}>
        <span class="field-label">Senha</span>
        <span class="value" class:empty={!pass}>{pass ? '•'.repeat(Math.min(pass.length, 12)) : 'sua senha'}</span>
      </button>
      {#if error}<p class="error" role="alert"><span class="bang" aria-hidden="true">!</span>{error}</p>{/if}
      <button type="submit" class="btn primary wide" data-nav disabled={busy}>
        <Icon name="cloud" /> {busy ? 'Entrando…' : 'Entrar com o servidor'}
      </button>
    </form>

    <div class="or" aria-hidden="true"><span>ou</span></div>
    <button class="btn wide" data-nav onclick={() => next()}><Icon name="play" /> Jogar offline</button>
    <p class="muted small">O outro servidor liga depois: RomM em Configurações › Servidor, Jellyfin no perfil › Contas.</p>
  </main>
</div>

<style>
  .welcome {
    position: fixed;
    inset: 0;
    display: grid;
    place-items: center;
    overflow-y: auto;
    padding: 1.5rem 1rem;
  }
  /* 🎬 cartão de contorno à mão (1º tipo de desenho) + carimbo (2º) */
  .card {
    position: relative;
    width: min(32rem, calc(100vw - 2rem));
    display: grid;
    gap: 1rem;
    padding: 1.75rem 2rem 1.5rem;
  }
  .stamp {
    position: absolute;
    inset-block-start: 1.1rem;
    inset-inline-end: 1.1rem;
    transform: rotate(6deg);
    color: var(--accent);
    border-color: var(--accent);
  }
  .logo {
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.3em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .cast {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    gap: 0.5rem;
  }
  .cast li {
    display: grid;
    place-items: center;
    width: 2.6rem;
    height: 2.6rem;
    background: var(--balloon);
    border: 2px solid var(--outline);
    border-radius: var(--radii-sm);
  }
  .cast img {
    width: 80%;
    height: 80%;
    object-fit: contain;
    image-rendering: pixelated;
  }
  h1 {
    font-size: clamp(2.6rem, 6vw, 3.6rem);
  }
  .lead {
    color: var(--muted);
  }

  /* grupo de alternância numa peça só: o escolhido fica verde-azulado */
  .kinds {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  .kind {
    display: grid;
    justify-items: center;
    gap: 0.2rem;
    min-height: 3.6rem;
    padding: 0.6rem;
    font: inherit;
    color: var(--text);
    background: none;
    border: none;
    border-radius: 0;
    cursor: pointer;
    transition: background var(--t) var(--ease);
  }
  .kind:not(:last-child) {
    border-inline-end: var(--edge) solid var(--outline);
  }
  .kind:first-child {
    border-start-start-radius: 6px;
    border-end-start-radius: 6px;
  }
  .kind:last-child {
    border-start-end-radius: 6px;
    border-end-end-radius: 6px;
  }
  .kind:hover {
    background: color-mix(in srgb, var(--outline) 12%, var(--surface));
  }
  .kind[aria-checked='true'] {
    background: var(--accent-2);
    color: var(--bg);
    box-shadow: inset 3px 3px 0 color-mix(in srgb, var(--outline) 45%, transparent);
  }
  .kind[data-nav]:focus-visible {
    outline-offset: -7px;
  }
  .kind[aria-checked='true'][data-nav]:focus-visible {
    outline-color: var(--bg);
  }
  .kind-name {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .kind-server {
    font-family: var(--font-hud);
    font-size: 0.8rem;
    letter-spacing: 0.04em;
  }

  .form {
    display: grid;
    gap: 0.9rem;
  }
  /* campo: rótulo em Bevan em cima, valor numa caixa de texto */
  .field {
    display: grid;
    gap: 0.35rem;
    padding: 0;
    font: inherit;
    color: var(--text);
    text-align: start;
    background: none;
    border: none;
    cursor: text;
  }
  .field-label {
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .value {
    display: block;
    min-height: 2.9rem;
    padding: 0.6rem 0.75rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    transition:
      border-color var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .field:hover .value,
  .field:focus-visible .value {
    border-color: var(--focus);
    box-shadow: 4px 4px 0 var(--focus);
  }
  .value.empty {
    color: var(--muted);
    font-style: italic;
  }
  .wide {
    width: 100%;
    justify-content: center;
  }
  .error {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    color: var(--danger);
  }
  .bang {
    flex: none;
    display: grid;
    place-items: center;
    width: 1.1rem;
    height: 1.1rem;
    font-family: var(--font-label);
    font-size: 0.75rem;
    color: var(--bg);
    background: var(--danger);
  }
  .or {
    display: flex;
    align-items: center;
    gap: 0.8rem;
    font-family: var(--font-hud);
    color: var(--muted);
  }
  .or::before,
  .or::after {
    content: '';
    flex: 1;
    border-top: var(--edge) solid var(--outline);
  }
  .small {
    font-size: 0.85rem;
    text-align: center;
  }
</style>
