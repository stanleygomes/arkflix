import React from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Home, Film, Tv, Bookmark } from 'lucide-react'
import { useTranslation } from '@/hooks'
import { cn } from '@/lib/utils'

export const MobileTabBar: React.FC = () => {
  const { t } = useTranslation()
  const location = useLocation()

  const tabs = [
    { label: t.nav.home, path: '/', icon: Home },
    { label: t.nav.movies, path: '/movies', icon: Film },
    { label: t.nav.series, path: '/series', icon: Tv },
    { label: t.common.myList, path: '/my-list', icon: Bookmark },
  ]

  const isWatchPage = location.pathname.startsWith('/watch')
  if (isWatchPage) return null

  return (
    <nav className="md:hidden fixed bottom-0 left-0 right-0 z-40 px-4 py-2 flex items-center justify-around border-t backdrop-blur-2xl bg-[#000000]/90 border-white/10 text-white shadow-2xl pb-safe">
      {tabs.map((tab) => {
        const isActive = location.pathname === tab.path
        const Icon = tab.icon

        return (
          <Link
            key={tab.path}
            to={tab.path}
            className={cn(
              'flex flex-col items-center justify-center py-1 px-3 rounded-squircle-sm transition-all text-[11px] font-medium gap-0.5',
              isActive
                ? 'text-white font-bold scale-105'
                : 'text-apple-subtext hover:text-white'
            )}
          >
            <Icon className={cn('w-5 h-5', isActive && 'text-white stroke-[2.5]')} />
            <span>{tab.label}</span>
          </Link>
        )
      })}
    </nav>
  )
}
