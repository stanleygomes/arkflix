<script lang="ts">
  import ControllerPreview from '../../components/ControllerPreview.svelte'
  import Glyph from '../../components/Glyph.svelte'
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { onPad, pushHandler, setCapture, type Captured } from '../../lib/input'
  import { glyph, keyLabel, retroArchKey } from '../../lib/keys'
  import { play } from '../../lib/sound'
  import { app, confirm, toast } from '../../lib/state.svelte'
  import type { Family, Mapping } from '../../lib/types'

  type Mode = 'xbox' | 'playstation' | '8bitdo' | 'keyboard'
  const modes: { id: Mode; label: string }[] = [
    { id: 'xbox', label: 'Xbox' },
    { id: 'playstation', label: 'PlayStation' },
    { id: '8bitdo', label: '8BitDo / SNES' },
    { id: 'keyboard', label: 'Teclado' },
  ]
  const firstPad = app.pads[0]?.family
  let mode = $state<Mode>(firstPad === 'playstation' || firstPad === '8bitdo' ? firstPad : firstPad === 'nintendo' ? '8bitdo' : 'xbox')
  let pressed = $state(new Set<string>())
  let focusedBtn = $state('')
  let waiting = $state('')
  let conflicts = $state<string[]>([])
  let timeout: number | undefined

  const mapping = $derived(app.input!)
  const family = $derived<Family>(mode)
  const highlight = $derived(waiting ? '' : mode === 'keyboard' ? '' : (mapping.pad[focusedBtn] ?? ''))

  $effect(() =>
    onPad((e) => {
      const next = new Set(pressed)
      if (e.pressed) next.add(e.button)
      else next.delete(e.button)
      pressed = next
      if (e.pressed && mode !== 'keyboard' && !waiting) {
        mode = e.family === 'playstation' ? 'playstation' : e.family === '8bitdo' || e.family === 'nintendo' ? '8bitdo' : 'xbox'
      }
    }),
  )

  $effect(() => () => {
    setCapture(null)
    clearTimeout(timeout)
  })

  // no controle, L1/R1/X/Y/Start servem pra testar a prévia: não trocam de seção
  // nem abrem nada (no teclado, Q/E continuam trocando de seção)
  $effect(() =>
    pushHandler((a, src) => src === 'pad' && (a === 'prevTab' || a === 'nextTab' || a === 'fav' || a === 'details' || a === 'menu')),
  )

  async function save(next: Mapping) {
    try {
      const res = await api.saveInput(next)
      app.input = res.mapping
      conflicts = res.conflicts
      play(res.conflicts.length ? 'error' : 'accept')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function stopWaiting() {
    waiting = ''
    setCapture(null)
    clearTimeout(timeout)
  }

  function remap(id: string, label: string) {
    waiting = id
    play('toggle')
    timeout = window.setTimeout(() => {
      stopWaiting()
      toast('Tempo esgotado: nada mudou', 'info', 2500)
    }, 6000)
    setCapture((c: Captured) => {
      stopWaiting()
      if (c.kind === 'pad') {
        if (mode === 'keyboard') return toast('Modo teclado: aperte uma tecla, não um botão', 'error')
        save({ ...mapping, pad: { ...mapping.pad, [id]: c.button } })
        toast(`${label} agora é ${glyph(c.button, family)}`, 'ok', 2500)
      } else {
        if (c.code === 'Escape') return toast('Cancelado', 'info', 2000)
        if (mode !== 'keyboard') return toast('Modo controle: aperte um botão do controle (ou troque pra Teclado)', 'error')
        const key = retroArchKey(c.code)
        if (!key) return toast('Essa tecla não pode ser usada no RetroArch', 'error')
        save({ ...mapping, keys: { ...mapping.keys, [id]: key } })
        toast(`${label} agora é ${keyLabel(key)}`, 'ok', 2500)
      }
    })
  }

  async function reset() {
    if (!(await confirm('Voltar controle e teclado pro mapeamento padrão?', 'Restaurar'))) return
    const res = await api.resetInput()
    app.input = res.mapping
    conflicts = res.conflicts
    toast('Mapeamento padrão restaurado', 'ok')
  }
</script>

<div class="panel-body">
  <h2>Controles</h2>
  <p class="muted">
    {app.pads.length
      ? `Conectado: ${app.pads.map((p) => p.name).join(', ')}.`
      : 'Nenhum controle conectado agora. Dá pra configurar mesmo assim, e o teclado sempre funciona.'}
    Este mapeamento vale pros jogos do RetroArch (NES, SNES, Mega Drive, GBC, N64, PS1, DS e arcade). PSP e PS2 usam as posições
    padrão do controle.
  </p>

  <div class="segmented" role="group" aria-label="Tipo de controle">
    {#each modes as m (m.id)}
      <button class="btn small" data-nav aria-pressed={mode === m.id} onclick={() => (mode = m.id)}>{m.label}</button>
    {/each}
  </div>

  <div class="grid">
    <div class="preview">
      {#if mode === 'keyboard'}
        <div class="keyboard" aria-label="Teclas configuradas">
          {#each app.retroPad as b (b.id)}
            <span class="keycap" class:focus={focusedBtn === b.id}><kbd>{keyLabel(mapping.keys[b.id])}</kbd><small>{b.label}</small></span>
          {/each}
        </div>
      {:else}
        <ControllerPreview {family} {pressed} {highlight} />
      {/if}
      <p class="live muted" aria-live="polite">
        {#if waiting}
          <strong class="wait">Aperte {mode === 'keyboard' ? 'a tecla' : 'o botão'} pra {app.retroPad.find((b) => b.id === waiting)?.label}…</strong>
        {:else if pressed.size}
          Apertado: {[...pressed].map((b) => glyph(b, family)).join(' + ')}
        {:else}
          Aperte qualquer botão do controle pra testar.
        {/if}
      </p>
    </div>

    <ul class="bindings" aria-label="Mapeamento do RetroPad">
      {#each app.retroPad as b (b.id)}
        <li>
          <button
            class="binding"
            class:waiting={waiting === b.id}
            data-nav
            data-retropad={b.id}
            aria-label={`${b.label}: ${mode === 'keyboard' ? keyLabel(mapping.keys[b.id]) : glyph(mapping.pad[b.id], family)}. Aperte pra trocar`}
            onfocus={() => (focusedBtn = b.id)}
            onclick={() => remap(b.id, b.label)}
          >
            <span>{b.label}</span>
            {#if mode === 'keyboard'}
              <kbd class="key">{keyLabel(mapping.keys[b.id])}</kbd>
            {:else}
              <Glyph button={mapping.pad[b.id]} {family} />
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  </div>

  {#if conflicts.length}
    <div class="conflicts" role="alert">
      <Icon name="warn" /> <span>Botão usado mais de uma vez: {conflicts.join('; ')}</span>
    </div>
  {/if}

  <div class="actions">
    <button class="btn" data-nav onclick={reset}>Restaurar padrão</button>
  </div>
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  /* grupo de alternância numa peça só (toggle group da noemi-ui) */
  .segmented {
    display: flex;
    flex-wrap: wrap;
    width: fit-content;
    max-width: 100%;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    overflow: hidden;
  }
  .segmented .btn {
    border: none;
    border-inline-end: var(--edge) solid var(--outline);
    border-radius: 0;
    box-shadow: none;
    transform: none;
    background: none;
  }
  .segmented .btn:last-child {
    border-inline-end: none;
  }
  .segmented .btn:hover {
    background: color-mix(in srgb, var(--outline) 12%, var(--surface));
  }
  .segmented .btn[aria-pressed='true'] {
    background: var(--accent-2);
    color: var(--bg);
    box-shadow: inset 3px 3px 0 color-mix(in srgb, var(--outline) 45%, transparent);
  }
  .segmented .btn[data-nav]:focus-visible {
    outline-offset: -7px;
  }
  .segmented .btn[aria-pressed='true'][data-nav]:focus-visible {
    outline-color: var(--bg);
  }
  .grid {
    display: grid;
    grid-template-columns: minmax(16rem, 1.2fr) minmax(16rem, 1fr);
    gap: 2rem;
    align-items: start;
  }
  .preview {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    position: sticky;
    top: 0;
    padding: 1.25rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
  }
  .live {
    text-align: center;
    min-height: 1.6em;
  }
  .wait {
    font-family: var(--font-label);
    font-weight: 400;
    color: var(--accent);
  }
  /* mapeamento: lista de itens, cada linha um botão; a que espera tecla fica
     com fundo mostarda 22% e barra na borda */
  .bindings {
    list-style: none;
    margin: 0;
    padding: 0;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
  }
  .bindings li + li {
    border-top: 2px solid color-mix(in srgb, var(--outline) 25%, transparent);
  }
  .binding {
    width: 100%;
    min-height: 3rem;
    padding: 0.45rem 0.9rem;
    font: inherit;
    color: var(--text);
    background: none;
    border: none;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    cursor: pointer;
  }
  .binding:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .binding[data-nav]:focus-visible {
    outline-offset: -4px;
  }
  .binding.waiting {
    background: color-mix(in srgb, var(--accent) 22%, transparent);
    box-shadow: inset 4px 0 0 var(--accent);
  }
  /* tecla (kbd da noemi-ui): Courier Prime, caixa de balão com sombra dura */
  .key,
  .keycap kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 1.8rem;
    min-height: 1.8rem;
    padding: 0 0.45rem;
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 0.85rem;
    line-height: 1;
    color: var(--text);
    background: var(--balloon);
    border: 2px solid var(--outline);
    border-radius: 4px 5px 4px 5px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .keyboard {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.6rem;
  }
  .keycap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
    padding: 0.45rem;
    border: 2px solid transparent;
    border-radius: var(--radii-sm);
  }
  .keycap.focus {
    background: color-mix(in srgb, var(--accent) 22%, transparent);
    border-color: var(--accent);
  }
  .keycap.focus kbd {
    transform: translate(2px, 2px);
    box-shadow: none;
    color: var(--on-accent);
    background: var(--accent);
  }
  .keycap small {
    font-family: var(--font-hud);
    font-size: 0.75rem;
    text-align: center;
    color: var(--muted);
  }
  /* alerta sóbrio de aviso: barra mostarda de 8px por dentro da borda */
  .conflicts {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    padding: 0.75rem 1rem 0.75rem 1.4rem;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard), inset 8px 0 0 var(--accent);
  }
  .conflicts :global(svg) {
    flex: none;
    color: var(--accent);
  }
  .actions {
    display: flex;
    gap: 0.8rem;
  }
</style>
