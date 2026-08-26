import React from 'react'
import { FeaturedCarousel } from '@/components/media/FeaturedCarousel'
import { MediaRow } from '@/components/media/MediaRow'
import { DetailModal } from '@/components/media/DetailModal'
import { useResumeItems, useLatestItems, useMovies, useSeries, useTranslation } from '@/hooks'

export const HomePage: React.FC = () => {
  const { t } = useTranslation()

  // 1ª Sessão Obrigatória: Continuar Assistindo (Resume Items)
  const { data: resumeItems, isLoading: loadingResume } = useResumeItems(16)

  // Recém Adicionados (Novidades)
  const { data: latestItems, isLoading: loadingLatest } = useLatestItems(undefined, 20)

  // Filmes
  const { data: moviesData, isLoading: loadingMovies } = useMovies({ limit: 24, sortBy: 'DateCreated' })

  // Séries
  const { data: seriesData, isLoading: loadingSeries } = useSeries({ limit: 24, sortBy: 'DateCreated' })

  // Filmes Melhor Avaliados
  const { data: topRatedMovies, isLoading: loadingTopMovies } = useMovies({
    limit: 20,
    sortBy: 'CommunityRating',
  })

  // Séries Mais Populares / Melhor Avaliadas
  const { data: topRatedSeries, isLoading: loadingTopSeries } = useSeries({
    limit: 20,
    sortBy: 'CommunityRating',
  })

  // Monta lista de destaques rotativos combinando filmes e séries recentes
  const featuredList = [
    ...(latestItems || []),
    ...(moviesData?.Items || []),
    ...(seriesData?.Items || []),
  ].filter(
    (item, index, self) => index === self.findIndex((t) => t.Id === item.Id)
  )

  const isHeroLoading = loadingLatest && loadingMovies

  return (
    <div className="pb-24">
      {/* Rotativo Dinâmico de Destaques (Hero Carousel da API) */}
      <FeaturedCarousel items={featuredList} isLoading={isHeroLoading} autoRotateInterval={7000} />

      {/* SESSÕES DA HOME - margem ajustada sem sobreposição aos botões do carousel */}
      <div className="-mt-4 sm:-mt-6 md:-mt-8 relative z-10 space-y-2 sm:space-y-4">
        {/* 1ª SESSÃO OBRIGATÓRIA: CONTINUAR ASSISTINDO */}
        {resumeItems && resumeItems.length > 0 && (
          <MediaRow
            title={t.home.continueWatching}
            items={resumeItems}
            isLoading={loadingResume}
          />
        )}

        {/* 2ª SESSÃO: NOVIDADES / RECÉM ADICIONADOS */}
        <MediaRow
          title={t.home.recentlyAdded}
          items={latestItems || []}
          isLoading={loadingLatest}
        />

        {/* 3ª SESSÃO: FILMES EM DESTAQUE */}
        <MediaRow
          title={t.home.featuredMovies}
          items={moviesData?.Items || []}
          isLoading={loadingMovies}
        />

        {/* 4ª SESSÃO: SÉRIES POPULARES */}
        <MediaRow
          title={t.home.popularSeries}
          items={seriesData?.Items || []}
          isLoading={loadingSeries}
        />

        {/* 5ª SESSÃO: FILMES MELHOR AVALIADOS (TOP CRÍTICA) */}
        {topRatedMovies?.Items && topRatedMovies.Items.length > 0 && (
          <MediaRow
            title="Filmes Aclamados pela Crítica"
            items={topRatedMovies.Items}
            isLoading={loadingTopMovies}
          />
        )}

        {/* 6ª SESSÃO: SÉRIES EM ALTA */}
        {topRatedSeries?.Items && topRatedSeries.Items.length > 0 && (
          <MediaRow
            title="Séries Mais Assistidas"
            items={topRatedSeries.Items}
            isLoading={loadingTopSeries}
          />
        )}
      </div>

      {/* Media Details Modal */}
      <DetailModal />
    </div>
  )
}
