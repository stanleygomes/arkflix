import React from 'react'
import { useNavigate } from 'react-router-dom'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/Button'

export const LoginPage: React.FC = () => {
  const [username, setUsername] = React.useState('nono')
  const [password, setPassword] = React.useState('BR#jf2026')
  const [error, setError] = React.useState('')
  const [loading, setLoading] = React.useState(false)

  const { setAuth } = useAuthStore()
  const navigate = useNavigate()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const data = await jellyfinService.authenticate(username, password)
      setAuth(data.User, data.AccessToken, data.ServerId)
      navigate('/')
    } catch (err: any) {
      setError('Credenciais inválidas ou erro ao conectar ao Jellyfin.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-screen w-full flex items-center justify-center bg-black bg-opacity-70 px-4">
      {/* Background Image / Overlay */}
      <div className="absolute inset-0 bg-[url('https://assets.nflxext.com/ffe/siteui/vlv3/7ca5b7c7-20aa-42a8-a278-e801b0d63e1c/bn-en-20240326-popsignuptwoweeks-perspective_alpha_website_large.jpg')] bg-cover bg-center -z-10 opacity-40"></div>
      <div className="absolute inset-0 bg-black/60 -z-10"></div>

      {/* Login Card */}
      <div className="w-full max-w-md bg-black/75 rounded-lg p-8 md:p-12 border border-white/10 shadow-2xl">
        <h1 className="text-3xl font-bold text-white mb-6">Entrar</h1>

        {error && (
          <div className="bg-netflix-red/20 border border-netflix-red text-netflix-lightGray p-3 rounded text-sm mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Usuário"
              required
              className="w-full bg-[#333] text-white px-4 py-3 rounded focus:outline-none focus:ring-2 focus:ring-netflix-red text-sm"
            />
          </div>

          <div>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Senha"
              required
              className="w-full bg-[#333] text-white px-4 py-3 rounded focus:outline-none focus:ring-2 focus:ring-netflix-red text-sm"
            />
          </div>

          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={loading}
            className="w-full bg-netflix-red text-white hover:bg-red-700 font-bold mt-2"
          >
            {loading ? 'Conectando...' : 'Entrar no Arkflix'}
          </Button>
        </form>

        <div className="mt-8 text-xs text-netflix-gray space-y-2">
          <p>Servidor Conectado: <strong className="text-white">ark-flix.duckdns.org</strong></p>
          <p>Dica: As credenciais padrão já vêm pré-preenchidas para teste.</p>
        </div>
      </div>
    </div>
  )
}
