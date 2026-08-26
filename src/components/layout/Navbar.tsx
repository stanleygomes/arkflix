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
        'fixed top-0 left-0 right-0 z-40 transition-all duration-300 px-4 sm:px-6 md:px-14 py-2.5 sm:py-3.5 flex items-center justify-between',
        isProfilePage
          ? isScrolled
            ? 'bg-white/80 backdrop-blur-2xl border-b border-black/[0.08] shadow-sm py-2'
            : 'bg-[#F5F5F7]'
          : isScrolled
            ? 'glass-nav py-2 shadow-apple'
            : 'bg-gradient-to-b from-black/90 via-black/40 to-transparent'
      )}
    >
      {/* Left: Distinctive Arkflix Brand Logo & Nav Tabs */}
      <div className="flex items-center gap-4 sm:gap-10">
        <Logo size="md" theme={isProfilePage ? 'light' : 'dark'} />

        {/* Apple TV Navigation Pill Tabs (Desktop only, mobile uses bottom tab bar) */}
        <nav
          className={cn(
            'hidden md:flex items-center gap-1 backdrop-blur-xl p-1 rounded-full border transition-colors',
            isProfilePage
              ? 'bg-black/[0.04] border-black/10'
              : 'bg-white/[0.06] border-white/10'
          )}
        >
          {navLinks.map((link) => {
            const isActive = location.pathname === link.path
            return (
              <Link
                key={link.path}
                to={link.path}
                className={cn(
                  'px-4 py-1.5 text-xs font-semibold rounded-full transition-all duration-300',
                  isProfilePage
                    ? isActive
                      ? 'bg-white text-black shadow-sm'
                      : 'text-[#6E6E73] hover:text-[#1D1D1F] hover:bg-black/[0.04]'
                    : isActive
                      ? 'bg-white/20 text-white shadow-sm'
                      : 'text-apple-subtext hover:text-white hover:bg-white/5'
                )}
              >
                {link.label}
              </Link>
            )
          })}
        </nav>
      </div>

      {/* Right: Search, Cast, Profile Direct Link */}
      <div className="flex items-center gap-2 sm:gap-4">
        {/* Search Bar */}
        <div className="relative flex items-center">
          {showSearch ? (
            <form
              onSubmit={handleSearchSubmit}
              className={cn(
                'flex items-center backdrop-blur-xl border px-3 py-1.5 rounded-full transition-all duration-300',
                isProfilePage
                  ? 'bg-white border-black/15 shadow-sm'
                  : 'bg-white/15 border-white/20'
              )}
            >
              <Search className={cn('w-3.5 h-3.5 mr-1.5 sm:mr-2', isProfilePage ? 'text-[#86868B]' : 'text-apple-subtext')} />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t.nav.searchPlaceholder}
                className={cn(
                  'bg-transparent text-xs placeholder-apple-subtext focus:outline-none w-28 sm:w-36 md:w-56',
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
                'p-1.5 sm:p-2 rounded-full transition-all',
                isProfilePage
                  ? 'text-[#6E6E73] hover:text-[#1D1D1F] hover:bg-black/5'
                  : 'text-apple-subtext hover:text-white hover:bg-white/10'
              )}
            >
              <Search className="w-4 h-4 sm:w-4 sm:h-4" />
            </button>
          )}
        </div>

        {/* Google Cast Button Launcher */}
        <div
          className={cn(
            'w-7 h-7 sm:w-8 sm:h-8 flex items-center justify-center p-1 rounded-full transition-colors',
            isProfilePage ? 'hover:bg-black/5' : 'hover:bg-white/10'
          )}
        >
          {React.createElement('google-cast-launcher', {
            class: cn(
              'w-4 h-4 cursor-pointer transition-opacity',
              isProfilePage ? 'opacity-80 hover:opacity-100 invert' : 'opacity-70 hover:opacity-100'
            ),
          })}
        </div>

        {/* User Profile Avatar */}
        <Link
          to="/profile"
          title="Perfis e Configurações"
          className="flex items-center cursor-pointer focus:outline-none group ml-1"
        >
          <div
            className={cn(
              'w-8 h-8 sm:w-10 sm:h-10 rounded-full overflow-hidden shadow-sm transition-all duration-300 group-hover:scale-105 bg-gradient-to-tr from-blue-500 to-indigo-600 flex items-center justify-center border-2',
              isProfilePage ? 'border-[#0071E3] ring-2 ring-blue-500/20' : 'border-white/30 group-hover:border-white'
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
              <span className="text-xs sm:text-sm font-bold text-white">
                {user?.Name?.charAt(0).toUpperCase() || <User className="w-3.5 h-3.5 sm:w-4 sm:h-4" />}
              </span>
            )}
          </div>
        </Link>
      </div>
    </header>
  )
}
