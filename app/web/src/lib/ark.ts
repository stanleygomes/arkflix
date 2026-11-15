// arkflix-server do perfil: quem tem (e o que dá pra fazer) e as listas do
// usuário de lá. O Go descobre pelo /info do Jellyfin ou do RomM ligado.
import { api } from './api'
import { ago } from './format'
import { app } from './state.svelte'

let seq = 0

/** Pergunta de novo ao Go (troca de perfil, conta vinculada ou desligada). */
export async function refreshArk(): Promise<void> {
  const mine = ++seq
  try {
    const info = await api.arkServer()
    if (mine !== seq) return
    app.ark = info.found ? info : null
    if (!app.ark?.lists) app.arkLists = []
  } catch {
    if (mine === seq) app.ark = null
  }
}

/** Busca as listas (com os itens) do usuário do servidor. */
export async function loadArkLists(): Promise<void> {
  if (!app.ark?.lists) {
    app.arkLists = []
    return
  }
  app.arkLists = await api.arkLists()
}

/** Pode mexer nas listas (o perfil infantil só vê). */
export function arkEditable(): boolean {
  return !!app.ark?.lists && !app.profile?.profiles.find((p) => p.id === app.profile?.current)?.kids
}

/** Admin com o servidor gravável: esconder e enviar arquivos. */
export function arkWritable(): boolean {
  return !!app.ark?.admin && app.ark.writable
}

/** Data do servidor (RFC 3339) em "há 5 min"; data zero do Go = "nunca". */
export function since(iso: string): string {
  const ms = Date.parse(iso)
  return ms > 0 ? ago(ms / 1000) : 'nunca'
}
