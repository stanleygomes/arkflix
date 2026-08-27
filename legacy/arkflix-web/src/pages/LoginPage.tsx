import React, { useState } from 'react'
import { Button, Input, Logo } from '@/components/ui'
import { User, Lock, Server } from 'lucide-react'
import { useAuth, useTranslation } from '@/hooks'

export const LoginPage: React.FC = () => {
  const { serverUrl, login, isLoading, error } = useAuth()
  const { t } = useTranslation()

  const [customServerUrl, setCustomServerUrl] = useState(serverUrl || '')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!customServerUrl.trim()) return
    await login(username, password, customServerUrl)
  }

  return (
    <div className="relative min-h-screen w-full flex items-center justify-center bg-[#000000] px-4 overflow-hidden selection:bg-white/20 selection:text-white">
      {/* Subtle Apple Dark Ambient Glow Elements */}
      <div className="absolute top-1/4 -left-48 w-96 h-96 bg-blue-600/10 rounded-full blur-[140px] pointer-events-none" />
      <div className="absolute bottom-1/4 -right-48 w-96 h-96 bg-indigo-600/10 rounded-full blur-[140px] pointer-events-none" />

      {/* Apple Dark Glass Card */}
      <div className="w-full max-w-sm glass-panel rounded-squircle-2xl p-8 md:p-10 relative z-10 flex flex-col items-center border border-white/10 shadow-2xl backdrop-blur-2xl">
        {/* Distinctive Logo Branding */}
        <div className="mb-6 flex flex-col items-center">
          <Logo size="lg" theme="dark" withLink={false} />
          <p className="text-xs text-apple-subtext mt-2 text-center">
            {t.login.subtitle}
          </p>
        </div>

        {error && (
          <div className="w-full bg-red-500/10 border border-red-500/20 text-red-400 p-3 rounded-squircle-sm text-xs mb-4 text-center font-medium animate-fadeIn">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="w-full space-y-4">
          {/* Server URL Input */}
          <div className="space-y-1.5">
            <Input
              type="text"
              label={t.login.serverLabel}
              value={customServerUrl}
              onChange={(e) => setCustomServerUrl(e.target.value)}
              placeholder="https://seu-servidor-jellyfin.com"
              icon={<Server className="w-4 h-4 text-apple-subtext" />}
              className="bg-white/10 text-white placeholder-apple-subtext border-white/10 focus:border-white/30"
              required
            />
          </div>

          <div className="space-y-1.5">
            <Input
              type="text"
              label={t.login.usernameLabel}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t.login.usernamePlaceholder}
              icon={<User className="w-4 h-4 text-apple-subtext" />}
              className="bg-white/10 text-white placeholder-apple-subtext border-white/10 focus:border-white/30"
              required
            />
          </div>

          <div className="space-y-1.5">
            <Input
              type="password"
              label={t.login.passwordLabel}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t.login.passwordPlaceholder}
              icon={<Lock className="w-4 h-4 text-apple-subtext" />}
              className="bg-white/10 text-white placeholder-apple-subtext border-white/10 focus:border-white/30"
              required
            />
          </div>

          <Button
            type="submit"
            variant="primary"
            size="lg"
            isLoading={isLoading}
            loadingText={t.login.connecting}
            className="w-full font-bold mt-3 shadow-apple py-3"
          >
            {t.login.submitButton}
          </Button>
        </form>
      </div>
    </div>
  )
}
