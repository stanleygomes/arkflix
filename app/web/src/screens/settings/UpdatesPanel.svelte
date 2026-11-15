<script lang="ts">
  // Atualizações: versão atual → nova, novidades, "Atualizar e reiniciar"
  // num clique, procurar de novo e o "atualizar sozinho".
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { ago, bytes } from '../../lib/format'
  import { app, isKids, toast } from '../../lib/state.svelte'
  import { shortVersion, updateNow } from '../../lib/update'
  import { saveSoon } from './save.svelte'

  let checking = $state(false)
  const u = $derived(app.update)

  async function check() {
    checking = true
    try {
      app.update = await api.checkUpdate()
      if (!app.update.available) toast(`Você já está na versão mais nova (${shortVersion(app.update.current)})`, 'ok', 3000)
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      checking = false
    }
  }

  async function downloadOnly() {
    try {
      await api.downloadUpdate()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function toggleAuto() {
    if (app.settings) saveSoon({ ...app.settings, autoUpdate: !app.settings.autoUpdate })
  }

  const busy = $derived(u?.state === 'downloading' || u?.state === 'restarting')
</script>

<div class="panel-body">
  <h2>Atualizações</h2>

  {#if u?.available}
    <section class="card new" aria-labelledby="upd-new">
      <p class="chip warn">Versão nova</p>
      <h3 id="upd-new" class="versions">
        <span class="old">{shortVersion(u.current)}</span>
        <span class="arrow" aria-label="para">→</span>
        <span class="neu">{u.latest}</span>
      </h3>
      {#if u.size}<p class="muted small">{bytes(u.size)} pra baixar</p>{/if}
      {#if u.state === 'downloading'}
        <p>{u.restart ? 'Baixando; reinicia sozinho no fim…' : 'Baixando…'} {Math.round(u.progress * 100)}%</p>
        <div class="energy" role="progressbar" aria-label="Baixando atualização" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(u.progress * 100)}>
          <span style:width={`${Math.floor(u.progress * 20) * 5}%`}></span>
        </div>
      {:else if u.state === 'ready'}
        <p class="ready"><Icon name="check" /> Já baixou. Reinicie agora ou ela instala quando você fechar o app.</p>
      {/if}
      {#if u.error}<p class="error" role="alert">{u.error}</p>{/if}
      {#if isKids()}
        <p class="muted">Peça pra um adulto atualizar.</p>
      {:else}
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default disabled={busy || u.asset === ''} onclick={updateNow}>
            <Icon name="download" /> {u.state === 'ready' ? 'Reiniciar e atualizar' : 'Atualizar e reiniciar'}
          </button>
          {#if u.state === 'idle' || u.state === 'error'}
            <button class="btn" data-nav disabled={u.asset === ''} onclick={downloadOnly}>Só baixar (instala ao fechar)</button>
          {/if}
        </div>
        {#if u.asset === ''}<p class="muted small">Essa versão não tem instalador pra este sistema ainda.</p>{/if}
      {/if}
      {#if u.notes}
        <h4>Novidades</h4>
        <pre class="notes">{u.notes}</pre>
      {/if}
      {#if u.url}<p class="muted small">Detalhes: {u.url}</p>{/if}
    </section>
  {:else}
    <section class="card" aria-labelledby="upd-cur">
      <p class="chip ok">Em dia</p>
      <h3 id="upd-cur" class="versions"><span class="neu">{shortVersion(u?.current ?? app.version)}</span></h3>
      <p class="muted">{u?.error || 'Você está na versão mais nova que encontramos.'}</p>
    </section>
  {/if}

  <div class="actions">
    <button class="btn" data-nav data-nav-default={u?.available ? undefined : ''} disabled={checking || busy} onclick={check}>
      <Icon name="resume" /> {checking ? 'Procurando…' : 'Procurar atualizações'}
    </button>
    {#if app.settings && !isKids()}
      <button class="switch" data-nav role="switch" aria-checked={app.settings.autoUpdate} onclick={toggleAuto}>
        <span class="knob" aria-hidden="true"></span><span>Baixar sozinho</span>
      </button>
    {/if}
  </div>
  <p class="muted small">
    {u?.checked ? `Última consulta ${ago(u.checked)}.` : 'Ainda não consultou nesta sessão.'}
    O app procura ao abrir e a cada 6 horas. Com "Baixar sozinho", a versão nova já vem baixada e instala ao fechar.
  </p>
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  /* cartão sóbrio; versão nova ganha a barra mostarda de 8px (destaque) */
  .card {
    display: grid;
    gap: 0.75rem;
    justify-items: start;
    max-width: 44rem;
    padding: 1.25rem 1.5rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
  }
  .card.new {
    padding-inline-start: 2rem;
    box-shadow: var(--hard), inset 8px 0 0 var(--accent);
  }
  .card p {
    margin: 0;
  }
  .versions {
    display: flex;
    align-items: baseline;
    gap: 1rem;
    margin: 0;
    font-family: var(--font-label);
  }
  .old {
    font-size: 1.3rem;
    color: var(--muted);
  }
  .arrow {
    font-size: 1.4rem;
    color: var(--accent);
  }
  .neu {
    font-size: clamp(2rem, 4vw, 2.8rem);
    line-height: 1;
    color: var(--text);
  }
  .card .energy {
    width: 100%;
  }
  .ready {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    color: var(--accent-2);
  }
  .error {
    color: var(--danger);
  }
  h4 {
    margin: 0.4rem 0 0;
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .notes {
    max-height: 12rem;
    width: 100%;
    overflow-y: auto;
    white-space: pre-wrap;
    font-family: var(--font-mono);
    font-size: 0.88rem;
    color: var(--muted);
    margin: 0;
    padding: 0.75rem;
    background: var(--bg);
    border: 2px solid color-mix(in srgb, var(--outline) 40%, transparent);
    border-radius: var(--radii-sm);
  }
  .actions {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .small {
    font-size: 0.85rem;
    overflow-wrap: anywhere;
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
