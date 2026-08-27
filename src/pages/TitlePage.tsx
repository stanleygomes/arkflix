import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getImageUrl } from '@/services/api'
import { Button, Select, Badge, RatingBadge, AppleSpinner } from '@/components/ui'
import { useSeasons, useEpisodes, useItemDetails, useTranslation, useToggleFavorite } from '@/hooks'
import { Play, Clock, Calendar, User, ArrowLeft, Plus, Check, Trash2, Bookmark } from 'lucide-react'

export const TitlePage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toggleFavorite = useToggleFavorite()
  const [selectedSeasonId, setSelectedSeasonId] = useState<string | null>(null)

  const { data: item, isLoading } = useItemDetails(id)
  const isSeries = item?.Type === 'Series'
  const isFavorite = !!item?.UserData?.IsFavorite

  // Fetch seasons for series
  const { data: seasons } = useSeasons(item?.Id, isSeries && !!item?.Id)
  const { data: episodes, isLoading: loadingEpisodes } = useEpisodes(
    item?.Id,
    selectedSeasonId || undefined,
    isSeries && !!selectedSeasonId
  )

  // Select first season by default
  useEffect(() => {
    if (seasons && seasons.length > 0 && !selectedSeasonId) {
      setSelectedSeasonId(seasons[0].Id)
    }
  }, [seasons, selectedSeasonId])

  // Scroll to top on enter
  useEffect(() => {
    window.scrollTo(0, 0)
  }, [id])

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#000000] flex items-center justify-center">
        <AppleSpinner size="lg" color="white" />
      </div>
    )
  }

  if (!item) {
    return (
      <div className="min-h-screen bg-[#000000] flex flex-col items-center justify-center text-white space-y-4">
        <p className="text-sm text-apple-subtext">{t.common.unavailable}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-apple-accent hover:underline text-xs"
        >
          {t.common.backToHome}
        </button>
      </div>
    )
  }

  const backdropUrl = getImageUrl(item.Id, 'Backdrop', { fillWidth: 1920, quality: 90 })
  const logoUrl = item.ImageTags?.Logo
    ? getImageUrl(item.Id, 'Logo', { fillWidth: 600, quality: 90 })
    : null

  const formatRuntime = (ticks?: number) => {
    if (!ticks) return null
    const totalMinutes = Math.round(ticks / 600000000)
    const hours = Math.floor(totalMinutes / 60)
    const minutes = totalMinutes % 60
    if (hours > 0) return `${hours}h ${minutes}min`
    return `${minutes}min`
  }

  const actors = item.People?.filter((p) => p.Type === 'Actor' || !p.Type).slice(0, 12) || []
  const director = item.People?.find((p) => p.Type === 'Director')?.Name
  const videoStream = item.MediaStreams?.find((s) => s.Type === 'Video')
  const audioStream = item.MediaStreams?.find((s) => s.Type === 'Audio')

  const seasonOptions =
    seasons?.map((s) => ({
      value: s.Id,
      label: s.Name,
    })) || []

  return (
    <div className="min-h-screen bg-[#000000] text-white pb-24 selection:bg-white/20 animate-fadeIn">
      {/* Hero Backdrop Banner */}
      <div className="relative h-[65vh] sm:h-[75vh] md:h-[82vh] w-full overflow-hidden">
        <img
          src={backdropUrl}
          alt={item.Name}
          className="w-full h-full object-cover object-center scale-[1.02]"
        />

        {/* Apple Cinematic Gradients */}
        <div className="absolute inset-0 bg-gradient-to-r from-black/95 via-black/50 to-transparent w-full md:w-3/5" />
        <div className="absolute inset-0 bg-gradient-to-t from-[#000000] via-[#000000]/40 to-transparent" />

        {/* Back Button */}
        <div className="absolute top-20 sm:top-24 left-4 sm:left-6 md:left-14 z-20">
          <button
            onClick={() => navigate(-1)}
            className="flex items-center gap-2 px-3.5 py-2 rounded-full bg-black/60 hover:bg-black/85 backdrop-blur-xl border border-white/20 text-white text-xs font-semibold transition-all hover:scale-105 active:scale-95 shadow-apple cursor-pointer"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Voltar</span>
          </button>
        </div>

        {/* Hero Title & Actions */}
        <div className="absolute bottom-10 sm:bottom-14 md:bottom-20 left-4 sm:left-6 md:left-14 right-4 sm:right-auto max-w-2xl z-10 space-y-3 sm:space-y-4">
          {logoUrl ? (
            <img
              src={logoUrl}
              alt={item.Name}
              className="max-h-16 sm:max-h-24 md:max-h-32 object-contain mb-2 drop-shadow-2xl"
            />
          ) : (
            <h1 className="text-2xl sm:text-4xl md:text-5xl font-extrabold text-white tracking-tight leading-tight">
              {item.Name}
            </h1>
          )}

          {item.Taglines && item.Taglines.length > 0 && (
            <p className="text-xs sm:text-sm text-white/80 italic font-medium drop-shadow">
              &ldquo;{item.Taglines[0]}&rdquo;
            </p>
          )}

          {/* Action Buttons */}
          <div className="flex flex-wrap items-center gap-3 pt-2">
            <Button
              variant="primary"
              size="lg"
              onClick={() => navigate(`/watch/${item.Id}`)}
              className="font-semibold shadow-apple text-xs sm:text-base py-2.5 sm:py-3.5 px-6 sm:px-8"
            >
              <Play className="w-4 h-4 fill-black text-black mr-1.5" /> {t.common.watch}
            </Button>

            {isFavorite ? (
              <Button
                variant="glass"
                size="lg"
                onClick={() => toggleFavorite.mutate({ itemId: item.Id, isFavorite: true })}
                className="font-medium text-white hover:text-red-400 hover:border-red-500/40 text-xs sm:text-base py-2.5 sm:py-3.5 px-5 sm:px-6 shadow-apple group"
              >
                <Check className="w-4 h-4 text-emerald-400 mr-2 stroke-[3] group-hover:hidden" />
                <Trash2 className="w-4 h-4 text-red-400 mr-2 hidden group-hover:block" />
                <span className="group-hover:hidden">Na Minha Lista</span>
                <span className="hidden group-hover:inline text-red-400">Remover da Lista</span>
              </Button>
            ) : (
              <Button
                variant="glass"
                size="lg"
                onClick={() => toggleFavorite.mutate({ itemId: item.Id, isFavorite: false })}
                className="font-medium text-white text-xs sm:text-base py-2.5 sm:py-3.5 px-5 sm:px-6 shadow-apple"
              >
                <Plus className="w-4 h-4 mr-2" /> Minha Lista
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Main Details Section */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 md:px-14 -mt-6 relative z-20 space-y-10">
        {/* Info Grid */}
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
                  <Calendar className="w-3.5 h-3.5 text-apple-subtext" />
                  {item.ProductionYear}
                </span>
              )}
              {item.RunTimeTicks && (
                <span className="flex items-center gap-1">
                  <Clock className="w-3.5 h-3.5 text-apple-subtext" />
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
              {audioStream?.Codec && (
                <span className="uppercase text-[10px] bg-white/10 px-2 py-0.5 rounded font-mono text-white/90">
                  {audioStream.Codec}
                </span>
              )}
            </div>

            {/* Synopsis */}
            <p className="text-sm sm:text-base text-[#F5F5F7] leading-relaxed font-normal">
              {item.Overview || t.common.noOverview}
            </p>
          </div>

          {/* Right Meta Column (Genres, Studios, Director) */}
          <div className="space-y-3.5 text-xs bg-white/[0.04] p-5 rounded-squircle-xl border border-white/10 shadow-sm">
            {item.Genres && item.Genres.length > 0 && (
              <div>
                <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Gêneros</span>
                <span className="text-white font-medium">{item.Genres.join(', ')}</span>
              </div>
            )}

            {director && (
              <div>
                <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Direção</span>
                <span className="text-white font-medium">{director}</span>
              </div>
            )}

            {item.Studios && item.Studios.length > 0 && (
              <div>
                <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Estúdio</span>
                <span className="text-white font-medium">{item.Studios.map((s) => s.Name).join(', ')}</span>
              </div>
            )}

            <div>
              <span className="text-apple-subtext block text-[11px] uppercase tracking-wider mb-0.5">Tipo</span>
              <span className="text-white font-medium">
                {item.Type === 'Series' ? t.common.series : t.common.movie}
              </span>
            </div>
          </div>
        </div>

        {/* Elenco / Atores (Actors Cinematic Cards) */}
        {actors.length > 0 && (
          <div className="pt-6 border-t border-white/10 space-y-4">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
              <User className="w-4 h-4 text-apple-accent" /> Elenco Principal
            </h3>

            <div className="flex gap-3 sm:gap-4 overflow-x-auto pb-3 no-scrollbar -mx-4 px-4 sm:mx-0 sm:px-0" style={{ WebkitOverflowScrolling: 'touch' }}>
              {actors.map((actor) => (
                <div
                  key={actor.Id || actor.Name}
                  className="flex-none flex flex-col items-center text-center p-3 rounded-squircle-lg bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 transition-all w-28 sm:w-32 group/actor shadow-sm"
                >
                  {/* High-Res Large Actor Portrait */}
                  <div className="w-20 h-20 sm:w-24 sm:h-24 rounded-full overflow-hidden mb-2.5 shadow-apple border-2 border-white/20 group-hover/actor:border-apple-accent transition-all">
                    {actor.PrimaryImageTag ? (
                      <img
                        src={getImageUrl(actor.Id, 'Primary', { fillWidth: 200, quality: 90 })}
                        alt={actor.Name}
                        className="w-full h-full object-cover group-hover/actor:scale-105 transition-transform duration-300"
                      />
                    ) : (
                      <div className="w-full h-full bg-gradient-to-tr from-blue-600/30 to-indigo-600/30 flex items-center justify-center text-lg sm:text-xl font-bold text-white">
                        {actor.Name.charAt(0)}
                      </div>
                    )}
                  </div>

                  <p className="text-xs font-bold text-white line-clamp-1 w-full leading-tight">{actor.Name}</p>
                  {actor.Role && (
                    <p className="text-[11px] text-apple-subtext line-clamp-1 w-full mt-0.5">{actor.Role}</p>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Episodes List (if Series) */}
        {isSeries && (
          <div className="pt-6 border-t border-white/10 space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <h3 className="text-lg sm:text-xl font-bold text-white tracking-tight">{t.common.episodes}</h3>
              {seasonOptions.length > 1 && (
                <div className="w-full sm:w-56">
                  <Select
                    value={selectedSeasonId || ''}
                    onChange={(e) => setSelectedSeasonId(e.target.value)}
                    options={seasonOptions}
                  />
                </div>
              )}
            </div>

            {loadingEpisodes ? (
              <div className="space-y-3">
                {Array.from({ length: 6 }).map((_, i) => (
                  <div key={i} className="h-20 rounded-squircle bg-white/[0.03] animate-pulse" />
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                {episodes?.map((ep, idx) => {
                  const isEpFavorite = !!ep.UserData?.IsFavorite

                  return (
                    <div
                      key={ep.Id}
                      onClick={() => navigate(`/watch/${ep.Id}`)}
                      className="group flex items-center gap-3 sm:gap-5 p-3.5 rounded-squircle bg-white/[0.03] hover:bg-white/[0.08] active:scale-[0.99] transition-all cursor-pointer border border-white/5 hover:border-white/15"
                    >
                      <span className="text-apple-subtext font-semibold text-sm sm:text-base w-6 sm:w-8 text-center flex-none">
                        {ep.IndexNumber || idx + 1}
                      </span>
                      <img
                        src={getImageUrl(ep.Id, 'Primary', { fillWidth: 240, quality: 80 })}
                        alt={ep.Name}
                        className="w-24 sm:w-36 aspect-video object-cover rounded-squircle-sm bg-black/40 shadow-sm flex-none"
                      />
                      <div className="flex-grow min-w-0">
                        <div className="flex items-center gap-2.5">
                          <h4 className="text-xs sm:text-sm font-semibold text-white group-hover:text-apple-accent transition-colors truncate">
                            {ep.Name}
                          </h4>
                          {ep.RunTimeTicks && (
                            <span className="text-[10px] text-apple-subtext flex-none">
                              {formatRuntime(ep.RunTimeTicks)}
                            </span>
                          )}
                        </div>
                        <p className="text-xs text-apple-subtext line-clamp-2 mt-1 font-normal">
                          {ep.Overview}
                        </p>
                      </div>

                      {/* Actions: Add to My List + Play */}
                      <div className="flex items-center gap-2 flex-none">
                        {/* Bookmark Button */}
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            toggleFavorite.mutate({ itemId: ep.Id, isFavorite: isEpFavorite })
                          }}
                          title={isEpFavorite ? 'Remover da minha lista' : 'Adicionar à minha lista'}
                          className={`w-9 h-9 rounded-full flex items-center justify-center transition-all cursor-pointer border ${
                            isEpFavorite
                              ? 'bg-blue-500/20 text-apple-accent border-blue-500/40 hover:bg-red-500 hover:text-white hover:border-red-500'
                              : 'bg-white/10 text-white/70 border-white/10 hover:bg-white/20 hover:text-white opacity-80 sm:opacity-0 group-hover:opacity-100'
                          }`}
                        >
                          {isEpFavorite ? (
                            <Check className="w-4 h-4 stroke-[3]" />
                          ) : (
                            <Bookmark className="w-4 h-4" />
                          )}
                        </button>

                        {/* Play Button */}
                        <div className="w-9 h-9 rounded-full bg-white/10 group-hover:bg-white flex items-center justify-center transition-all opacity-80 sm:opacity-0 group-hover:opacity-100 shadow-sm">
                          <Play className="w-4 h-4 fill-white group-hover:fill-black text-white group-hover:text-black ml-0.5 transition-colors" />
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
