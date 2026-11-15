// Animações de tela e diálogo (noemi-ui): em degraus, curtas, ligadas a um
// evento e desligadas com "reduzir animações" (do app ou do sistema). O que
// sai fica inerte: o foco do controle nunca cai numa tela de saída.

export function reduced(): boolean {
  return (
    document.documentElement.classList.contains('reduce-motion') ||
    (typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches)
  )
}

/** Arredonda o progresso em n degraus (animação desenhada, quadro a quadro). */
export const stepped = (t: number, n: number) => Math.min(1, Math.floor(t * n) / n)

type Css = { duration: number; delay?: number; css?: (t: number, u: number) => string }

export type ScreenMotion = { kind: 'swap' | 'none'; dir: 1 | -1 }

/** Entrada de tela: chega 8px de lado (na direção da navegação), em 3 degraus. */
export function screenIn(_node: Element, p: ScreenMotion): Css {
  if (reduced() || p.kind === 'none') return { duration: 0 }
  return {
    duration: 150,
    css: (t) => `transform: translateX(${(1 - stepped(t, 3)) * 8 * p.dir}px); opacity: ${stepped(t, 3)};`,
  }
}

/** Saída de tela: apaga em 2 degraus e fica inerte. */
export function screenOut(node: Element, p: ScreenMotion): Css {
  ;(node as HTMLElement).inert = true
  if (reduced() || p.kind === 'none') return { duration: 0 }
  return { duration: 100, css: (t) => `opacity: ${stepped(t, 2)};` }
}

/** Diálogo: cai de 12px em 3 degraus. */
export function panelIn(_node: Element): Css {
  if (reduced()) return { duration: 0 }
  return { duration: 180, css: (t) => `transform: translateY(${(1 - stepped(t, 3)) * -12}px); opacity: ${stepped(t, 3)};` }
}

export function panelOut(node: Element): Css {
  ;(node as HTMLElement).inert = true
  if (reduced()) return { duration: 0 }
  return { duration: 120, css: (t) => `transform: translateY(${(1 - stepped(t, 2)) * -12}px); opacity: ${stepped(t, 2)};` }
}

/** Fundo dos diálogos: escurece em degraus. */
export function shade(node: Element, { out = false } = {}): Css {
  if (out) (node as HTMLElement).inert = true
  if (reduced()) return { duration: 0 }
  return { duration: 150, css: (t) => `opacity: ${stepped(t, 3)};` }
}

/** Estrela que explode em pixels a partir do elemento (favoritar). */
export function burst(from: Element | null | undefined, color = 'var(--focus)') {
  if (!from || reduced()) return
  const r = from.getBoundingClientRect()
  const cx = r.left + r.width / 2
  const cy = r.top + r.height / 2
  for (let i = 0; i < 8; i++) {
    const a = (Math.PI * 2 * i) / 8
    const px = document.createElement('span')
    px.className = 'burst-pixel'
    px.setAttribute('aria-hidden', 'true')
    px.style.left = `${cx}px`
    px.style.top = `${cy}px`
    px.style.background = color
    px.style.setProperty('--dx', `${Math.round(Math.cos(a) * 46)}px`)
    px.style.setProperty('--dy', `${Math.round(Math.sin(a) * 46)}px`)
    document.body.appendChild(px)
    setTimeout(() => px.remove(), 650)
  }
}
