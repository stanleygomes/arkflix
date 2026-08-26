import React from 'react'
import { motion } from 'framer-motion'
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
              'relative flex items-center gap-2 px-4 py-1.5 text-xs font-semibold rounded-full transition-colors duration-200 whitespace-nowrap z-10',
              isActive ? 'text-white' : 'text-apple-subtext hover:text-white'
            )}
          >
            {/* Sliding Pill Background with Apple Spring Physics */}
            {isActive && (
              <motion.div
                layoutId="activeTabPill"
                transition={{
                  type: 'spring',
                  stiffness: 450,
                  damping: 35,
                }}
                className="absolute inset-0 bg-white/20 rounded-full shadow-sm -z-10 border border-white/10"
              />
            )}
            {tab.icon && <span className="w-3.5 h-3.5">{tab.icon}</span>}
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}
