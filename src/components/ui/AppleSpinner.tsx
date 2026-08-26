import React from 'react'
import { cn } from '@/lib/utils'

export interface AppleSpinnerProps {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  color?: 'white' | 'gray' | 'blue'
  className?: string
}

export const AppleSpinner: React.FC<AppleSpinnerProps> = ({
  size = 'md',
  color = 'white',
  className,
}) => {
  const sizeClasses = {
    xs: 'w-3.5 h-3.5',
    sm: 'w-4 h-4',
    md: 'w-6 h-6',
    lg: 'w-8 h-8',
    xl: 'w-12 h-12',
  }

  const colorClasses = {
    white: 'text-white',
    gray: 'text-apple-subtext',
    blue: 'text-apple-accent',
  }

  // iOS-style 8-segment circular loader
  const segments = Array.from({ length: 8 })

  return (
    <div
      role="status"
      aria-label="Carregando"
      className={cn('relative inline-flex items-center justify-center', sizeClasses[size], className)}
    >
      <svg
        className={cn('w-full h-full animate-spin', colorClasses[color])}
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        style={{ animationDuration: '0.9s', animationTimingFunction: 'steps(8, end)' }}
      >
        {segments.map((_, i) => {
          const rotation = i * 45
          const opacity = (i + 1) / 8
          return (
            <line
              key={i}
              x1="12"
              y1="3"
              x2="12"
              y2="6.5"
              stroke="currentColor"
              strokeWidth="2.2"
              strokeLinecap="round"
              transform={`rotate(${rotation} 12 12)`}
              style={{ opacity }}
            />
          )
        })}
      </svg>
    </div>
  )
}
