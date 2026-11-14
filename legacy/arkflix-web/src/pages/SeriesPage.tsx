import React, { useState } from 'react'
import { useSeries, useRefreshLibrary } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { MediaCardSkeleton, PullToRefresh, EmptyState } from '@/components/ui'
import { Clock, Star, ArrowDownAZ, Calendar, Tv } from 'lucide-react'
import { cn } from '@/lib/utils'

type SortOption = 'DateCreated' | 'CommunityRating' | 'SortName' | 'PremiereDate'

export const SeriesPage: React.FC = () => {
  const [sortBy, setSortBy] = useState<SortOption>('DateCreated')
  const refreshLibraryMutation = useRefreshLibrary()

  const { data: seriesData, isLoading, refetch } = useSeries({
    limit: 60,
    sortBy,
    sortOrder: 'Descending',
  })

  // Deduplicate and filter out empty ghost series and duplicate names
  const rawSeries = seriesData?.Items || []
  
  // 1. Exclude series that have 0 recursive items (empty ghost records)
  const validSeries = rawSeries.filter((item) => {
    if (typeof item.RecursiveItemCount === 'number') {
      return item.RecursiveItemCount > 0
    }
    return true
  })

  // 2. Consolidate series with the exact same name keeping the one with the most items/seasons
  const seriesByName = new Map<string, (typeof rawSeries)[0]>()
  for (const item of validSeries) {
    const key = (item.Name || '').trim().toLowerCase()
    const existing = seriesByName.get(key)
    if (!existing) {
      seriesByName.set(key, item)
    } else {
      const existingCount = existing.RecursiveItemCount ?? existing.ChildCount ?? 0
      const currentCount = item.RecursiveItemCount ?? item.ChildCount ?? 0
      if (currentCount > existingCount) {
        seriesByName.set(key, item)
      }
    }
  }

  const series = Array.from(seriesByName.values())

  const filterChips: { value: SortOption; label: string; icon: React.FC<{ className?: string }> }[] = [
    { value: 'DateCreated', label: 'Recentes', icon: Clock },
    { value: 'CommunityRating', label: 'Mais Votadas', icon: Star },
    { value: 'SortName', label: 'Ordem A-Z', icon: ArrowDownAZ },
    { value: 'PremiereDate', label: 'Ano', icon: Calendar },
  ]

  const handleRefresh = async () => {
    try {
      await refreshLibraryMutation.mutateAsync()
    } catch {}
    await refetch()
  }

  return (
    <PullToRefresh onRefresh={handleRefresh}>
      <div className="min-h-screen pt-[max(env(safe-area-inset-top,0px)+4.5rem,5.25rem)] sm:pt-28 pb-20 px-4 sm:px-6 md:px-14 max-w-7xl mx-auto space-y-4 sm:space-y-6 animate-fadeIn">
        {/* Apple Horizontal Segmented Filter Chips */}
        <div className="flex items-center justify-between gap-4 border-b border-white/[0.08] pb-3 sm:pb-4">
          <div className="flex items-center gap-2 overflow-x-auto pb-1 no-scrollbar -mx-4 px-4 sm:mx-0 sm:px-0 flex-1">
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
          <span className="text-[11px] sm:text-xs text-apple-subtext font-medium hidden sm:inline flex-none">
            {isLoading ? 'Carregando...' : `${series.length} séries`}
          </span>
        </div>

        {/* Series Grid */}
        {isLoading ? (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
            {Array.from({ length: 12 }).map((_, i) => (
              <MediaCardSkeleton key={i} />
            ))}
          </div>
        ) : series.length === 0 ? (
          <EmptyState
            icon={Tv}
            title="Nenhuma série encontrada"
            description="Não encontramos séries disponíveis na sua biblioteca Jellyfin. Puxe para atualizar ou sincronize seu servidor."
            actionLabel="Sincronizar Biblioteca"
            onAction={handleRefresh}
          />
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
            {series.map((item) => (
              <MediaCard key={item.Id} item={item} layout="grid" />
            ))}
          </div>
        )}
      </div>
    </PullToRefresh>
  )
}
