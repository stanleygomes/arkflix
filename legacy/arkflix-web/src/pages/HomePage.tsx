import React from 'react'
import { FeaturedCarousel } from '@/components/media/FeaturedCarousel'
import { MediaRow } from '@/components/media/MediaRow'
import { DetailModal } from '@/components/media/DetailModal'
import { useResumeItems, useLatestItems, useMovies, useSeries, useFavorites, useTranslation } from '@/hooks'

export const HomePage: React.FC = () => {
  const { t } = useTranslation()

  // 1ª Sessão: Continuar Assistindo (Resume Items)
  const { data: resumeItems, isLoading: loadingResume } = useResumeItems(16)

  // 2ª Sessão: Minha Lista (Favoritos)
  const { data: favoritesData, isLoading: loadingFavorites } = useFavorites(24)
  const rawFavoriteItems = favoritesData?.Items || []
  const favoriteItems = rawFavoriteItems
    .filter((item) => !(item.Type === 'Series' && item.RecursiveItemCount === 0))
    .filter((item, index, self) => index === self.findIndex((t) => (t.Name || '').toLowerCase() === (item.Name || '').toLowerCase()))

  // 3ª Sessão: Novidades / Adicionados Recentemente
  const { data: latestItems, isLoading: loadingLatest } = useLatestItems(undefined, 20)

  // 4ª Sessão: Filmes em Destaque
  const { data: moviesData, isLoading: loadingMovies } = useMovies({ limit: 24, sortBy: 'DateCreated' })

  // 5ª Sessão: Séries Populares
  const { data: seriesData, isLoading: loadingSeries } = useSeries({ limit: 24, sortBy: 'DateCreated' })
  const rawSeries = seriesData?.Items || []
  const seriesItems = rawSeries
    .filter((item) => !(item.Type === 'Series' && item.RecursiveItemCount === 0))
    .filter((item, index, self) => index === self.findIndex((t) => (t.Name || '').toLowerCase() === (item.Name || '').toLowerCase()))

  // Destaques Rotativos no Hero Banner:
  // Se o usuário tiver itens em "Minha Lista", eles são os destaques principais.
  // Caso a lista esteja vazia, usa os títulos recentes com imagem como fallback.
  const featuredList = favoriteItems.length > 0
    ? favoriteItems
    : [
        ...(latestItems || []),
        ...(moviesData?.Items || []),
        ...seriesItems,
      ].filter((item, index, self) => index === self.findIndex((t) => t.Id === item.Id))

  const isHeroLoading = loadingFavorites && loadingLatest

  return (
    <div className="pb-24">
      {/* Rotativo Dinâmico de Destaques (Hero Carousel alimentado por Minha Lista) */}
      <FeaturedCarousel items={featuredList} isLoading={isHeroLoading} autoRotateInterval={7000} />

      {/* SESSÕES DA HOME ORDENADAS */}
      <div className="-mt-4 sm:-mt-6 md:-mt-8 relative z-10 space-y-2 sm:space-y-4">
        {/* 1ª SESSÃO OBRIGATÓRIA: CONTINUAR ASSISTINDO */}
        {resumeItems && resumeItems.length > 0 && (
          <MediaRow
            title={t.home.continueWatching}
            items={resumeItems}
            isLoading={loadingResume}
          />
        )}

        {/* 2ª SESSÃO: MINHA LISTA (Logo abaixo de Continuar Assistindo) */}
        {favoriteItems && favoriteItems.length > 0 && (
          <MediaRow
            title={t.home.myList}
            items={favoriteItems}
            isLoading={loadingFavorites}
          />
        )}

        {/* 3ª SESSÃO: NOVIDADES / RECÉM ADICIONADOS */}
        <MediaRow
          title={t.home.recentlyAdded}
          items={latestItems || []}
          isLoading={loadingLatest}
        />

        {/* 4ª SESSÃO: FILMES EM DESTAQUE */}
        <MediaRow
          title={t.home.featuredMovies}
          items={moviesData?.Items || []}
          isLoading={loadingMovies}
        />

        {/* 5ª SESSÃO: SÉRIES POPULARES */}
        <MediaRow
          title={t.home.popularSeries}
          items={seriesItems}
          isLoading={loadingSeries}
        />
      </div>

      {/* Media Details Modal */}
      <DetailModal />
    </div>
  )
}
