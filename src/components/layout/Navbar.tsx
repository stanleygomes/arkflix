import React from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Search, User } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation } from '@/hooks'
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
  const navigate = useNavigate()
  const location = useLocation()

  const isProfilePage = location.pathname === '/profile'

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

  const navLinks = [
    { label: t.nav.home, path: '/' },
    { label: t.nav.movies, path: '/movies' },
    { label: t.nav.series, path: '/series' },
    { label: t.nav.library, path: '/my-list' },
  ]

  const userAvatarUrl = user?.Id ? getUserAvatarUrl(user.Id, user.PrimaryImageTag) : ''

  return (
    <header
      className={cn(
        'fixed top-0 left-0 right-0 z-40 transition-all duration-300 px-6 sm:px-10 md:px-16 py-4 md:py-5 flex items-center justify-between',
        isProfilePage
          ? isScrolled
            ? 'bg-white/90 backdrop-blur-2xl border-b border-black/[0.08] shadow-sm py-3.5'
            : 'bg-[#F5F5F7]'
          : isScrolled
            ? 'glass-nav py-3.5 shadow-apple'
            : 'bg-gradient-to-b from-black/95 via-black/50 to-transparent'
      )}
    >
      {/* Left: Prominent Arkflix Brand Logo & Nav Tabs */}
      <div className="flex items-center gap-8 lg:gap-12">
        <Logo size="md" theme={isProfilePage ? 'light' : 'dark'} />

        {/* Apple TV Navigation Pill Tabs (Larger typography & touch targets) */}
        <nav
          className={cn(
            'hidden md:flex items-center gap-1.5 backdrop-blur-2xl p-1.5 rounded-full border transition-all shadow-sm',
            isProfilePage
              ? 'bg-black/[0.05] border-black/10'
              : 'bg-white/[0.08] border-white/15'
          )}
        >
          {navLinks.map((link) => {
            const isActive = location.pathname === link.path
            return (
              <Link
                key={link.path}
                to={link.path}
                className={cn(
                  'px-5 py-2 text-sm font-bold rounded-full transition-all duration-200 tracking-tight',
                  isProfilePage
                    ? isActive
                      ? 'bg-white text-black shadow-md scale-105'
                      : 'text-[#515154] hover:text-[#1D1D1F] hover:bg-black/[0.04]'
                    : isActive
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

      {/* Right: Search, Cast, Profile Direct Link (Tight, balanced spacing) */}
      <div className="flex items-center gap-3 md:gap-4">
        {/* Search Bar */}
        <div className="relative flex items-center">
          {showSearch ? (
            <form
              onSubmit={handleSearchSubmit}
              className={cn(
                'flex items-center backdrop-blur-xl border px-3.5 py-2 rounded-full transition-all duration-300 shadow-sm',
                isProfilePage
                  ? 'bg-white border-black/20 shadow-md'
                  : 'bg-white/15 border-white/25'
              )}
            >
              <Search className={cn('w-4 h-4 mr-2', isProfilePage ? 'text-[#6E6E73]' : 'text-white/80')} />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t.nav.searchPlaceholder}
                className={cn(
                  'bg-transparent text-sm placeholder-apple-subtext focus:outline-none w-36 sm:w-48 md:w-64 font-medium',
                  isProfilePage ? 'text-black placeholder-[#86868B]' : 'text-white'
                )}
                autoFocus
                onBlur={() => !searchQuery && setShowSearch(false)}
              />
            </form>
          ) : (
            <button
              onClick={() => setShowSearch(true)}
              aria-label="Buscar"
              className={cn(
                'p-2.5 rounded-full transition-all hover:scale-105 active:scale-95',
                isProfilePage
                  ? 'text-[#1D1D1F] hover:bg-black/5'
                  : 'text-white/90 hover:text-white hover:bg-white/15'
              )}
            >
              <Search className="w-5 h-5" />
            </button>
          )}
        </div>

        {/* Google Cast Button Launcher */}
        <div
          className={cn(
            'w-10 h-10 flex items-center justify-center p-1 rounded-full transition-all hover:scale-105 active:scale-95',
            isProfilePage ? 'hover:bg-black/5' : 'hover:bg-white/15'
          )}
        >
          {React.createElement('google-cast-launcher', {
            class: cn(
              'w-5 h-5 cursor-pointer transition-opacity',
              isProfilePage ? 'opacity-90 hover:opacity-100 invert' : 'opacity-85 hover:opacity-100'
            ),
          })}
        </div>

        {/* User Profile Avatar (Prominent & High-Accessibility: 48px) */}
        <Link
          to="/profile"
          title="Perfis e Configurações"
          className="flex items-center cursor-pointer focus:outline-none group pl-1"
        >
          <div
            className={cn(
              'w-11 h-11 md:w-12 md:h-12 rounded-full overflow-hidden shadow-apple transition-all duration-300 group-hover:scale-105 group-hover:shadow-[0_0_20px_rgba(41,151,255,0.4)] bg-gradient-to-tr from-blue-500 to-indigo-600 flex items-center justify-center border-2',
              isProfilePage ? 'border-[#0071E3] ring-4 ring-blue-500/30' : 'border-white/40 group-hover:border-white'
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
              <span className="text-base md:text-lg font-extrabold text-white">
                {user?.Name?.charAt(0).toUpperCase() || <User className="w-5 h-5" />}
              </span>
            )}
          </div>
        </Link>
      </div>
    </header>
  )
}
