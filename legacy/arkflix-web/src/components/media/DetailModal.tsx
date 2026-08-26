import React from 'react'
import { X, Play, Star } from 'lucide-react'
import { useModalStore } from '@/stores/modalStore'
import { getImageUrl } from '@/services/api'
import { Button } from '@/components/ui/Button'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'

export const DetailModal: React.FC = () => {
  const { isOpen, selectedItem, closeModal } = useModalStore()
  const { user } = useAuthStore()
  const navigate = useNavigate()
  const [selectedSeasonId, setSelectedSeasonId] = React.useState<string | null>(null)

  // Fetch seasons if it's a Series
  const { data: seasons } = useQuery({
    queryKey: ['seasons', selectedItem?.Id],
    queryFn: () => jellyfinService.getSeasons(user!.Id, selectedItem!.Id),
    enabled: isOpen && !!user && selectedItem?.Type === 'Series',
  })

  // Select first season by default
  React.useEffect(() => {
    if (seasons && seasons.length > 0 && !selectedSeasonId) {
      setSelectedSeasonId(seasons[0].Id)
    }
  }, [seasons, selectedSeasonId])

  // Fetch episodes if a season is selected
  const { data: episodes } = useQuery({
    queryKey: ['episodes', selectedItem?.Id, selectedSeasonId],
    queryFn: () => jellyfinService.getEpisodes(user!.Id, selectedItem!.Id, selectedSeasonId!),
    enabled: isOpen && !!user && !!selectedSeasonId,
  })

  if (!isOpen || !selectedItem) return null

  const backdropUrl = getImageUrl(selectedItem.Id, 'Backdrop', { fillWidth: 1280, quality: 85 })
  const logoUrl = selectedItem.ImageTags?.Logo
    ? getImageUrl(selectedItem.Id, 'Logo', { fillWidth: 500, quality: 90 })
    : null

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/80 flex justify-center items-start pt-12 pb-16 px-4 backdrop-blur-sm">
      <div
        className="relative w-full max-w-4xl bg-netflix-dark rounded-lg overflow-hidden shadow-2xl border border-white/10"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close Button */}
        <button
          onClick={closeModal}
          className="absolute top-4 right-4 z-20 w-9 h-9 rounded-full bg-netflix-black/80 text-white flex items-center justify-center hover:bg-white hover:text-black transition-colors"
          aria-label="Fechar"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Hero Header with Backdrop */}
        <div className="relative aspect-video w-full">
          <img
            src={backdropUrl}
            alt={selectedItem.Name}
            className="w-full h-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-netflix-dark via-transparent to-black/20" />

          {/* Action buttons inside header */}
          <div className="absolute bottom-6 left-6 md:left-10 z-10 space-y-3">
            {logoUrl ? (
              <img src={logoUrl} alt={selectedItem.Name} className="max-h-20 object-contain mb-2 drop-shadow" />
            ) : (
              <h2 className="text-2xl md:text-3xl font-black text-white">{selectedItem.Name}</h2>
            )}

            <div className="flex items-center gap-3">
              <Button
                variant="primary"
                onClick={() => {
                  closeModal()
                  navigate(`/watch/${selectedItem.Id}`)
                }}
              >
                <Play className="w-5 h-5 fill-black" /> Assistir
              </Button>
            </div>
          </div>
        </div>

        {/* Details Content */}
        <div className="p-6 md:p-10 space-y-6 text-sm text-netflix-lightGray">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="md:col-span-2 space-y-4">
              <div className="flex items-center gap-3 text-xs text-gray-400 font-medium">
                {selectedItem.CommunityRating && (
                  <span className="flex items-center gap-1 text-green-400 font-bold">
                    <Star className="w-3.5 h-3.5 fill-green-400" /> {selectedItem.CommunityRating.toFixed(1)}
                  </span>
                )}
                {selectedItem.ProductionYear && <span>{selectedItem.ProductionYear}</span>}
                {selectedItem.OfficialRating && (
                  <span className="border border-gray-600 px-1 py-0.5 rounded text-[10px] uppercase">
                    {selectedItem.OfficialRating}
                  </span>
                )}
                <span className="uppercase text-[10px] bg-white/10 px-1.5 py-0.5 rounded">HD / 4K</span>
              </div>

              <p className="text-base text-white/90 leading-relaxed">
                {selectedItem.Overview || 'Sem sinopse disponível para este título.'}
              </p>
            </div>

            <div className="space-y-2 text-xs">
              {selectedItem.Genres && selectedItem.Genres.length > 0 && (
                <div>
                  <span className="text-gray-500">Gêneros: </span>
                  <span className="text-white">{selectedItem.Genres.join(', ')}</span>
                </div>
              )}
              <div>
                <span className="text-gray-500">Tipo: </span>
                <span className="text-white">{selectedItem.Type === 'Series' ? 'Série' : 'Filme'}</span>
              </div>
            </div>
          </div>

          {/* Episodes List (if Series) */}
          {selectedItem.Type === 'Series' && (
            <div className="pt-6 border-t border-white/10 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-white">Episódios</h3>
                {seasons && seasons.length > 1 && (
                  <select
                    value={selectedSeasonId || ''}
                    onChange={(e) => setSelectedSeasonId(e.target.value)}
                    className="bg-netflix-black border border-white/20 rounded px-3 py-1 text-white text-xs focus:outline-none"
                  >
                    {seasons.map((season) => (
                      <option key={season.Id} value={season.Id}>
                        {season.Name}
                      </option>
                    ))}
                  </select>
                )}
              </div>

              <div className="space-y-3">
                {episodes?.map((ep, idx) => (
                  <div
                    key={ep.Id}
                    onClick={() => {
                      closeModal()
                      navigate(`/watch/${ep.Id}`)
                    }}
                    className="flex items-center gap-4 p-3 rounded-md hover:bg-white/10 transition-colors cursor-pointer border-b border-white/5"
                  >
                    <span className="text-gray-400 font-bold text-base w-6 text-center">{idx + 1}</span>
                    <img
                      src={getImageUrl(ep.Id, 'Primary', { fillWidth: 200, quality: 70 })}
                      alt={ep.Name}
                      className="w-24 aspect-video object-cover rounded bg-black/40"
                    />
                    <div className="flex-grow">
                      <h4 className="text-sm font-semibold text-white">{ep.Name}</h4>
                      <p className="text-xs text-gray-400 line-clamp-2 mt-0.5">{ep.Overview}</p>
                    </div>
                    <Play className="w-4 h-4 text-gray-400 hover:text-white" />
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
