import React from 'react'
import { useFavorites, useRefreshLibrary } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { MediaCardSkeleton, PullToRefresh } from '@/components/ui'
import { Film } from 'lucide-react'

export const MyListPage: React.FC = () => {
  const { data: favoritesData, isLoading, refetch } = useFavorites(80)
  const refreshLibraryMutation = useRefreshLibrary()
  const rawItems = favoritesData?.Items || []

  // Filter out series with 0 recursive items and deduplicate identical names
  const validItems = rawItems.filter((item) => {
    if (item.Type === 'Series' && typeof item.RecursiveItemCount === 'number') {
      return item.RecursiveItemCount > 0
    }
    return true
  })

  const itemsByName = new Map<string, (typeof rawItems)[0]>()
  for (const item of validItems) {
    const key = (item.Name || '').trim().toLowerCase()
    const existing = itemsByName.get(key)
    if (!existing) {
      itemsByName.set(key, item)
    } else {
      const existingCount = existing.RecursiveItemCount ?? existing.ChildCount ?? 0
      const currentCount = item.RecursiveItemCount ?? item.ChildCount ?? 0
      if (currentCount > existingCount) {
        itemsByName.set(key, item)
      }
    }
  }

  const items = Array.from(itemsByName.values())

  const handleRefresh = async () => {
    try {
      await refreshLibraryMutation.mutateAsync()
    } catch {}
    await refetch()
  }

  return (
    <PullToRefresh onRefresh={handleRefresh}>
      <div className="min-h-screen pt-[max(env(safe-area-inset-top,0px)+3.8rem,4.6rem)] sm:pt-24 pb-20 px-4 sm:px-6 md:px-14 max-w-7xl mx-auto space-y-4 sm:space-y-6 animate-fadeIn">
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
    </PullToRefresh>
  )
}
