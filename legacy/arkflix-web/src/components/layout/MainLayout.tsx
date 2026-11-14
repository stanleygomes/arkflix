import React from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Navbar } from './Navbar'
import { MobileTabBar } from './MobileTabBar'
import { PageTransition } from './PageTransition'

export const MainLayout: React.FC = () => {
  const location = useLocation()

  return (
    <div className="min-h-screen bg-[#000000] flex flex-col selection:bg-white/20 pb-16 md:pb-0">
      <Navbar />
      <main className="flex-grow flex flex-col">
        <PageTransition key={location.pathname}>
          <Outlet />
        </PageTransition>
      </main>
      <MobileTabBar />
    </div>
  )
}
