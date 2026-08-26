import React from 'react'
import { cn } from '@/lib/utils'

export interface TabItem {
  id: string
  label: string
  icon?: React.ReactNode
}

export interface TabsProps {
  tabs: TabItem[]
  activeTab: string
  onChange: (tabId: string) => void
  className?: string
}

export const Tabs: React.FC<TabsProps> = ({ tabs, activeTab, onChange, className }) => {
  return (
    <div className={cn('flex items-center gap-2 border-b border-white/10 overflow-x-auto no-scrollbar', className)}>
      {tabs.map((tab) => {
        const isActive = tab.id === activeTab
        return (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            className={cn(
              'flex items-center gap-2 px-4 py-2.5 text-sm font-semibold transition-all border-b-2 whitespace-nowrap',
              isActive
                ? 'text-white border-netflix-red'
                : 'text-netflix-gray border-transparent hover:text-white hover:border-white/30'
            )}
          >
            {tab.icon && <span className="w-4 h-4">{tab.icon}</span>}
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}
