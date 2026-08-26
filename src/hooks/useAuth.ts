import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'

export function useAuth() {
  const { user, token, serverId, isAuthenticated, setAuth, logout } = useAuthStore()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  const login = async (username: string, password: string) => {
    setIsLoading(true)
    setError(null)

    try {
      const data = await jellyfinService.authenticate(username, password)
      setAuth(data.User, data.AccessToken, data.ServerId)
      navigate('/')
      return true
    } catch (err: any) {
      const message =
        err?.response?.data?.message ||
        'Credenciais inválidas ou erro ao conectar ao servidor Jellyfin.'
      setError(message)
      return false
    } finally {
      setIsLoading(false)
    }
  }

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return {
    user,
    token,
    serverId,
    isAuthenticated,
    isLoading,
    error,
    login,
    logout: handleLogout,
  }
}
