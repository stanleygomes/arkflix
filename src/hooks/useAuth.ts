import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'

export function useAuth() {
  const {
    user,
    token,
    serverId,
    serverUrl,
    isAuthenticated,
    profiles,
    setServerUrl,
    setAuth,
    switchProfile,
    removeProfile,
    logout,
  } = useAuthStore()

  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  const login = async (username: string, password: string, customServerUrl?: string) => {
    setIsLoading(true)
    setError(null)

    if (customServerUrl) {
      setServerUrl(customServerUrl)
    }

    try {
      const data = await jellyfinService.authenticate(username, password)
      setAuth(data.User, data.AccessToken, data.ServerId)
      navigate('/')
      return true
    } catch (err: any) {
      const message =
        err?.response?.data?.message ||
        'Não foi possível conectar ao Jellyfin. Verifique o endereço do servidor e suas credenciais.'
      setError(message)
      return false
    } finally {
      setIsLoading(false)
    }
  }

  const handleSwitchProfile = (profileId: string) => {
    switchProfile(profileId)
    navigate('/')
  }

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return {
    user,
    token,
    serverId,
    serverUrl,
    isAuthenticated,
    profiles,
    isLoading,
    error,
    login,
    setServerUrl,
    switchProfile: handleSwitchProfile,
    removeProfile,
    logout: handleLogout,
  }
}
