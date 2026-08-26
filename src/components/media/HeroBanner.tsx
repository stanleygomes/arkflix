import React from 'react'
import { Play, Info } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Button } from '@/components/ui/Button'
import { useModalStore } from '@/stores/modalStore'
import { useNavigate } from 'react-router-dom'

interface HeroBannerProps {
  item?: MediaItem
}

export const HeroBanner: React.FC<HeroBannerProps> = ({ item }) => {
  const { openModal } = useModalStore()
  const navigate = useNavigate()

  if (!item) {
    return (
      <div className="relative h-[65vh] md:h-[80vh] w-full bg-netflix-dark animate-pulse flex items-end pb-24 px-4 md:px-12">
        <div className="space-y-4 max-w-xl">
          <div className="h-10 bg-white/10 rounded w-3/4"></div>
          <div className="h-4 bg-white/10 rounded w-full"></div>
          <div className="h-4 bg-white/10 rounded w-2/3"></div>
        </div>
      </div>
    )
  }

  const backdropUrl = getImageUrl(item.Id, 'Backdrop', { fillWidth: 1920, quality: 85 })
  const hasLogo = item.ImageTags?.Logo
  const logoUrl = hasLogo ? getImageUrl(item.Id, 'Logo', { fillWidth: 600, quality: 90 }) : null

  return (
    <div className="relative h-[65vh] md:h-[85vh] w-full select-none">
      {/* Backdrop Image */}
      <img
        src={backdropUrl}
        alt={item.Name}
        className="absolute inset-0 w-full h-full object-cover object-center"
      />

      {/* Gradients to blend into Netflix background */}
      <div className="absolute inset-0 bg-gradient-to-r from-netflix-black via-netflix-black/40 to-transparent w-full md:w-2/3"></div>
      <div className="absolute inset-0 bg-gradient-to-t from-netflix-black via-transparent to-black/30"></div>

      {/* Content */}
      <div className="absolute bottom-12 md:bottom-20 left-4 md:left-12 max-w-2xl z-10 space-y-4">
        {logoUrl ? (
          <img
            src={logoUrl}
            alt={item.Name}
            className="max-h-28 md:max-h-36 object-contain mb-2 drop-shadow-xl"
          />
        ) : (
          <h1 className="text-3xl md:text-5xl font-black text-white drop-shadow-lg tracking-tight">
            {item.Name}
          </h1>
        )}

        {item.Overview && (
          <p className="text-sm md:text-base text-netflix-lightGray line-clamp-3 md:line-clamp-4 drop-shadow">
            {item.Overview}
          </p>
        )}

        <div className="flex items-center gap-3 pt-2">
          <Button
            variant="primary"
            size="lg"
            onClick={() => navigate(`/watch/${item.Id}`)}
            className="shadow-lg"
          >
            <Play className="w-5 h-5 fill-black" /> Assistir
          </Button>

          <Button
            variant="secondary"
            size="lg"
            onClick={() => openModal(item)}
            className="shadow-lg"
          >
            <Info className="w-5 h-5" /> Mais Informações
          </Button>
        </div>
      </div>
    </div>
  )
}
