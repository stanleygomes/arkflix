// Cores: o tema é fixo (noemi-ui, cinema antigo escuro, em app.css); o
// console não pinta mais a tela. O contraste (WCAG) é calculado aqui pros
// testes. O alto contraste é a classe high-contrast no <body>.
import type { Palette } from './types'

/** Tema do app (os tokens de app.css). */
export const homePalette: Palette = {
  bg: '#17110c',
  surface: '#241a13',
  accent: '#e8b34a',
  accent2: '#5fa8a1',
  text: '#f1e4c6',
  muted: '#b9a588',
}

/** Alto contraste: preto, branco e amarelo, igual em todo console. */
export const highContrastPalette: Palette = {
  bg: '#000000',
  surface: '#141414',
  accent: '#ffd400',
  accent2: '#7fdcff',
  text: '#ffffff',
  muted: '#e6e6e6',
}

export function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const n = Number.parseInt(full, 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function channel(c: number): number {
  const s = c / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

export function luminance(hex: string): number {
  const [r, g, b] = hexToRgb(hex)
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

/** Razão de contraste WCAG (1 a 21). */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

/** Cor de texto legível sobre o fundo dado (tinta escura ou o creme). */
export function onColor(bg: string, light = '#f1e4c6'): string {
  return contrast(bg, '#24170f') >= contrast(bg, light) ? '#24170f' : light
}

const PROPS = ['--bg', '--surface', '--accent', '--accent-2', '--text', '--muted', '--on-accent']

/** Antes pintava a tela com a cor do console; hoje só limpa o que sobrou
 * (o tema vem do CSS). Fica pra as telas não precisarem saber disso. */
export function applyPalette(_p?: Palette, el: HTMLElement = document.documentElement): void {
  for (const prop of PROPS) el.style.removeProperty(prop)
}
