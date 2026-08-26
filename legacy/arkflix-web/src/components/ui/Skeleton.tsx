import React from 'react'
import { cn } from '@/lib/utils'

export const Skeleton: React.FC<{ className?: string }> = ({ className }) => {
  return <div className={cn('animate-pulse bg-white/10 rounded', className)} />
}

export const MediaCardSkeleton: React.FC = () => {
  return (
    <div className="flex-none w-[160px] sm:w-[200px] md:w-[240px] aspect-[2/3] bg-netflix-dark rounded-md animate-pulse border border-white/5" />
  )
}

export const HeroBannerSkeleton: React.FC = () => {
  return (
    <div className="relative h-[65vh] md:h-[80vh] w-full bg-netflix-dark animate-pulse flex items-end pb-24 px-4 md:px-12">
      <div className="space-y-4 max-w-xl">
        <Skeleton className="h-10 w-3/4" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-2/3" />
        <div className="flex gap-3 pt-2">
          <Skeleton className="h-10 w-32" />
          <Skeleton className="h-10 w-40" />
        </div>
      </div>
    </div>
  )
}
