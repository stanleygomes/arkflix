import React from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Search, User, Cast } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation, useChromecast } from '@/hooks'
import { getUserAvatarUrl } from '@/services/api'
import { Logo } from '@/components/ui'
import { cn } from '@/lib/utils'

export const Navbar: React.FC = () => {
  const [isScrolled, setIsScrolled] = React.useState(false)
  const [showSearch, setShowSearch] = React.useState(false)
  const [searchQuery, setSearchQuery] = React.useState('')
  const [avatarError, setAvatarError] = React.useState(false)
  const { user } = useAuthStore()
  const { t } = useTranslation()
  const chromecast = useChromecast()
  const navigate = useNavigate()
  const location = useLocation()

  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 20)
    }
    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery)}`)
      setShowSearch(false)
    }
  }

  const handleCastClick = () => {
    if (window.cast?.framework) {
      try {
        window.cast.framework.CastContext.getInstance().requestSession()
      } catch (err) {
        console.log('Cast session request:', err)
      }
    }
  }

  const navLinks = [
    { label: t.nav.home, path: '/' },
    { label: t.nav.movies, path: '/movies' },
    { label: t.nav.series, path: '/series' },
    { label: t.common.myList, path: '/my-list' },
  ]

  const userAvatarUrl = user?.Id ? getUserAvatarUrl(user.Id, user.PrimaryImageTag) : ''

  return (
    <header
      className={cn(
        'fixed top-0 left-0 right-0 z-40 transition-all duration-300 px-4 sm:px-10 md:px-16 pt-[max(env(safe-area-inset-top,0px)+0.75rem,0.85rem)] pb-3 md:py-5 flex items-center justify-between',
        isScrolled
          ? 'glass-nav shadow-apple'
          : 'bg-gradient-to-b from-black/95 via-black/50 to-transparent'
      )}
    >
      {/* Left: Prominent Arkflix Brand Logo & Nav Tabs */}
      <div className="flex items-center gap-8 lg:gap-12">
        <Logo size="md" theme="dark" />

        {/* Apple TV Navigation Pill Tabs */}
        <nav className="hidden md:flex items-center gap-1.5 backdrop-blur-2xl p-1.5 rounded-full border bg-white/[0.08] border-white/15 transition-all shadow-sm">
          {navLinks.map((link) => {
            const isActive = location.pathname === link.path
            return (
              <Link
                key={link.path}
                to={link.path}
                className={cn(
                  'px-5 py-2 text-sm font-bold rounded-full transition-all duration-200 tracking-tight',
                  isActive
                    ? 'bg-white text-black shadow-apple scale-105'
                    : 'text-[#A1A1A6] hover:text-white hover:bg-white/10'
                )}
              >
                {link.label}
              </Link>
            )
          })}
        </nav>
      </div>

      {/* Right: Search, Cast, Profile Direct Link */}
      <div className="flex items-center gap-1.5 sm:gap-2">
        {/* Search Bar */}
        <div className="relative flex items-center">
          {showSearch ? (
            <form
              onSubmit={handleSearchSubmit}
              className="flex items-center backdrop-blur-xl border px-3 py-1.5 rounded-full transition-all duration-300 shadow-sm bg-white/15 border-white/25"
            >
              <Search className="w-4 h-4 mr-1.5 text-white/80" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t.nav.searchPlaceholder}
                className="bg-transparent text-sm placeholder-apple-subtext focus:outline-none w-32 sm:w-44 md:w-56 font-medium text-white"
                autoFocus
                onBlur={() => !searchQuery && setShowSearch(false)}
              />
            </form>
          ) : (
            <button
              onClick={() => setShowSearch(true)}
              aria-label="Buscar"
              className="p-2 rounded-full transition-all hover:scale-105 active:scale-95 text-white/90 hover:text-white hover:bg-white/15"
            >
              <Search className="w-5 h-5" />
            </button>
          )}
        </div>

        {/* Google Cast Button Launcher */}
        <div
          onClick={handleCastClick}
          title={chromecast.isConnected ? `Conectado a ${chromecast.deviceName}` : 'Transmitir via Google Cast'}
          className="relative w-9 h-9 flex items-center justify-center p-1 rounded-full transition-all hover:scale-105 active:scale-95 cursor-pointer hover:bg-white/15 text-white/90"
        >
          <Cast className={cn('w-5 h-5 transition-colors', chromecast.isConnected ? 'text-apple-accent stroke-[2.5]' : '')} />

          <div className="absolute inset-0 opacity-0 cursor-pointer overflow-hidden flex items-center justify-center">
            {React.createElement('google-cast-launcher', {
              class: 'w-full h-full cursor-pointer',
            })}
          </div>
        </div>

        {/* User Profile Avatar */}
        <Link
          to="/profile"
          title="Perfis e Configurações"
          className="flex items-center cursor-pointer focus:outline-none group ml-1"
        >
          <div
            className={cn(
              'w-10 h-10 md:w-11 md:h-11 rounded-full overflow-hidden shadow-apple transition-all duration-300 group-hover:scale-105 bg-gradient-to-tr from-blue-500 to-indigo-600 flex items-center justify-center border-2',
              location.pathname === '/profile'
                ? 'border-apple-accent ring-4 ring-blue-500/30'
                : 'border-white/40 group-hover:border-white'
            )}
          >
            {userAvatarUrl && !avatarError ? (
              <img
                src={userAvatarUrl}
                alt={user?.Name || 'Avatar'}
                onError={() => setAvatarError(true)}
                className="w-full h-full object-cover"
              />
            ) : (
              <span className="text-sm md:text-base font-extrabold text-white">
                {user?.Name?.charAt(0).toUpperCase() || <User className="w-5 h-5" />}
              </span>
            )}
          </div>
        </Link>
      </div>
    </header>
  )
}
