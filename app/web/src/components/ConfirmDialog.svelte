<script lang="ts">
  // Diálogo de confirmação. [data-nav-scope] prende a navegação aqui dentro;
  // "voltar" cancela.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { app } from '../lib/state.svelte'

  const c = $derived(app.confirm)
  const now = $derived(c ? new Date().toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }) : '')
  let previous: HTMLElement | null = null

  function close(v: boolean) {
    const cur = app.confirm
    app.confirm = null
    cur?.resolve(v)
    previous?.focus()
  }

  $effect(() => {
    if (!c) return
    previous = document.activeElement as HTMLElement | null
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        close(false)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })
</script>

{#if c}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="dialog" in:panelIn out:panelOut role="alertdialog" aria-modal="true" aria-labelledby="confirm-text" data-nav-scope>
      <div class="body">
        {#if c.danger}
          <!-- 🎬 confirmação destrutiva escrita como telegrama -->
          <div class="telegram sketched">
            <p class="band" aria-hidden="true">Telegrama</p>
            <p id="confirm-text" class="wire">{c.text}</p>
            <p class="from meta" aria-hidden="true">arkflix · {now}</p>
          </div>
        {:else}
          <p id="confirm-text" class="text">{c.text}</p>
        {/if}
      </div>
      <div class="actions">
        <button class="btn" data-nav data-nav-default onclick={() => close(false)}>Cancelar</button>
        <button class="btn" class:danger={c.danger} class:primary={!c.danger} data-nav onclick={() => close(true)}>{c.ok}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Diálogo sóbrio (noemi-ui): fundo de tinta a 55%, sem blur; painel de
     cartão com borda reta e sombra dura; rodapé com linha e botões à
     direita, o perigoso por último. */
  .backdrop {
    position: fixed;
    inset: 0;
    /* tinta (#24170f, o contorno do tema claro) a 55%, sem blur: o creme
       do contorno escuro lavaria a tela */
    background: color-mix(in srgb, var(--on-accent) 55%, transparent);
    display: grid;
    place-items: center;
    padding: var(--safe);
    z-index: 100;
  }
  .dialog {
    width: min(34rem, 100%);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: 8px 8px 0 var(--outline);
  }
  .body {
    padding: 1.4rem 1.4rem 1.6rem;
  }
  .text {
    font-size: 1.15rem;
  }
  .actions {
    display: flex;
    gap: 0.75rem;
    justify-content: flex-end;
    padding: 1rem 1.25rem;
    border-block-start: var(--edge) solid var(--outline);
  }

  /* 🎬 telegrama: faixa escura em cima, texto datilografado, "STOP." */
  .telegram {
    background: var(--balloon);
    border: var(--edge) solid transparent;
    overflow: hidden;
    max-width: 46ch;
  }
  .band {
    font-family: var(--font-label);
    font-size: 0.78rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    background: var(--text);
    color: var(--bg);
    padding: 0.35rem 0.75rem;
  }
  .wire {
    font-family: var(--font-hud);
    font-size: 0.95rem;
    line-height: 1.6;
    text-transform: uppercase;
    padding: 0.8rem 0.9rem 0.3rem;
    animation: wire 0.9s steps(9) both;
  }
  .wire::after {
    content: ' STOP.' / '';
  }
  .from {
    font-size: 0.75rem;
    color: var(--muted);
    text-align: end;
    padding: 0 0.9rem 0.6rem;
  }
  /* bate à máquina uma vez, sem travar os botões */
  @keyframes wire {
    from {
      clip-path: inset(0 100% 0 0);
    }
    to {
      clip-path: inset(0);
    }
  }
</style>
