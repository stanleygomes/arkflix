import React from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { MediaCard } from './MediaCard'
import { MediaCardSkeleton, Skeleton } from '@/components/ui'

interface MediaRowProps {
  title: string
  items: MediaItem[]
  isLoading?: boolean
}

export const MediaRow: React.FC<MediaRowProps> = ({ title, items, isLoading }) => {
  const rowRef = React.useRef<HTMLDivElement>(null)

  const handleScroll = (direction: 'left' | 'right') => {
    if (rowRef.current) {
      const { scrollLeft, clientWidth } = rowRef.current
      const scrollAmount = direction === 'left' ? -clientWidth * 0.75 : clientWidth * 0.75
      rowRef.current.scrollTo({ left: scrollLeft + scrollAmount, behavior: 'smooth' })
    }
  }

  if (isLoading) {
    return (
      <div className="my-6 sm:my-8 px-4 sm:px-6 md:px-14 space-y-3">
        <Skeleton className="h-5 sm:h-6 w-36 sm:w-44" />
        <div className="flex gap-3 sm:gap-4 overflow-hidden">
          {Array.from({ length: 4 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      </div>
    )
  }

  if (!items || items.length === 0) return null

  return (
    <div className="relative my-5 sm:my-8 px-4 sm:px-6 md:px-14">
      <div className="flex items-center justify-between mb-2.5 sm:mb-3.5">
        <h2 className="text-base sm:text-lg md:text-xl font-bold text-white tracking-tight">
          {title}
        </h2>
      </div>

      <div className="relative group/row">
        {/* Apple Glass Scroll Left Button */}
        <button
          onClick={() => handleScroll('left')}
          className="hidden md:flex absolute left-0 top-1/2 -translate-y-1/2 z-30 w-10 h-10 -ml-5 bg-black/70 backdrop-blur-xl border border-white/15 text-white rounded-full opacity-0 group-hover/row:opacity-100 hover:bg-black/90 hover:scale-110 active:scale-95 transition-all items-center justify-center shadow-apple"
          aria-label="Rolar para esquerda"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>

        {/* Horizontal Row with touch momentum scroll on mobile */}
        <div
          ref={rowRef}
          className="flex gap-3 sm:gap-4 overflow-x-auto scrollbar-hide py-2 scroll-smooth no-scrollbar active:cursor-grabbing"
          style={{ scrollbarWidth: 'none', msOverflowStyle: 'none', WebkitOverflowScrolling: 'touch' }}
        >
          {items.map((item) => (
            <MediaCard key={item.Id} item={item} layout="carousel" />
          ))}
        </div>

        {/* Apple Glass Scroll Right Button */}
        <button
          onClick={() => handleScroll('right')}
          className="hidden md:flex absolute right-0 top-1/2 -translate-y-1/2 z-30 w-10 h-10 -mr-5 bg-black/70 backdrop-blur-xl border border-white/15 text-white rounded-full opacity-0 group-hover/row:opacity-100 hover:bg-black/90 hover:scale-110 active:scale-95 transition-all items-center justify-center shadow-apple"
          aria-label="Rolar para direita"
        >
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>
    </div>
  )
}
