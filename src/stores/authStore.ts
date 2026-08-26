import { create } from 'zustand'
import { JellyfinUser } from '@/types/jellyfin'

interface AuthState {
  user: JellyfinUser | null
  token: string | null
  serverId: string | null
  isAuthenticated: boolean
  setAuth: (user: JellyfinUser, token: string, serverId: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: JSON.parse(localStorage.getItem('arkflix_user') || 'null'),
  token: localStorage.getItem('arkflix_token'),
  serverId: localStorage.getItem('arkflix_server_id'),
  isAuthenticated: !!localStorage.getItem('arkflix_token'),

  setAuth: (user, token, serverId) => {
    localStorage.setItem('arkflix_user', JSON.stringify(user))
    localStorage.setItem('arkflix_token', token)
    localStorage.setItem('arkflix_server_id', serverId)
    set({ user, token, serverId, isAuthenticated: true })
  },

  logout: () => {
    localStorage.removeItem('arkflix_user')
    localStorage.removeItem('arkflix_token')
    localStorage.removeItem('arkflix_server_id')
    set({ user: null, token: null, serverId: null, isAuthenticated: false })
  },
}))
