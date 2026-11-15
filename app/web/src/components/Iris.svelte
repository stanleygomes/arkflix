<script lang="ts">
  // 🎬 Íris de desenho antigo: ao recomeçar a navegação (trocar de perfil,
  // entrar no app), a tela nova abre num círculo que cresce. Só nesses
  // momentos grandes; com "reduzir animações", não aparece.
  import { reduced } from '../lib/motion'
  import { app } from '../lib/state.svelte'

  let run = $state(0)
  let last = app.nav.seq

  $effect(() => {
    const { seq, dir } = app.nav
    if (seq === last) return
    last = seq
    if (dir !== 'reset' || reduced()) return
    run++
    const mine = run
    setTimeout(() => {
      if (run === mine) run = -mine
    }, 480)
  })
</script>

{#if run > 0}
  {#key run}<div class="iris" aria-hidden="true"></div>{/key}
{/if}

<style>
  .iris {
    position: fixed;
    left: 50%;
    top: 50%;
    z-index: 9000;
    width: 0;
    height: 0;
    border-radius: 50%;
    pointer-events: none;
    box-shadow: 0 0 0 300vmax #0b0806;
    transform: translate(-50%, -50%);
    animation: iris-open 0.45s steps(7) forwards;
  }
  @keyframes iris-open {
    to {
      width: 300vmax;
      height: 300vmax;
    }
  }
</style>
