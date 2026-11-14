<script lang="ts">
  // Contas online do perfil: Jellyfin (filmes e séries, uma conta por perfil)
  // e RomM (um servidor pro app; aqui só escolhe qual perfil sincroniza os
  // saves). A senha vai direto pro login: o app guarda um token.
  import { api } from '../lib/api'
  import { play } from '../lib/sound'
  import { app, confirm, go, prompt, toast } from '../lib/state.svelte'
  import type { Accounts } from '../lib/types'
  import Icon from './Icon.svelte'

  let { profile }: { profile: string } = $props()

  let acc = $state<Accounts | null>(null)
  let host = $state('')
  let user = $state('')
  let busy = $state(false)

  async function load() {
    try {
      acc = await api.profileAccounts(profile)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() => {
    void profile
    load()
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

  // o perfil em uso ganha a conta na hora (a aba Filmes e séries já abre)
  async function syncCurrent() {
    if (profile === app.profile?.current) app.watch = await api.watchStatus()
  }

  async function link() {
    if (!host || !user) {
      toast('Informe o endereço e o usuário', 'info')
      return
    }
    busy = true
    let linked = false
    await prompt('Senha do Jellyfin', '', 'Vincular', async (pass) => {
      try {
        await api.connectWatch(profile, host, user, pass)
        linked = true
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 200, secret: true })
    busy = false
    if (linked) {
      play('accept')
      await load()
      await syncCurrent()
      toast(`Jellyfin vinculado: ${acc?.jellyfin.server || acc?.jellyfin.base}`, 'ok')
    }
  }

  async function unlink() {
    const ok = await confirm('Desvincular o Jellyfin deste perfil? Filmes e séries somem daqui; o histórico continua no servidor.', 'Desvincular', true)
    if (!ok) return
    try {
      await api.disconnectWatch(profile)
      await load()
      await syncCurrent()
      toast('Jellyfin desvinculado', 'ok')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function takeRomM() {
    try {
      acc = await api.linkRomM(profile)
      play('accept')
      toast('Os saves do RomM agora sincronizam neste perfil', 'ok')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<fieldset class="accounts">
  <legend>Contas vinculadas</legend>
  {#if acc}
    <div class="account">
      <h3><Icon name="film" /> Jellyfin <span class="muted">filmes e séries</span></h3>
      {#if acc.jellyfin.connected}
        <p class="status">
          Conectado a <strong>{acc.jellyfin.server || acc.jellyfin.base}</strong> como <strong>{acc.jellyfin.user}</strong>.
        </p>
        <p class="muted">O que este perfil assistiu, a lista e o controle dos pais vêm dessa conta.</p>
        <div class="row">
          <button class="btn danger" data-nav onclick={unlink}>Desvincular</button>
        </div>
      {:else}
        <p class="muted">Cada perfil usa a própria conta. Pro perfil infantil, use uma conta com controle dos pais no Jellyfin.</p>
        <button class="btn field" data-nav onclick={async () => (host = (await ask('Endereço do Jellyfin', host)) ?? host)}>
          <span class="muted">Endereço</span> <strong>{host || 'ex.: filmes.duckdns.org'}</strong>
        </button>
        <button class="btn field" data-nav onclick={async () => (user = (await ask('Usuário do Jellyfin', user, 60)) ?? user)}>
          <span class="muted">Usuário</span> <strong>{user || 'escolher'}</strong>
        </button>
        <div class="row">
          <button class="btn primary" data-nav disabled={busy || !host || !user} onclick={link}><Icon name="film" /> Vincular</button>
        </div>
      {/if}
    </div>

    <div class="account">
      <h3><Icon name="cloud" /> RomM <span class="muted">jogos e saves</span></h3>
      {#if !acc.romm.connected}
        <p class="muted">Nenhum servidor RomM ligado no app.</p>
        <div class="row">
          <button class="btn" data-nav onclick={() => go({ name: 'settings', tab: 'server' })}><Icon name="cloud" /> Ligar o servidor</button>
        </div>
      {:else if acc.romm.mine}
        <p class="status">Os saves deste perfil sincronizam com <strong>{acc.romm.user}</strong> em {acc.romm.base}.</p>
      {:else}
        <p class="muted">Os saves sincronizam no perfil {acc.romm.owner || 'que ligou o servidor'}. Aqui os jogos baixam, mas os saves ficam neste PC.</p>
        <div class="row">
          <button class="btn" data-nav onclick={takeRomM}><Icon name="save" /> Sincronizar saves neste perfil</button>
        </div>
      {/if}
    </div>
  {/if}
</fieldset>

<style>
  .accounts {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }
  .account {
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
  }
  h3 {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1.1rem;
  }
  h3 .muted {
    font-weight: 400;
    font-size: 0.9rem;
  }
  .field {
    justify-content: flex-start;
    gap: 1rem;
    width: fit-content;
  }
  .row {
    display: flex;
    gap: 0.8rem;
    flex-wrap: wrap;
  }
</style>
