import React from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Search, Bell, User, LogOut, Settings } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation } from '@/hooks'
import { Logo } from '@/components/ui'
import { cn } from '@/lib/utils'

export const Navbar: React.FC = () => {
  const [isScrolled, setIsScrolled] = React.useState(false)
  const [showSearch, setShowSearch] = React.useState(false)
  const [searchQuery, setSearchQuery] = React.useState('')
  const { user, logout } = useAuthStore()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()

  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 30)
    }
    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery)}`)
    }
  }

  const navLinks = [
    { label: t.nav.home, path: '/' },
    { label: t.nav.movies, path: '/movies' },
    { label: t.nav.series, path: '/series' },
    { label: t.nav.trending, path: '/latest' },
    { label: t.nav.library, path: '/my-list' },
  ]

  return (
    <header
      className={cn(
        'fixed top-0 left-0 right-0 z-50 transition-all duration-500 px-6 md:px-14 py-3.5 flex items-center justify-between',
        isScrolled
          ? 'glass-nav py-2.5 shadow-apple'
          : 'bg-gradient-to-b from-black/85 via-black/30 to-transparent'
      )}
    >
      {/* Left: Distinctive Arkflix Brand Logo & Nav Tabs */}
      <div className="flex items-center gap-10">
        <Logo size="md" />

        {/* Apple TV Navigation Pill Tabs */}
        <nav className="hidden md:flex items-center gap-1 bg-white/[0.06] backdrop-blur-xl p-1 rounded-full border border-white/10">
          {navLinks.map((link) => {
            const isActive = location.pathname === link.path
            return (
              <Link
                key={link.path}
                to={link.path}
                className={cn(
                  'px-4 py-1.5 text-xs font-semibold rounded-full transition-all duration-300',
                  isActive
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

      {/* Right: Search, Cast, Profile Menu */}
      <div className="flex items-center gap-4 text-white">
        {/* Apple Glass Search Bar */}
        <div className="relative flex items-center">
          {showSearch ? (
            <form onSubmit={handleSearchSubmit} className="flex items-center bg-white/10 backdrop-blur-xl border border-white/15 px-3 py-1.5 rounded-full transition-all duration-300">
              <Search className="w-3.5 h-3.5 text-apple-subtext mr-2" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t.nav.searchPlaceholder}
                className="bg-transparent text-xs text-white placeholder-apple-subtext focus:outline-none w-36 md:w-56"
                autoFocus
                onBlur={() => !searchQuery && setShowSearch(false)}
              />
            </form>
          ) : (
            <button
              onClick={() => setShowSearch(true)}
              aria-label="Buscar"
              className="p-2 rounded-full hover:bg-white/10 text-apple-subtext hover:text-white transition-all"
            >
              <Search className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Google Cast Button Launcher */}
        <div className="w-7 h-7 flex items-center justify-center p-1 rounded-full hover:bg-white/10 transition-colors">
          {React.createElement('google-cast-launcher', {
            class: 'w-4 h-4 cursor-pointer opacity-70 hover:opacity-100 transition-opacity',
          })}
        </div>

        <button
          aria-label="Notificações"
          className="p-2 rounded-full hover:bg-white/10 text-apple-subtext hover:text-white transition-colors hidden sm:flex"
        >
          <Bell className="w-4 h-4" />
        </button>

        {/* User Profile Avatar with Apple Dropdown */}
        <div className="relative group">
          <button
            onClick={() => navigate('/profile')}
            className="flex items-center gap-2 cursor-pointer focus:outline-none"
          >
            <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-blue-500 to-indigo-600 flex items-center justify-center text-xs font-semibold text-white border border-white/20 shadow-sm transition-transform group-hover:scale-105">
              {user?.Name?.charAt(0).toUpperCase() || <User className="w-3.5 h-3.5" />}
            </div>
          </button>

          {/* Apple Glass Dropdown */}
          <div className="absolute right-0 top-full mt-3 w-56 glass-panel rounded-squircle-lg p-2 shadow-apple hidden group-hover:block transition-all animate-fadeIn">
            <div className="px-3 py-2 border-b border-white/10">
              <p className="text-[11px] text-apple-subtext">{t.common.connectedAs}</p>
              <p className="text-xs font-semibold text-white truncate">{user?.Name || t.common.guest}</p>
            </div>

            <Link
              to="/profile"
              className="w-full mt-1 px-3 py-2 text-left text-xs font-medium text-white/90 hover:bg-white/10 rounded-squircle-sm flex items-center gap-2 transition-colors"
            >
              <Settings className="w-3.5 h-3.5 text-apple-subtext" /> {t.nav.profileSettings}
            </Link>

            <button
              onClick={logout}
              className="w-full px-3 py-2 text-left text-xs font-medium text-red-400 hover:bg-white/10 rounded-squircle-sm flex items-center gap-2 transition-colors"
            >
              <LogOut className="w-3.5 h-3.5" /> {t.common.logout}
            </button>
          </div>
        </div>
      </div>
    </header>
  )
}
