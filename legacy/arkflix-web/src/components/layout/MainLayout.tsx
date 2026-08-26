import React from 'react'
import { Outlet } from 'react-router-dom'
import { Navbar } from './Navbar'
import { useTranslation } from '@/hooks'

export const MainLayout: React.FC = () => {
  const { t } = useTranslation()

  return (
    <div className="min-h-screen bg-apple-bg flex flex-col justify-between selection:bg-white/20">
      <Navbar />
      <main className="flex-grow">
        <Outlet />
      </main>
      <footer className="px-6 md:px-14 py-10 text-xs text-apple-subtext border-t border-white/[0.06] flex flex-col md:flex-row justify-between items-center gap-4">
        <p>{t.footer.copyright}</p>
        <div className="flex items-center gap-6">
          <span className="hover:text-white transition-colors cursor-pointer">{t.footer.privacy}</span>
          <span className="hover:text-white transition-colors cursor-pointer">{t.footer.terms}</span>
          <span className="hover:text-white transition-colors cursor-pointer">{t.footer.support}</span>
        </div>
      </footer>
    </div>
  )
}
