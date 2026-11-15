<script lang="ts">
  // Teclado de PIN (4 números): controle navega no teclado da tela, teclado
  // físico digita direto. Com 4 números confere; errou, limpa e avisa.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  const ask = $derived(app.pin)
  let digits = $state('')
  let busy = $state(false)
  let shake = $state(false) // errou: a fileira treme uma vez
  let previous: HTMLElement | null = null

  function close(v: string | null) {
    const cur = app.pin
    app.pin = null
    digits = ''
    cur?.resolve(v)
    previous?.focus()
  }

  async function type(d: string) {
    if (busy || digits.length >= 4) return
    digits += d
    play('move')
    if (digits.length < 4 || !app.pin) return
    busy = true
    const err = await app.pin.submit(digits)
    busy = false
    if (!app.pin) return
    if (err) {
      play('error')
      app.pin.error = err
      digits = ''
      shake = true
      return
    }
    play('accept')
    close(digits)
  }

  function erase() {
    digits = digits.slice(0, -1)
  }

  $effect(() => {
    if (!ask) return
    previous = document.activeElement as HTMLElement | null
    digits = ''
    tick().then(focusDefault)
    const onKey = (ev: KeyboardEvent) => {
      if (/^[0-9]$/.test(ev.key)) {
        ev.preventDefault()
        ev.stopPropagation()
        type(ev.key)
      } else if (ev.key === 'Backspace') {
        ev.preventDefault()
        erase()
      }
    }
    window.addEventListener('keydown', onKey, { capture: true })
    const off = pushHandler((a) => {
      if (a === 'back') {
        play('back')
        close(null)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
    return () => {
      window.removeEventListener('keydown', onKey, { capture: true })
      off()
    }
  })
</script>

{#if ask}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="dialog" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="pin-title" data-nav-scope>
      <h2 id="pin-title">{ask.title}</h2>
      <p class="dots" class:shake class:bad={!!ask.error && !digits} aria-live="polite" onanimationend={() => (shake = false)}>
        <span class="sr-only">{digits.length} de 4 números</span>
        {#each [0, 1, 2, 3] as i (i)}<span class="cell" class:on={i < digits.length} class:next={i === digits.length && !busy} aria-hidden="true">{i < digits.length ? '•' : ''}</span>{/each}
      </p>
      {#if ask.error}<p class="error" role="alert">{ask.error}</p>{/if}
      <div class="keys">
        {#each ['1', '2', '3', '4', '5', '6', '7', '8', '9'] as d (d)}
          <button class="btn key digit" data-nav data-nav-default={d === '1' ? '' : undefined} onclick={() => type(d)}>{d}</button>
        {/each}
        <button class="btn key" data-nav onclick={erase} aria-label="Apagar número"><Icon name="back" /></button>
        <button class="btn key digit" data-nav onclick={() => type('0')}>0</button>
        <button class="btn key flat" data-nav onclick={() => close(null)}>Cancelar</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Diálogo sóbrio com o PIN em casas de código (input otp da noemi-ui) e
     o teclado numérico grande, pra mirar do sofá. */
  .backdrop {
    position: fixed;
    inset: 0;
    /* tinta (#24170f, o contorno do tema claro) a 55%, sem blur: o creme
       do contorno escuro lavaria a tela */
    background: color-mix(in srgb, var(--on-accent) 55%, transparent);
    display: grid;
    place-items: center;
    padding: var(--safe);
    z-index: 95;
  }
  .dialog {
    display: grid;
    gap: 1.1rem;
    justify-items: center;
    padding: 1.4rem 1.8rem 1.6rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 8px 8px 0 var(--outline);
  }
  h2 {
    font-size: 1.2rem;
  }
  .dots {
    display: flex;
    gap: 0.5rem;
  }
  .cell {
    width: 3rem;
    height: 3.5rem;
    display: grid;
    place-items: center;
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 2.6rem;
    line-height: 1;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
  }
  /* casa da vez: borda e sombra verde-azuladas */
  .cell.next {
    border-color: var(--focus);
    box-shadow: 4px 4px 0 var(--focus);
  }
  .cell.on {
    animation: cell-pop 120ms steps(2);
  }
  .bad .cell {
    border-color: var(--danger);
    box-shadow: 3px 3px 0 var(--danger);
  }
  .shake {
    animation: shake 0.3s steps(4);
  }
  @keyframes cell-pop {
    from {
      transform: scale(0.85);
    }
  }
  @keyframes shake {
    25% {
      transform: translateX(-4px);
    }
    75% {
      transform: translateX(4px);
    }
  }
  .error {
    color: var(--danger);
    font-weight: 700;
    max-width: 18rem;
    text-align: center;
  }
  .keys {
    display: grid;
    grid-template-columns: repeat(3, 5.5rem);
    gap: 0.75rem;
  }
  .key {
    justify-content: center;
    min-height: 3.4rem;
    padding-inline: 0.5rem;
  }
  .key.flat {
    font-size: 0.68rem;
    padding-inline: 0.2rem;
  }
  .digit {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.5rem;
  }
</style>
