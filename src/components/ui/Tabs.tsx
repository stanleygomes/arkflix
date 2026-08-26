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
    <div
      className={cn(
        'inline-flex items-center p-1 bg-[#1C1C1E]/80 backdrop-blur-xl border border-white/10 rounded-full select-none overflow-x-auto no-scrollbar',
        className
      )}
    >
      {tabs.map((tab) => {
        const isActive = tab.id === activeTab
        return (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            className={cn(
              'relative flex items-center gap-2 px-4 py-1.5 text-xs font-semibold rounded-full transition-all duration-300 whitespace-nowrap',
              isActive
                ? 'bg-white/20 text-white shadow-sm'
                : 'text-apple-subtext hover:text-white hover:bg-white/5'
            )}
          >
            {tab.icon && <span className="w-3.5 h-3.5">{tab.icon}</span>}
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}
