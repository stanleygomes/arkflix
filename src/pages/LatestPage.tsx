import React from 'react'
import { useLatestItems, useTranslation } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { MediaCardSkeleton } from '@/components/ui'
import { Flame } from 'lucide-react'

export const LatestPage: React.FC = () => {
  const { t } = useTranslation()
  const { data: latestItems, isLoading } = useLatestItems(undefined, 48)

  return (
    <div className="min-h-screen pt-24 pb-20 px-6 md:px-14 max-w-7xl mx-auto space-y-8 animate-fadeIn">
      {/* Page Header */}
      <div className="flex items-center gap-3 border-b border-white/[0.08] pb-6">
        <div className="w-10 h-10 rounded-squircle bg-amber-500/15 text-amber-400 flex items-center justify-center border border-amber-500/25">
          <Flame className="w-5 h-5" />
        </div>
        <div>
          <h1 className="text-2xl md:text-3xl font-extrabold text-white tracking-tight">{t.nav.trending}</h1>
          <p className="text-xs text-apple-subtext mt-0.5">Títulos recém-adicionados e novidades na sua biblioteca</p>
        </div>
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
          {Array.from({ length: 18 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      ) : !latestItems || latestItems.length === 0 ? (
        <div className="py-20 text-center space-y-2">
          <p className="text-sm text-apple-subtext">Nenhum item recente encontrado.</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
          {latestItems.map((item) => (
            <MediaCard key={item.Id} item={item} />
          ))}
        </div>
      )}

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
