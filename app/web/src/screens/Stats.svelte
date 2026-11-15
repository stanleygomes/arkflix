<script lang="ts">
  import BarChart from '../components/BarChart.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { playTime } from '../lib/format'
  import { app, system, toast, currentProfile } from '../lib/state.svelte'
  import type { Stats } from '../lib/types'

  const periods = [
    { id: 'all', label: 'Sempre' },
    { id: '30d', label: '30 dias' },
    { id: '7d', label: '7 dias' },
  ]
  let period = $state('all')
  let data = $state<Stats | null>(null)

  $effect(() => {
    api
      .stats(period)
      .then((s) => (data = s))
      .catch((err) => toast((err as Error).message, 'error'))
  })

  const weekdays = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']
  // tempo total na vida (todos os jogos), fora do período escolhido
  const lifetime = $derived(app.games.reduce((sum, g) => sum + (g.playTime || 0), 0))
  const owned = $derived(app.games.filter((g) => !g.remote).length)
  const dur = (s: number) => (s > 0 ? playTime(s) : '0 min')
  // número gigante: algarismos em texto, unidades (h, min) em mostarda
  const parts = (t: string) => t.split(/(\d+)/).filter(Boolean).map((p) => ({ p, unit: !/^\d/.test(p) }))
  const dayLabel = (d: string) => d.slice(8, 10) + '/' + d.slice(5, 7)
  // embedded: aberta como painel da tela Você (lista sempre visível)
  let { embedded = false }: { embedded?: boolean } = $props()
</script>

<div class="page" class:embedded>
  {#if !embedded}<TopBar actions={false} />{/if}
  <main class="body">
    <p class="summary meta">
      <span>{currentProfile()?.name ?? 'Jogador'}</span>
      <span>{owned} jogo{owned === 1 ? '' : 's'} no PC</span>
      <span>{dur(lifetime)} na vida toda</span>
    </p>
    <header class="head">
      {#if !embedded}<h1>Estatísticas</h1>{/if}
      <div class="segmented" role="group" aria-label="Período">
        {#each periods as p, i (p.id)}
          <button class="btn small" data-nav data-nav-default={i === 0 ? '' : undefined} aria-pressed={period === p.id} onclick={() => (period = p.id)}>
            {p.label}
          </button>
        {/each}
      </div>
    </header>
    {#if data}
      <dl class="tiles">
        <div class="giant"><dt>Tempo jogando</dt><dd>{#each parts(dur(data.totalSeconds)) as t, i (i)}<span class:unit={t.unit}>{t.p}</span>{/each}</dd></div>
        <div class="tile"><dt>Partidas</dt><dd>{data.plays}</dd></div>
        <div class="tile"><dt>Jogos jogados</dt><dd>{data.gamesPlayed} <small>de {data.libraryGames}</small></dd></div>
        <div class="tile">
          <dt>Dia com mais jogo</dt>
          <dd>{data.bestDay.seconds ? `${dayLabel(data.bestDay.date)} · ${dur(data.bestDay.seconds)}` : '—'}</dd>
        </div>
      </dl>
      {#if data.sessions === 0}
        <p class="empty">Nenhuma partida neste período. Jogue alguma coisa e volte aqui.</p>
      {:else}
        <div class="grid">
          <BarChart
            title="Jogos com mais tempo"
            items={data.byGame.slice(0, 10).map((g) => ({
              label: g.name,
              value: g.seconds,
              display: dur(g.seconds),
              detail: `${g.plays} partida${g.plays === 1 ? '' : 's'} · ${system(g.system)?.short ?? g.system}`,
            }))}
          />
          <BarChart
            title="Tempo por console"
            items={data.bySystem
              .filter((s) => s.games > 0 || s.seconds > 0)
              .map((s) => ({
                label: system(s.system)?.name ?? s.system,
                value: s.seconds,
                display: dur(s.seconds),
                detail: `${s.played} de ${s.games} jogos jogados${s.games ? ` (${Math.round((s.played / s.games) * 100)}%)` : ''}`,
              }))}
          />
          <BarChart
            title="Dia da semana"
            vertical
            items={data.byWeekday.map((v, i) => ({ label: weekdays[i], value: v, display: dur(v) }))}
          />
          <BarChart
            title="Últimos 30 dias"
            vertical
            every={5}
            items={data.byDay.map((d) => ({ label: dayLabel(d.date), value: d.seconds, display: dur(d.seconds) }))}
          />
        </div>
      {/if}
    {/if}
  </main>
  {#if !embedded}<HelpBar hints={[{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />{/if}
</div>

<style>
  /* embutida na tela Você: sem barra, sem rodapé, sem rolagem própria */
  .page.embedded {
    height: auto;
    display: block;
  }
  .page.embedded > main {
    padding: 0;
    overflow: visible;
  }

  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.5rem 2.5rem 1.5rem 3.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  .summary {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem 0;
    color: var(--muted);
  }
  .summary span + span::before {
    content: '·' / '';
    margin-inline: 0.7rem;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    flex-wrap: wrap;
  }
  h1 {
    font-size: clamp(2.4rem, 4vw, 3.2rem);
  }
  .segmented {
    display: flex;
    gap: 0.6rem;
  }
  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: 1.25rem;
    margin: 0;
  }
  /* o número gigante da tela (um só): tempo jogando no período. Sem
     moldura, é layout; um fio reto embaixo separa dos cartões */
  .giant {
    grid-column: 1 / -1;
    padding-bottom: 1.1rem;
    border-bottom: var(--edge) solid var(--outline);
  }
  .giant dd {
    font-family: var(--font-label);
    font-size: clamp(44px, 8vw, 92px);
    line-height: 1;
    font-variant-numeric: tabular-nums;
    letter-spacing: 0;
    color: var(--text);
    margin-top: 0.4rem;
  }
  .giant .unit {
    color: var(--accent);
    font-size: 0.55em;
  }
  .tile {
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    padding: 1rem 1.25rem;
  }
  dt {
    font-family: var(--font-hud);
    font-size: 0.8rem;
    letter-spacing: 0.04em;
    color: var(--muted);
  }
  dd {
    margin: 0.35rem 0 0;
    font-family: var(--font-label);
    font-size: 1.5rem;
    line-height: 1.15;
    font-variant-numeric: tabular-nums;
  }
  dd small {
    font-family: var(--font-body);
    font-size: 1rem;
    color: var(--muted);
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(30rem, 1fr));
    gap: 1.75rem;
  }
  /* vazio: caixa tracejada, texto do porquê */
  .empty {
    padding: 2rem 1.25rem;
    max-width: 30rem;
    color: var(--muted);
    font-size: 1.05rem;
    border: var(--edge) dashed var(--muted);
    border-radius: var(--radii-sm);
  }
</style>
