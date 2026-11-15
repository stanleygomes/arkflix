<script lang="ts" module>
  // o modo gerenciar sobrevive à ida e volta do editor
  let keepManaging = false
</script>

<script lang="ts">
  // "Quem vai jogar?": escolher, criar e (no modo gerenciar) editar perfis.
  import { tick } from 'svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, applyBoot, askPIN, back, go, isKids, resetTo, toast } from '../lib/state.svelte'
  import type { Profile } from '../lib/types'

  const MAX = 8
  let managing = $state(keepManaging)
  $effect(() => {
    keepManaging = managing
  })
  const list = $derived(app.profile?.profiles ?? [])
  const canGoBack = $derived(app.stack.length > 1)

  $effect(() => {
    tick().then(focusDefault)
    // a lista pode ter mudado no editor
    api.profiles().then((p) => (app.profile = p)).catch(() => {})
  })

  $effect(() =>
    pushHandler((a) => {
      if (app.pin || app.confirm) return false
      if (a === 'back') {
        if (managing) {
          managing = false
          return true
        }
        return !canGoBack // na abertura, escolher é obrigatório
      }
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random' || a === 'menu'
    }),
  )

  async function enter(p: Profile) {
    if (managing) {
      go({ name: 'profile', id: p.id })
      return
    }
    let pin = ''
    if (p.hasPIN) {
      const got = await askPIN(`PIN de ${p.name}`, async (v) => {
        try {
          await api.checkProfilePIN(p.id, v)
          return null
        } catch (err) {
          return (err as Error).message
        }
      })
      if (got === null) return
      pin = got
    }
    try {
      const boot = await api.selectProfile(p.id, pin)
      applyBoot(boot)
      resetTo({ name: 'home' })
      play('start')
      toast(`Olá, ${p.name}!`, 'ok', 2500)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="page">
  <main class="who" aria-labelledby="who-title">
    <h1 id="who-title">{managing ? 'Gerenciar perfis' : 'Quem vai jogar?'}</h1>
    <ul class="cards">
      {#each list as p (p.id)}
        <li>
          <button
            class="card"
            data-nav
            data-profile={p.id}
            data-nav-default={p.id === app.profile?.current ? '' : undefined}
            onclick={() => enter(p)}
          >
            <span class="art">
              <Avatar avatar={p.avatar} color={p.color} size="7rem" mark />
              {#if managing}<span class="edit" aria-hidden="true"><Icon name="pencil" /></span>{/if}
            </span>
            <span class="name">
              {#if managing}<span class="sr-only">Editar </span>{/if}{p.name}
            </span>
            {#if p.hasPIN || p.kids}
              <span class="tags">
                {#if p.hasPIN}<span class="chip"><Icon name="lock" size="0.9em" /> PIN</span>{/if}
                {#if p.kids}<span class="chip ok">Infantil</span>{/if}
              </span>
            {/if}
          </button>
        </li>
      {/each}
      {#if list.length < MAX && !isKids()}
        <li>
          <button class="card add" data-nav onclick={() => go({ name: 'profile' })}>
            <span class="art plus" aria-hidden="true"><Icon name="plus" size="2.6rem" /></span>
            <span class="name">Adicionar perfil</span>
          </button>
        </li>
      {/if}
    </ul>
    <div class="actions">
      {#if !isKids()}
        <button class="btn" data-nav aria-pressed={managing} onclick={() => (managing = !managing)}>
          <Icon name="sliders" /> {managing ? 'Pronto' : 'Gerenciar perfis'}
        </button>
      {/if}
      {#if canGoBack}
        <button class="btn" data-nav onclick={() => { managing = false; keepManaging = false; back() }}><Icon name="back" /> Voltar</button>
      {/if}
    </div>
  </main>
  <HelpBar hints={[{ button: 'south', label: managing ? 'Editar' : 'Entrar' }, ...(canGoBack || managing ? [{ button: 'east', label: 'Voltar' }] : [])]} />
</div>

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .who {
    flex: 1;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 2.4rem;
    padding: 2rem;
    overflow-y: auto;
  }
  h1 {
    font-size: clamp(2.6rem, 6vw, 4.6rem);
    text-align: center;
  }
  .cards {
    list-style: none;
    margin: 0;
    padding: 0.75rem;
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 1.75rem;
  }
  /* cartão sóbrio: fundo de cartão, borda reta, sombra dura; sobe no foco */
  .card {
    display: grid;
    justify-items: center;
    align-content: start;
    gap: 0.75rem;
    width: 11rem;
    padding: 1.25rem 0.75rem 1rem;
    font: inherit;
    color: var(--text);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .card:hover,
  .card:focus-visible {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .card:active {
    transform: translate(4px, 4px);
    box-shadow: none;
  }
  .art {
    position: relative;
    display: grid;
    place-items: center;
  }
  .add {
    background: transparent;
    border-style: dashed;
    box-shadow: none;
  }
  .plus {
    width: 7rem;
    height: 7rem;
    color: var(--muted);
  }
  .add:hover .plus,
  .add:focus-visible .plus {
    color: var(--accent);
  }
  /* gerenciar: etiqueta de lápis no canto, em mostarda (ação) */
  .edit {
    position: absolute;
    inset-block-start: -0.6rem;
    inset-inline-end: -0.9rem;
    display: grid;
    place-items: center;
    width: 2.4rem;
    height: 2.4rem;
    font-size: 1.2rem;
    color: var(--on-accent);
    background: var(--accent);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  .name {
    font-family: var(--font-label);
    font-size: 1.05rem;
    letter-spacing: 0.02em;
    text-align: center;
    overflow-wrap: anywhere;
  }
  .tags {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 0.4rem;
  }
  .actions {
    display: flex;
    gap: 1rem;
  }
</style>
