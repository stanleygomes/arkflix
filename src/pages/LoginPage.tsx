import React, { useState } from 'react'
import { Button, Input } from '@/components/ui'
import { User, Lock, Server, Tv, Globe } from 'lucide-react'
import { useAuth } from '@/hooks'

export const LoginPage: React.FC = () => {
  const { serverUrl, login, isLoading, error } = useAuth()

  const [customServerUrl, setCustomServerUrl] = useState(serverUrl)
  const [username, setUsername] = useState('nono')
  const [password, setPassword] = useState('BR#jf2026')
  const [showServerInput, setShowServerInput] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    await login(username, password, customServerUrl)
  }

  return (
    <div className="relative min-h-screen w-full flex items-center justify-center bg-apple-bg px-4 overflow-hidden selection:bg-white/20">
      {/* Ambient Apple Glows */}
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
          Conecte sua biblioteca Jellyfin
        </p>

        {error && (
          <div className="w-full bg-red-500/10 border border-red-500/20 text-red-300 p-3 rounded-squircle-sm text-xs mb-4 text-center font-medium animate-fadeIn">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="w-full space-y-4">
          {/* Server URL Input */}
          <div>
            <div className="flex items-center justify-between mb-1.5 px-1">
              <label className="text-xs font-medium text-apple-subtext">Servidor Jellyfin</label>
              <button
                type="button"
                onClick={() => setShowServerInput(!showServerInput)}
                className="text-[11px] text-apple-accent hover:underline"
              >
                {showServerInput ? 'Ocultar' : 'Alterar'}
              </button>
            </div>

            {showServerInput ? (
              <Input
                type="text"
                value={customServerUrl}
                onChange={(e) => setCustomServerUrl(e.target.value)}
                placeholder="https://seu-jellyfin.com"
                icon={<Server className="w-4 h-4" />}
                required
              />
            ) : (
              <div
                onClick={() => setShowServerInput(true)}
                className="flex items-center gap-2 px-3.5 py-2.5 bg-white/[0.04] border border-white/10 rounded-squircle text-xs text-apple-subtext hover:bg-white/[0.08] cursor-pointer transition-colors"
              >
                <Globe className="w-3.5 h-3.5 text-apple-accent" />
                <span className="truncate text-white/90 font-mono text-[11px]">{customServerUrl}</span>
              </div>
            )}
          </div>

          <Input
            type="text"
            label="Usuário"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Nome de usuário"
            icon={<User className="w-4 h-4" />}
            required
          />

          <Input
            type="password"
            label="Senha"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Sua senha"
            icon={<Lock className="w-4 h-4" />}
            required
          />

          <Button
            type="submit"
            variant="primary"
            size="lg"
            isLoading={isLoading}
            loadingText="Conectando..."
            className="w-full font-semibold mt-3 shadow-sm"
          >
            Conectar
          </Button>
        </form>
      </div>
    </div>
  )
}
