import React from 'react'
import { Outlet } from 'react-router-dom'
import { Navbar } from './Navbar'

export const MainLayout: React.FC = () => {
  return (
    <div className="min-h-screen bg-apple-bg flex flex-col justify-between selection:bg-white/20">
      <Navbar />
      <main className="flex-grow">
        <Outlet />
      </main>
      <footer className="px-6 md:px-14 py-10 text-xs text-apple-subtext border-t border-white/[0.06] flex flex-col md:flex-row justify-between items-center gap-4">
        <p>© 2026 Arkflix. Experiência de Streaming Jellyfin com Apple Human Interface Guidelines.</p>
        <div className="flex items-center gap-6">
          <span className="hover:text-white transition-colors cursor-pointer">Privacidade</span>
          <span className="hover:text-white transition-colors cursor-pointer">Termos</span>
          <span className="hover:text-white transition-colors cursor-pointer">Suporte</span>
        </div>
      </footer>
    </div>
  )
}
