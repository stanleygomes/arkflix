import React, { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useSearchMedia } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { MediaCardSkeleton, Input } from '@/components/ui'
import { Search } from 'lucide-react'

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
        <div className="py-20 text-center space-y-2">
          <Search className="w-12 h-12 text-apple-subtext mx-auto opacity-40" />
          <p className="text-sm font-semibold text-white">Pesquise em toda a sua coleção Jellyfin</p>
          <p className="text-xs text-apple-subtext">Encontre filmes, séries, episódios e gêneros instantaneamente.</p>
        </div>
      ) : items.length === 0 ? (
        <div className="py-20 text-center space-y-3">
          <p className="text-sm sm:text-base font-semibold text-white">Nenhum resultado para &ldquo;{initialQuery}&rdquo;</p>
          <p className="text-xs text-apple-subtext max-w-md mx-auto">
            Tente pesquisar com palavras-chave diferentes, nomes de atores ou diretores.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
          {items.map((item) => (
            <MediaCard key={item.Id} item={item} layout="grid" />
          ))}
        </div>
      )}

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
