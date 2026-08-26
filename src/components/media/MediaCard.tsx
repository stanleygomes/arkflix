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
      whileHover={{ y: -6, scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      transition={{ type: 'spring', stiffness: 400, damping: 28 }}
      className={cn(
        'group relative aspect-[2/3] rounded-squircle-lg overflow-hidden bg-[#1C1C1E] cursor-pointer shadow-sm hover:shadow-apple border border-black/10 dark:border-white/10 hover:border-black/30 dark:hover:border-white/30 will-change-transform',
        layout === 'grid' ? 'w-full' : 'flex-none w-[160px] sm:w-[195px] md:w-[230px]',
        className
      )}
    >
      <img
        src={imageUrl}
        alt={item.Name}
        loading="lazy"
        className="w-full h-full object-cover transition-transform duration-700 ease-out group-hover:scale-105"
      />

      {/* Progress Bar (Continuar Assistindo) */}
      {progressPercent > 0 && (
        <div className="absolute bottom-0 left-0 right-0 h-1 bg-black/60 z-10">
          <div
            className="h-full bg-apple-accent"
            style={{ width: `${Math.min(progressPercent, 100)}%` }}
          />
        </div>
      )}

      {/* Subtle Apple Glass Gradient on Hover */}
      <div className="absolute inset-0 bg-gradient-to-t from-black/95 via-black/30 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex flex-col justify-end p-3.5 backdrop-blur-[2px]">
        <h4 className="text-xs font-semibold text-white line-clamp-1 mb-1">{item.Name}</h4>

        <div className="flex items-center gap-1.5 text-[10px] text-apple-subtext mb-2">
          {item.ProductionYear && <span>{item.ProductionYear}</span>}
          {item.CommunityRating && <RatingBadge rating={item.CommunityRating} />}
          {item.OfficialRating && <Badge variant="rating">{item.OfficialRating}</Badge>}
        </div>

        <div className="flex items-center gap-2 pt-1">
          <button
            onClick={(e) => {
              e.stopPropagation()
              navigate(`/watch/${item.Id}`)
            }}
            className="flex-1 py-1.5 bg-white text-black text-xs font-semibold rounded-squircle-sm flex items-center justify-center gap-1 hover:bg-white/90 active:scale-95 transition-all shadow-sm"
          >
            <Play className="w-3 h-3 fill-black text-black" /> {t.common.watch}
          </button>
        </div>
      </div>
    </motion.div>
  )
}
