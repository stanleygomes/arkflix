import React from 'react'
import { Play, Info } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Button, HeroBannerSkeleton } from '@/components/ui'
import { useModalStore } from '@/stores/modalStore'
import { useTranslation } from '@/hooks'
import { useNavigate } from 'react-router-dom'

interface HeroBannerProps {
  item?: MediaItem
  isLoading?: boolean
}

export const HeroBanner: React.FC<HeroBannerProps> = ({ item, isLoading }) => {
  const { openModal } = useModalStore()
  const { t } = useTranslation()
  const navigate = useNavigate()

  if (isLoading || !item) {
    return <HeroBannerSkeleton />
  }

  const backdropUrl = getImageUrl(item.Id, 'Backdrop', { fillWidth: 1920, quality: 85 })
  const hasLogo = item.ImageTags?.Logo
  const logoUrl = hasLogo ? getImageUrl(item.Id, 'Logo', { fillWidth: 600, quality: 90 }) : null

  return (
    <div className="relative h-[70vh] md:h-[82vh] w-full select-none overflow-hidden">
      {/* Backdrop Image */}
      <img
        src={backdropUrl}
        alt={item.Name}
        className="absolute inset-0 w-full h-full object-cover object-center scale-[1.02] transition-transform duration-1000 ease-out"
      />

      {/* Apple Subtle Vignette & Bottom Blends */}
      <div className="absolute inset-0 bg-gradient-to-r from-black/90 via-black/40 to-transparent w-full md:w-3/5"></div>
      <div className="absolute inset-0 bg-gradient-to-t from-apple-bg via-apple-bg/30 to-transparent"></div>

      {/* Content Container */}
      <div className="absolute bottom-12 md:bottom-20 left-6 md:left-14 max-w-2xl z-10 space-y-4">
        {logoUrl ? (
          <img
            src={logoUrl}
            alt={item.Name}
            className="max-h-24 md:max-h-32 object-contain mb-3 drop-shadow-2xl"
          />
        ) : (
          <h1 className="text-3xl md:text-5xl font-extrabold text-white tracking-tight leading-tight">
            {item.Name}
          </h1>
        )}

        {/* Metadata tag pills */}
        <div className="flex items-center gap-2 text-xs text-apple-subtext font-medium">
          {item.ProductionYear && <span>{item.ProductionYear}</span>}
          {item.OfficialRating && (
            <span className="bg-white/10 text-white px-2 py-0.5 rounded-squircle-sm text-[10px] uppercase font-semibold">
              {item.OfficialRating}
            </span>
          )}
          {item.CommunityRating && (
            <span className="text-amber-400 font-semibold">★ {item.CommunityRating.toFixed(1)}</span>
          )}
          <span className="text-white/40">•</span>
          <span className="text-white/80">{item.Type === 'Series' ? t.common.series : t.common.movie}</span>
        </div>

        {item.Overview && (
          <p className="text-sm md:text-base text-[#D1D1D6] line-clamp-3 leading-relaxed font-normal">
            {item.Overview}
          </p>
        )}

        {/* Apple HIG CTA Buttons */}
        <div className="flex items-center gap-3 pt-3">
          <Button
            variant="primary"
            size="lg"
            onClick={() => navigate(`/watch/${item.Id}`)}
            className="font-semibold shadow-apple"
          >
            <Play className="w-4 h-4 fill-black text-black mr-1" /> {t.common.watch}
          </Button>

          <Button
            variant="glass"
            size="lg"
            onClick={() => openModal(item)}
            className="font-medium text-white"
          >
            <Info className="w-4 h-4 mr-1 text-apple-subtext" /> {t.common.details}
          </Button>
        </div>
      </div>
    </div>
  )
}
