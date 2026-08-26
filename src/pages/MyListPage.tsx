import React from 'react'
import { useFavorites, useTranslation } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { MediaCardSkeleton } from '@/components/ui'
import { Bookmark, Film } from 'lucide-react'

export const MyListPage: React.FC = () => {
  const { t } = useTranslation()
  const { data: favoritesData, isLoading } = useFavorites(80)
  const items = favoritesData?.Items || []

  return (
    <div className="min-h-screen pt-20 sm:pt-24 pb-20 px-4 sm:px-6 md:px-14 max-w-7xl mx-auto space-y-6 sm:space-y-8 animate-fadeIn">
      {/* Page Header */}
      <div className="flex items-center gap-3 border-b border-white/[0.08] pb-4 sm:pb-6">
        <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-squircle bg-blue-500/15 text-apple-accent flex items-center justify-center border border-blue-500/25 flex-none">
          <Bookmark className="w-4 h-4 sm:w-5 sm:h-5" />
        </div>
        <div>
          <h1 className="text-xl sm:text-2xl md:text-3xl font-extrabold text-white tracking-tight">{t.common.myList}</h1>
          <p className="text-[11px] sm:text-xs text-apple-subtext mt-0.5">
            {isLoading
              ? 'Carregando sua lista...'
              : `${items.length} títulos salvos para assistir`}
          </p>
        </div>
      </div>

      {/* Favorites Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
          {Array.from({ length: 12 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="py-24 text-center space-y-3">
          <Film className="w-12 h-12 text-apple-subtext mx-auto opacity-30" />
          <p className="text-base font-semibold text-white">Sua lista está vazia</p>
          <p className="text-xs text-apple-subtext max-w-sm mx-auto">
            Adicione filmes e séries aos seus favoritos clicando no botão de marcador para acessá-los facilmente aqui.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
          {items.map((item) => (
            <MediaCard key={item.Id} item={item} layout="grid" />
          ))}
        </div>
      )}
    </div>
  )
}
