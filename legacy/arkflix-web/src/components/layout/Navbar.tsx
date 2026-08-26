import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Search, Bell, User, LogOut } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { cn } from '@/lib/utils'

export const Navbar: React.FC = () => {
  const [isScrolled, setIsScrolled] = React.useState(false)
  const [showSearch, setShowSearch] = React.useState(false)
  const [searchQuery, setSearchQuery] = React.useState('')
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 50)
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

  return (
    <header
      className={cn(
        'fixed top-0 left-0 right-0 z-50 transition-colors duration-300 px-4 md:px-12 py-3 flex items-center justify-between',
        isScrolled ? 'bg-netflix-black shadow-lg' : 'bg-gradient-to-b from-black/80 via-black/40 to-transparent'
      )}
    >
      {/* Left section: Logo & Nav Links */}
      <div className="flex items-center gap-8">
        <Link to="/" className="text-netflix-red font-black text-2xl md:text-3xl tracking-tighter uppercase">
          Arkflix
        </Link>

        <nav className="hidden md:flex items-center gap-5 text-sm text-netflix-lightGray">
          <Link to="/" className="hover:text-white font-medium transition-colors">Início</Link>
          <Link to="/series" className="hover:text-white transition-colors">Séries</Link>
          <Link to="/movies" className="hover:text-white transition-colors">Filmes</Link>
          <Link to="/latest" className="hover:text-white transition-colors">Bombando</Link>
          <Link to="/my-list" className="hover:text-white transition-colors">Minha Lista</Link>
        </nav>
      </div>

      {/* Right section: Search, Cast, User Profile */}
      <div className="flex items-center gap-4 text-white">
        {/* Search Bar */}
        <div className="relative flex items-center">
          {showSearch ? (
            <form onSubmit={handleSearchSubmit} className="flex items-center border border-white/40 bg-black/60 px-2 py-1 rounded">
              <Search className="w-4 h-4 text-gray-400 mr-2" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Títulos, séries, filmes..."
                className="bg-transparent text-sm text-white focus:outline-none w-36 md:w-56"
                autoFocus
                onBlur={() => !searchQuery && setShowSearch(false)}
              />
            </form>
          ) : (
            <button
              onClick={() => setShowSearch(true)}
              aria-label="Buscar"
              className="p-1 hover:text-gray-300 transition-colors"
            >
              <Search className="w-5 h-5" />
            </button>
          )}
        </div>

        {/* Google Cast Button Launcher */}
        <div className="w-6 h-6 flex items-center justify-center">
          {React.createElement('google-cast-launcher', {
            class: 'w-6 h-6 cursor-pointer opacity-80 hover:opacity-100 transition-opacity',
          })}
        </div>

        <button aria-label="Notificações" className="p-1 hover:text-gray-300 transition-colors hidden sm:block">
          <Bell className="w-5 h-5" />
        </button>

        {/* User Menu */}
        <div className="relative group">
          <div className="flex items-center gap-2 cursor-pointer">
            <div className="w-8 h-8 rounded bg-netflix-red flex items-center justify-center font-bold text-sm">
              {user?.Name?.charAt(0).toUpperCase() || <User className="w-4 h-4" />}
            </div>
          </div>

          {/* Dropdown */}
          <div className="absolute right-0 top-full mt-2 w-48 bg-netflix-dark border border-white/10 rounded shadow-2xl py-2 hidden group-hover:block">
            <div className="px-4 py-2 text-xs text-gray-400 border-b border-white/10">
              Conectado como <strong className="text-white block truncate">{user?.Name || 'Convidado'}</strong>
            </div>
            <button
              onClick={logout}
              className="w-full px-4 py-2 text-left text-sm text-netflix-lightGray hover:bg-white/10 flex items-center gap-2"
            >
              <LogOut className="w-4 h-4" /> Sair do Arkflix
            </button>
          </div>
        </div>
      </div>
    </header>
  )
}
