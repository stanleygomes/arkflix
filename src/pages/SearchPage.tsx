import React from 'react'
import { useSearchParams } from 'react-router-dom'
import { useSearchMedia } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { MediaCardSkeleton } from '@/components/ui'
import { Search } from 'lucide-react'

export const SearchPage: React.FC = () => {
  const [searchParams] = useSearchParams()
  const query = searchParams.get('q') || ''

  const { data: resultsData, isLoading } = useSearchMedia(query)
  const items = resultsData?.Items || []

  return (
    <div className="min-h-screen pt-24 pb-20 px-6 md:px-14 max-w-7xl mx-auto space-y-8 animate-fadeIn">
      {/* Page Header */}
      <div className="flex items-center gap-3 border-b border-white/[0.08] pb-6">
        <div className="w-10 h-10 rounded-squircle bg-white/10 flex items-center justify-center text-white border border-white/15">
          <Search className="w-5 h-5 text-apple-accent" />
        </div>
        <div>
          <h1 className="text-2xl md:text-3xl font-extrabold text-white tracking-tight">
            Resultados para &ldquo;<span className="text-apple-accent">{query}</span>&rdquo;
          </h1>
          <p className="text-xs text-apple-subtext mt-0.5">
            {isLoading ? 'Pesquisando...' : `${items.length} títulos encontrados na sua biblioteca`}
          </p>
        </div>
      </div>

      {/* Results Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
          {Array.from({ length: 12 }).map((_, i) => (
            <MediaCardSkeleton key={i} />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="py-24 text-center space-y-3">
          <p className="text-base font-semibold text-white">Nenhum resultado para &ldquo;{query}&rdquo;</p>
          <p className="text-xs text-apple-subtext max-w-md mx-auto">
            Tente pesquisar com palavras-chave diferentes, nomes de atores, diretores ou títulos em inglês.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
          {items.map((item) => (
            <MediaCard key={item.Id} item={item} />
          ))}
        </div>
      )}

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
