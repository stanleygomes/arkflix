import React from 'react'
import { cn } from '@/lib/utils'

export const Skeleton: React.FC<{ className?: string }> = ({ className }) => {
  return <div className={cn('animate-pulse bg-white/[0.07] rounded-squircle', className)} />
}

export const MediaCardSkeleton: React.FC = () => {
  return (
    <div className="flex-none w-[170px] sm:w-[210px] md:w-[260px] aspect-[16/10] bg-white/[0.06] rounded-squircle-lg animate-pulse border border-white/5" />
  )
}

export const HeroBannerSkeleton: React.FC = () => {
  return (
    <div className="relative h-[65vh] md:h-[78vh] w-full bg-[#111113] animate-pulse flex items-end pb-20 px-6 md:px-16">
      <div className="space-y-4 max-w-xl">
        <Skeleton className="h-10 w-3/4 rounded-squircle" />
        <Skeleton className="h-4 w-full rounded-squircle-sm" />
        <Skeleton className="h-4 w-2/3 rounded-squircle-sm" />
        <div className="flex gap-3 pt-3">
          <Skeleton className="h-11 w-32 rounded-squircle" />
          <Skeleton className="h-11 w-40 rounded-squircle" />
        </div>
      </div>
    </div>
  )
}
