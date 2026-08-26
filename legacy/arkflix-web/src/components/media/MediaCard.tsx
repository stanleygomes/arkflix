import React from 'react'
import { Play, ChevronDown } from 'lucide-react'
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

  const posterUrl = getImageUrl(item.Id, 'Primary', { fillWidth: 400, quality: 80 })
  const progressPercent = item.UserData?.PlayedPercentage || 0

  return (
    <div
      onClick={() => openModal(item)}
      className="group relative flex-none w-[160px] sm:w-[200px] md:w-[240px] aspect-[2/3] rounded-md overflow-hidden bg-netflix-dark cursor-pointer transition-transform duration-300 hover:scale-105 hover:z-20 shadow-md"
    >
      <img
        src={posterUrl}
        alt={item.Name}
        loading="lazy"
        className="w-full h-full object-cover transition-opacity duration-300 group-hover:opacity-90"
      />

      {/* Progress Bar */}
      {progressPercent > 0 && (
        <div className="absolute bottom-0 left-0 right-0 h-1.5 bg-black/60 z-10">
          <div
            className="h-full bg-netflix-red"
            style={{ width: `${Math.min(progressPercent, 100)}%` }}
          />
        </div>
      )}

      {/* Overlay on Hover */}
      <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex flex-col justify-end p-3">
        <h4 className="text-sm font-bold text-white line-clamp-1">{item.Name}</h4>
        
        <div className="flex items-center gap-2 text-[11px] text-gray-300 my-1">
          {item.ProductionYear && <span>{item.ProductionYear}</span>}
          {item.OfficialRating && (
            <Badge variant="rating">{item.OfficialRating}</Badge>
          )}
          {item.CommunityRating && (
            <RatingBadge rating={item.CommunityRating} />
          )}
        </div>

        <div className="flex items-center justify-between pt-1">
          <button
            onClick={(e) => {
              e.stopPropagation()
              navigate(`/watch/${item.Id}`)
            }}
            className="w-8 h-8 rounded-full bg-white text-black flex items-center justify-center hover:bg-white/80 transition-colors"
            title="Assistir agora"
          >
            <Play className="w-4 h-4 fill-black ml-0.5" />
          </button>

          <button
            onClick={(e) => {
              e.stopPropagation()
              openModal(item)
            }}
            className="w-8 h-8 rounded-full border border-white/40 text-white flex items-center justify-center hover:border-white transition-colors"
            title="Mais detalhes"
          >
            <ChevronDown className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
