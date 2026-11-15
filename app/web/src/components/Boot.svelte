<script lang="ts">
  // Ao abrir (só no app desktop): a TV liga com o cartão de teste e o som de
  // boot, ~1,6 s (components/Crt.svelte). Qualquer botão pula. Com "reduzir
  // animações", não aparece.
  import { onMount } from 'svelte'
  import { play, unlock } from '../lib/sound'
  import { app } from '../lib/state.svelte'
  import { tubeMoment } from '../lib/tube.svelte'

  onMount(() => {
    if (app.mode !== 'desktop' || app.settings?.a11y.reduceMotion) return
    unlock()
    play('boot')
    tubeMoment('on', 'Jogos · filmes · séries')
  })
</script>
