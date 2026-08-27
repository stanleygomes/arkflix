import React, { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useSearchMedia } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { MediaCardSkeleton, Input, EmptyState } from '@/components/ui'
import { Search, SearchX } from 'lucide-react'

export const SearchPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams()
  const initialQuery = searchParams.get('q') || ''
  const [searchInput, setSearchInput] = useState(initialQuery)

  const { data: resultsData, isLoading } = useSearchMedia(initialQuery)
  const items = resultsData?.Items || []

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchInput.trim()) {
      setSearchParams({ q: searchInput.trim() })
    }
  }

  return (
    <div className="min-h-screen pt-[max(env(safe-area-inset-top,0px)+4.5rem,5.5rem)] sm:pt-28 pb-20 px-4 sm:px-6 md:px-14 max-w-7xl mx-auto space-y-6 sm:space-y-8 animate-fadeIn">
      {/* Mobile Inline Search Bar */}
      <form onSubmit={handleSearchSubmit} className="md:hidden">
        <Input
          type="text"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder="Buscar filmes, séries e episódios..."
          icon={<Search className="w-4 h-4" />}
          className="bg-white/10 text-white placeholder-apple-subtext"
        />
      </form>

      {/* Page Header */}
      <div className="flex items-center gap-3 border-b border-white/[0.08] pb-4 sm:pb-6">
        <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-squircle bg-white/10 flex items-center justify-center text-white border border-white/15 flex-none">
          <Search className="w-4 h-4 sm:w-5 sm:h-5 text-apple-accent" />
        </div>
        <div className="min-w-0">
          <h1 className="text-xl sm:text-2xl md:text-3xl font-extrabold text-white tracking-tight truncate">
            {initialQuery ? (
              <>
                Resultados para &ldquo;<span className="text-apple-accent">{initialQuery}</span>&rdquo;
              </>
            ) : (
              'Busca Global'
            )}
          </h1>
          <p className="text-[11px] sm:text-xs text-apple-subtext mt-0.5">
            {isLoading
              ? 'Pesquisando...'
              : initialQuery
              ? `${items.length} títulos encontrados na sua biblioteca`
              : 'Digite um termo para pesquisar'}
          </p>
        </div>
      </div>

      {/* Results Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
          {Array.from({ length: 12 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      ) : !initialQuery ? (
        <EmptyState
          icon={Search}
          title="Pesquise no Arkflix"
          description="Encontre filmes, séries, episódios e gêneros instantaneamente em toda a sua biblioteca Jellyfin."
        />
      ) : items.length === 0 ? (
        <EmptyState
          icon={SearchX}
          title={`Nenhum resultado para "${initialQuery}"`}
          description="Tente pesquisar com outras palavras-chave, títulos originais ou nomes de elenco."
        />
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
