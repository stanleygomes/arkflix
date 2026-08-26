import React from 'react'
import { HeroBanner } from '@/components/media/HeroBanner'
import { MediaRow } from '@/components/media/MediaRow'
import { DetailModal } from '@/components/media/DetailModal'
import { useResumeItems, useLatestItems, useMovies, useSeries } from '@/hooks'

export const HomePage: React.FC = () => {
  // Chamadas desacopladas via Hooks
  const { data: resumeItems, isLoading: loadingResume } = useResumeItems(12)
  const { data: latestItems, isLoading: loadingLatest } = useLatestItems(undefined, 16)
  const { data: moviesData, isLoading: loadingMovies } = useMovies({ limit: 16 })
  const { data: seriesData, isLoading: loadingSeries } = useSeries({ limit: 16 })

  const heroItem = latestItems?.[0] || moviesData?.Items?.[0]
  const isHeroLoading = loadingLatest && loadingMovies

  return (
    <div className="pb-16">
      {/* Hero Banner */}
      <HeroBanner item={heroItem} isLoading={isHeroLoading} />

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
