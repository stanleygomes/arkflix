import React, { useState, useRef, useEffect } from 'react'
import { motion, useSpring, useTransform } from 'framer-motion'
import { AppleSpinner } from './AppleSpinner'
import { ArrowDown } from 'lucide-react'

interface PullToRefreshProps {
  onRefresh: () => Promise<any> | void
  children: React.ReactNode
  threshold?: number
  className?: string
}

export const PullToRefresh: React.FC<PullToRefreshProps> = ({
  onRefresh,
  children,
  threshold = 70,
  className = '',
}) => {
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [pullProgress, setPullProgress] = useState(0) // 0 to 1+
  const isPullingRef = useRef(false)
  const startYRef = useRef(0)
  const currentYRef = useRef(0)

  const pullY = useSpring(0, {
    stiffness: 400,
    damping: 35,
    mass: 0.8,
  })

  const spinnerOpacity = useTransform(pullY, [0, 20, threshold], [0, 0.4, 1])
  const spinnerScale = useTransform(pullY, [0, threshold], [0.6, 1])
  const arrowRotation = useTransform(pullY, [0, threshold], [0, 180])

  const handleTouchStart = (e: TouchEvent) => {
    if (isRefreshing) return
    const scrollTop = window.scrollY || document.documentElement.scrollTop
    if (scrollTop <= 2) {
      isPullingRef.current = true
      startYRef.current = e.touches[0].clientY
      currentYRef.current = e.touches[0].clientY
    }
  }

  const handleTouchMove = (e: TouchEvent) => {
    if (!isPullingRef.current || isRefreshing) return
    const scrollTop = window.scrollY || document.documentElement.scrollTop
    if (scrollTop > 2) {
      isPullingRef.current = false
      pullY.set(0)
      setPullProgress(0)
      return
    }

    currentYRef.current = e.touches[0].clientY
    const rawDistance = Math.max(0, currentYRef.current - startYRef.current)

    if (rawDistance > 0) {
      // iOS rubber-band damping
      const dampingFactor = 0.45
      const dampedDistance = Math.min(rawDistance * dampingFactor, threshold * 1.5)
      pullY.set(dampedDistance)
      setPullProgress(Math.min(dampedDistance / threshold, 1.2))

      if (e.cancelable && dampedDistance > 10) {
        e.preventDefault()
      }
    }
  }

  const handleTouchEnd = async () => {
    if (!isPullingRef.current || isRefreshing) return
    isPullingRef.current = false

    const rawDistance = Math.max(0, currentYRef.current - startYRef.current)
    const dampedDistance = rawDistance * 0.45

    if (dampedDistance >= threshold) {
      setIsRefreshing(true)
      pullY.set(50)

      if (navigator.vibrate) {
        try {
          navigator.vibrate(12)
        } catch {}
      }

      try {
        await Promise.resolve(onRefresh())
      } finally {
        setTimeout(() => {
          setIsRefreshing(false)
          pullY.set(0)
          setPullProgress(0)
        }, 400)
      }
    } else {
      pullY.set(0)
      setPullProgress(0)
    }
  }

  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    const onTouchMove = (e: TouchEvent) => handleTouchMove(e)
    const onTouchStart = (e: TouchEvent) => handleTouchStart(e)
    const onTouchEnd = () => handleTouchEnd()

    el.addEventListener('touchstart', onTouchStart, { passive: true })
    el.addEventListener('touchmove', onTouchMove, { passive: false })
    el.addEventListener('touchend', onTouchEnd, { passive: true })

    return () => {
      el.removeEventListener('touchstart', onTouchStart)
      el.removeEventListener('touchmove', onTouchMove)
      el.removeEventListener('touchend', onTouchEnd)
    }
  }, [isRefreshing, threshold, onRefresh])

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      {/* Pull Indicator */}
      <motion.div
        style={{
          y: pullY,
          opacity: spinnerOpacity,
        }}
        className="absolute top-0 left-0 right-0 z-30 flex items-center justify-center pointer-events-none -mt-8"
      >
        <motion.div
          style={{ scale: spinnerScale }}
          className="w-9 h-9 rounded-full bg-black/85 backdrop-blur-xl border border-white/20 shadow-apple flex items-center justify-center text-white"
        >
          {isRefreshing ? (
            <AppleSpinner size="sm" color="blue" />
          ) : pullProgress >= 1 ? (
            <motion.div style={{ rotate: arrowRotation }}>
              <ArrowDown className="w-4 h-4 text-apple-accent stroke-[2.5]" />
            </motion.div>
          ) : (
            <motion.div style={{ rotate: arrowRotation }}>
              <ArrowDown className="w-4 h-4 text-white/70 stroke-[2]" />
            </motion.div>
          )}
        </motion.div>
      </motion.div>

      {/* Content wrapper with spring transform */}
      <motion.div style={{ y: pullY }}>
        {children}
      </motion.div>
    </div>
  )
}
