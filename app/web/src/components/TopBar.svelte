<script lang="ts">
  import type { Snippet } from 'svelte'
  import { unread } from '../lib/activity.svelte'
  import { versionArrow } from '../lib/update'
  import { app, back, currentProfile, glyphFamily, go, isKids } from '../lib/state.svelte'
  import { play } from '../lib/sound'
  import Avatar from './Avatar.svelte'
  import Glyph from './Glyph.svelte'
  import Icon from './Icon.svelte'
  import WindowControls from './WindowControls.svelte'
  import { api } from '../lib/api'

  let { children, actions = true }: { children?: Snippet; actions?: boolean } = $props()
  let now = $state(new Date())
  $effect(() => {
    const t = setInterval(() => (now = new Date()), 15000)
    return () => clearInterval(t)
  })
  const me = $derived(currentProfile())
  const news = $derived(unread())
  const clock = $derived(now.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }))
  // "Seg, 02 de nov" (só a primeira letra maiúscula)
  const date = $derived.by(() => {
    const d = now.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short' }).replace(/\./g, '')
    return d.charAt(0).toUpperCase() + d.slice(1)
  })
  const canBack = $derived(app.stack.length > 1)
  // voltando pro Início, o botão diz isso (fica claro onde ele leva)
  const toHome = $derived(app.stack.length > 1 && app.stack[app.stack.length - 2].name === 'home')
  function goBack() {
    play('back')
    back()
  }
  // a barra é a de título da janela: duplo clique no vazio maximiza/restaura
  function onTitleDblClick(e: MouseEvent) {
    if (app.mode !== 'desktop' || (e.target as HTMLElement).closest('button, a, input')) return
    api.toggleMaximise().catch(() => {})
  }
  const padLabel = $derived(
    app.pads.length === 0 ? 'Nenhum controle conectado' : `${app.pads.length} controle(s): ${app.pads.map((p) => p.name).join(', ')}`,
  )
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<header class="top" ondblclick={onTitleDblClick}>
  {#if canBack}
    <!-- voltar sempre no mesmo lugar, em toda tela que veio de outra -->
    <button class="back-btn" data-nav onclick={goBack}>
      <Icon name={toHome ? 'home' : 'back'} size="1.2em" />
      <span class="sub">{toHome ? 'Início' : 'Voltar'}</span>
      <span aria-hidden="true"><Glyph button="east" family={glyphFamily()} /></span>
    </button>
  {/if}
  <div class="brand" aria-hidden="true">Ark<span>flix</span></div>
  <div class="middle">{@render children?.()}</div>
  <div class="right">
    {#if app.update?.available && !isKids()}
      <!-- aviso de versão nova: leva pra aba Atualizações -->
      <button class="upd-btn" data-nav title="Atualização disponível" onclick={() => go({ name: 'settings', tab: 'updates' })}>
        <Icon name="download" size="1.2em" /> <span>Atualizar</span> <span class="ver">{versionArrow(app.update)}</span>
      </button>
    {/if}
    {#if actions}
      <!-- como o canto do PS5: busca e avisos em blocos iguais, perfil com nome -->
      <button class="icon-btn" data-nav aria-label="Buscar jogos" title="Buscar" onclick={() => go({ name: 'search' })}>
        <Icon name="search" size="1.35em" />
      </button>
      <button class="icon-btn" data-nav aria-label={news ? `Central: ${news} aviso${news === 1 ? '' : 's'} novo${news === 1 ? '' : 's'}` : 'Central de downloads e avisos'} title="Central" onclick={() => go({ name: 'activity' })}>
        <Icon name="bell" size="1.35em" />
        {#if news}<span class="dot" aria-hidden="true"></span>{/if}
      </button>
      {#if me}
        <!-- o perfil abre "Você": perfil, configurações, estatísticas, energia -->
        <button class="profile-btn" data-nav aria-label={`Você: ${me.name} (perfil e configurações)`} onclick={() => go({ name: 'you' })}>
          <Avatar avatar={me.avatar} color={me.color} size="2.4rem" />
          <span class="who sub">{me.name}</span>
        </button>
      {/if}
      <span class="sep" aria-hidden="true"></span>
    {/if}
    <span class="status" class:on={app.pads.length > 0} role="img" aria-label={padLabel} title={padLabel}>
      <Icon name="pad" size="1.3em" />
      {#if app.pads.length}<span class="count">{app.pads.length}</span>{/if}
    </span>
    {#if app.mode === 'desktop'}
      <!-- energia sempre à mão: minimizar, tela cheia, fechar -->
      <button class="icon-btn power" data-nav aria-label="Energia" title="Energia" onclick={() => (app.power = true)}>
        <Icon name="power" size="1.35em" />
      </button>
    {/if}
    <span class="when">
      <time class="clock sub" datetime={now.toISOString()}>{clock}</time>
      <span class="date">{date}</span>
    </span>
    <WindowControls />
  </div>
</header>

<style>
  /* arrasta a janela pela barra (desktop sem moldura); o que é clicável não */
  .top {
    --wails-draggable: drag;
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 1rem 2.5rem 0.9rem;
    min-height: 4.5rem;
    border-bottom: var(--edge) solid color-mix(in srgb, var(--outline) 22%, transparent);
  }
  .top :global(button),
  .top :global(a),
  .top :global(input) {
    --wails-draggable: no-drag;
  }
  /* marca: letra de cartaz, sem brilho */
  .brand {
    font-family: var(--font-title);
    font-size: 1.8rem;
    line-height: 1;
    white-space: nowrap;
    letter-spacing: 0.02em;
    color: var(--text);
  }
  .brand span {
    color: var(--accent);
  }
  .middle {
    flex: 1;
    min-width: 0;
  }
  .right {
    display: flex;
    align-items: center;
    gap: 0.9rem;
  }
  /* botões da barra: sóbrios (cartão, borda reta, sombra dura pequena) */
  .icon-btn,
  .back-btn,
  .profile-btn,
  .upd-btn {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    height: 3rem;
    font-family: var(--font-label);
    font-size: 0.85rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    line-height: 1;
    color: var(--text);
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    box-shadow: var(--hard-sm);
    cursor: pointer;
    transition:
      transform var(--t) var(--ease),
      box-shadow var(--t) var(--ease);
  }
  .icon-btn:hover,
  .back-btn:hover,
  .profile-btn:hover,
  .upd-btn:hover,
  .top [data-nav]:focus {
    transform: translate(-2px, -2px);
    box-shadow: var(--hard-up);
  }
  .icon-btn:active,
  .back-btn:active,
  .profile-btn:active,
  .upd-btn:active {
    transform: translate(3px, 3px);
    box-shadow: none;
  }
  .icon-btn {
    width: 3rem;
    padding: 0;
  }
  .back-btn {
    padding: 0 0.9rem;
  }
  /* tecla do atalho (kbd da noemi-ui), sem a cor do botão do controle */
  .back-btn :global(kbd.glyph) {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 0.75rem;
    line-height: 1;
    min-width: 1.6rem;
    height: 1.6rem;
    padding: 0 0.35rem;
    color: var(--text);
    background: var(--balloon);
    border: 2px solid var(--outline);
    border-radius: 4px 5px 4px 5px;
    box-shadow: 1px 1px 0 var(--outline);
  }
  .back-btn :global(kbd.glyph.round) {
    padding: 0;
    width: 1.6rem;
    border-radius: 50%;
  }
  .profile-btn {
    padding: 0 1rem 0 0.25rem;
  }
  /* versão nova = ação: mostarda, parado (nada pisca em loop) */
  .upd-btn {
    padding: 0 1rem;
    color: var(--on-accent);
    background: var(--accent);
  }
  .upd-btn .ver {
    font-family: var(--font-mono);
    font-weight: 700;
    text-transform: none;
    letter-spacing: 0;
  }
  /* aviso novo na Central: ponto mostarda no canto */
  .dot {
    position: absolute;
    top: -0.45rem;
    inset-inline-end: -0.45rem;
    width: 0.9rem;
    height: 0.9rem;
    background: var(--accent);
    border: 2px solid var(--bg);
    border-radius: 50%;
  }
  .who {
    font-size: 0.85rem;
    max-width: 9rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sep {
    width: var(--edge);
    height: 2rem;
    background: color-mix(in srgb, var(--outline) 22%, transparent);
  }
  .status {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    color: var(--muted);
  }
  /* controle conectado = ok (verde-azulado) */
  .status.on {
    color: var(--accent-2);
  }
  .count {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.1rem;
  }
  .when {
    display: grid;
    justify-items: end;
    line-height: 1;
    gap: 0.3rem;
  }
  .clock {
    font-size: 1.45rem;
    font-variant-numeric: tabular-nums;
    color: var(--text);
  }
  .date {
    font-family: var(--font-hud);
    font-size: 0.8rem;
    letter-spacing: 0.04em;
    color: var(--muted);
  }
</style>
