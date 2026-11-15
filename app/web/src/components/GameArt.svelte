<script lang="ts" module>
  // nome da mídia, datilografado no rodapé do cartaz
  const MEDIA: Record<string, string> = {
    cartridge: 'cartucho',
    disc: 'disco',
    umd: 'UMD',
    card: 'cartão',
    arcade: 'fliperama',
  }
</script>

<script lang="ts">
  // Capa do jogo. Sem capa, um cartaz de cinema antigo: moldura dupla,
  // console datilografado em cima, título em letra de cartaz, mídia e ano
  // embaixo. Só creme, mostarda e o fundo de cartão (sem cor por console).
  import type { Game, System } from '../lib/types'

  let { game, system, large = false }: { game: Game; system?: System; large?: boolean } = $props()
  let failed = $state(false)
  const media = $derived(MEDIA[system?.media ?? 'cartridge'] ?? 'cartucho')
  // tamanho do título pela palavra mais comprida: cabe sem quebrar no meio
  const longest = $derived(Math.max(4, ...game.name.split(/\s+/).map((w) => w.length)))
  const fit = $derived(Math.min(13, 68 / (0.74 * longest)))
</script>

<div class="art" class:large style:--fit={fit}>
  {#if game.cover && !failed}
    <img src={game.cover} alt="" loading="lazy" decoding="async" onerror={() => (failed = true)} />
  {:else}
    <!-- texto em conteúdo gerado (CSS): é enfeite, quem lê o nome é o rótulo
         do cartão em volta (sem duplicar nem brigar com o nome acessível) -->
    <div class="poster" aria-hidden="true">
      <span class="p-tag" data-text={system?.short ?? game.system}></span>
      <span class="p-name" data-text={game.name}></span>
      <span class="p-rule"></span>
      <span class="p-foot" data-text={`${media}${game.year ? ` · ${game.year}` : ''}`}></span>
    </div>
  {/if}
</div>

<style>
  .art {
    container-type: inline-size;
    aspect-ratio: 3 / 4;
    width: 100%;
    position: relative;
    background: var(--surface);
    border: var(--edge) solid var(--outline);
    border-radius: var(--radii-sm);
    overflow: hidden;
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  /* moldura dupla de cartaz: linha fina por dentro da borda */
  .poster {
    position: absolute;
    inset: 6cqi;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: space-between;
    gap: 4cqi;
    padding: 7cqi 6cqi 6cqi;
    text-align: center;
    border: 3px double color-mix(in srgb, var(--outline) 55%, transparent);
    background:
      radial-gradient(ellipse at 50% 35%, color-mix(in srgb, var(--accent) 10%, transparent), transparent 70%);
  }
  .p-tag::before,
  .p-name::before,
  .p-foot::before {
    content: attr(data-text);
  }
  .p-tag {
    font-family: var(--font-hud);
    font-size: clamp(0.6rem, 9cqi, 1.1rem);
    letter-spacing: 0.1em;
    text-transform: uppercase;
    line-height: 1;
    color: var(--text);
    border: 2px solid var(--outline);
    border-radius: 4px;
    padding: 0.3em 0.5em 0.15em;
  }
  .p-name {
    flex: 1;
    display: flex;
    align-items: center;
    font-family: var(--font-title);
    font-size: clamp(0.7rem, calc(var(--fit) * 1cqi), 2.4rem);
    line-height: 1.05;
    color: var(--accent);
    overflow-wrap: break-word;
    overflow: hidden;
  }
  /* fio com estrela no meio */
  .p-rule {
    align-self: stretch;
    display: flex;
    justify-content: center;
    color: var(--muted);
    font-size: clamp(0.5rem, 7cqi, 0.9rem);
    line-height: 1;
    background: linear-gradient(color-mix(in srgb, var(--outline) 45%, transparent), color-mix(in srgb, var(--outline) 45%, transparent)) center / 100% 2px no-repeat;
  }
  .p-rule::after {
    content: '★';
    padding: 0 3cqi;
    background: var(--surface);
  }
  .p-foot {
    font-family: var(--font-hud);
    font-size: clamp(0.55rem, 7.5cqi, 0.95rem);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    line-height: 1;
    color: var(--muted);
  }
</style>
