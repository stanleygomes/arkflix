import React, { useState } from 'react'
import { useAuth, useTheme, useTranslation } from '@/hooks'
import { Button, Input, Tabs } from '@/components/ui'
import { getUserAvatarUrl } from '@/services/api'
import { Plus, Moon, Sun, Laptop, Trash2, Check, Server, LogOut, User } from 'lucide-react'
import { ThemeMode } from '@/stores/themeStore'

export const ProfilePage: React.FC = () => {
  const { user, profiles, switchProfile, removeProfile, login, serverUrl, logout } = useAuth()
  const { theme, setTheme } = useTheme()
  const { t } = useTranslation()

  const [showAddProfile, setShowAddProfile] = useState(false)
  const [newUsername, setNewUsername] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [addError, setAddError] = useState('')
  const [isAdding, setIsAdding] = useState(false)
  const [imgErrors, setImgErrors] = useState<Record<string, boolean>>({})

  const handleAddProfile = async (e: React.FormEvent) => {
    e.preventDefault()
    setAddError('')
    setIsAdding(true)

    const success = await login(newUsername, newPassword)
    setIsAdding(false)
    if (success) {
      setShowAddProfile(false)
      setNewUsername('')
      setNewPassword('')
    } else {
      setAddError('Falha ao autenticar o novo perfil.')
    }
  }

  const handleImgError = (profileId: string) => {
    setImgErrors((prev) => ({ ...prev, [profileId]: true }))
  }

  const themeTabs = [
    { id: 'dark', label: t.profile.dark, icon: <Moon className="w-3.5 h-3.5" /> },
    { id: 'light', label: t.profile.light, icon: <Sun className="w-3.5 h-3.5" /> },
    { id: 'auto', label: t.profile.auto, icon: <Laptop className="w-3.5 h-3.5" /> },
  ]

  return (
    <div className="min-h-screen pt-24 pb-20 px-6 md:px-14 max-w-4xl mx-auto space-y-10 animate-fadeIn">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-extrabold text-apple-text tracking-tight">{t.profile.title}</h1>
        <p className="text-xs text-apple-subtext mt-1">
          {t.profile.subtitle}
        </p>
      </div>

      {/* Section 1: Perfis Cadastrados */}
      <div className="glass-panel p-6 md:p-8 rounded-squircle-xl space-y-6 shadow-apple">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-bold text-apple-text tracking-tight">{t.profile.whoIsWatching}</h2>
            <p className="text-xs text-apple-subtext">{t.profile.whoIsWatchingDesc}</p>
          </div>

          <Button
            variant="glass"
            size="sm"
            onClick={() => setShowAddProfile(!showAddProfile)}
            className="text-xs"
          >
            <Plus className="w-3.5 h-3.5 mr-1" /> {t.profile.addProfile}
          </Button>
        </div>

        {/* Profiles Grid */}
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
          {profiles.map((profile) => {
            const isCurrent = profile.id === user?.Id
            const avatarUrl = getUserAvatarUrl(profile.id, profile.primaryImageTag)
            const hasImgError = imgErrors[profile.id]

            return (
              <div
                key={profile.id}
                onClick={() => !isCurrent && switchProfile(profile.id)}
                className={`relative group flex flex-col items-center p-4 rounded-squircle transition-all duration-300 cursor-pointer ${
                  isCurrent
                    ? 'bg-apple-accent/15 border-2 border-apple-accent shadow-sm'
                    : 'bg-black/5 dark:bg-white/[0.04] hover:bg-black/10 dark:hover:bg-white/[0.08] border border-black/10 dark:border-white/10 hover:scale-105'
                }`}
              >
                {/* Profile Avatar Image or Fallback */}
                <div className="relative w-16 h-16 rounded-full overflow-hidden mb-2.5 shadow-md border border-white/20">
                  {!hasImgError ? (
                    <img
                      src={avatarUrl}
                      alt={profile.name}
                      onError={() => handleImgError(profile.id)}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div
                      className={`w-full h-full bg-gradient-to-tr ${profile.avatarColor} flex items-center justify-center text-xl font-bold text-white`}
                    >
                      {profile.name.charAt(0).toUpperCase()}
                    </div>
                  )}
                </div>

                <span className="text-xs font-semibold text-apple-text truncate max-w-[120px]">
                  {profile.name}
                </span>

                {isCurrent && (
                  <span className="mt-1 inline-flex items-center gap-1 text-[10px] text-emerald-400 font-bold bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                    <Check className="w-2.5 h-2.5" /> {t.common.active}
                  </span>
                )}

                {/* Remove button (if more than 1 profile) */}
                {profiles.length > 1 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      removeProfile(profile.id)
                    }}
                    className="absolute top-2 right-2 p-1.5 rounded-full bg-black/60 text-apple-subtext hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity"
                    title={t.profile.removeTitle}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            )
          })}
        </div>

        {/* Add Profile Form Sheet */}
        {showAddProfile && (
          <form
            onSubmit={handleAddProfile}
            className="p-5 rounded-squircle bg-black/5 dark:bg-white/[0.03] border border-black/10 dark:border-white/10 space-y-4 animate-fadeIn"
          >
            <h3 className="text-xs font-bold text-apple-text uppercase tracking-wider">
              {t.profile.connectNewProfile}
            </h3>

            {addError && <p className="text-xs text-red-400">{addError}</p>}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <Input
                label={t.login.usernameLabel}
                placeholder={t.login.usernamePlaceholder}
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                required
              />
              <Input
                type="password"
                label={t.login.passwordLabel}
                placeholder={t.login.passwordPlaceholder}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setShowAddProfile(false)}
              >
                {t.common.cancel}
              </Button>
              <Button
                type="submit"
                variant="primary"
                size="sm"
                isLoading={isAdding}
                loadingText={t.profile.adding}
              >
                {t.profile.saveProfile}
              </Button>
            </div>
          </form>
        )}
      </div>

      {/* Section 2: Aparência (Modo Escuro / Claro / Auto) */}
      <div className="glass-panel p-6 md:p-8 rounded-squircle-xl space-y-4 shadow-apple">
        <div>
          <h2 className="text-base font-bold text-apple-text tracking-tight">{t.profile.appearanceTitle}</h2>
          <p className="text-xs text-apple-subtext">
            {t.profile.appearanceDesc}
          </p>
        </div>

        <div className="pt-2">
          <Tabs
            tabs={themeTabs}
            activeTab={theme}
            onChange={(newTheme) => setTheme(newTheme as ThemeMode)}
          />
        </div>
      </div>

      {/* Section 3: Servidor Conectado & Logout */}
      <div className="glass-panel p-6 md:p-8 rounded-squircle-xl space-y-4 shadow-apple flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4 text-apple-accent" />
            <h3 className="text-sm font-semibold text-apple-text">{t.profile.serverTitle}</h3>
          </div>
          <p className="text-xs text-apple-subtext font-mono">{serverUrl}</p>
        </div>

        <Button
          variant="glass"
          size="sm"
          onClick={logout}
          className="text-red-400 hover:text-red-300 border-red-500/20 hover:bg-red-500/10"
        >
          <LogOut className="w-4 h-4 mr-1.5" /> {t.common.logout}
        </Button>
      </div>
    </div>
  )
}
