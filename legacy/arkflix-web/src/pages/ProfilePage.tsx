import React, { useState } from 'react'
import { useAuth, useTranslation } from '@/hooks'
import { Button, Input } from '@/components/ui'
import { getUserAvatarUrl } from '@/services/api'
import { Plus, Trash2, Check, Server, LogOut, User } from 'lucide-react'

export const ProfilePage: React.FC = () => {
  const { user, profiles, switchProfile, removeProfile, login, serverUrl, logout } = useAuth()
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

  return (
    <div className="min-h-screen bg-[#000000] text-white pt-[max(env(safe-area-inset-top,0px)+4.5rem,5.5rem)] sm:pt-28 pb-20 px-4 sm:px-6 md:px-14 selection:bg-white/20">
      <div className="max-w-4xl mx-auto space-y-6 sm:space-y-8 animate-fadeIn">
        {/* Header da Página */}
        <div className="border-b border-white/[0.08] pb-4 sm:pb-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 sm:w-11 sm:h-11 rounded-squircle bg-gradient-to-tr from-blue-500/20 to-indigo-500/20 flex items-center justify-center text-apple-accent border border-blue-500/30 shadow-sm flex-none">
              <User className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight leading-tight">
                {t.profile.title}
              </h1>
              <p className="text-xs text-apple-subtext mt-1">
                {t.profile.subtitle}
              </p>
            </div>
          </div>
        </div>

        {/* Section 1: Perfis Cadastrados */}
        <div className="glass-panel p-5 sm:p-6 md:p-8 rounded-squircle-xl space-y-6 border border-white/10 shadow-apple">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-sm sm:text-base font-bold text-white tracking-tight">{t.profile.whoIsWatching}</h2>
              <p className="text-xs text-apple-subtext">{t.profile.whoIsWatchingDesc}</p>
            </div>

            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowAddProfile(!showAddProfile)}
              className="text-xs font-semibold"
            >
              <Plus className="w-3.5 h-3.5 mr-1" /> {t.profile.addProfile}
            </Button>
          </div>

          {/* Profiles Grid */}
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3 sm:gap-4">
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
                      ? 'bg-white/[0.08] border-2 border-apple-accent shadow-apple'
                      : 'bg-white/[0.03] hover:bg-white/[0.07] border border-white/10 hover:scale-105 shadow-sm'
                  }`}
                >
                  {/* Profile Avatar Image or Fallback */}
                  <div className="relative w-16 h-16 rounded-full overflow-hidden mb-2.5 shadow-md border-2 border-white/20">
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

                  <span className="text-xs font-semibold text-white truncate max-w-[120px]">
                    {profile.name}
                  </span>

                  {isCurrent && (
                    <span className="mt-1 inline-flex items-center gap-1 text-[10px] text-apple-accent font-bold bg-blue-500/20 px-2 py-0.5 rounded-full border border-blue-500/30">
                      <Check className="w-2.5 h-2.5 stroke-[3]" /> {t.common.active}
                    </span>
                  )}

                  {/* Remove button (if more than 1 profile) */}
                  {profiles.length > 1 && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        removeProfile(profile.id)
                      }}
                      className="absolute top-2 right-2 p-1.5 rounded-full bg-black/60 text-white hover:bg-red-500 opacity-0 group-hover:opacity-100 transition-all"
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
              className="p-5 rounded-squircle bg-white/[0.03] border border-white/10 space-y-4 animate-fadeIn"
            >
              <h3 className="text-xs font-bold text-white uppercase tracking-wider">
                {t.profile.connectNewProfile}
              </h3>

              {addError && <p className="text-xs text-red-400">{addError}</p>}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Input
                  label={t.login.usernameLabel}
                  placeholder={t.login.usernamePlaceholder}
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  className="bg-white/10 text-white border-white/10 placeholder-apple-subtext"
                  required
                />
                <Input
                  type="password"
                  label={t.login.passwordLabel}
                  placeholder={t.login.passwordPlaceholder}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="bg-white/10 text-white border-white/10 placeholder-apple-subtext"
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

        {/* Section 2: Servidor Conectado & Logout */}
        <div className="glass-panel p-5 sm:p-6 md:p-8 rounded-squircle-xl space-y-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border border-white/10 shadow-apple">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Server className="w-4 h-4 text-apple-accent" />
              <h3 className="text-sm font-semibold text-white">{t.profile.serverTitle}</h3>
            </div>
            <p className="text-xs text-apple-subtext font-mono">{serverUrl}</p>
          </div>

          <Button
            variant="ghost"
            size="sm"
            onClick={logout}
            className="text-red-400 hover:text-red-300 hover:bg-red-500/10 border border-red-500/20"
          >
            <LogOut className="w-4 h-4 mr-1.5" /> {t.common.logout}
          </Button>
        </div>
      </div>
    </div>
  )
}
