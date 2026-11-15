<script lang="ts">
  import { play } from '../../lib/sound'
  import { sleep } from '../../lib/idle.svelte'
  import { app } from '../../lib/state.svelte'
  import type { Accessibility, ScreensaverSettings } from '../../lib/types'
  import { saveSoon } from './save.svelte'

  const a = $derived(app.settings!.a11y)
  const scales = [1, 1.25, 1.5, 1.75]

  function set<K extends keyof Accessibility>(key: K, value: Accessibility[K]) {
    saveSoon({ ...app.settings!, a11y: { ...a, [key]: value } })
    play('toggle')
  }

  const saver = $derived(app.settings!.screensaver)
  function setSaver<K extends keyof ScreensaverSettings>(key: K, value: ScreensaverSettings[K]) {
    saveSoon({ ...app.settings!, screensaver: { ...saver, [key]: value } })
    play('toggle')
  }
  const minutes = [1, 5, 10, 15, 30]
  const modes: { id: ScreensaverSettings['mode']; label: string }[] = [
    { id: 'snake', label: 'Cobrinha' },
    { id: 'showcase', label: 'Vitrine' },
  ]

  const switches: { key: keyof Accessibility; label: string; help: string }[] = [
    { key: 'highContrast', label: 'Alto contraste', help: 'Preto, branco e amarelo em toda a interface; ignora a cor dos consoles.' },
    { key: 'reduceMotion', label: 'Reduzir animações', help: 'Sem transições, cursor piscando ou rolagem suave.' },
    { key: 'sounds', label: 'Sons de navegação', help: 'Bipes 16-bit ao mover, confirmar e voltar.' },
    { key: 'music', label: 'Música de fundo', help: 'Trilha chiptune baixa nos menus; para durante o jogo.' },
    { key: 'videos', label: 'Vídeo dos jogos', help: 'No Início, o vídeo do jogo toca no fundo depois de 2 s. Nunca com "Reduzir animações".' },
    { key: 'videoSound', label: 'Som dos vídeos', help: 'Desligado, os vídeos tocam mudos.' },
  ]
</script>

<div class="panel-body">
  <h2>Acessibilidade</h2>

  <fieldset>
    <legend>Tamanho do texto</legend>
    <div class="segmented" role="group" aria-label="Tamanho do texto">
      {#each scales as s (s)}
        <button class="btn" data-nav aria-pressed={a.fontScale === s} onclick={() => set('fontScale', s)}>{Math.round(s * 100)}%</button>
      {/each}
    </div>
  </fieldset>

  <ul class="switches">
    {#each switches as sw (sw.key)}
      <li>
        <button
          class="switch"
          data-nav
          role="switch"
          aria-checked={a[sw.key] as boolean}
          aria-describedby={`help-${sw.key}`}
          onclick={() => set(sw.key, !a[sw.key])}
        >
          <span class="knob" aria-hidden="true"></span>
          <span class="label">{sw.label}</span>
        </button>
        <p class="muted help" id={`help-${sw.key}`}>{sw.help}</p>
      </li>
    {/each}
  </ul>

  {#each [{ key: 'soundVolume' as const, label: 'Volume dos sons' }, { key: 'musicVolume' as const, label: 'Volume da música' }] as v (v.key)}
    <fieldset>
      <legend>{v.label}</legend>
      <div class="stepper">
        <button class="btn small" data-nav aria-label={`Diminuir ${v.label.toLowerCase()}`} onclick={() => set(v.key, Math.max(0, a[v.key] - 10))}>−</button>
        <span class="meter" role="meter" aria-label={v.label} aria-valuenow={a[v.key]} aria-valuemin="0" aria-valuemax="100">
          {#each Array(10) as _, i (i)}<span class="bar" class:on={a[v.key] > i * 10}></span>{/each}
        </span>
        <span class="level">{a[v.key]}</span>
        <button class="btn small" data-nav aria-label={`Aumentar ${v.label.toLowerCase()}`} onclick={() => set(v.key, Math.min(100, a[v.key] + 10))}>+</button>
      </div>
    </fieldset>
  {/each}

  <fieldset>
    <legend>Protetor de tela</legend>
    <div class="saver">
      <button class="switch" data-nav role="switch" aria-checked={saver.enabled} aria-describedby="help-saver" onclick={() => setSaver('enabled', !saver.enabled)}>
        <span class="knob" aria-hidden="true"></span>
        <span class="label">Ligar quando parado</span>
      </button>
      <p class="muted help" id="help-saver">A cobrinha joga sozinha, ou as capas passam como cartaz de cinema. Nunca com jogo aberto; qualquer botão volta.</p>
      {#if saver.enabled}
        <div class="segmented" role="group" aria-label="Minutos parado">
          {#each minutes as m (m)}
            <button class="btn" data-nav aria-pressed={saver.minutes === m} onclick={() => setSaver('minutes', m)}>{m} min</button>
          {/each}
        </div>
        <div class="segmented" role="group" aria-label="Estilo do protetor">
          {#each modes as m (m.id)}
            <button class="btn" data-nav aria-pressed={saver.mode === m.id} onclick={() => setSaver('mode', m.id)}>{m.label}</button>
          {/each}
          <button class="btn" data-nav onclick={() => sleep()}>Ver agora</button>
        </div>
        <div class="segmented" role="group" aria-label="Descanso: tela preta depois de">
          {#each [0, 15, 30, 60] as r (r)}
            <button class="btn" data-nav aria-pressed={(saver.rest ?? 0) === r} onclick={() => setSaver('rest', r)}>{r ? `Descanso em ${r} min` : 'Sem descanso'}</button>
          {/each}
        </div>
      {/if}
    </div>
  </fieldset>
</div>

<style>
  .saver {
    display: grid;
    gap: 0.9rem;
  }
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.6rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  /* grupo: sem caixa, legenda em Bevan */
  fieldset {
    border: none;
    padding: 0;
    margin: 0;
  }
  legend {
    padding: 0;
    margin-bottom: 0.6rem;
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
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
  .stepper {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .switches {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.9rem;
  }
  /* interruptor: trilho de 3px, botão que pula pro lado; ligado = verde-azulado */
  .switch {
    display: flex;
    gap: 0.9rem;
    align-items: center;
    width: fit-content;
    min-height: 3rem;
    padding: 0.3rem 0.5rem;
    font: inherit;
    font-weight: 700;
    color: var(--text);
    background: none;
    border: none;
    border-radius: var(--radii-sm);
    cursor: pointer;
  }
  .knob {
    position: relative;
    flex: none;
    width: 3.4rem;
    height: 1.8rem;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 999px;
    box-shadow: 2px 2px 0 var(--outline);
  }
  .knob::after {
    content: '';
    position: absolute;
    top: 50%;
    inset-inline-start: 0.2rem;
    width: 1.05rem;
    height: 1.05rem;
    translate: 0 -50%;
    background: var(--outline);
    border-radius: var(--radii-sm);
    transition: inset-inline-start 150ms steps(3);
  }
  .switch[aria-checked='true'] .knob {
    background: var(--accent-2);
  }
  .switch[aria-checked='true'] .knob::after {
    inset-inline-start: calc(100% - 1.25rem);
    background: var(--bg);
  }
  .switch:hover .knob {
    box-shadow: 3px 3px 0 var(--outline);
  }
  /* rótulo do interruptor em Alegreya (o .label global é Bevan) */
  .switch .label {
    font-family: var(--font-body);
    font-size: 1.05rem;
    font-weight: 700;
    letter-spacing: 0;
    text-transform: none;
  }
  .help {
    margin-inline-start: 4.9rem;
    font-size: 0.9rem;
  }
  /* volume: trilho de 10 degraus; os ligados em mostarda, número escrito do lado */
  .meter {
    display: inline-flex;
    gap: 3px;
    padding: 3px;
    background: var(--balloon);
    border: var(--edge) solid var(--outline);
    border-radius: 4px 5px 4px 5px;
  }
  .bar {
    width: 0.8rem;
    height: 1.2rem;
    border-radius: 2px;
    background: color-mix(in srgb, var(--outline) 14%, transparent);
  }
  .bar.on {
    background: var(--accent);
  }
  .level {
    min-width: 2.5em;
    font-family: var(--font-mono);
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    text-align: end;
  }
</style>
