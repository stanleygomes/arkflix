<script lang="ts">
  // Músicas › Servidor (conta admin num perfil adulto): scan da biblioteca
  // (vira tarefa da Central), contagem e usuários. No Navidrome os usuários
  // são só leitura (ele cuida deles na interface web dele); num
  // arkflix-server, a aba Usuários e o envio de arquivos do painel do
  // Servidor arkflix fazem o resto.
  import Icon from '../../components/Icon.svelte'
  import { api, on } from '../../lib/api'
  import { since } from '../../lib/ark'
  import { play } from '../../lib/sound'
  import { app, go, toast } from '../../lib/state.svelte'
  import type { MusicAdminInfo, TaskEvent } from '../../lib/types'

  let info = $state<MusicAdminInfo | null>(null)
  let failed = $state('')
  let scanning = $state(false)
  const srv = $derived(app.musicServer)

  async function load() {
    try {
      info = await api.musicAdmin()
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    load()
    return on<TaskEvent>('task', (ev) => {
      if (ev.id !== 'musicscan') return
      scanning = ev.state === 'running'
      if (!scanning) load()
    })
  })

  async function scan() {
    try {
      await api.musicScan()
      scanning = true
      play('accept')
      toast('Scan das músicas começou: o aviso chega quando terminar', 'info', 3000)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  const running = $derived(scanning || !!info?.scan.scanning)
  const navidrome = $derived(!srv?.ark)
</script>

<div class="panel-body">
  <h2>Servidor de músicas</h2>
  {#if failed}
    <p class="err" role="alert">{failed}</p>
    <div class="actions"><button class="btn" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button></div>
  {:else if !info}
    <p class="muted" role="status">Perguntando ao servidor…</p>
  {:else}
    <p class="status">
      <Icon name="music" />
      <span><strong>{srv?.server || 'Subsonic'}</strong> {srv?.version} em <strong>{srv?.base}</strong> · você é <strong>{srv?.user}</strong> (admin)</span>
    </p>

    <section class="scan" aria-labelledby="music-scan">
      <h3 id="music-scan">Scan</h3>
      <p role="status">
        {#if running}
          Escaneando as músicas… (acompanhe na Central)
        {:else}
          Último scan {info.scan.lastScan ? since(info.scan.lastScan) : 'nunca'}: {info.scan.count} faixas{info.scan.folderCount ? `, ${info.scan.folderCount} pastas/álbuns` : ''}.
        {/if}
      </p>
      <div class="actions">
        <button class="btn primary" data-nav data-nav-default disabled={running} onclick={scan}><Icon name="resume" /> Escanear músicas</button>
      </div>
    </section>

    <section aria-labelledby="music-users">
      <h3 id="music-users">Usuários</h3>
      {#if srv?.ark}
        <p>Este servidor é um arkflix-server: os usuários (criar, trocar senha, admin) ficam no painel do Servidor arkflix.</p>
        {#if srv.arkPanel}
          <div class="actions">
            <button class="btn" data-nav onclick={() => go({ name: 'you', item: 'arkusers' })}><Icon name="user" /> Abrir os usuários do servidor</button>
          </div>
        {:else}
          <p class="muted">Pra abrir o painel, vincule a conta do Jellyfin (ou o RomM) desse mesmo servidor com um usuário admin.</p>
        {/if}
      {:else}
        <p class="muted">Só leitura: o Navidrome cuida dos usuários na interface web dele (Configurações › Usuários).</p>
        {#if info.error}
          <p class="err">{info.error}</p>
        {:else}
          <div class="table-wrap">
            <table>
              <thead><tr><th scope="col">Usuário</th><th scope="col">Papel</th><th scope="col">Pode</th></tr></thead>
              <tbody>
                {#each info.users as u (u.username)}
                  <tr>
                    <th scope="row">{u.username}</th>
                    <td><span class="chip" class:ok={u.adminRole}>{u.adminRole ? 'Admin' : 'Comum'}</span></td>
                    <td>{[u.streamRole && 'ouvir', u.downloadRole && 'baixar', u.playlistRole && 'playlists'].filter(Boolean).join(', ') || '—'}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      {/if}
    </section>

    {#if srv?.upload}
      <section aria-labelledby="music-upload">
        <h3 id="music-upload">Enviar músicas</h3>
        <p>O servidor está gravável: mande arquivos de música pelo envio do Servidor arkflix (tipo “Música”).</p>
        <div class="actions">
          <button class="btn" data-nav onclick={() => go({ name: 'you', item: 'arkupload' })}><Icon name="download" /> Enviar arquivos</button>
        </div>
      </section>
    {:else if navidrome}
      <p class="muted">Pra pôr músicas novas no Navidrome, copie os arquivos pra pasta dele e escaneie.</p>
    {/if}
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
    margin: 0 0 0.4rem;
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
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
    margin-top: 0.8rem;
  }
</style>
