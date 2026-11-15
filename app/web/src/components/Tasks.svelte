<script lang="ts">
  // Tarefas longas em andamento (downloads, capas, instalação): barra de
  // progresso flutuante. O aviso do fim e o histórico ficam na central.
  import { activity } from '../lib/activity.svelte'
  import { app } from '../lib/state.svelte'

  // na própria central a barra seria repetida
  const tasks = $derived(app.stack[app.stack.length - 1].name === 'activity' ? [] : activity.tasks.filter((t) => t.state === 'running').slice(0, 3))
</script>

{#if tasks.length}
  <div class="tasks" aria-live="polite">
    {#each tasks as t (t.id)}
      <div class="task">
        <span class="name">{t.label}…</span>
        <div class="energy" role="progressbar" aria-label={t.label} aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(t.progress * 100)}>
          <span style:width={`${Math.floor(t.progress * 20) * 5}%`}></span>
        </div>
        <span class="pct">{Math.round(t.progress * 100)}%</span>
      </div>
    {/each}
  </div>
{/if}

<style>
  /* Cartão sóbrio por tarefa: nome, barra que anda em degraus (.energy) e
     a porcentagem datilografada. */
  .tasks {
    position: fixed;
    inset-inline-start: calc(1rem + var(--safe));
    bottom: calc(4.5rem + var(--safe));
    z-index: 50;
    display: grid;
    gap: 0.5rem;
    max-width: calc(100vw - 2rem);
  }
  .task {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard);
    padding: 0.55rem 0.9rem;
  }
  .name {
    font-weight: 700;
    max-width: 16rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .energy {
    width: 12rem;
    flex: none;
  }
  .pct {
    font-family: var(--font-hud);
    font-size: 0.85rem;
    min-width: 3ch;
    text-align: end;
  }
</style>
