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
  const { data: episodes } = useEpisodes(
    selectedItem?.Id,
    selectedSeasonId || undefined,
    isOpen && isSeries && !!selectedSeasonId
  )

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
      <div className="relative aspect-video w-full overflow-hidden">
        <img
          src={backdropUrl}
          alt={selectedItem.Name}
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#1C1C1E] via-black/20 to-transparent" />

        {/* Action buttons inside header */}
        <div className="absolute bottom-6 left-6 md:left-10 z-10 space-y-3">
          {logoUrl ? (
            <img
              src={logoUrl}
              alt={selectedItem.Name}
              className="max-h-20 object-contain mb-2 drop-shadow-2xl"
            />
          ) : (
            <h2 className="text-2xl md:text-4xl font-extrabold text-white tracking-tight">
              {selectedItem.Name}
            </h2>
          )}

          <div className="flex items-center gap-3">
            <Button
              variant="primary"
              size="md"
              onClick={() => {
                closeModal()
                navigate(`/watch/${selectedItem.Id}`)
              }}
              className="font-semibold shadow-apple"
            >
              <Play className="w-4 h-4 fill-black text-black mr-1" /> Assistir
            </Button>
          </div>
        </div>
      </div>

      {/* Details Content */}
      <div className="p-6 md:p-10 space-y-6 text-sm text-[#D1D1D6]">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          <div className="md:col-span-2 space-y-4">
            <div className="flex items-center gap-2.5 text-xs text-apple-subtext font-medium">
              {selectedItem.CommunityRating && (
                <RatingBadge rating={selectedItem.CommunityRating} />
              )}
              {selectedItem.ProductionYear && <span>{selectedItem.ProductionYear}</span>}
              {selectedItem.OfficialRating && (
                <Badge variant="rating">{selectedItem.OfficialRating}</Badge>
              )}
              <Badge variant="hd">4K HDR</Badge>
            </div>

            <p className="text-sm md:text-base text-[#F5F5F7] leading-relaxed font-normal">
              {selectedItem.Overview || 'Sem sinopse disponível para este título.'}
            </p>
          </div>

          <div className="space-y-2 text-xs bg-white/[0.04] p-4 rounded-squircle border border-white/10">
            {selectedItem.Genres && selectedItem.Genres.length > 0 && (
              <div>
                <span className="text-apple-subtext">Gêneros: </span>
                <span className="text-[#F5F5F7] font-medium">{selectedItem.Genres.join(', ')}</span>
              </div>
            )}
            <div>
              <span className="text-apple-subtext">Tipo: </span>
              <span className="text-[#F5F5F7] font-medium">
                {selectedItem.Type === 'Series' ? 'Série' : 'Filme'}
              </span>
            </div>
          </div>
        </div>

        {/* Episodes List (if Series) */}
        {isSeries && (
          <div className="pt-6 border-t border-white/10 space-y-5">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white tracking-tight">Episódios</h3>
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

            <div className="space-y-2.5">
              {episodes?.map((ep, idx) => (
                <div
                  key={ep.Id}
                  onClick={() => {
                    closeModal()
                    navigate(`/watch/${ep.Id}`)
                  }}
                  className="group flex items-center gap-4 p-3 rounded-squircle bg-white/[0.03] hover:bg-white/[0.08] transition-all cursor-pointer border border-white/5 hover:border-white/15"
                >
                  <span className="text-apple-subtext font-semibold text-sm w-6 text-center">
                    {idx + 1}
                  </span>
                  <img
                    src={getImageUrl(ep.Id, 'Primary', { fillWidth: 200, quality: 75 })}
                    alt={ep.Name}
                    className="w-24 sm:w-28 aspect-video object-cover rounded-squircle-sm bg-black/40 shadow-sm"
                  />
                  <div className="flex-grow">
                    <h4 className="text-xs sm:text-sm font-semibold text-white group-hover:text-apple-accent transition-colors">
                      {ep.Name}
                    </h4>
                    <p className="text-xs text-apple-subtext line-clamp-2 mt-0.5 font-normal">
                      {ep.Overview}
                    </p>
                  </div>
                  <div className="w-8 h-8 rounded-full bg-white/10 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                    <Play className="w-3.5 h-3.5 fill-white text-white ml-0.5" />
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}
