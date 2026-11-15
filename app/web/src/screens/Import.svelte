<script lang="ts">
  import { tick } from 'svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { bytes } from '../lib/format'
  import { focusDefault, focusWithin } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, system, toast } from '../lib/state.svelte'
  import type { ImportItem, ImportResult } from '../lib/types'

  let items = $state<ImportItem[]>([])
  let results = $state<ImportResult[]>([])
  let path = $state('')
  let busy = $state(false)

  $effect(() => {
    if (!embedded) tick().then(focusDefault)
  })

  const ready = $derived(items.filter((i) => !i.problem && i.system))

  async function analyze(paths: string[]) {
    if (!paths.length) return
    busy = true
    try {
      const found = await api.analyzeImport(paths)
      const known = new Set(items.map((i) => i.source))
      items = [...items, ...found.filter((f) => !known.has(f.source))]
      results = []
      play('accept')
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function pick() {
    try {
      const files = await api.pickFiles()
      if (files?.length) await analyze(files)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function addPath(ev: SubmitEvent) {
    ev.preventDefault()
    const p = path.trim()
    if (!p) return
    path = ''
    analyze([p])
  }

  function cycleSystem(i: number) {
    const it = items[i]
    const options = it.candidates?.length ? it.candidates : app.systems.map((s) => s.slug)
    const next = options[(options.indexOf(it.system) + 1) % options.length]
    items[i] = { ...it, system: next, problem: it.problem?.startsWith('extensão desconhecida') ? '' : it.problem }
    play('toggle')
  }

  function removeItem(i: number) {
    items.splice(i, 1)
    tick().then(() => (embedded ? focusWithin(root) : focusDefault()))
  }

  async function run() {
    busy = true
    try {
      results = await api.importItems(ready)
      const ok = results.filter((r) => r.ok).length
      const bad = results.length - ok
      toast(`${ok} jogo(s) importado(s)${bad ? `, ${bad} com problema` : ''}`, bad ? 'error' : 'ok')
      items = items.filter((i) => !results.some((r) => r.ok && r.source === i.source))
      play(bad ? 'error' : 'start')
      app.games = await api.refresh()
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }
  // embedded: aberta como painel da tela Você (lista sempre visível)
  let { embedded = false }: { embedded?: boolean } = $props()
  let root = $state<HTMLElement | null>(null)
</script>

<div class="page" class:embedded bind:this={root}>
  {#if !embedded}<TopBar actions={false} />{/if}
  <main class="body">
    {#if !embedded}<h1>Importar jogos</h1>{/if}
    <p class="intro">Os jogos são copiados pra <code>{app.settings?.importDir}</code>, numa pasta por console. Zip com disco vira pasta; o resto vai como está.</p>

    <div class="sources panel">
      {#if app.mode === 'desktop'}
        <button class="btn primary" data-nav data-nav-default disabled={busy} onclick={pick}><Icon name="folder" /> Escolher arquivos</button>
      {/if}
      <form class="path" onsubmit={addPath}>
        <label class="label" for="import-path">Caminho de um arquivo ou pasta</label>
        <div class="row">
          <input id="import-path" type="text" data-nav data-nav-default={app.mode === 'web' ? '' : undefined} bind:value={path} placeholder="~/Downloads/Super Mario World.zip" autocomplete="off" spellcheck="false" />
          <button class="btn" data-nav type="submit" disabled={busy || !path.trim()}><Icon name="plus" /> Adicionar</button>
        </div>
      </form>
    </div>

    {#if items.length}
      <section aria-labelledby="found-title">
        <h2 id="found-title">Encontrados ({items.length})</h2>
        <ul class="items">
          {#each items as it, i (it.source)}
            {@const sys = system(it.system)}
            <li class="item" class:bad={!!it.problem}>
              <div class="what">
                <strong>{it.title}</strong>
                <span class="file">{it.source}</span>
                <span class="meta muted">{bytes(it.size)}{it.extract ? ' · vira pasta' : ''}</span>
                {#if it.problem}<span class="problem"><Icon name="warn" /> {it.problem}</span>{/if}
              </div>
              <button
                class="btn small console"
                data-nav
                disabled={!!it.problem && !it.problem.startsWith('extensão desconhecida')}
                onclick={() => cycleSystem(i)}
              >
                <span class="sr-only">Console: </span>{sys?.short ?? 'Escolher console'}<span class="sr-only"> ({sys?.name ?? 'nenhum'}; aperte pra trocar)</span>
              </button>
              <button class="btn small" data-nav aria-label={`Tirar ${it.title} da lista`} onclick={() => removeItem(i)}><Icon name="trash" /></button>
            </li>
          {/each}
        </ul>
        <div class="go">
          <button class="btn primary" data-nav disabled={busy || ready.length === 0} onclick={run}>
            <Icon name="check" /> Importar {ready.length} jogo{ready.length === 1 ? '' : 's'}
          </button>
          <button class="btn" data-nav disabled={busy} onclick={() => (items = [])}>Limpar lista</button>
        </div>
      </section>
    {/if}

    {#if results.length}
      <section aria-labelledby="results-title">
        <h2 id="results-title">Resultado</h2>
        <ul class="results">
          {#each results as r (r.source)}
            <li class:ok={r.ok}>
              <span class="lead"><Icon name={r.ok ? 'check' : 'warn'} /></span>
              <strong>{r.title}</strong>
              <span class="why">{r.ok ? `em ${system(r.system)?.name}` : r.problem}</span>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </main>
  {#if !embedded}<HelpBar hints={[{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />{/if}
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
  .body {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem 2.5rem 2rem 3.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    max-width: 70rem;
  }
  h1 {
    font-size: clamp(2.4rem, 4vw, 3.2rem);
  }
  h2 {
    font-size: 1rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-bottom: 0.8rem;
  }
  .intro {
    max-width: 48rem;
    color: var(--muted);
  }
  code {
    color: var(--text);
    font-size: 0.9em;
    padding: 0 0.3rem;
    background: var(--balloon);
    border-radius: 4px;
  }
  .sources {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1.25rem;
  }
  /* campo: rótulo em Bevan em cima, controle e botão na mesma linha */
  .path {
    display: grid;
    gap: 0.5rem;
    width: 100%;
  }
  .row {
    display: flex;
    gap: 0.8rem;
    align-items: center;
  }
  .row input {
    min-height: 3.2rem;
    font-size: 1.05rem;
  }
  /* lista de itens: moldura reta, linhas separadas por fio */
  .items,
  .results {
    list-style: none;
    padding: 0;
    margin: 0;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
  }
  .item,
  .results li {
    border-bottom: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  .item:last-child,
  .results li:last-child {
    border-bottom: 0;
  }
  .item {
    display: flex;
    gap: 1rem;
    align-items: center;
    padding: 0.9rem 1rem;
  }
  .item.bad {
    box-shadow: inset 6px 0 0 var(--danger);
  }
  .what {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    min-width: 0;
  }
  .file {
    overflow-wrap: anywhere;
    font-family: var(--font-mono);
    font-size: 0.85rem;
    color: var(--muted);
  }
  .meta {
    font-size: 0.85rem;
  }
  .problem {
    color: var(--danger);
    display: inline-flex;
    gap: 0.4rem;
    align-items: center;
  }
  .console {
    min-width: 8rem;
    justify-content: center;
  }
  .go {
    display: flex;
    gap: 0.75rem;
    margin-top: 1.25rem;
    padding-top: 1.25rem;
    border-top: var(--edge) solid var(--outline);
  }
  .results li {
    display: grid;
    grid-template-columns: auto auto 1fr;
    gap: 0.75rem;
    align-items: center;
    padding: 0.75rem 1rem;
  }
  .lead {
    display: grid;
    place-items: center;
    width: 2.4rem;
    height: 2.4rem;
    color: var(--danger);
    background: var(--bg);
    border: var(--edge) solid var(--danger);
    border-radius: var(--radii-sm);
  }
  .results li.ok .lead {
    color: var(--accent-2);
    border-color: var(--outline);
  }
  .why {
    color: var(--muted);
  }
</style>
