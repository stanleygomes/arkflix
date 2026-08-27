import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Play, Info, ChevronLeft, ChevronRight, Plus, Check, Trash2 } from 'lucide-react'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { Button, HeroBannerSkeleton, RatingBadge } from '@/components/ui'
import { useTranslation, useToggleFavorite, useFavorites } from '@/hooks'
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
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toggleFavorite = useToggleFavorite()
  const { data: favoritesData } = useFavorites(80)

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
  const backdropUrl = getImageUrl(currentItem.Id, 'Backdrop', { fillWidth: 1280, quality: 85 })
  const hasLogo = currentItem.ImageTags?.Logo
  const logoUrl = hasLogo ? getImageUrl(currentItem.Id, 'Logo', { fillWidth: 500, quality: 90 }) : null

  // Check favorite dynamically against latest favoritesData cache or item UserData
  const isFavorite = Boolean(
    favoritesData?.Items?.some((fav) => fav.Id === currentItem.Id) ||
    currentItem.UserData?.IsFavorite
  )

  const handlePrev = () => {
    setCurrentIndex((prev) => (prev === 0 ? featuredItems.length - 1 : prev - 1))
  }

  const handleNext = () => {
    setCurrentIndex((prev) => (prev + 1) % featuredItems.length)
  }

  return (
    <div
      className="relative h-[65vh] sm:h-[75vh] md:h-[84vh] w-full select-none overflow-hidden group/hero"
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

      {/* Apple Vignettes - Heavy on mobile for readability */}
      <div className="absolute inset-0 bg-gradient-to-r from-black/95 via-black/60 to-transparent w-full md:w-3/5 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-t from-[#000000] via-[#000000]/40 to-transparent pointer-events-none" />

      {/* Content Container */}
      <div className="absolute bottom-16 sm:bottom-20 md:bottom-28 left-4 sm:left-6 md:left-14 right-4 sm:right-auto max-w-2xl z-10 space-y-3 sm:space-y-4">
        <AnimatePresence mode="wait">
          <motion.div
            key={currentItem.Id}
            initial={{ opacity: 0, y: 15 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="space-y-2 sm:space-y-3.5"
          >
            {logoUrl ? (
              <img
                src={logoUrl}
                alt={currentItem.Name}
                className="max-h-16 sm:max-h-24 md:max-h-32 object-contain mb-2 drop-shadow-2xl"
              />
            ) : (
              <h1 className="text-2xl sm:text-3xl md:text-5xl font-extrabold text-white tracking-tight leading-tight line-clamp-2">
                {currentItem.Name}
              </h1>
            )}

            {/* Metadata Pills */}
            <div className="flex flex-wrap items-center gap-2 text-[11px] sm:text-xs text-apple-subtext font-medium">
              {currentItem.ProductionYear && <span>{currentItem.ProductionYear}</span>}
              {currentItem.OfficialRating && (
                <span className="bg-white/10 text-white px-1.5 py-0.5 rounded-squircle-sm text-[9px] sm:text-[10px] uppercase font-semibold border border-white/10">
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
                  <span className="text-white/40 hidden sm:inline">•</span>
                  <span className="text-apple-subtext truncate max-w-[150px] sm:max-w-[200px] hidden sm:inline">
                    {currentItem.Genres.slice(0, 2).join(', ')}
                  </span>
                </>
              )}
            </div>

            {currentItem.Overview && (
              <p className="text-xs sm:text-sm md:text-base text-[#D1D1D6] line-clamp-2 sm:line-clamp-3 leading-relaxed font-normal max-w-xl">
                {currentItem.Overview}
              </p>
            )}
          </motion.div>
        </AnimatePresence>

        {/* Action Buttons */}
        <div className="flex items-center gap-2 sm:gap-3 pt-2">
          {/* 1. Main Watch Button */}
          <Button
            variant="primary"
            size="md"
            onClick={() => navigate(`/watch/${currentItem.Id}`)}
            className="font-bold shadow-apple text-xs sm:text-sm py-2 sm:py-2.5 px-4 sm:px-6 flex-none"
          >
            <Play className="w-4 h-4 fill-black text-black mr-1.5" /> {t.common.watch}
          </Button>

          {/* 2. My List Quick Action with Explicit Hover Remove state */}
          <button
            onClick={() => toggleFavorite.mutate({ itemId: currentItem.Id, isFavorite })}
            className={`group/fav h-9 sm:h-10 px-3 sm:px-4 rounded-full border backdrop-blur-xl flex items-center justify-center gap-1.5 text-xs sm:text-sm font-semibold transition-all active:scale-95 cursor-pointer ${
              isFavorite
                ? 'bg-blue-500/20 text-apple-accent border-blue-500/35 hover:bg-rose-500 hover:text-white hover:border-rose-500'
                : 'bg-white/10 hover:bg-white/20 text-white border-white/15'
            }`}
            title={isFavorite ? 'Remover da Minha Lista' : 'Adicionar à Minha Lista'}
          >
            {isFavorite ? (
              <>
                <Check className="w-4 h-4 text-apple-accent stroke-[3] group-hover/fav:hidden" />
                <Trash2 className="w-4 h-4 text-white hidden group-hover/fav:block" />
                <span className="group-hover/fav:hidden">Na Lista</span>
                <span className="hidden group-hover/fav:inline">Remover</span>
              </>
            ) : (
              <>
                <Plus className="w-4 h-4" />
                <span>Minha Lista</span>
              </>
            )}
          </button>

          {/* 3. Details Button */}
          <button
            onClick={() => navigate(`/title/${currentItem.Id}`)}
            className="h-9 sm:h-10 px-3 sm:px-4 rounded-full bg-white/10 hover:bg-white/20 text-white border border-white/15 backdrop-blur-xl flex items-center justify-center gap-1.5 text-xs sm:text-sm font-semibold transition-all active:scale-95 cursor-pointer"
            title="Ver detalhes"
          >
            <Info className="w-4 h-4 text-apple-subtext" />
            <span>Detalhes</span>
          </button>
        </div>
      </div>

      {/* Carousel Navigation Indicators (Pill Dots) & Arrows */}
      <div className="absolute bottom-6 sm:bottom-8 md:bottom-12 right-4 sm:right-6 md:right-14 z-30 flex items-center gap-2 sm:gap-3 pointer-events-auto">
        {/* Previous */}
        <button
          onClick={handlePrev}
          className="w-8 h-8 sm:w-9 sm:h-9 rounded-full bg-black/60 hover:bg-black/90 backdrop-blur-xl border border-white/20 text-white hidden sm:flex items-center justify-center opacity-0 group-hover/hero:opacity-100 transition-all hover:scale-110 active:scale-95 cursor-pointer shadow-apple"
          aria-label="Anterior"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>

        {/* Carousel Dots */}
        <div className="flex items-center gap-1.5 sm:gap-2 bg-black/60 backdrop-blur-xl px-3 py-2 rounded-full border border-white/15 shadow-apple">
          {featuredItems.map((item, index) => {
            const isActive = index === currentIndex
            return (
              <button
                key={item.Id}
                onClick={() => setCurrentIndex(index)}
                aria-label={`Slide ${index + 1}`}
                className={`transition-all duration-300 rounded-full cursor-pointer p-0.5 ${
                  isActive ? 'w-6 h-2 bg-white shadow-sm' : 'w-2 h-2 bg-white/40 hover:bg-white/80 hover:scale-125'
                }`}
              />
            )
          })}
        </div>

        {/* Next */}
        <button
          onClick={handleNext}
          className="w-8 h-8 sm:w-9 sm:h-9 rounded-full bg-black/60 hover:bg-black/90 backdrop-blur-xl border border-white/20 text-white hidden sm:flex items-center justify-center opacity-0 group-hover/hero:opacity-100 transition-all hover:scale-110 active:scale-95 cursor-pointer shadow-apple"
          aria-label="Próximo"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}
