import React from 'react'
import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface BadgeProps {
  children: React.ReactNode
  variant?: 'default' | 'rating' | 'hd' | 'match' | 'outline'
  className?: string
}

export const Badge: React.FC<BadgeProps> = ({ children, variant = 'default', className }) => {
  const base = 'inline-flex items-center gap-1 text-[11px] font-semibold rounded px-1.5 py-0.5 tracking-wide uppercase'

  const variants = {
    default: 'bg-white/10 text-white',
    rating: 'border border-gray-500 text-gray-300',
    hd: 'bg-transparent border border-white/40 text-white text-[9px] font-bold px-1',
    match: 'text-green-400 font-bold bg-transparent',
    outline: 'border border-white/20 text-netflix-lightGray',
  }

  return <span className={cn(base, variants[variant], className)}>{children}</span>
}

export const RatingBadge: React.FC<{ rating: number; className?: string }> = ({ rating, className }) => {
  return (
    <span className={cn('inline-flex items-center gap-1 text-xs font-bold text-green-400', className)}>
      <Star className="w-3.5 h-3.5 fill-green-400" />
      {rating.toFixed(1)}
    </span>
  )
}
