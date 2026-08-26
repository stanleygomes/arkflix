import React, { useState, useEffect } from 'react'
import { useModalStore } from '@/stores/modalStore'
import { getImageUrl } from '@/services/api'
import { Button, Modal, Select, Badge, RatingBadge } from '@/components/ui'
import { useNavigate } from 'react-router-dom'
import { useSeasons, useEpisodes } from '@/hooks'
import { Play } from 'lucide-react'

export const DetailModal: React.FC = () => {
  const { isOpen, selectedItem, closeModal } = useModalStore()
  const navigate = useNavigate()
  const [selectedSeasonId, setSelectedSeasonId] = useState<string | null>(null)

  const isSeries = selectedItem?.Type === 'Series'

  // Chamadas desacopladas via Hooks
  const { data: seasons } = useSeasons(selectedItem?.Id, isOpen && isSeries)
  const { data: episodes } = useEpisodes(selectedItem?.Id, selectedSeasonId || undefined, isOpen && isSeries && !!selectedSeasonId)

  // Select first season by default
  useEffect(() => {
    if (seasons && seasons.length > 0 && !selectedSeasonId) {
      setSelectedSeasonId(seasons[0].Id)
    }
  }, [seasons, selectedSeasonId])

  if (!isOpen || !selectedItem) return null

  const backdropUrl = getImageUrl(selectedItem.Id, 'Backdrop', { fillWidth: 1280, quality: 85 })
  const logoUrl = selectedItem.ImageTags?.Logo
    ? getImageUrl(selectedItem.Id, 'Logo', { fillWidth: 500, quality: 90 })
    : null

  const seasonOptions =
    seasons?.map((s) => ({
      value: s.Id,
      label: s.Name,
    })) || []

  return (
    <Modal isOpen={isOpen} onClose={closeModal} maxWidth="4xl">
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
                <RatingBadge rating={selectedItem.CommunityRating} />
              )}
              {selectedItem.ProductionYear && <span>{selectedItem.ProductionYear}</span>}
              {selectedItem.OfficialRating && (
                <Badge variant="rating">{selectedItem.OfficialRating}</Badge>
              )}
              <Badge variant="hd">Ultra HD 4K</Badge>
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
        {isSeries && (
          <div className="pt-6 border-t border-white/10 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white">Episódios</h3>
              {seasonOptions.length > 1 && (
                <div className="w-48">
                  <Select
                    value={selectedSeasonId || ''}
                    onChange={(e) => setSelectedSeasonId(e.target.value)}
                    options={seasonOptions}
                  />
                </div>
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
    </Modal>
  )
}
