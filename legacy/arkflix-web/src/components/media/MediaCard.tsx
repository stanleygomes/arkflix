import React from 'react'
import { Play } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Badge, RatingBadge } from '@/components/ui'
import { useModalStore } from '@/stores/modalStore'
import { useNavigate } from 'react-router-dom'

interface MediaCardProps {
  item: MediaItem
}

export const MediaCard: React.FC<MediaCardProps> = ({ item }) => {
  const { openModal } = useModalStore()
  const navigate = useNavigate()

  const imageUrl = getImageUrl(item.Id, 'Primary', { fillWidth: 400, quality: 85 })
  const progressPercent = item.UserData?.PlayedPercentage || 0

  return (
    <div
      onClick={() => openModal(item)}
      className="group relative flex-none w-[160px] sm:w-[195px] md:w-[230px] aspect-[2/3] rounded-squircle-lg overflow-hidden bg-[#1C1C1E] cursor-pointer transition-all duration-300 hover:scale-105 hover:shadow-apple border border-white/10 hover:border-white/30"
    >
      <img
        src={imageUrl}
        alt={item.Name}
        loading="lazy"
        className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
      />

      {/* Progress Bar (Continuar Assistindo) */}
      {progressPercent > 0 && (
        <div className="absolute bottom-0 left-0 right-0 h-1 bg-black/60 z-10">
          <div
            className="h-full bg-white"
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
            <Play className="w-3 h-3 fill-black text-black" /> Assistir
          </button>
        </div>
      </div>
    </div>
  )
}
