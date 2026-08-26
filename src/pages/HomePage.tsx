import React from 'react'
import { HeroBanner } from '@/components/media/HeroBanner'
import { MediaRow } from '@/components/media/MediaRow'
import { DetailModal } from '@/components/media/DetailModal'
import { useResumeItems, useLatestItems, useMovies, useSeries, useTranslation } from '@/hooks'

export const HomePage: React.FC = () => {
  const { t } = useTranslation()

  // Chamadas desacopladas via Hooks
  const { data: resumeItems, isLoading: loadingResume } = useResumeItems(12)
  const { data: latestItems, isLoading: loadingLatest } = useLatestItems(undefined, 16)
  const { data: moviesData, isLoading: loadingMovies } = useMovies({ limit: 16 })
  const { data: seriesData, isLoading: loadingSeries } = useSeries({ limit: 16 })

  const heroItem = latestItems?.[0] || moviesData?.Items?.[0]
  const isHeroLoading = loadingLatest && loadingMovies

  return (
    <div className="pb-20">
      {/* Hero Banner */}
      <HeroBanner item={heroItem} isLoading={isHeroLoading} />

      {/* Media Carousels */}
      <div className="-mt-16 md:-mt-24 relative z-20 space-y-6">
        {resumeItems && resumeItems.length > 0 && (
          <MediaRow title={t.home.continueWatching} items={resumeItems} isLoading={loadingResume} />
        )}

        <MediaRow title={t.home.recentlyAdded} items={latestItems || []} isLoading={loadingLatest} />

        <MediaRow title={t.home.featuredMovies} items={moviesData?.Items || []} isLoading={loadingMovies} />

        <MediaRow title={t.home.popularSeries} items={seriesData?.Items || []} isLoading={loadingSeries} />
      </div>

      {/* Media Details Modal */}
      <DetailModal />
    </div>
  )
}
