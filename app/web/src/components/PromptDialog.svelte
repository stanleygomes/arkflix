<script lang="ts">
  // Diálogo de texto: campo + teclado na tela. Voltar cancela.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app } from '../lib/state.svelte'
  import Keyboard from './Keyboard.svelte'

  const p = $derived(app.prompt)
  let previous: HTMLElement | null = null
  let busy = $state(false)
  const typing = $derived(app.family === 'keyboard')

  function close(v: string | null) {
    const cur = app.prompt
    app.prompt = null
    cur?.resolve(v)
    previous?.focus()
  }

  async function submit() {
    if (!app.prompt || busy) return
    busy = true
    const err = await app.prompt.submit(app.prompt.value)
    busy = false
    if (err) {
      if (app.prompt) app.prompt.error = err
      play('error')
      return
    }
    play('accept')
    close(app.prompt?.value ?? null)
  }

  $effect(() => {
    if (!p) return
    previous = document.activeElement as HTMLElement | null
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        play('back')
        close(null)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })
</script>

{#if p}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="dialog" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="prompt-title" data-nav-scope>
      <h2 id="prompt-title" class="head">{p.title}</h2>
      <div class="body">
      <label class="sr-only" for="prompt-input">{p.title}</label>
      {#if p.multiline}
        <textarea
          id="prompt-input"
          data-nav
          data-nav-default={typing ? '' : undefined}
          bind:value={p.value}
          rows="6"
          maxlength={p.max}
          aria-invalid={p.error ? 'true' : undefined}
          aria-describedby={p.error ? 'prompt-error' : undefined}
        ></textarea>
      {:else}
      <input
        id="prompt-input"
        type={p.secret ? 'password' : 'text'}
        maxlength={p.max}
        data-nav
        data-nav-default={typing ? '' : undefined}
        bind:value={p.value}
        autocomplete="off"
        spellcheck="false"
        aria-invalid={p.error ? 'true' : undefined}
        aria-describedby={p.error ? 'prompt-error' : undefined}
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            submit()
          }
        }}
      />
      {/if}
      {#if p.error}<p id="prompt-error" class="error" role="alert">{p.error}</p>{/if}
      <Keyboard bind:value={p.value} autofocus={!typing} max={p.max} />
      </div>
      <div class="actions">
        <button class="btn" data-nav onclick={() => close(null)}>Cancelar</button>
        <button class="btn primary" data-nav disabled={busy} onclick={submit}>{p.ok}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Diálogo sóbrio (noemi-ui) com campo: cabeçalho em Bevan, corpo com o
     campo e o teclado da tela, rodapé com linha e botões à direita. */
  .backdrop {
    position: fixed;
    inset: 0;
    /* tinta (#24170f, o contorno do tema claro) a 55%, sem blur: o creme
       do contorno escuro lavaria a tela */
    background: color-mix(in srgb, var(--on-accent) 55%, transparent);
    display: grid;
    place-items: center;
    padding: var(--safe);
    z-index: 110;
  }
  .dialog {
    max-width: 100%;
    max-height: 100%;
    overflow-y: auto;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 8px 8px 0 var(--outline);
  }
  .head {
    font-size: 1.2rem;
    padding: 1.1rem 1.25rem 0;
  }
  .body {
    display: grid;
    gap: 0.9rem;
    padding: 1rem 1.25rem 1.25rem;
  }
  textarea {
    width: min(40rem, 80vw);
    resize: vertical;
  }
  .error {
    color: var(--danger);
    font-weight: 700;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    border-block-start: var(--edge) solid var(--outline);
  }
</style>
