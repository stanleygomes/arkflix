import { create } from 'zustand'
import { JellyfinUser } from '@/types/jellyfin'
import { getServerUrl, setServerUrl } from '@/services/api'

export interface ProfileAccount {
  id: string
  name: string
  avatarColor: string
  token: string
  serverId: string
}

interface AuthState {
  serverUrl: string
  user: JellyfinUser | null
  token: string | null
  serverId: string | null
  isAuthenticated: boolean
  profiles: ProfileAccount[]
  setServerUrl: (url: string) => void
  setAuth: (user: JellyfinUser, token: string, serverId: string) => void
  switchProfile: (profileId: string) => void
  addProfile: (profile: ProfileAccount) => void
  removeProfile: (profileId: string) => void
  logout: () => void
}

const AVATAR_COLORS = [
  'from-blue-500 to-indigo-600',
  'from-purple-500 to-pink-600',
  'from-amber-500 to-orange-600',
  'from-emerald-500 to-teal-600',
  'from-rose-500 to-red-600',
]

export const useAuthStore = create<AuthState>((set, get) => ({
  serverUrl: getServerUrl(),
  user: JSON.parse(localStorage.getItem('arkflix_user') || 'null'),
  token: localStorage.getItem('arkflix_token'),
  serverId: localStorage.getItem('arkflix_server_id'),
  isAuthenticated: !!localStorage.getItem('arkflix_token'),
  profiles: JSON.parse(localStorage.getItem('arkflix_profiles') || '[]'),

  setServerUrl: (url: string) => {
    setServerUrl(url)
    set({ serverUrl: url })
  },

  setAuth: (user, token, serverId) => {
    localStorage.setItem('arkflix_user', JSON.stringify(user))
    localStorage.setItem('arkflix_token', token)
    localStorage.setItem('arkflix_server_id', serverId)

    const profiles = get().profiles
    const existingIndex = profiles.findIndex((p) => p.id === user.Id)
    const color = AVATAR_COLORS[profiles.length % AVATAR_COLORS.length]

    const newProfile: ProfileAccount = {
      id: user.Id,
      name: user.Name,
      avatarColor: existingIndex >= 0 ? profiles[existingIndex].avatarColor : color,
      token,
      serverId,
    }

    let updatedProfiles = [...profiles]
    if (existingIndex >= 0) {
      updatedProfiles[existingIndex] = newProfile
    } else {
      updatedProfiles.push(newProfile)
    }

    localStorage.setItem('arkflix_profiles', JSON.stringify(updatedProfiles))
    set({ user, token, serverId, isAuthenticated: true, profiles: updatedProfiles })
  },

  switchProfile: (profileId: string) => {
    const profile = get().profiles.find((p) => p.id === profileId)
    if (!profile) return

    const userObj: JellyfinUser = {
      Id: profile.id,
      Name: profile.name,
      ServerId: profile.serverId,
    }

    localStorage.setItem('arkflix_user', JSON.stringify(userObj))
    localStorage.setItem('arkflix_token', profile.token)
    localStorage.setItem('arkflix_server_id', profile.serverId)

    set({
      user: userObj,
      token: profile.token,
      serverId: profile.serverId,
      isAuthenticated: true,
    })
  },

  addProfile: (profile) => {
    const updated = [...get().profiles, profile]
    localStorage.setItem('arkflix_profiles', JSON.stringify(updated))
    set({ profiles: updated })
  },

  removeProfile: (profileId) => {
    const updated = get().profiles.filter((p) => p.id !== profileId)
    localStorage.setItem('arkflix_profiles', JSON.stringify(updated))

    if (get().user?.Id === profileId) {
      if (updated.length > 0) {
        get().switchProfile(updated[0].id)
      } else {
        get().logout()
      }
    } else {
      set({ profiles: updated })
    }
  },

  logout: () => {
    localStorage.removeItem('arkflix_user')
    localStorage.removeItem('arkflix_token')
    localStorage.removeItem('arkflix_server_id')
    set({ user: null, token: null, serverId: null, isAuthenticated: false })
  },
}))
