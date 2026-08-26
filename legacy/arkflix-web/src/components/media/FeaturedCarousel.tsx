import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Play, Info, ChevronLeft, ChevronRight } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Button, HeroBannerSkeleton, RatingBadge } from '@/components/ui'
import { useModalStore } from '@/stores/modalStore'
import { useTranslation } from '@/hooks'
import { useNavigate } from 'react-router-dom'

interface FeaturedCarouselProps {
  items: MediaItem[]
  isLoading?: boolean
  autoRotateInterval?: number
}

export const FeaturedCarousel: React.FC<FeaturedCarouselProps> = ({
  items = [],
  isLoading,
  autoRotateInterval = 8000,
}) => {
  const { openModal } = useModalStore()
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [currentIndex, setCurrentIndex] = useState(0)
  const [isPaused, setIsPaused] = useState(false)

  const featuredItems = items.slice(0, 6)

  // Auto rotate timer
  useEffect(() => {
    if (featuredItems.length <= 1 || isPaused) return

    const timer = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % featuredItems.length)
    }, autoRotateInterval)

    return () => clearInterval(timer)
  }, [featuredItems.length, isPaused, autoRotateInterval])

  if (isLoading || featuredItems.length === 0) {
    return <HeroBannerSkeleton />
  }

  const currentItem = featuredItems[currentIndex] || featuredItems[0]
  const backdropUrl = getImageUrl(currentItem.Id, 'Backdrop', { fillWidth: 1920, quality: 85 })
  const hasLogo = currentItem.ImageTags?.Logo
  const logoUrl = hasLogo ? getImageUrl(currentItem.Id, 'Logo', { fillWidth: 600, quality: 90 }) : null

  const handlePrev = () => {
    setCurrentIndex((prev) => (prev === 0 ? featuredItems.length - 1 : prev - 1))
  }

  const handleNext = () => {
    setCurrentIndex((prev) => (prev + 1) % featuredItems.length)
  }

  return (
    <div
      className="relative h-[72vh] md:h-[84vh] w-full select-none overflow-hidden group/hero"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
    >
      {/* Animated Backdrop Carousel with crossfade */}
      <AnimatePresence mode="wait">
        <motion.div
          key={currentItem.Id}
          initial={{ opacity: 0, scale: 1.04 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.9, ease: [0.25, 1, 0.5, 1] }}
          className="absolute inset-0 w-full h-full"
        >
          <img
            src={backdropUrl}
            alt={currentItem.Name}
            className="w-full h-full object-cover object-center"
          />
        </motion.div>
      </AnimatePresence>

      {/* Apple Subtle Vignette & Bottom Blends */}
      <div className="absolute inset-0 bg-gradient-to-r from-black/90 via-black/45 to-transparent w-full md:w-3/5 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-t from-apple-bg via-apple-bg/30 to-transparent pointer-events-none" />

      {/* Content Container */}
      <div className="absolute bottom-14 md:bottom-24 left-6 md:left-14 max-w-2xl z-10 space-y-4">
        <AnimatePresence mode="wait">
          <motion.div
            key={currentItem.Id}
            initial={{ opacity: 0, y: 15 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="space-y-3.5"
          >
            {logoUrl ? (
              <img
                src={logoUrl}
                alt={currentItem.Name}
                className="max-h-24 md:max-h-32 object-contain mb-3 drop-shadow-2xl"
              />
            ) : (
              <h1 className="text-3xl md:text-5xl font-extrabold text-white tracking-tight leading-tight">
                {currentItem.Name}
              </h1>
            )}

            {/* Metadata Pills */}
            <div className="flex items-center gap-2.5 text-xs text-apple-subtext font-medium">
              {currentItem.ProductionYear && <span>{currentItem.ProductionYear}</span>}
              {currentItem.OfficialRating && (
                <span className="bg-white/10 text-white px-2 py-0.5 rounded-squircle-sm text-[10px] uppercase font-semibold border border-white/10">
                  {currentItem.OfficialRating}
                </span>
              )}
              {currentItem.CommunityRating && (
                <RatingBadge rating={currentItem.CommunityRating} />
              )}
              <span className="text-white/40">•</span>
              <span className="text-white/80">
                {currentItem.Type === 'Series' ? t.common.series : t.common.movie}
              </span>
              {currentItem.Genres && currentItem.Genres.length > 0 && (
                <>
                  <span className="text-white/40">•</span>
                  <span className="text-apple-subtext truncate max-w-[200px]">
                    {currentItem.Genres.slice(0, 3).join(', ')}
                  </span>
                </>
              )}
            </div>

            {currentItem.Overview && (
              <p className="text-sm md:text-base text-[#D1D1D6] line-clamp-3 leading-relaxed font-normal max-w-xl">
                {currentItem.Overview}
              </p>
            )}
          </motion.div>
        </AnimatePresence>

        {/* Action Buttons */}
        <div className="flex items-center gap-3 pt-3">
          <Button
            variant="primary"
            size="lg"
            onClick={() => navigate(`/watch/${currentItem.Id}`)}
            className="font-semibold shadow-apple"
          >
            <Play className="w-4 h-4 fill-black text-black mr-1" /> {t.common.watch}
          </Button>

          <Button
            variant="glass"
            size="lg"
            onClick={() => openModal(currentItem)}
            className="font-medium text-white"
          >
            <Info className="w-4 h-4 mr-1 text-apple-subtext" /> {t.common.details}
          </Button>
        </div>
      </div>

      {/* Carousel Navigation Indicators (Pill Dots) & Arrows */}
      <div className="absolute bottom-6 right-6 md:right-14 z-20 flex items-center gap-3">
        {/* Previous */}
        <button
          onClick={handlePrev}
          className="w-9 h-9 rounded-full bg-black/40 hover:bg-black/70 backdrop-blur-xl border border-white/10 text-white flex items-center justify-center opacity-0 group-hover/hero:opacity-100 transition-all hover:scale-105 active:scale-95"
          aria-label="Anterior"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>

        {/* Carousel Dots */}
        <div className="flex items-center gap-2 bg-black/40 backdrop-blur-xl px-3 py-2 rounded-full border border-white/10">
          {featuredItems.map((item, index) => {
            const isActive = index === currentIndex
            return (
              <button
                key={item.Id}
                onClick={() => setCurrentIndex(index)}
                aria-label={`Slide ${index + 1}`}
                className={`transition-all duration-300 rounded-full ${
                  isActive ? 'w-6 h-2 bg-white' : 'w-2 h-2 bg-white/30 hover:bg-white/60'
                }`}
              />
            )
          })}
        </div>

        {/* Next */}
        <button
          onClick={handleNext}
          className="w-9 h-9 rounded-full bg-black/40 hover:bg-black/70 backdrop-blur-xl border border-white/10 text-white flex items-center justify-center opacity-0 group-hover/hero:opacity-100 transition-all hover:scale-105 active:scale-95"
          aria-label="Próximo"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}
