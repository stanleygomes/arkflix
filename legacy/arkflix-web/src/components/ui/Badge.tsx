import React from 'react'
import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface BadgeProps {
  children: React.ReactNode
  variant?: 'default' | 'rating' | 'hd' | 'match' | 'glass'
  className?: string
}

export const Badge: React.FC<BadgeProps> = ({ children, variant = 'default', className }) => {
  const base =
    'inline-flex items-center gap-1 text-[11px] font-semibold rounded-squircle-sm px-2 py-0.5 tracking-tight'

  const variants = {
    default: 'bg-white/10 text-white backdrop-blur-md border border-white/10',
    rating: 'bg-[#1C1C1E] text-apple-subtext border border-white/15',
    hd: 'bg-white/15 text-white font-bold text-[10px] tracking-wider px-1.5 border border-white/20',
    match: 'text-emerald-400 bg-emerald-500/10 font-bold border border-emerald-500/20',
    glass: 'bg-white/10 text-[#F5F5F7] backdrop-blur-md border border-white/10',
  }

  return <span className={cn(base, variants[variant], className)}>{children}</span>
}

export const RatingBadge: React.FC<{ rating: number; className?: string }> = ({ rating, className }) => {
  return (
    <span className={cn('inline-flex items-center gap-1 text-xs font-semibold text-amber-400 bg-amber-400/10 border border-amber-400/20 px-2 py-0.5 rounded-squircle-sm', className)}>
      <Star className="w-3 h-3 fill-amber-400" />
      {rating.toFixed(1)}
    </span>
  )
}
