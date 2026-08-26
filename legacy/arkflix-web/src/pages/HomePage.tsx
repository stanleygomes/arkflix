import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'
import { HeroBanner } from '@/components/media/HeroBanner'
import { MediaRow } from '@/components/media/MediaRow'
import { DetailModal } from '@/components/media/DetailModal'

export const HomePage: React.FC = () => {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  // Continue watching (Resume)
  const { data: resumeItems, isLoading: loadingResume } = useQuery({
    queryKey: ['resumeItems', userId],
    queryFn: () => jellyfinService.getResumeItems(userId, 12),
    enabled: !!userId,
  })

  // Latest items
  const { data: latestItems, isLoading: loadingLatest } = useQuery({
    queryKey: ['latestItems', userId],
    queryFn: () => jellyfinService.getLatestItems(userId, undefined, 16),
    enabled: !!userId,
  })

  // Movies
  const { data: moviesData, isLoading: loadingMovies } = useQuery({
    queryKey: ['movies', userId],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        includeItemTypes: 'Movie',
        sortBy: 'DateCreated',
        sortOrder: 'Descending',
        limit: 16,
      }),
    enabled: !!userId,
  })

  // Series
  const { data: seriesData, isLoading: loadingSeries } = useQuery({
    queryKey: ['series', userId],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        includeItemTypes: 'Series',
        sortBy: 'DateCreated',
        sortOrder: 'Descending',
        limit: 16,
      }),
    enabled: !!userId,
  })

  const heroItem = latestItems?.[0] || moviesData?.Items?.[0]

  return (
    <div className="pb-16">
      {/* Hero Banner */}
      <HeroBanner item={heroItem} />

      {/* Media Carousels */}
      <div className="-mt-16 md:-mt-28 relative z-20 space-y-4">
        {resumeItems && resumeItems.length > 0 && (
          <MediaRow title="Continuar Assistindo" items={resumeItems} isLoading={loadingResume} />
        )}

        <MediaRow title="Adicionados Recentemente" items={latestItems || []} isLoading={loadingLatest} />

        <MediaRow title="Filmes em Destaque" items={moviesData?.Items || []} isLoading={loadingMovies} />

        <MediaRow title="Séries Populares" items={seriesData?.Items || []} isLoading={loadingSeries} />
      </div>

      {/* Media Details Modal */}
      <DetailModal />
    </div>
  )
}
