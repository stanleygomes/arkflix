<script lang="ts">
  // Servidor arkflix › Enviar arquivos (só com o servidor gravável): escolhe o
  // tipo, preenche o destino e os arquivos; o Go envia em segundo plano, com
  // progresso na Central e dá pra cancelar. O servidor põe cada coisa na pasta
  // certa e escaneia depois.
  import Icon from '../../components/Icon.svelte'
  import { api, on } from '../../lib/api'
  import { cancelTask } from '../../lib/activity.svelte'
  import { play } from '../../lib/sound'
  import { app, prompt, toast } from '../../lib/state.svelte'
  import type { ArkPlatform, ArkUploadKind, TaskEvent, WatchItem } from '../../lib/types'
  import { displayName } from '../../lib/watch'

  type Field = { id: string; label: string; required?: boolean; number?: boolean; hint: string }
  const KINDS: { id: ArkUploadKind; label: string; note: string; fields: Field[] }[] = [
    {
      id: 'movie',
      label: 'Filme',
      note: 'Vai pra pasta de filmes como "Título (Ano)". Sem título, fica o nome do arquivo.',
      fields: [
        { id: 'title', label: 'Título', hint: 'nome do arquivo' },
        { id: 'year', label: 'Ano', number: true, hint: 'opcional' },
      ],
    },
    {
      id: 'episode',
      label: 'Episódio',
      note: 'Vai pra pasta da série, na temporada, como "Série - S01E02".',
      fields: [
        { id: 'series', label: 'Série', required: true, hint: 'obrigatório' },
        { id: 'season', label: 'Temporada', required: true, number: true, hint: 'obrigatório' },
        { id: 'episode', label: 'Episódio', number: true, hint: 'nome do arquivo' },
        { id: 'title', label: 'Título do episódio', hint: 'opcional' },
      ],
    },
    { id: 'rom', label: 'ROM', note: 'Vai pra pasta de ROMs do console escolhido.', fields: [] },
    { id: 'bios', label: 'BIOS', note: 'Vai pra pasta de BIOS do console escolhido.', fields: [] },
    { id: 'music', label: 'Música', note: 'Vai pra pasta de músicas, dentro da pasta escolhida (ou "Avulsas").', fields: [{ id: 'folder', label: 'Pasta', hint: 'Avulsas' }] },
    { id: 'subtitle', label: 'Legenda', note: 'Fica ao lado do vídeo do filme escolhido.', fields: [{ id: 'lang', label: 'Idioma', hint: 'ex.: pt-BR' }] },
  ]

  let kind = $state<ArkUploadKind>('movie')
  let values = $state<Record<string, string>>({})
  let platform = $state('')
  let platforms = $state<ArkPlatform[]>([])
  let target = $state<WatchItem | null>(null)
  let found = $state<WatchItem[]>([])
  let files = $state<string[]>([])
  let path = $state('')
  let task = $state<TaskEvent | null>(null)

  const spec = $derived(KINDS.find((k) => k.id === kind)!)
  const kinds = $derived(KINDS.filter((k) => k.id !== 'subtitle' || app.ark?.videos))
  const needsPlatform = $derived(kind === 'rom' || kind === 'bios')
  const sending = $derived(task?.state === 'running')
  const missing = $derived(
    spec.fields.some((f) => f.required && !values[f.id]?.trim()) || (needsPlatform && !platform) || (kind === 'subtitle' && !target),
  )

  $effect(() => {
    api.arkPlatforms().then((p) => (platforms = p)).catch((err) => toast((err as Error).message, 'error'))
    return on<TaskEvent>('task', (ev) => {
      if (ev.id !== 'arkupload') return
      task = ev
      if (ev.state === 'done') play('accept')
    })
  })

  function pickKind(id: ArkUploadKind) {
    if (kind === id) return
    kind = id
    values = {}
    platform = ''
    target = null
    found = []
  }

  async function ask(f: Field) {
    await prompt(f.label, values[f.id] ?? '', 'Usar', async (v) => {
      v = v.trim()
      if (f.required && !v) return 'Não pode ficar vazio'
      if (f.number && v && !/^\d+$/.test(v)) return 'Use só números'
      values = { ...values, [f.id]: v }
      return null
    }, { max: 120 })
  }

  async function searchTarget() {
    await prompt('Buscar o filme', '', 'Buscar', async (v) => {
      try {
        found = (await api.watchSearch(v)).filter((it) => it.type === 'Movie')
        return found.length ? null : 'Nenhum filme com esse nome'
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 80 })
  }

  function addFiles(list: string[]) {
    const known = new Set(files)
    files = [...files, ...list.filter((f) => f && !known.has(f))]
  }

  async function pick() {
    try {
      const got = await api.arkPickFiles()
      if (got?.length) addFiles(got)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function addPath(ev: SubmitEvent) {
    ev.preventDefault()
    if (path.trim()) addFiles([path.trim()])
    path = ''
  }

  async function send() {
    const fields: Record<string, string> = { ...values }
    if (needsPlatform) fields.platform = platform
    if (kind === 'subtitle' && target) fields.item = target.id
    try {
      await api.arkUpload(kind, fields, files)
      toast(`Enviando ${files.length} arquivo${files.length === 1 ? '' : 's'}: acompanhe aqui ou na Central`, 'info', 3500)
      files = []
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  const base = (p: string) => p.split(/[\\/]/).pop() ?? p
</script>

<div class="panel-body">
  <h2>Enviar arquivos</h2>
  <p class="muted">Os arquivos vão do seu PC direto pro servidor. Nome repetido não é sobrescrito: o servidor recusa.</p>

  <div class="kinds" role="radiogroup" aria-label="Tipo de arquivo">
    {#each kinds as k, i (k.id)}
      <button class="btn small" role="radio" aria-checked={kind === k.id} data-nav data-nav-default={i === 0 ? '' : undefined} onclick={() => pickKind(k.id)}>{k.label}</button>
    {/each}
  </div>
  <p class="note">{spec.note}</p>

  {#if spec.fields.length}
    <div class="fields">
      {#each spec.fields as f (f.id)}
        <button class="field" data-nav onclick={() => ask(f)}>
          <span class="field-label">{f.label}{f.required ? ' *' : ''}</span>
          <span class="value" class:empty={!values[f.id]}>{values[f.id] || f.hint}</span>
        </button>
      {/each}
    </div>
  {/if}

  {#if needsPlatform}
    <div class="group" role="radiogroup" aria-label="Console *">
      <span class="field-label" aria-hidden="true">Console *</span>
      <div class="chips">
        {#each platforms as p (p.slug)}
          <button class="btn small" role="radio" aria-checked={platform === p.slug} data-nav onclick={() => (platform = p.slug)}>{p.name}</button>
        {/each}
      </div>
    </div>
  {/if}

  {#if kind === 'subtitle'}
    <div class="group">
      <button class="field" data-nav onclick={searchTarget}>
        <span class="field-label">Filme *</span>
        <span class="value" class:empty={!target}>{target ? displayName(target) : 'buscar pelo nome'}</span>
      </button>
      {#if found.length && !target}
        <div class="chips" role="group" aria-label="Filmes encontrados">
          {#each found.slice(0, 8) as it (it.id)}
            <button class="btn small" data-nav onclick={() => ((target = it), (found = []))}>{displayName(it)}{it.year ? ` (${it.year})` : ''}</button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <section class="files" aria-labelledby="ark-files">
    <h3 id="ark-files">Arquivos ({files.length})</h3>
    <div class="sources">
      {#if app.mode === 'desktop'}
        <button class="btn" data-nav onclick={pick}><Icon name="folder" /> Escolher arquivos</button>
      {/if}
      <form class="path" onsubmit={addPath}>
        <label class="field-label" for="ark-path">Caminho de um arquivo</label>
        <div class="row">
          <input id="ark-path" type="text" data-nav bind:value={path} placeholder="~/Downloads/Filme (2024).mkv" autocomplete="off" spellcheck="false" />
          <button class="btn" data-nav type="submit" disabled={!path.trim()}><Icon name="plus" /> Adicionar</button>
        </div>
      </form>
    </div>
    {#if files.length}
      <ul class="list">
        {#each files as f (f)}
          <li>
            <span class="file" title={f}>{base(f)}</span>
            <button class="btn small flat" data-nav aria-label={`Tirar ${base(f)}`} onclick={() => (files = files.filter((x) => x !== f))}><Icon name="close" /> Tirar</button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  {#if task}
    <div class="progress" role="status">
      <span>
        {#if task.state === 'running'}
          {task.label}: {Math.round(task.progress * 100)}%
        {:else if task.state === 'done'}
          Envio terminado: {task.message}
        {:else if task.state === 'cancelled'}
          Envio cancelado
        {:else}
          Envio falhou: {task.message}
        {/if}
      </span>
      {#if sending}
        <span class="energy" aria-hidden="true"><span style:width={`${Math.floor(task.progress * 20) * 5}%`}></span></span>
        <button class="btn small" data-nav onclick={() => cancelTask('arkupload')}><Icon name="close" /> Cancelar envio</button>
      {/if}
    </div>
  {/if}

  <div class="actions">
    <button class="btn primary" data-nav disabled={sending || missing || !files.length} onclick={send}><Icon name="cloud" /> Enviar pro servidor</button>
  </div>
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
    max-width: 60rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  h3 {
    font-family: var(--font-label);
    font-size: 0.8rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    margin: 0;
  }
  p {
    margin: 0;
  }
  .kinds,
  .chips {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
  }
  .note {
    font-style: italic;
    color: var(--muted);
  }
  .group {
    display: grid;
    gap: 0.5rem;
  }
  .fields {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .field {
    display: grid;
    gap: 0.35rem;
    width: min(100%, 16rem);
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
  .files {
    display: grid;
    gap: 0.7rem;
  }
  .sources {
    display: flex;
    gap: 1rem;
    align-items: end;
    flex-wrap: wrap;
  }
  .path {
    display: grid;
    gap: 0.35rem;
    flex: 1;
    min-width: 18rem;
  }
  .row {
    display: flex;
    gap: 0.6rem;
  }
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    border: var(--edge) solid var(--outline);
  }
  .list li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.8rem;
    padding: 0.3rem 0.4rem 0.3rem 0.75rem;
  }
  .list li + li {
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  .file {
    font-family: var(--font-mono);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .progress {
    display: flex;
    gap: 1rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
  }
</style>
