import React, { useState } from 'react'
import { Button, Input } from '@/components/ui'
import { User, Lock } from 'lucide-react'
import { useAuth } from '@/hooks'

export const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('nono')
  const [password, setPassword] = useState('BR#jf2026')

  // Hook desacoplado de autenticação
  const { login, isLoading, error } = useAuth()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    await login(username, password)
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

        <form onSubmit={handleLogin} className="space-y-5">
          <Input
            type="text"
            label="Usuário"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Digite seu usuário"
            icon={<User className="w-4 h-4" />}
            required
          />

          <Input
            type="password"
            label="Senha"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Digite sua senha"
            icon={<Lock className="w-4 h-4" />}
            required
          />

          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={isLoading}
            className="w-full bg-netflix-red text-white hover:bg-red-700 font-bold mt-2"
          >
            {isLoading ? 'Conectando...' : 'Entrar no Arkflix'}
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
