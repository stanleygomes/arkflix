import React, { useState, useEffect } from 'react'
import { useModalStore } from '@/stores/modalStore'
import { getImageUrl } from '@/services/api'
import { Button, Modal, Select, Badge, RatingBadge } from '@/components/ui'
import { useNavigate } from 'react-router-dom'
import { useSeasons, useEpisodes, useItemDetails, useTranslation } from '@/hooks'
import { Play, Clock, Calendar, User } from 'lucide-react'

export const DetailModal: React.FC = () => {
  const { isOpen, selectedItem, closeModal } = useModalStore()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [selectedSeasonId, setSelectedSeasonId] = useState<string | null>(null)

  // Fetch full item details (includes People/Actors, Studios, MediaStreams, Taglines, etc.)
  const { data: fullItem } = useItemDetails(isOpen ? selectedItem?.Id : undefined)
  const item = fullItem || selectedItem

  const isSeries = item?.Type === 'Series'

  // Fetch seasons for series
  const { data: seasons } = useSeasons(item?.Id, isOpen && isSeries)
  const { data: episodes } = useEpisodes(
    item?.Id,
    selectedSeasonId || undefined,
    isOpen && isSeries && !!selectedSeasonId
  )

  // Select first season by default
  useEffect(() => {
    if (seasons && seasons.length > 0 && !selectedSeasonId) {
      setSelectedSeasonId(seasons[0].Id)
    }
  }, [seasons, selectedSeasonId])

  if (!isOpen || !item) return null

  const backdropUrl = getImageUrl(item.Id, 'Backdrop', { fillWidth: 1280, quality: 85 })
  const logoUrl = item.ImageTags?.Logo
    ? getImageUrl(item.Id, 'Logo', { fillWidth: 500, quality: 90 })
    : null

  const formatRuntime = (ticks?: number) => {
    if (!ticks) return null
    const totalMinutes = Math.round(ticks / 600000000)
    const hours = Math.floor(totalMinutes / 60)
    const minutes = totalMinutes % 60
    if (hours > 0) return `${hours}h ${minutes}min`
    return `${minutes}min`
  }

  const actors = item.People?.filter((p) => p.Type === 'Actor' || !p.Type).slice(0, 10) || []
  const director = item.People?.find((p) => p.Type === 'Director')?.Name
  const videoStream = item.MediaStreams?.find((s) => s.Type === 'Video')
  const audioStream = item.MediaStreams?.find((s) => s.Type === 'Audio')

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
          alt={item.Name}
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#1C1C1E] via-black/30 to-transparent" />

        {/* Action buttons inside header */}
        <div className="absolute bottom-6 left-6 md:left-10 z-10 space-y-3">
          {logoUrl ? (
            <img
              src={logoUrl}
              alt={item.Name}
              className="max-h-20 object-contain mb-2 drop-shadow-2xl"
            />
          ) : (
            <h2 className="text-2xl md:text-4xl font-extrabold text-white tracking-tight">
              {item.Name}
            </h2>
          )}

          {item.Taglines && item.Taglines.length > 0 && (
            <p className="text-xs text-white/80 italic font-medium drop-shadow">
              &ldquo;{item.Taglines[0]}&rdquo;
            </p>
          )}

          <div className="flex items-center gap-3 pt-1">
            <Button
              variant="primary"
              size="md"
              onClick={() => {
                closeModal()
                navigate(`/watch/${item.Id}`)
              }}
              className="font-semibold shadow-apple"
            >
              <Play className="w-4 h-4 fill-black text-black mr-1" /> {t.common.watch}
            </Button>
          </div>
        </div>
      </div>

      {/* Comprehensive Details Content */}
      <div className="p-6 md:p-10 space-y-8 text-sm text-[#D1D1D6]">
        {/* Top Info Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          <div className="md:col-span-2 space-y-4">
            {/* Metadata Badges & Specs */}
            <div className="flex flex-wrap items-center gap-2.5 text-xs text-apple-subtext font-medium">
              {item.CommunityRating && (
                <RatingBadge rating={item.CommunityRating} />
              )}
              {item.CriticRating && (
                <span className="text-xs font-semibold text-rose-400 bg-rose-500/10 border border-rose-500/20 px-2 py-0.5 rounded-squircle-sm">
                  🍅 {item.CriticRating}%
                </span>
              )}
              {item.ProductionYear && (
                <span className="flex items-center gap-1">
                  <Calendar className="w-3 h-3 text-apple-subtext" />
                  {item.ProductionYear}
                </span>
              )}
              {item.RunTimeTicks && (
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3 text-apple-subtext" />
                  {formatRuntime(item.RunTimeTicks)}
                </span>
              )}
              {item.OfficialRating && (
                <Badge variant="rating">{item.OfficialRating}</Badge>
              )}
              {videoStream && (
                <Badge variant="hd">
                  {videoStream.Height && videoStream.Height >= 2160
                    ? '4K UHD'
                    : videoStream.Height && videoStream.Height >= 1080
                    ? '1080p FHD'
                    : 'HD'}
                </Badge>
              )}
              {videoStream?.Codec && (
                <span className="uppercase text-[10px] bg-white/10 px-1.5 py-0.5 rounded font-mono text-white/90">
                  {videoStream.Codec}
                </span>
              )}
              {audioStream?.Codec && (
                <span className="uppercase text-[10px] bg-white/10 px-1.5 py-0.5 rounded font-mono text-white/90">
                  {audioStream.Codec}
                </span>
              )}
            </div>

            {/* Synopsis */}
            <p className="text-sm md:text-base text-[#F5F5F7] leading-relaxed font-normal">
              {item.Overview || t.common.noOverview}
            </p>
          </div>

          {/* Right Meta Column (Genres, Studios, Director) */}
          <div className="space-y-3 text-xs bg-white/[0.04] p-5 rounded-squircle-lg border border-white/10">
            {item.Genres && item.Genres.length > 0 && (
              <div>
                <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Gêneros</span>
                <span className="text-[#F5F5F7] font-medium">{item.Genres.join(', ')}</span>
              </div>
            )}

            {director && (
              <div>
                <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Direção</span>
                <span className="text-[#F5F5F7] font-medium">{director}</span>
              </div>
            )}

            {item.Studios && item.Studios.length > 0 && (
              <div>
                <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Estúdio</span>
                <span className="text-[#F5F5F7] font-medium">{item.Studios.map((s) => s.Name).join(', ')}</span>
              </div>
            )}

            <div>
              <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Tipo</span>
              <span className="text-[#F5F5F7] font-medium">
                {item.Type === 'Series' ? t.common.series : t.common.movie}
              </span>
            </div>
          </div>
        </div>

        {/* Elenco / Atores (Actors Carousel/Grid) */}
        {actors.length > 0 && (
          <div className="pt-2 border-t border-white/10 space-y-3">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
              <User className="w-4 h-4 text-apple-accent" /> Elenco Principal
            </h3>

            <div className="flex gap-3 overflow-x-auto pb-2 no-scrollbar">
              {actors.map((actor) => (
                <div
                  key={actor.Id || actor.Name}
                  className="flex-none flex items-center gap-2.5 p-2 pr-4 rounded-squircle bg-white/[0.03] border border-white/5"
                >
                  {actor.PrimaryImageTag ? (
                    <img
                      src={getImageUrl(actor.Id, 'Primary', { fillWidth: 80, quality: 75 })}
                      alt={actor.Name}
                      className="w-10 h-10 rounded-full object-cover shadow-sm"
                    />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center text-xs font-bold text-white">
                      {actor.Name.charAt(0)}
                    </div>
                  )}
                  <div>
                    <p className="text-xs font-semibold text-white truncate max-w-[120px]">{actor.Name}</p>
                    {actor.Role && (
                      <p className="text-[10px] text-apple-subtext truncate max-w-[120px]">{actor.Role}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Episodes List (if Series) */}
        {isSeries && (
          <div className="pt-4 border-t border-white/10 space-y-5">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white tracking-tight">{t.common.episodes}</h3>
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
                    {ep.IndexNumber || idx + 1}
                  </span>
                  <img
                    src={getImageUrl(ep.Id, 'Primary', { fillWidth: 200, quality: 75 })}
                    alt={ep.Name}
                    className="w-24 sm:w-28 aspect-video object-cover rounded-squircle-sm bg-black/40 shadow-sm"
                  />
                  <div className="flex-grow">
                    <div className="flex items-center gap-2">
                      <h4 className="text-xs sm:text-sm font-semibold text-white group-hover:text-apple-accent transition-colors">
                        {ep.Name}
                      </h4>
                      {ep.RunTimeTicks && (
                        <span className="text-[10px] text-apple-subtext">
                          {formatRuntime(ep.RunTimeTicks)}
                        </span>
                      )}
                    </div>
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
