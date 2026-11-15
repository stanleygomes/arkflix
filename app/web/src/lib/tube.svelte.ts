// Tubo de TV em momento curto (noemi-ui: CRT só em momento, nunca fixo):
// liga ao abrir o app e ao começar um filme, desliga ao fechar um jogo.
// Dura ~1,6 s e some; com "reduzir animações", não aparece.
import { reduced } from './motion'

export type TubeKind = 'on' | 'off'

export const tube = $state<{ seq: number; kind: TubeKind; caption: string }>({ seq: 0, kind: 'on', caption: '' })

/** Mostra o tubo ligando (ou desligando) com a legenda dada. */
export function tubeMoment(kind: TubeKind, caption = ''): void {
  if (reduced()) return
  tube.kind = kind
  tube.caption = caption
  tube.seq++
}
