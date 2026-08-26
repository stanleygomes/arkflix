import React from 'react'
import { motion } from 'framer-motion'
import { Play } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Badge, RatingBadge } from '@/components/ui'
import { useModalStore } from '@/stores/modalStore'
import { useTranslation } from '@/hooks'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface MediaCardProps {
  item: MediaItem
  className?: string
  layout?: 'grid' | 'carousel'
}

export const MediaCard: React.FC<MediaCardProps> = ({ item, className, layout = 'grid' }) => {
  const { openModal } = useModalStore()
  const { t } = useTranslation()
  const navigate = useNavigate()

  const imageUrl = getImageUrl(item.Id, 'Primary', { fillWidth: 400, quality: 85 })
  const progressPercent = item.UserData?.PlayedPercentage || 0

  return (
    <motion.div
      onClick={() => openModal(item)}
      whileHover={{ y: -6, scale: 1.03 }}
      whileTap={{ scale: 0.97 }}
      transition={{ type: 'spring', stiffness: 400, damping: 25 }}
      className={cn(
        'group relative aspect-[2/3] rounded-squircle-lg overflow-hidden bg-[#1C1C1E] cursor-pointer shadow-sm hover:shadow-apple border border-white/10 hover:border-white/30 will-change-transform',
        layout === 'grid' ? 'w-full' : 'flex-none w-[160px] sm:w-[195px] md:w-[230px]',
        className
      )}
    >
      {/* Crisp Cover Image (Always 100% sharp and clear) */}
      <img
        src={imageUrl}
        alt={item.Name}
        loading="lazy"
        className="w-full h-full object-cover transition-transform duration-500 ease-out group-hover:scale-105"
      />

      {/* Progress Bar (Continuar Assistindo) */}
      {progressPercent > 0 && (
        <div className="absolute bottom-0 left-0 right-0 h-1 bg-black/80 z-20">
          <div
            className="h-full bg-apple-accent"
            style={{ width: `${Math.min(progressPercent, 100)}%` }}
          />
        </div>
      )}

      {/* Discrete Bottom Info Sheet on Hover (No full blur, crystal clear image) */}
      <div className="absolute inset-x-0 bottom-0 pt-16 pb-3.5 px-3.5 bg-gradient-to-t from-black/95 via-black/70 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex flex-col justify-end z-10">
        <h4 className="text-xs font-bold text-white line-clamp-1 mb-1 drop-shadow">
          {item.Name}
        </h4>

        <div className="flex items-center gap-1.5 text-[10px] text-apple-subtext mb-2">
          {item.ProductionYear && <span>{item.ProductionYear}</span>}
          {item.CommunityRating && <RatingBadge rating={item.CommunityRating} />}
          {item.OfficialRating && <Badge variant="rating">{item.OfficialRating}</Badge>}
        </div>

        <button
          onClick={(e) => {
            e.stopPropagation()
            navigate(`/watch/${item.Id}`)
          }}
          className="w-full py-1.5 bg-white text-black text-xs font-semibold rounded-squircle-sm flex items-center justify-center gap-1 hover:bg-white/90 active:scale-95 transition-all shadow-sm"
        >
          <Play className="w-3 h-3 fill-black text-black" /> {t.common.watch}
        </button>
      </div>
    </motion.div>
  )
}
