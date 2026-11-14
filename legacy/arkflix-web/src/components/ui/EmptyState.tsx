import React from 'react'
import { LucideIcon } from 'lucide-react'
import { Button } from './Button'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description?: string
  actionLabel?: string
  onAction?: () => void
  className?: string
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon: Icon,
  title,
  description,
  actionLabel,
  onAction,
  className,
}) => {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center px-4 py-16 sm:py-24 max-w-md mx-auto space-y-4 animate-fadeIn',
        className
      )}
    >
      {/* Icon Squircle Badge with Glow */}
      <div className="w-16 h-16 sm:w-20 sm:h-20 rounded-squircle-lg bg-white/[0.04] border border-white/10 flex items-center justify-center text-apple-subtext shadow-apple backdrop-blur-xl relative group">
        <div className="absolute inset-0 rounded-squircle-lg bg-blue-500/10 opacity-0 group-hover:opacity-100 transition-opacity blur-md" />
        <Icon className="w-7 h-7 sm:w-9 sm:h-9 text-apple-subtext/70 relative z-10 transition-transform group-hover:scale-110" />
      </div>

      {/* Typography */}
      <div className="space-y-1.5">
        <h3 className="text-base sm:text-lg font-bold text-white tracking-tight">
          {title}
        </h3>
        {description && (
          <p className="text-xs sm:text-sm text-apple-subtext max-w-xs sm:max-w-sm mx-auto leading-relaxed">
            {description}
          </p>
        )}
      </div>

      {/* Optional CTA Action Button */}
      {actionLabel && onAction && (
        <div className="pt-2">
          <Button
            variant="glass"
            size="md"
            onClick={onAction}
            className="text-xs sm:text-sm font-semibold px-5 shadow-apple text-white hover:text-apple-accent hover:border-blue-500/40"
          >
            {actionLabel}
          </Button>
        </div>
      )}
    </div>
  )
}
