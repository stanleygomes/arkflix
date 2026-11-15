// "Baixar pro PC" de álbum e playlist: a tarefa vem da Central (cancelável)
// e, no fim, quem mostra o selo "No PC" recarrega.
import { activity, type Task } from './activity.svelte'
import { api, on } from './api'
import { play } from './sound'
import { confirm, toast } from './state.svelte'
import type { TaskEvent } from './types'

export type SetKind = 'album' | 'playlist'

export function taskId(kind: SetKind, id: string): string {
  return `musicdl:${kind}:${id}`
}

/** A tarefa de download em andamento (ou undefined). */
export function downloading(kind: SetKind, id: string): Task | undefined {
  const t = activity.tasks.find((x) => x.id === taskId(kind, id))
  return t?.state === 'running' ? t : undefined
}

/** Chama fn quando o download do conjunto termina (devolve o cancelamento). */
export function onDownloaded(kind: SetKind, id: string, fn: (ev: TaskEvent) => void): () => void {
  return on<TaskEvent>('task', (ev) => {
    if (ev.id === taskId(kind, id) && ev.state !== 'running') fn(ev)
  })
}

export async function download(kind: SetKind, id: string, name: string) {
  try {
    await api.musicDownload(kind, id)
    play('accept')
    toast(`Baixando ${name}: acompanhe na Central`, 'info', 3000)
  } catch (err) {
    toast((err as Error).message, 'error')
  }
}

export async function cancelDownload(kind: SetKind, id: string) {
  await api.cancelTask(taskId(kind, id)).catch((err) => toast((err as Error).message, 'error'))
}

/** Apaga a cópia do PC (com confirmação). Devolve true se apagou. */
export async function removeLocal(kind: SetKind, id: string, name: string): Promise<boolean> {
  const ok = await confirm(`Apagar a cópia de "${name}" deste PC? No servidor continua tudo.`, 'Apagar do PC', true)
  if (!ok) return false
  try {
    await api.musicRemoveLocal(kind, id)
    toast(`${name} saiu do PC`, 'ok', 2500)
    return true
  } catch (err) {
    toast((err as Error).message, 'error')
    return false
  }
}
