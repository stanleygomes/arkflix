<script lang="ts">
  // Servidor arkflix › Usuários: quem entra no servidor. Criar (nome, senha,
  // admin), trocar a senha, ligar/desligar admin e apagar (com telegrama).
  // As senhas vão direto pro servidor; o app não guarda nenhuma.
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { since } from '../../lib/ark'
  import { play } from '../../lib/sound'
  import { app, confirm, prompt, toast } from '../../lib/state.svelte'
  import type { ArkUser } from '../../lib/types'

  let users = $state<ArkUser[] | null>(null)
  let failed = $state('')
  let name = $state('')
  let pass = $state('')
  let admin = $state(false)
  let busy = $state(false)

  async function load() {
    try {
      users = await api.arkUsers()
      failed = ''
    } catch (err) {
      failed = (err as Error).message
    }
  }

  $effect(() => {
    load()
  })

  function put(u: ArkUser) {
    if (!users) return
    const i = users.findIndex((x) => x.id === u.id)
    if (i >= 0) users[i] = u
    else users.push(u)
  }

  async function askName() {
    await prompt('Nome do usuário', name, 'Usar', async (v) => {
      if (!v.trim()) return 'Não pode ficar vazio'
      name = v.trim()
      return null
    }, { max: 40 })
  }

  async function askPass() {
    await prompt('Senha do usuário', '', 'Usar', async (v) => {
      if (v.length < 4) return 'Use pelo menos 4 caracteres'
      pass = v
      return null
    }, { max: 120, secret: true })
  }

  async function create() {
    if (!name || !pass) {
      toast('Informe o nome e a senha', 'info')
      return
    }
    busy = true
    try {
      put(await api.arkCreateUser(name, pass, admin))
      play('accept')
      toast(`Usuário ${name} criado`, 'ok', 2500)
      name = ''
      pass = ''
      admin = false
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function password(u: ArkUser) {
    await prompt(`Senha nova de ${u.name}`, '', 'Trocar senha', async (v) => {
      if (v.length < 4) return 'Use pelo menos 4 caracteres'
      try {
        put(await api.arkSetPassword(u.id, v))
        toast(`Senha de ${u.name} trocada`, 'ok', 2500)
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 120, secret: true })
  }

  async function toggleAdmin(u: ArkUser) {
    try {
      put(await api.arkSetAdmin(u.id, !u.admin))
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function remove(u: ArkUser) {
    const ok = await confirm(`Apagar o usuário ${u.name} do servidor? As listas e o progresso dele somem junto. Não dá pra desfazer.`, 'Apagar usuário', true)
    if (!ok) return
    try {
      await api.arkDeleteUser(u.id)
      users = users?.filter((x) => x.id !== u.id) ?? null
      toast(`Usuário ${u.name} apagado`, 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="panel-body">
  <h2>Usuários do servidor</h2>
  {#if failed}
    <p class="err" role="alert">{failed}</p>
    <div class="actions"><button class="btn" data-nav data-nav-default onclick={load}><Icon name="resume" /> Tentar de novo</button></div>
  {:else if !users}
    <p class="muted" role="status">Buscando os usuários…</p>
  {:else}
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th scope="col">Nome</th><th scope="col">Último acesso</th><th scope="col">Ações</th></tr>
        </thead>
        <tbody>
          {#each users as u, i (u.id)}
            {@const me = u.name === app.ark?.user}
            <tr>
              <th scope="row">
                {u.name}
                {#if u.admin}<span class="chip ok">Admin</span>{/if}
                {#if me}<span class="chip">Você</span>{/if}
              </th>
              <td class="meta">{since(u.lastLogin)}</td>
              <td class="row-actions">
                <button class="btn small flat" data-nav data-nav-default={i === 0 ? '' : undefined} aria-label={`Trocar senha de ${u.name}`} onclick={() => password(u)}>Trocar senha</button>
                {#if !me}
                  <button class="btn small flat" data-nav aria-label={`${u.admin ? 'Tirar admin' : 'Tornar admin'} de ${u.name}`} onclick={() => toggleAdmin(u)}>
                    {u.admin ? 'Tirar admin' : 'Tornar admin'}
                  </button>
                  <button class="btn small flat del" data-nav aria-label={`Apagar ${u.name}`} onclick={() => remove(u)}><Icon name="trash" /> Apagar</button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <section class="new" aria-labelledby="ark-new-user">
      <h3 id="ark-new-user">Novo usuário</h3>
      <div class="fields">
        <button class="field" data-nav onclick={askName}>
          <span class="field-label">Nome</span> <span class="value" class:empty={!name}>{name || 'escolher'}</span>
        </button>
        <button class="field" data-nav onclick={askPass}>
          <span class="field-label">Senha</span> <span class="value" class:empty={!pass}>{pass ? '••••••••' : 'escolher'}</span>
        </button>
      </div>
      <button class="choice" role="switch" aria-checked={admin} data-nav onclick={() => (admin = !admin)}>
        <span class="track" aria-hidden="true"></span> Administrador (pode mexer no servidor)
      </button>
      <div class="actions">
        <button class="btn primary" data-nav disabled={busy || !name || !pass} onclick={create}><Icon name="plus" /> Criar usuário</button>
      </div>
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
    margin: 0;
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
    padding: 0.5rem 0.75rem;
    text-align: start;
    vertical-align: middle;
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  tbody th {
    font-weight: 700;
  }
  tbody th .chip {
    margin-inline-start: 0.4rem;
  }
  tbody tr:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .row-actions {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .del {
    color: var(--danger);
  }
  .new {
    display: grid;
    gap: 0.9rem;
    justify-items: start;
  }
  .fields {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }
  /* campo: rótulo em Bevan em cima, valor numa caixa de texto */
  .field {
    display: grid;
    gap: 0.35rem;
    width: min(100%, 18rem);
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
  /* interruptor sóbrio: trilho com bolinha, mostarda quando ligado */
  .choice {
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
    min-height: 3rem;
    padding: 0.3rem 0.5rem;
    font: inherit;
    color: var(--text);
    background: none;
    border: 0;
    cursor: pointer;
  }
  .track {
    position: relative;
    width: 46px;
    height: 26px;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 999px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .track::after {
    content: '';
    position: absolute;
    top: 2px;
    inset-inline-start: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--outline);
  }
  [aria-checked='true'] > .track {
    background: var(--accent);
  }
  [aria-checked='true'] > .track::after {
    inset-inline-start: 22px;
    background: var(--on-accent);
  }
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
  }
</style>
