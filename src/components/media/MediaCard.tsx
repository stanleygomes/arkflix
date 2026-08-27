import React from 'react'
import { motion } from 'framer-motion'
import { Play, Bookmark, Check, X } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Badge, RatingBadge } from '@/components/ui'
import { useTranslation, useToggleFavorite, useFavorites } from '@/hooks'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface MediaCardProps {
  item: MediaItem
  className?: string
  layout?: 'grid' | 'carousel'
}

export const MediaCard: React.FC<MediaCardProps> = ({ item, className, layout = 'grid' }) => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toggleFavorite = useToggleFavorite()
  const { data: favoritesData } = useFavorites(80)

  const imageUrl = getImageUrl(item.Id, 'Primary', { fillWidth: 400, quality: 85 })
  const progressPercent = item.UserData?.PlayedPercentage || 0

  // Calculate isFavorite dynamically against latest favorites cache or item UserData
  const isFavorite = Boolean(
    favoritesData?.Items?.some((fav) => fav.Id === item.Id) ||
    item.UserData?.IsFavorite
  )

  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    toggleFavorite.mutate({ itemId: item.Id, isFavorite })
  }

  return (
    <motion.div
      onClick={() => navigate(`/title/${item.Id}`)}
      whileHover={{ y: -6, scale: 1.03 }}
      whileTap={{ scale: 0.97 }}
      transition={{ type: 'spring', stiffness: 400, damping: 25 }}
      className={cn(
        'group/card relative aspect-[2/3] rounded-squircle-lg overflow-hidden bg-[#1C1C1E] cursor-pointer shadow-sm hover:shadow-apple border border-white/10 hover:border-white/30 will-change-transform',
        layout === 'grid' ? 'w-full' : 'flex-none w-[160px] sm:w-[195px] md:w-[230px]',
        className
      )}
    >
      {/* Crisp Cover Image */}
      <img
        src={imageUrl}
        alt={item.Name}
        loading="lazy"
        className="w-full h-full object-cover transition-transform duration-500 ease-out group-hover/card:scale-105"
      />

      {/* Interactive Favorite Button (Hover transforms Check into explicit Red Close 'X') */}
      <button
        onClick={handleFavoriteClick}
        aria-label={isFavorite ? 'Remover da minha lista' : 'Adicionar à minha lista'}
        title={isFavorite ? 'Remover da minha lista' : 'Adicionar à minha lista'}
        className={cn(
          'absolute top-2.5 right-2.5 z-20 w-8 h-8 rounded-full backdrop-blur-xl flex items-center justify-center transition-all duration-200 shadow-md group/favbtn cursor-pointer',
          isFavorite
            ? 'bg-blue-500 text-white hover:bg-rose-500 hover:scale-110 hover:shadow-rose-500/30'
            : 'bg-black/60 text-white/80 opacity-0 group-hover/card:opacity-100 hover:bg-black/80 hover:text-white border border-white/20 hover:scale-110'
        )}
      >
        {isFavorite ? (
          <>
            {/* Standard: Checkmark | Hover: Explicit 'X' to Remove */}
            <Check className="w-4 h-4 stroke-[3] group-hover/favbtn:hidden" />
            <X className="w-4 h-4 stroke-[3] hidden group-hover/favbtn:block" />
          </>
        ) : (
          <Bookmark className="w-4 h-4" />
        )}
      </button>

      {/* Progress Bar (Continuar Assistindo) */}
      {progressPercent > 0 && (
        <div className="absolute bottom-0 left-0 right-0 h-1 bg-black/80 z-20">
          <div
            className="h-full bg-apple-accent"
            style={{ width: `${Math.min(progressPercent, 100)}%` }}
          />
        </div>
      )}

      {/* Discrete Bottom Info Sheet on Hover */}
      <div className="absolute inset-x-0 bottom-0 pt-16 pb-3.5 px-3.5 bg-gradient-to-t from-black/95 via-black/70 to-transparent opacity-0 group-hover/card:opacity-100 transition-opacity duration-200 flex flex-col justify-end z-10 pointer-events-none group-hover/card:pointer-events-auto">
        <h4 className="text-xs font-bold text-white line-clamp-1 mb-1 drop-shadow">
          {item.Name}
        </h4>

        <div className="flex items-center gap-1.5 text-[10px] text-apple-subtext mb-2">
          {item.ProductionYear && <span>{item.ProductionYear}</span>}
          {item.CommunityRating && <RatingBadge rating={item.CommunityRating} />}
          {item.OfficialRating && <Badge variant="rating">{item.OfficialRating}</Badge>}
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={(e) => {
              e.stopPropagation()
              navigate(`/watch/${item.Id}`)
            }}
            className="flex-1 py-1.5 bg-white text-black text-xs font-semibold rounded-squircle-sm flex items-center justify-center gap-1 hover:bg-white/90 active:scale-95 transition-all shadow-sm cursor-pointer"
          >
            <Play className="w-3 h-3 fill-black text-black" /> {t.common.watch}
          </button>

          {isFavorite && (
            <button
              onClick={handleFavoriteClick}
              title="Remover da lista"
              className="px-2.5 py-1.5 bg-red-500/20 hover:bg-red-500 text-red-400 hover:text-white text-xs font-semibold rounded-squircle-sm flex items-center justify-center transition-all cursor-pointer border border-red-500/30 active:scale-95"
            >
              <X className="w-3.5 h-3.5 stroke-[2.5]" />
            </button>
          )}
        </div>
      </div>
    </motion.div>
  )
}
