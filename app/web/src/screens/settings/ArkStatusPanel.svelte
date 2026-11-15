<script lang="ts">
  // Servidor arkflix › Estado: bibliotecas (itens e se a pasta está no ar),
  // scan, metadados e quem está usando. "Escanear agora" vira uma tarefa da
  // Central, que avisa no fim; o painel recarrega quando ela termina.
  import Icon from '../../components/Icon.svelte'
  import { api, on } from '../../lib/api'
  import { since } from '../../lib/ark'
  import { play } from '../../lib/sound'
  import { app, toast } from '../../lib/state.svelte'
  import type { ArkLibrary, ArkStatus, TaskEvent } from '../../lib/types'

  let st = $state<ArkStatus | null>(null)
  let failed = $state('')
  let scanning = $state(false)

  async function load() {
    try {
      st = await api.arkStatus()
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    load()
    return on<TaskEvent>('task', (ev) => {
      if (ev.id !== 'arkscan') return
      scanning = ev.state === 'running'
      if (!scanning) load()
    })
  })

  async function scan() {
    try {
      await api.arkScan()
      scanning = true
      play('accept')
      toast('Scan começou: o aviso chega quando terminar', 'info', 3000)
      load()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function extra(l: ArkLibrary): string {
    const parts = []
    if (l.episodes) parts.push(`${l.episodes} episódios`)
    if (l.albums) parts.push(`${l.albums} álbuns`)
    if (l.artists) parts.push(`${l.artists} artistas`)
    if (l.platforms) parts.push(`${l.platforms} consoles`)
    return parts.join(' · ')
  }

  const running = $derived(scanning || !!st?.scan.running)
</script>

<div class="panel-body">
  <h2>Estado do servidor</h2>
  {#if failed}
    <p class="err" role="alert">{failed}</p>
    <div class="actions"><button class="btn" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button></div>
  {:else if !st}
    <p class="muted" role="status">Perguntando ao servidor…</p>
  {:else}
    <p class="status">
      <Icon name="cloud" />
      <span><strong>{app.ark?.name}</strong> {st.version} em <strong>{app.ark?.base}</strong> · você é <strong>{app.ark?.user}</strong></span>
      <span class="chip" class:ok={st.writable}>{st.writable ? 'Gravável' : 'Só leitura'}</span>
    </p>

    <section aria-labelledby="ark-libs">
      <h3 id="ark-libs">Bibliotecas</h3>
      <div class="table-wrap">
        <table>
          <thead>
            <tr><th scope="col">Biblioteca</th><th scope="col" class="num">Itens</th><th scope="col">Pasta</th><th scope="col">Situação</th></tr>
          </thead>
          <tbody>
            {#each st.libraries as l (l.id)}
              <tr>
                <th scope="row">{l.name}{#if extra(l)}<span class="sub muted">{extra(l)}</span>{/if}</th>
                <td class="num">{l.count}</td>
                <td class="path">{l.path}</td>
                <td><span class="chip" class:ok={l.online} class:error={!l.online}>{l.online ? 'No ar' : 'Fora do ar'}</span></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </section>

    <section class="scan" aria-labelledby="ark-scan">
      <h3 id="ark-scan">Scan</h3>
      <p role="status">
        {#if running}
          Escaneando as pastas… (acompanhe na Central)
        {:else if st.scan.error}
          <span class="err">Último scan falhou: {st.scan.error}</span>
        {:else}
          Último scan {since(st.scan.finishedAt)}: {st.scan.count} itens.
        {/if}
      </p>
      <div class="actions">
        <button class="btn primary" data-nav data-nav-default disabled={running} onclick={scan}><Icon name="resume" /> Escanear agora</button>
      </div>
    </section>

    <section aria-labelledby="ark-meta">
      <h3 id="ark-meta">Metadados</h3>
      <p>
        {#if !st.metadata.enabled}
          Busca de metadados desligada no servidor.
        {:else}
          {st.metadata.running ? 'Buscando agora' : 'Ligada'}{st.metadata.tmdb ? ' (com TMDB)' : ''} · {st.metadata.pending} na fila.
        {/if}
        {#if st.metadata.error}<span class="err"> {st.metadata.error}</span>{/if}
      </p>
    </section>

    <section aria-labelledby="ark-sessions">
      <h3 id="ark-sessions">Em uso agora ({st.sessions.length})</h3>
      {#if st.sessions.length}
        <ul class="sessions">
          {#each st.sessions as x, i (i)}
            <li><strong>{x.user}</strong> <span class="muted">{x.client} · {x.device} · {since(x.lastSeen)}</span></li>
          {/each}
        </ul>
      {:else}
        <p class="muted">Ninguém conectado.</p>
      {/if}
    </section>
  {/if}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
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
    margin: 0 0 0.6rem;
  }
  section p {
    margin: 0;
  }
  .status {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
    margin: 0;
  }
  .status :global(svg) {
    color: var(--accent-2);
  }
  .err {
    color: var(--danger);
    font-weight: 700;
  }
  /* tabela sóbria: moldura reta, cabeçalho escuro, linhas com fio */
  .table-wrap {
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    border: var(--edge) solid var(--outline);
  }
  thead th {
    background: var(--text);
    color: var(--bg);
    font-family: var(--font-label);
    font-weight: 400;
    font-size: 0.75rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    text-align: start;
    padding: 0.6rem 0.75rem;
  }
  tbody th,
  td {
    padding: 0.6rem 0.75rem;
    text-align: start;
    vertical-align: top;
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  tbody th {
    font-weight: 700;
  }
  .sub {
    display: block;
    font-weight: 400;
    font-size: 0.85rem;
  }
  .num {
    text-align: end;
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .path {
    font-family: var(--font-mono);
    font-size: 0.85rem;
    overflow-wrap: anywhere;
  }
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
    margin-top: 0.8rem;
  }
  .sessions {
    margin: 0;
    padding-inline-start: 1.2rem;
    display: grid;
    gap: 0.4rem;
  }
  .sessions li::marker {
    color: var(--accent-2);
  }
</style>
