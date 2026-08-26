import React from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface LogoProps {
  size?: 'sm' | 'md' | 'lg' | 'xl'
  theme?: 'dark' | 'light' | 'auto'
  animated?: boolean
  withLink?: boolean
  className?: string
}

export const Logo: React.FC<LogoProps> = ({
  size = 'md',
  theme = 'dark',
  animated = true,
  withLink = true,
  className,
}) => {
  const sizeMap = {
    sm: {
      text: 'text-xl font-bold tracking-tight',
      iconContainer: 'w-8 h-8',
    },
    md: {
      text: 'text-2xl md:text-3xl font-extrabold tracking-tight',
      iconContainer: 'w-10 h-10 md:w-11 md:h-11',
    },
    lg: {
      text: 'text-3xl md:text-4xl font-extrabold tracking-tight',
      iconContainer: 'w-14 h-14 md:w-16 md:h-16',
    },
    xl: {
      text: 'text-4xl md:text-5xl font-extrabold tracking-tight',
      iconContainer: 'w-18 h-18 md:w-20 md:h-20',
    },
  }

  const isLight = theme === 'light'

  const content = (
    <motion.div
      whileHover={animated ? { scale: 1.04 } : undefined}
      whileTap={animated ? { scale: 0.96 } : undefined}
      transition={{ type: 'spring', stiffness: 450, damping: 25 }}
      className={cn('inline-flex items-center gap-3 select-none cursor-pointer group', className)}
    >
      {/* High-Impact Popcorn Bucket Emblem */}
      <div
        className={cn(
          'relative flex items-center justify-center filter drop-shadow-md transition-all duration-300 group-hover:drop-shadow-[0_0_15px_rgba(255,215,0,0.4)]',
          sizeMap[size].iconContainer
        )}
      >
        <svg
          viewBox="0 0 48 48"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          className="w-full h-full overflow-visible"
        >
          {/* Glowing Popcorn Kernels / Top */}
          <circle cx="16" cy="14" r="5.5" fill="#FFD000" />
          <circle cx="24" cy="10.5" r="6.5" fill="#FFE566" />
          <circle cx="32" cy="14" r="5.5" fill="#FFD000" />
          <circle cx="20" cy="15" r="5" fill="#FFF280" />
          <circle cx="28" cy="15" r="5" fill="#FFE566" />

          {/* Golden Kernel Highlights */}
          <ellipse cx="23" cy="9.5" rx="2.5" ry="1.5" fill="#FFFFFF" opacity="0.7" />
          <ellipse cx="15" cy="13" rx="2" ry="1.2" fill="#FFFFFF" opacity="0.6" />

          {/* Classic Cinema Popcorn Bucket (Red/White or Monochromatic Crisp) */}
          <path
            d="M10.5 19H37.5L34.2 41.5C34 42.8 32.9 43.8 31.6 43.8H16.4C15.1 43.8 14 42.8 13.8 41.5L10.5 19Z"
            fill={isLight ? '#1D1D1F' : '#E50914'}
          />

          {/* Bucket Stripes */}
          <path d="M19 19.5L17.5 43.5H21.5L22.5 19.5H19Z" fill="#FFFFFF" />
          <path d="M29 19.5L30.5 43.5H26.5L25.5 19.5H29Z" fill="#FFFFFF" />

          {/* Top Bucket Rim */}
          <rect
            x="9"
            y="17"
            width="30"
            height="3"
            rx="1.5"
            fill={isLight ? '#000000' : '#B81D24'}
          />
        </svg>
      </div>

      {/* Modern High-End Apple Typography */}
      <div className="flex items-baseline font-sans">
        <span className={cn(isLight ? 'text-[#1D1D1F]' : 'text-white', sizeMap[size].text)}>
          Ark<span className={isLight ? 'font-medium text-[#6E6E73]' : 'font-light text-[#86868B]'}>flix</span>
        </span>
      </div>
    </motion.div>
  )

  if (withLink) {
    return <Link to="/">{content}</Link>
  }

  return content
}
