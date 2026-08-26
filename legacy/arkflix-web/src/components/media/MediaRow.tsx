import React from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { MediaCard } from './MediaCard'

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
      <div className="my-6 px-4 md:px-12 space-y-3">
        <div className="h-6 w-48 bg-white/10 rounded animate-pulse"></div>
        <div className="flex gap-3 overflow-hidden">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="flex-none w-[160px] sm:w-[200px] md:w-[240px] aspect-[2/3] bg-netflix-dark rounded animate-pulse"
            />
          ))}
        </div>
      </div>
    )
  }

  if (!items || items.length === 0) return null

  return (
    <div className="group relative my-6 px-4 md:px-12">
      <h2 className="text-lg md:text-xl font-bold text-white mb-3 hover:text-netflix-red transition-colors cursor-pointer">
        {title}
      </h2>

      <div className="relative">
        {/* Scroll Left Button */}
        <button
          onClick={() => handleScroll('left')}
          className="absolute left-0 top-0 bottom-0 z-30 w-10 bg-black/60 text-white opacity-0 group-hover:opacity-100 hover:bg-black/80 transition-all flex items-center justify-center -ml-4 md:-ml-12 rounded-r"
          aria-label="Rolar para esquerda"
        >
          <ChevronLeft className="w-8 h-8" />
        </button>

        {/* Horizontal Row */}
        <div
          ref={rowRef}
          className="flex gap-3 overflow-x-auto scrollbar-hide py-3 scroll-smooth no-scrollbar"
          style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
        >
          {items.map((item) => (
            <MediaCard key={item.Id} item={item} />
          ))}
        </div>

        {/* Scroll Right Button */}
        <button
          onClick={() => handleScroll('right')}
          className="absolute right-0 top-0 bottom-0 z-30 w-10 bg-black/60 text-white opacity-0 group-hover:opacity-100 hover:bg-black/80 transition-all flex items-center justify-center -mr-4 md:-mr-12 rounded-l"
          aria-label="Rolar para direita"
        >
          <ChevronRight className="w-8 h-8" />
        </button>
      </div>
    </div>
  )
}
