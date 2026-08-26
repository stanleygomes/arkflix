import React, { useState } from 'react'
import { useSeries, useTranslation } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { Select, MediaCardSkeleton } from '@/components/ui'
import { Tv, Filter } from 'lucide-react'

export const SeriesPage: React.FC = () => {
  const { t } = useTranslation()
  const [sortBy, setSortBy] = useState<'DateCreated' | 'CommunityRating' | 'SortName' | 'PremiereDate'>('DateCreated')

  const { data: seriesData, isLoading } = useSeries({
    limit: 60,
    sortBy,
    sortOrder: 'Descending',
  })

  const series = seriesData?.Items || []

  return (
    <div className="min-h-screen pt-24 pb-20 px-6 md:px-14 max-w-7xl mx-auto space-y-8 animate-fadeIn">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-white/[0.08] pb-6">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-squircle bg-white/10 flex items-center justify-center text-white border border-white/15">
            <Tv className="w-5 h-5" />
          </div>
          <div>
            <h1 className="text-2xl md:text-3xl font-extrabold text-white tracking-tight">{t.nav.series}</h1>
            <p className="text-xs text-apple-subtext mt-0.5">Explore temporadas e episódios das suas séries favoritas</p>
          </div>
        </div>

        {/* Sort & Filter Controls */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 text-xs text-apple-subtext">
            <Filter className="w-3.5 h-3.5" />
            <span>Ordenar por:</span>
          </div>

          <div className="w-44">
            <Select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as any)}
              options={[
                { value: 'DateCreated', label: 'Adicionadas Recentemente' },
                { value: 'CommunityRating', label: 'Melhor Avaliadas' },
                { value: 'SortName', label: 'Ordem Alfabética (A-Z)' },
                { value: 'PremiereDate', label: 'Ano de Lançamento' },
              ]}
            />
          </div>
        </div>
      </div>

      {/* Series Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
          {Array.from({ length: 18 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      ) : series.length === 0 ? (
        <div className="py-20 text-center space-y-2">
          <p className="text-sm text-apple-subtext">Nenhuma série encontrada na sua biblioteca.</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
          {series.map((item) => (
            <MediaCard key={item.Id} item={item} />
          ))}
        </div>
      )}

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
