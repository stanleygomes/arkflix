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
      <div className="my-8 px-6 md:px-14 space-y-3">
        <Skeleton className="h-6 w-44" />
        <div className="flex gap-4 overflow-hidden">
          {Array.from({ length: 6 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      </div>
    )
  }

  if (!items || items.length === 0) return null

  return (
    <div className="group relative my-8 px-6 md:px-14">
      <div className="flex items-center justify-between mb-3.5">
        <h2 className="text-lg md:text-xl font-bold text-white tracking-tight">
          {title}
        </h2>
      </div>

      <div className="relative">
        {/* Apple Glass Scroll Left Button */}
        <button
          onClick={() => handleScroll('left')}
          className="absolute left-0 top-1/2 -translate-y-1/2 z-30 w-10 h-10 -ml-5 bg-black/60 backdrop-blur-xl border border-white/10 text-white rounded-full opacity-0 group-hover:opacity-100 hover:bg-black/80 hover:scale-110 active:scale-95 transition-all flex items-center justify-center shadow-apple"
          aria-label="Rolar para esquerda"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>

        {/* Horizontal Row */}
        <div
          ref={rowRef}
          className="flex gap-4 overflow-x-auto scrollbar-hide py-2 scroll-smooth no-scrollbar"
          style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
        >
          {items.map((item) => (
            <MediaCard key={item.Id} item={item} />
          ))}
        </div>

        {/* Apple Glass Scroll Right Button */}
        <button
          onClick={() => handleScroll('right')}
          className="absolute right-0 top-1/2 -translate-y-1/2 z-30 w-10 h-10 -mr-5 bg-black/60 backdrop-blur-xl border border-white/10 text-white rounded-full opacity-0 group-hover:opacity-100 hover:bg-black/80 hover:scale-110 active:scale-95 transition-all flex items-center justify-center shadow-apple"
          aria-label="Rolar para direita"
        >
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>
    </div>
  )
}
