<script lang="ts">
  // Criar ou editar perfil: nome, avatar, cor, PIN e apagar.
  import { tick } from 'svelte'
  import Accounts from '../components/Accounts.svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault, focusWithin } from '../lib/nav'
  import { play } from '../lib/sound'
  import { avatarName } from '../lib/avatars'
  import { app, askPIN, back, confirm, prompt, resetTo, toast } from '../lib/state.svelte'

  // setup: primeira vez do app (configurar o perfil principal antes de tudo)
  let { id, setup = false, suggest = '', embedded = false, onclose }: { id?: string; setup?: boolean; suggest?: string; embedded?: boolean; onclose?: () => void } = $props()

  const COLOR_NAMES = ['Vermelho', 'Laranja', 'Amarelo', 'Verde', 'Azul', 'Roxo', 'Rosa', 'Turquesa']

  const avatars = $derived(app.profile?.avatars ?? [])
  const colors = $derived(app.profile?.colors ?? [])
  const existing = $derived(id ? app.profile?.profiles.find((p) => p.id === id) : undefined)
  let name = $state('')
  let avatar = $state('')
  let color = $state('')
  let hasPIN = $state(false)
  let kids = $state(false)
  let daily = $state(0)
  let bedtime = $state('')
  const DAILY = [
    { v: 0, label: 'Sem limite' },
    { v: 30, label: '30 min' },
    { v: 60, label: '1 h' },
    { v: 120, label: '2 h' },
    { v: 180, label: '3 h' },
  ]
  const BEDTIMES = ['', '20:00', '21:00', '22:00']

  $effect(() => {
    const p = existing
    const used = new Set(app.profile?.profiles.map((x) => x.color))
    name = setup ? suggest : (p?.name ?? '')
    avatar = p?.avatar ?? avatars[(app.profile?.profiles.length ?? 0) % avatars.length] ?? 'knight'
    color = p?.color ?? colors.find((c) => !used.has(c)) ?? colors[0] ?? '#3a86d8'
    hasPIN = !!p?.hasPIN
    kids = !!p?.kids
    daily = p?.dailyMinutes ?? 0
    bedtime = p?.bedtime ?? ''
  })

  async function setLimits(minutes: number, bed: string) {
    if (!existing) return
    try {
      const p = await api.setProfileLimits(existing.id, minutes, bed)
      daily = p.dailyMinutes
      bedtime = p.bedtime
      await refresh()
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function toggleKids() {
    if (!existing) return
    try {
      const p = await api.setProfileKids(existing.id, !kids)
      kids = p.kids
      await refresh()
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() => {
    if (!embedded) tick().then(focusDefault)
  })

  $effect(() =>
    pushHandler((a) => {
      if (app.pin || app.prompt || app.confirm) return false
      if (setup && (a === 'back' || a === 'menu' || a === 'search')) return true // primeiro configura
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'random'
    }),
  )

  // embutida: concluir volta pra lista da tela Você; sozinha, volta de tela
  function close() {
    if (embedded) onclose?.()
    else back()
  }

  async function editName() {
    await prompt('Nome do perfil', name, 'Usar este nome', async (v) => {
      if (!v.trim()) return 'O perfil precisa de um nome'
      name = v.trim()
      return null
    }, { max: 20 })
    tick().then(() => (embedded ? focusWithin(root) : focusDefault()))
  }

  async function refresh() {
    app.profile = await api.profiles()
  }

  async function save() {
    try {
      if (existing) await api.updateProfile(existing.id, name, avatar, color)
      else await api.createProfile(name, avatar, color)
      await refresh()
      play('accept')
      if (setup) {
        toast(`Olá, ${name}!`, 'ok', 2500)
        resetTo({ name: 'home' })
        return
      }
      toast(existing ? 'Perfil salvo' : `Perfil "${name}" criado`, 'ok', 2500)
      close()
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  const check = (pid: string) => async (pin: string) => {
    try {
      await api.checkProfilePIN(pid, pin)
      return null
    } catch (err) {
      return (err as Error).message
    }
  }

  async function changePIN(remove: boolean) {
    if (!existing) return
    let current = ''
    if (hasPIN) {
      const got = await askPIN('PIN atual', check(existing.id))
      if (got === null) return
      current = got
    }
    let next = ''
    if (!remove) {
      const first = await askPIN('Novo PIN', async () => null)
      if (first === null) return
      const again = await askPIN('Repita o novo PIN', async (v) => (v === first ? null : 'Os PINs não batem'))
      if (again === null) return
      next = first
    }
    try {
      const p = await api.setProfilePIN(existing.id, current, next)
      hasPIN = p.hasPIN
      await refresh()
      toast(remove ? 'PIN removido' : 'PIN definido', 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
    tick().then(() => (embedded ? focusWithin(root) : focusDefault()))
  }

  async function remove() {
    if (!existing) return
    const ok = await confirm(
      `Apagar o perfil "${existing.name}"? Favoritos, listas, histórico e saves do RetroArch dele saem do app (a pasta fica guardada em profiles/apagados).`,
      'Apagar perfil',
      true,
    )
    if (!ok) return
    let pin = ''
    if (hasPIN) {
      const got = await askPIN(`PIN de ${existing.name}`, check(existing.id))
      if (got === null) return
      pin = got
    }
    try {
      await api.deleteProfile(existing.id, pin)
      await refresh()
      toast('Perfil apagado', 'ok', 2500)
      close()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
  let root = $state<HTMLElement | null>(null)
</script>

<div class="page" class:embedded bind:this={root}>
  {#if !embedded}<TopBar actions={false} />{/if}
  <main class="editor" aria-labelledby={embedded ? undefined : 'editor-title'}>
    <div class="preview">
      <Avatar {avatar} {color} size="9rem" variant="cameo" mark />
      <p class="plate">{name || 'Sem nome'}</p>
    </div>
    <div class="form">
      {#if !embedded}<h1 id="editor-title">{setup ? 'Boas-vindas! Monte seu perfil' : existing ? 'Editar perfil' : 'Novo perfil'}</h1>{/if}
      {#if setup}<p class="muted">Escolha seu nome e um personagem. Dá pra mudar depois e criar perfis pra mais gente.</p>{/if}
      <button class="field" data-nav data-nav-default onclick={editName}>
        <span class="field-label">Nome</span> <span class="value" class:empty={!name}>{name || 'Escolher nome'}</span>
      </button>

      <fieldset>
        <legend>Avatar</legend>
        <div class="choices" role="radiogroup" aria-label="Avatar">
          {#each avatars as a (a)}
            <button class="choice" data-nav role="radio" aria-checked={avatar === a} aria-label={avatarName(a)} onclick={() => (avatar = a)}>
              <Avatar avatar={a} size="3.4rem" />
              {#if avatar === a}<span class="tick" aria-hidden="true">✓</span>{/if}
            </button>
          {/each}
        </div>
      </fieldset>

      <fieldset>
        <legend>Cor</legend>
        <p class="muted hint">Só uma marquinha no canto do avatar, pra achar o seu perfil rápido.</p>
        <div class="choices" role="radiogroup" aria-label="Cor">
          {#each colors as c, i (c)}
            <button class="choice swatch" data-nav role="radio" aria-checked={color === c} aria-label={COLOR_NAMES[i] ?? c} onclick={() => (color = c)}>
              <span class="ink" style:background={c}></span>
              {#if color === c}<span class="tick" aria-hidden="true">✓</span>{/if}
            </button>
          {/each}
        </div>
      </fieldset>

      {#if existing && !setup}
        <fieldset>
          <legend>PIN</legend>
          <p class="muted">Com PIN, só quem sabe os 4 números entra neste perfil.</p>
          <div class="row">
            <button class="btn" data-nav onclick={() => changePIN(false)}><Icon name="lock" /> {hasPIN ? 'Trocar PIN' : 'Definir PIN'}</button>
            {#if hasPIN}<button class="btn" data-nav onclick={() => changePIN(true)}>Remover PIN</button>{/if}
          </div>
        </fieldset>
      {/if}

      {#if existing && !setup}<Accounts profile={existing.id} />{/if}

      {#if existing && !existing.main && existing.id !== app.profile?.current}
        <fieldset>
          <legend>Infantil</legend>
          <button class="switch" data-nav role="switch" aria-checked={kids} aria-describedby="kids-help" onclick={toggleKids}>
            <span class="knob" aria-hidden="true"></span>
            <span>Perfil infantil</span>
          </button>
          <p class="muted" id="kids-help">
            Só jogos liberados (no editor de cada jogo), botões maiores e nada de configurações, importar, apagar saves
            ou editar. Pra sair, o PIN do perfil adulto (defina um).
          </p>
          {#if kids}
            <p class="limit-label" id="daily-label">Tempo por dia</p>
            <div class="segmented" role="group" aria-labelledby="daily-label">
              {#each DAILY as d (d.v)}
                <button class="btn small" data-nav aria-pressed={daily === d.v} onclick={() => setLimits(d.v, bedtime)}>{d.label}</button>
              {/each}
            </div>
            <p class="limit-label" id="bed-label">Hora de dormir</p>
            <div class="segmented" role="group" aria-labelledby="bed-label">
              {#each BEDTIMES as b (b)}
                <button class="btn small" data-nav aria-pressed={bedtime === b} onclick={() => setLimits(daily, b)}>{b || 'Sem'}</button>
              {/each}
            </div>
          {/if}
        </fieldset>
      {/if}

      <div class="row actions">
        <button class="btn primary" data-nav disabled={!name} onclick={save}><Icon name="check" /> {setup ? 'Começar' : existing ? 'Salvar' : 'Criar perfil'}</button>
        {#if existing && !existing.main && existing.id !== app.profile?.current}
          <button class="btn danger" data-nav onclick={remove}><Icon name="trash" /> Apagar perfil</button>
        {/if}
      </div>
    </div>
  </main>
  {#if !embedded}<HelpBar hints={setup ? [{ button: 'south', label: 'Escolher' }] : [{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />{/if}
</div>

<style>
  /* embutida na tela Você: sem barra, sem rodapé, sem rolagem própria */
  .page.embedded {
    height: auto;
    display: block;
  }
  .page.embedded > main {
    padding: 0;
    overflow: visible;
  }

  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .editor {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: grid;
    grid-template-columns: minmax(12rem, 16rem) 1fr;
    gap: 3rem;
    padding: 1rem 3rem 2rem;
    align-items: start;
  }
  /* 🎬 retrato único da tela (cameo) com a plaquinha do nome embaixo */
  .preview {
    position: sticky;
    top: 0;
    display: grid;
    justify-items: center;
    gap: 1.25rem;
    padding-top: 2.5rem;
  }
  .plate {
    max-width: 100%;
    padding: 0.3rem 0.8rem 0.15rem;
    font-family: var(--font-hud);
    font-size: 0.95rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    text-align: center;
    overflow-wrap: anywhere;
    color: var(--bg);
    background: var(--text);
  }
  .form {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    max-width: 46rem;
  }
  h1 {
    font-size: clamp(2.2rem, 4vw, 3.4rem);
  }

  /* campo: rótulo em Bevan em cima, valor numa caixa de texto */
  .field {
    display: grid;
    gap: 0.35rem;
    width: min(100%, 24rem);
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
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  .field:hover .value,
  .field:focus-visible .value {
    border-color: var(--focus);
    box-shadow: 4px 4px 0 var(--focus);
  }
  .value.empty {
    color: var(--muted);
    font-style: italic;
    font-weight: 400;
  }

  /* grupo: sem caixa, régua de 3px em cima, legenda em Bevan */
  fieldset {
    border: none;
    border-top: var(--edge) solid var(--outline);
    padding: 1rem 0 0;
    margin: 0;
    display: grid;
    gap: 0.75rem;
  }
  legend {
    float: left;
    width: 100%;
    font-family: var(--font-label);
    font-size: 0.9rem;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .hint {
    font-size: 0.9rem;
  }
  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding: 0.3rem;
  }
  /* escolha: a moldura sobe no foco; a escolhida afunda e ganha ✓ */
  .choice {
    position: relative;
    padding: 0;
    background: none;
    border: none;
    border-radius: var(--radii-sm);
    cursor: pointer;
    transition: transform var(--t) var(--ease);
  }
  .choice:hover,
  .choice:focus-visible {
    transform: translate(-2px, -2px);
  }
  .choice[aria-checked='true'] {
    transform: translate(3px, 3px);
  }
  .choice[aria-checked='true'] :global(.avatar) {
    box-shadow: none;
    background: var(--accent-2);
  }
  .tick {
    position: absolute;
    inset-block-start: -0.5rem;
    inset-inline-end: -0.5rem;
    display: grid;
    place-items: center;
    width: 1.4rem;
    height: 1.4rem;
    font-family: var(--font-label);
    font-size: 0.8rem;
    color: var(--bg);
    background: var(--accent-2);
    border: 2px solid var(--outline);
    border-radius: 4px;
  }
  .ink {
    display: block;
    width: 2.8rem;
    height: 2.8rem;
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  .swatch[aria-checked='true'] .ink {
    box-shadow: none;
  }

  /* interruptor: trilho de 3px, botão que pula pro lado; ligado = verde-azulado */
  .switch {
    display: flex;
    gap: 0.9rem;
    align-items: center;
    width: fit-content;
    min-height: 3rem;
    padding: 0.3rem 0.5rem;
    font: inherit;
    font-weight: 700;
    color: var(--text);
    background: none;
    border: none;
    border-radius: var(--radii-sm);
    cursor: pointer;
  }
  .knob {
    position: relative;
    flex: none;
    width: 3.4rem;
    height: 1.8rem;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 999px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .knob::after {
    content: '';
    position: absolute;
    top: 50%;
    inset-inline-start: 0.2rem;
    width: 1.05rem;
    height: 1.05rem;
    translate: 0 -50%;
    background: var(--outline);
    border-radius: var(--radii-sm);
    transition: inset-inline-start 150ms steps(3);
  }
  .switch[aria-checked='true'] .knob {
    background: var(--accent-2);
  }
  .switch[aria-checked='true'] .knob::after {
    inset-inline-start: calc(100% - 1.25rem);
    background: var(--bg);
  }
  .switch:hover .knob {
    box-shadow: 3px 3px 0 var(--outline);
  }

  .limit-label {
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    margin: 0.4rem 0 0;
  }
  /* grupo de alternância numa peça só (toggle group da noemi-ui) */
  .segmented {
    display: flex;
    flex-wrap: wrap;
    width: fit-content;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    overflow: hidden;
  }
  .segmented .btn {
    border: none;
    border-inline-end: var(--edge) solid var(--outline);
    border-radius: 0;
    box-shadow: none;
    transform: none;
    background: none;
  }
  .segmented .btn:last-child {
    border-inline-end: none;
  }
  .segmented .btn:hover {
    background: color-mix(in srgb, var(--outline) 12%, var(--surface));
  }
  .segmented .btn[aria-pressed='true'] {
    background: var(--accent-2);
    color: var(--bg);
    box-shadow: inset 3px 3px 0 color-mix(in srgb, var(--outline) 45%, transparent);
  }
  .segmented .btn[data-nav]:focus-visible {
    outline-offset: -7px;
  }
  .segmented .btn[aria-pressed='true'][data-nav]:focus-visible {
    outline-color: var(--bg);
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
  }
  .actions {
    padding-top: 1.25rem;
    border-top: var(--edge) solid var(--outline);
  }
  @media (max-width: 760px) {
    .editor {
      grid-template-columns: 1fr;
    }
    .preview {
      position: static;
    }
  }
</style>
