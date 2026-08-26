import React from 'react'
import { Outlet } from 'react-router-dom'
import { Navbar } from './Navbar'

export const MainLayout: React.FC = () => {
  return (
    <div className="min-h-screen bg-netflix-black flex flex-col justify-between">
      <Navbar />
      <main className="flex-grow">
        <Outlet />
      </main>
      <footer className="px-6 md:px-12 py-8 text-xs text-netflix-gray border-t border-white/5">
        <p className="mb-2">Arkflix — Cliente Web Open Source para Jellyfin com Google Cast.</p>
        <p>© 2026 Arkflix. Interface inspirada na experiência Netflix.</p>
      </footer>
    </div>
  )
}
