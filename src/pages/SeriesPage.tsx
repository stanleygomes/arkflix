import React, { useState } from 'react'
import { useSeries, useTranslation } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { MediaCardSkeleton } from '@/components/ui'
import { Tv, Clock, Star, ArrowDownAZ, Calendar } from 'lucide-react'
import { cn } from '@/lib/utils'

type SortOption = 'DateCreated' | 'CommunityRating' | 'SortName' | 'PremiereDate'

export const SeriesPage: React.FC = () => {
  const { t } = useTranslation()
  const [sortBy, setSortBy] = useState<SortOption>('DateCreated')

  const { data: seriesData, isLoading } = useSeries({
    limit: 60,
    sortBy,
    sortOrder: 'Descending',
  })

  const series = seriesData?.Items || []

  const filterChips: { value: SortOption; label: string; icon: React.FC<{ className?: string }> }[] = [
    { value: 'DateCreated', label: 'Recentes', icon: Clock },
    { value: 'CommunityRating', label: 'Mais Votadas', icon: Star },
    { value: 'SortName', label: 'Ordem A-Z', icon: ArrowDownAZ },
    { value: 'PremiereDate', label: 'Ano', icon: Calendar },
  ]

  return (
    <div className="min-h-screen pt-[max(env(safe-area-inset-top,0px)+4.5rem,5.5rem)] sm:pt-28 pb-20 px-4 sm:px-6 md:px-14 max-w-7xl mx-auto space-y-5 sm:space-y-7 animate-fadeIn">
      {/* Page Header - Apple iOS Styled */}
      <div className="flex flex-col gap-4 border-b border-white/[0.08] pb-4 sm:pb-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 sm:w-11 sm:h-11 rounded-squircle bg-gradient-to-tr from-purple-500/20 to-indigo-500/20 flex items-center justify-center text-purple-400 border border-purple-500/30 shadow-sm flex-none">
              <Tv className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight leading-none">{t.nav.series}</h1>
              <p className="text-xs text-apple-subtext mt-1">
                {isLoading ? 'Carregando séries...' : `${series.length} séries com temporadas`}
              </p>
            </div>
          </div>
        </div>

        {/* Apple Horizontal Segmented Filter Chips for Mobile & Desktop */}
        <div className="flex items-center gap-2 overflow-x-auto pb-1 no-scrollbar -mx-4 px-4 sm:mx-0 sm:px-0">
          {filterChips.map((chip) => {
            const isActive = sortBy === chip.value
            const Icon = chip.icon

            return (
              <button
                key={chip.value}
                onClick={() => setSortBy(chip.value)}
                className={cn(
                  'flex items-center gap-1.5 px-3.5 py-1.5 sm:py-2 rounded-full text-xs font-semibold whitespace-nowrap transition-all duration-200 cursor-pointer flex-none border active:scale-95',
                  isActive
                    ? 'bg-white text-black border-white shadow-apple font-bold'
                    : 'bg-white/[0.06] text-apple-subtext hover:text-white hover:bg-white/10 border-white/10'
                )}
              >
                <Icon className={cn('w-3.5 h-3.5', isActive ? 'text-black' : 'text-apple-subtext')} />
                <span>{chip.label}</span>
              </button>
            )
          })}
        </div>
      </div>

      {/* Series Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
          {Array.from({ length: 12 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      ) : series.length === 0 ? (
        <div className="py-20 text-center space-y-2">
          <p className="text-sm text-apple-subtext">Nenhuma série encontrada na sua biblioteca.</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
          {series.map((item) => (
            <MediaCard key={item.Id} item={item} layout="grid" />
          ))}
        </div>
      )}

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
