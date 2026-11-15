<script lang="ts">
  import { untrack } from 'svelte'
  // Servidor RomM: entrar (endereço, usuário e senha), ver o estado, atualizar
  // a lista e sair. A senha vai direto pro login; o app guarda só um token.
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { refreshArk } from '../../lib/ark'
  import { ago } from '../../lib/format'
  import { play } from '../../lib/sound'
  import { app, confirm, prompt, toast } from '../../lib/state.svelte'
  import type { RemoteInfo } from '../../lib/types'

  let info = $state<RemoteInfo | null>(null)
  let host = $state('')
  let user = $state('')
  let busy = $state(false)

  async function load() {
    try {
      info = await api.remoteStatus()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() => {
    load()
  })

  // a lista nova chega pelo evento "library"
  // (sem ler o info aqui dentro: o load muda o info e o efeito rodaria em loop)
  $effect(() => {
    void app.games
    if (untrack(() => info?.connected)) load()
  })

  async function ask(title: string, value: string, max = 120) {
    let got = ''
    const ok = await prompt(title, value, 'Usar', async (v) => {
      if (!v.trim()) return 'Não pode ficar vazio'
      got = v.trim()
      return null
    }, { max })
    return ok === null ? null : got
  }

  async function connect() {
    if (!host || !user) {
      toast('Informe o endereço e o usuário', 'info')
      return
    }
    busy = true
    await prompt('Senha do RomM', '', 'Entrar', async (pass) => {
      if (!pass) return 'Digite a senha'
      try {
        info = await api.connectRemote(host, user, pass)
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 200, secret: true })
    busy = false
    if (info?.connected) {
      refreshArk()
      play('accept')
      toast(`Conectado a ${info.base}`, 'ok')
    }
  }

  async function disconnect() {
    const ok = await confirm(
      'Sair do servidor? Os jogos que só estão lá saem da biblioteca; os baixados e os saves deste PC ficam. O token do app continua no RomM até você apagar lá.',
      'Sair do servidor',
      true,
    )
    if (!ok) return
    try {
      await api.disconnectRemote()
      info = await api.remoteStatus()
      refreshArk()
      toast('Servidor desconectado', 'ok')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function refresh() {
    try {
      await api.refreshRemote()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="panel-body">
  <h2>Servidor RomM</h2>
  {#if info?.connected}
    <p class="status"><Icon name="cloud" /> Conectado a <strong>{info.base}</strong> como <strong>{info.user}</strong>.</p>
    <ul class="facts">
      <li>{info.games} jogo{info.games === 1 ? '' : 's'} no servidor{info.updated ? ` · lista de ${ago(info.updated)}` : ''}</li>
      <li>
        {#if info.mine}
          Saves do RetroArch deste perfil sincronizam: descem antes de jogar (se os do servidor forem mais novos) e sobem ao fechar o jogo.
        {:else}
          Os saves sincronizam só no perfil {info.owner || 'que conectou'}; neste perfil, os jogos baixam mas os saves ficam neste PC.
        {/if}
      </li>
      <li class="muted">Saves de PSP e PS2 não sincronizam (ficam em pastas e memory cards que o RomM não guarda).</li>
    </ul>
    <div class="actions">
      <button class="btn" data-nav data-nav-default onclick={refresh}><Icon name="resume" /> Atualizar lista</button>
      <button class="btn danger" data-nav onclick={disconnect}>Sair do servidor</button>
    </div>
  {:else if info}
    <p class="muted">
      Entre no seu RomM pra ver os jogos de lá na biblioteca, baixar pra jogar aqui e sincronizar os saves. A senha só é
      usada pra entrar: o app guarda um token próprio, que você pode apagar no RomM.
    </p>
    <button class="field" data-nav data-nav-default onclick={async () => (host = (await ask('Endereço do servidor', host)) ?? host)}>
      <span class="field-label">Endereço</span> <span class="value" class:empty={!host}>{host || 'ex.: meu-romm.duckdns.org'}</span>
    </button>
    <button class="field" data-nav onclick={async () => (user = (await ask('Usuário do RomM', user, 60)) ?? user)}>
      <span class="field-label">Usuário</span> <span class="value" class:empty={!user}>{user || 'escolher'}</span>
    </button>
    <div class="actions">
      <button class="btn primary" data-nav disabled={busy || !host || !user} onclick={connect}><Icon name="cloud" /> Entrar</button>
    </div>
  {/if}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  .status {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .status :global(svg) {
    color: var(--accent-2);
  }
  .facts {
    margin: 0;
    padding-inline-start: 1.2rem;
    display: grid;
    gap: 0.5rem;
  }
  .facts li::marker {
    color: var(--accent-2);
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
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
  }
</style>
