import React, { useState } from 'react'
import { Button, Input } from '@/components/ui'
import { User, Lock, Tv } from 'lucide-react'
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
    <div className="relative min-h-screen w-full flex items-center justify-center bg-apple-bg px-4 overflow-hidden selection:bg-white/20">
      {/* Subtle Ambient Apple Glows in the background */}
      <div className="absolute top-1/4 -left-48 w-96 h-96 bg-blue-600/20 rounded-full blur-[140px] pointer-events-none" />
      <div className="absolute bottom-1/4 -right-48 w-96 h-96 bg-purple-600/15 rounded-full blur-[140px] pointer-events-none" />

      {/* Apple Glass Card */}
      <div className="w-full max-w-sm glass-panel rounded-squircle-2xl p-8 md:p-10 shadow-apple border border-white/15 relative z-10 flex flex-col items-center">
        {/* Apple TV Icon Badge */}
        <div className="w-14 h-14 rounded-squircle-lg bg-white text-black flex items-center justify-center mb-6 shadow-md transition-transform hover:scale-105">
          <Tv className="w-7 h-7 fill-black text-black" />
        </div>

        <h1 className="text-2xl font-bold text-white tracking-tight text-center">
          Entrar no Arkflix
        </h1>
        <p className="text-xs text-apple-subtext mt-1.5 mb-6 text-center">
          Conecte sua conta do Jellyfin
        </p>

        {error && (
          <div className="w-full bg-red-500/10 border border-red-500/20 text-red-300 p-3 rounded-squircle-sm text-xs mb-4 text-center font-medium animate-fadeIn">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="w-full space-y-4">
          <Input
            type="text"
            label="ID do Usuário"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Usuário"
            icon={<User className="w-4 h-4" />}
            required
          />

          <Input
            type="password"
            label="Senha"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Senha"
            icon={<Lock className="w-4 h-4" />}
            required
          />

          <Button
            type="submit"
            variant="primary"
            size="lg"
            isLoading={isLoading}
            loadingText="Conectando..."
            className="w-full font-semibold mt-2 shadow-sm"
          >
            Continuar
          </Button>
        </form>

        <div className="mt-8 text-center text-[11px] text-apple-subtext space-y-1">
          <p>Servidor: <strong className="text-[#F5F5F7]">ark-flix.duckdns.org</strong></p>
        </div>
      </div>
    </div>
  )
}
