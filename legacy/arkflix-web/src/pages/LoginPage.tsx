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
    <div className="relative min-h-screen w-full flex items-center justify-center bg-[#F5F5F7] px-4 overflow-hidden selection:bg-blue-500/20 selection:text-black">
      {/* Subtle Apple Light Ambient Elements */}
      <div className="absolute top-1/4 -left-48 w-96 h-96 bg-blue-400/15 rounded-full blur-[140px] pointer-events-none" />
      <div className="absolute bottom-1/4 -right-48 w-96 h-96 bg-indigo-400/10 rounded-full blur-[140px] pointer-events-none" />

      {/* Apple Clean White Glass Card */}
      <div className="w-full max-w-sm apple-light-card rounded-squircle-2xl p-8 md:p-10 relative z-10 flex flex-col items-center">
        {/* Distinctive Logo Branding with crisp dark contrast for white theme */}
        <div className="mb-6 flex flex-col items-center">
          <Logo size="lg" theme="light" withLink={false} />
          <p className="text-xs text-[#6E6E73] mt-2 text-center">
            {t.login.subtitle}
          </p>
        </div>

        {error && (
          <div className="w-full bg-red-50 border border-red-200 text-red-600 p-3 rounded-squircle-sm text-xs mb-4 text-center font-medium animate-fadeIn">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="w-full space-y-4">
          {/* Server URL Input (Always explicit and editable by user) */}
          <div className="space-y-1.5">
            <Input
              type="text"
              label={t.login.serverLabel}
              value={customServerUrl}
              onChange={(e) => setCustomServerUrl(e.target.value)}
              placeholder="https://seu-servidor-jellyfin.com"
              icon={<Server className="w-4 h-4 text-[#86868B]" />}
              className="bg-white text-black border-black/10 focus:bg-white focus:ring-blue-500/20"
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
              icon={<User className="w-4 h-4 text-[#86868B]" />}
              className="bg-white text-black border-black/10 focus:bg-white focus:ring-blue-500/20"
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
              icon={<Lock className="w-4 h-4 text-[#86868B]" />}
              className="bg-white text-black border-black/10 focus:bg-white focus:ring-blue-500/20"
              required
            />
          </div>

          <Button
            type="submit"
            variant="apple-blue"
            size="lg"
            isLoading={isLoading}
            loadingText={t.login.connecting}
            className="w-full font-semibold mt-3 shadow-sm text-white"
          >
            {t.login.submitButton}
          </Button>
        </form>
      </div>
    </div>
  )
}
