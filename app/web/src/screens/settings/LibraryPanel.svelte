<script lang="ts">
  import Icon from '../../components/Icon.svelte'
  import { activity } from '../../lib/activity.svelte'
  import { api } from '../../lib/api'
  import { play } from '../../lib/sound'
  import { HIDDEN } from '../../lib/sources'
  import { app, confirm, go, toast } from '../../lib/state.svelte'
  import type { EmulatorPaths, SaveDirs, Settings } from '../../lib/types'

  let draft = $state<Settings>($state.snapshot(app.settings!) as Settings)
  let busy = $state(false)
  const hiddenCount = $derived(app.games.filter((g) => g.hidden).length)
  const dirty = $derived(JSON.stringify(draft) !== JSON.stringify(app.settings))

  function addDir() {
    draft.libraryDirs.push({ path: '', label: 'Nova pasta', writable: false })
  }

  async function save() {
    busy = true
    try {
      draft.libraryDirs = draft.libraryDirs.filter((d) => d.path.trim())
      app.settings = await api.saveSettings($state.snapshot(draft) as Settings)
      draft = $state.snapshot(app.settings) as Settings
      app.games = await api.refresh()
      toast(`Salvo. ${app.games.length} jogos na biblioteca.`, 'ok')
      play('accept')
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function rescan() {
    busy = true
    try {
      app.games = await api.refresh()
      toast(`${app.games.length} jogos encontrados`, 'ok')
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function toggleCovers() {
    const enabled = !draft.covers.enabled
    draft.covers.enabled = enabled
    try {
      app.settings = await api.saveSettings({ ...(app.settings as Settings), covers: { ...app.settings!.covers, enabled } })
      play('toggle')
      if (enabled) await api.updateCovers(false)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function updateCovers(all: boolean) {
    try {
      await api.updateCovers(all)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  // jogos sem descrição/ano/gênero que dá pra procurar na internet
  let missing = $state(0)
  const metaTask = $derived(activity.tasks.find((t) => t.id === 'meta'))
  $effect(() => {
    void metaTask?.state // relê quando a tarefa termina
    if (metaTask?.state === 'running') return
    api.missingMeta().then((n) => (missing = n)).catch(() => {})
  })

  async function completeMeta() {
    try {
      await api.completeMeta()
      play('accept')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function reset() {
    if (!(await confirm('Voltar todas as configurações ao padrão? Pastas, acessibilidade e emuladores.', 'Restaurar'))) return
    app.settings = await api.resetSettings()
    draft = $state.snapshot(app.settings) as Settings
    app.games = await api.refresh()
    toast('Configurações restauradas', 'ok')
  }
</script>

<form class="panel-body" onsubmit={(e) => (e.preventDefault(), save())}>
  <h2>Biblioteca</h2>
  <p class="muted">Cada pasta tem uma subpasta por console: <code>snes/</code>, <code>psx/</code>, <code>ps2/</code>... (os mesmos nomes do ES-DE).</p>

  <fieldset>
    <legend>Pastas de jogos</legend>
    <ul class="dirs">
      {#each draft.libraryDirs as dir, i (i)}
        <li>
          <label class="sr-only" for={`dir-${i}`}>Pasta {i + 1}</label>
          <input id={`dir-${i}`} type="text" data-nav bind:value={dir.path} spellcheck="false" />
          <button
            type="button"
            class="btn small"
            data-nav
            aria-label={`Remover pasta ${dir.path}`}
            disabled={draft.libraryDirs.length === 1}
            onclick={() => draft.libraryDirs.splice(i, 1)}
          >
            <Icon name="trash" />
          </button>
        </li>
      {/each}
    </ul>
    <button type="button" class="btn small" data-nav onclick={addDir}><Icon name="plus" /> Adicionar pasta</button>
  </fieldset>

  <button type="button" class="btn small" data-nav onclick={() => go({ name: 'library', source: HIDDEN })}>
    <Icon name="close" /> Jogos ocultos ({hiddenCount})
  </button>

  <div class="field">
    <label for="import-dir">Pasta onde os jogos importados ficam</label>
    <input id="import-dir" type="text" data-nav bind:value={draft.importDir} spellcheck="false" />
  </div>

  <div class="field">
    <label for="esde-dir">Pasta do ES-DE</label>
    <input id="esde-dir" type="text" data-nav bind:value={draft.esdeDir} spellcheck="false" aria-describedby="esde-help" />
    <p class="muted help" id="esde-help">Capas, descrições e anos vêm dos gamelists e da mídia de lá.</p>
  </div>

  <details>
    <summary data-nav>Emuladores e saves (avançado)</summary>
    <p class="muted">
      {app.platform === 'linux'
        ? 'Vazio = os Flatpaks (make emulators instala).'
        : 'Vazio = procura nos lugares onde os instaladores oficiais põem (retroarch.com, ppsspp.org, pcsx2.net).'}
    </p>
    {#each [{ id: 'retroarch', label: 'RetroArch (executável)' }, { id: 'cores', label: 'Pasta dos cores do RetroArch' }, { id: 'ppsspp', label: 'PPSSPP (executável)' }, { id: 'pcsx2', label: 'PCSX2 (executável)' }] as f (f.id)}
      <div class="field">
        <label for={`emu-${f.id}`}>{f.label}</label>
        <input id={`emu-${f.id}`} type="text" data-nav bind:value={draft.emulators[f.id as keyof EmulatorPaths]} placeholder="padrão" spellcheck="false" />
      </div>
    {/each}
    {#each [{ id: 'retroarchSaves', label: 'Saves do RetroArch' }, { id: 'retroarchStates', label: 'Save states do RetroArch' }, { id: 'ppssppSaveData', label: 'Saves do PPSSPP (SAVEDATA)' }, { id: 'ppssppStates', label: 'Save states do PPSSPP' }, { id: 'pcsx2Memcards', label: 'Memory cards do PCSX2' }, { id: 'pcsx2States', label: 'Save states do PCSX2' }] as f (f.id)}
      <div class="field">
        <label for={`save-${f.id}`}>{f.label}</label>
        <input id={`save-${f.id}`} type="text" data-nav bind:value={draft.saves[f.id as keyof SaveDirs]} spellcheck="false" />
      </div>
    {/each}
  </details>

  <fieldset>
    <legend>Capas</legend>
    <button type="button" class="switch" data-nav role="switch" aria-checked={draft.covers.enabled} aria-describedby="covers-help" onclick={toggleCovers}>
      <span class="knob" aria-hidden="true"></span>
      <span>Baixar capas automaticamente</span>
    </button>
    <p id="covers-help" class="muted">
      Capa, print e tela de título do acervo aberto do RetroArch (thumbnails.libretro.com), sem conta. O jogo é reconhecido pelo
      conteúdo da ROM ou pelo código do disco. Precisa de internet só pra baixar; depois fica tudo no PC. Capa sua em
      <code>covers/</code> ou do ES-DE tem prioridade.
    </p>
    <div class="actions">
      <button type="button" class="btn small" data-nav disabled={!draft.covers.enabled} onclick={() => updateCovers(false)}>Atualizar capas agora</button>
      <button type="button" class="btn small" data-nav disabled={!draft.covers.enabled} onclick={() => updateCovers(true)}>Tentar de novo as que faltam</button>
    </div>
  </fieldset>

  <fieldset>
    <legend>Informações dos jogos</legend>
    <p id="meta-help" class="muted">
      Descrição, ano e gêneros que faltam vêm do Wikidata e da Wikipedia (abertos, sem conta). Só preenche campo vazio: o que você
      editou fica. Acompanhe e cancele na Central.
    </p>
    <button type="button" class="btn small" data-nav aria-describedby="meta-help" disabled={!missing || metaTask?.state === 'running'} onclick={completeMeta}>
      <Icon name="globe" />
      {metaTask?.state === 'running' ? 'Completando informações…' : `Completar informações que faltam (${missing} jogo${missing === 1 ? '' : 's'})`}
    </button>
  </fieldset>

  <button type="button" class="switch" data-nav role="switch" aria-checked={draft.fullscreen} onclick={() => (draft.fullscreen = !draft.fullscreen)}>
    <span class="knob" aria-hidden="true"></span>
    <span>Abrir jogos em tela cheia</span>
  </button>

  <div class="actions">
    <button type="submit" class="btn primary" data-nav disabled={busy || !dirty}><Icon name="check" /> Salvar</button>
    <button type="button" class="btn" data-nav disabled={busy} onclick={rescan}><Icon name="resume" /> Procurar jogos de novo</button>
    <button type="button" class="btn" data-nav disabled={busy} onclick={reset}>Restaurar padrão</button>
  </div>
</form>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  .panel-body > .btn {
    align-self: flex-start;
  }
  code {
    padding: 0 0.3rem;
    font-size: 0.9em;
    background: var(--balloon);
    border: 1px solid color-mix(in srgb, var(--outline) 40%, transparent);
    border-radius: 4px;
    overflow-wrap: anywhere;
  }
  /* grupo: régua de 3px em cima, legenda em Bevan */
  fieldset {
    border: none;
    border-top: var(--edge) solid var(--outline);
    padding: 1rem 0 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    align-items: flex-start;
  }
  legend {
    float: left;
    width: 100%;
    padding: 0;
    font-family: var(--font-label);
    font-size: 0.9rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .dirs {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.75rem;
    width: 100%;
  }
  .dirs li {
    display: flex;
    gap: 0.75rem;
    align-items: center;
  }
  input {
    font-family: var(--font-mono);
    font-size: 0.95rem;
  }
  .field {
    display: grid;
    gap: 0.4rem;
    max-width: 40rem;
  }
  label {
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .help {
    font-size: 0.9rem;
  }
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
  }
  details {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
  }
  summary {
    width: fit-content;
    min-height: 2.8rem;
    padding: 0.6rem 0.9rem;
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    cursor: pointer;
  }
  details[open] summary {
    margin-bottom: 0.9rem;
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
</style>
