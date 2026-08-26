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
      text: 'text-lg font-bold tracking-tight',
      icon: 'w-7 h-7',
      iconInner: 'w-4 h-4',
    },
    md: {
      text: 'text-2xl md:text-3xl font-extrabold tracking-tight',
      icon: 'w-10 h-10 md:w-11 md:h-11',
      iconInner: 'w-6 h-6',
    },
    lg: {
      text: 'text-3xl md:text-4xl font-extrabold tracking-tight',
      icon: 'w-12 h-12 md:w-14 md:h-14',
      iconInner: 'w-7 h-7',
    },
    xl: {
      text: 'text-4xl md:text-5xl font-extrabold tracking-tight',
      icon: 'w-16 h-16',
      iconInner: 'w-8 h-8',
    },
  }

  const isLight = theme === 'light'

  const content = (
    <motion.div
      whileHover={animated ? { scale: 1.03 } : undefined}
      whileTap={animated ? { scale: 0.97 } : undefined}
      transition={{ type: 'spring', stiffness: 400, damping: 25 }}
      className={cn('inline-flex items-center gap-3 select-none cursor-pointer group', className)}
    >
      {/* Icon Squircle with Popcorn Bucket Artwork */}
      <div
        className={cn(
          'rounded-squircle flex items-center justify-center shadow-md relative overflow-hidden transition-all duration-300',
          isLight
            ? 'bg-black text-white shadow-lg'
            : 'bg-white text-black shadow-apple group-hover:shadow-[0_0_20px_rgba(255,255,255,0.45)]',
          sizeMap[size].icon
        )}
      >
        <svg
          viewBox="0 0 24 24"
          fill="currentColor"
          className={cn(isLight ? 'text-white' : 'text-black', sizeMap[size].iconInner)}
        >
          {/* Popcorn Fluffs Top */}
          <path d="M7 6.5a2.2 2.2 0 0 1 2.2-2.2c.4 0 .76.11 1.08.31a2.3 2.3 0 0 1 3.44 0A2.2 2.2 0 0 1 14.8 4.3a2.2 2.2 0 0 1 2.2 2.2c0 .48-.15.93-.41 1.3A2.3 2.3 0 0 1 18 10a2.3 2.3 0 0 1-2.3 2.3H8.3A2.3 2.3 0 0 1 6 10c0-.85.46-1.6 1.15-1.99A2.2 2.2 0 0 1 7 6.5Z" />
          {/* Popcorn Bucket Base */}
          <path d="M6.6 11.5h10.8l-1.3 9.4a1.8 1.8 0 0 1-1.78 1.6H9.68a1.8 1.8 0 0 1-1.78-1.6L6.6 11.5Zm3.2 2v6m4.4-6v6" fillRule="evenodd" stroke={isLight ? '#000000' : '#FFFFFF'} strokeWidth="1.2" strokeLinecap="round" />
        </svg>
      </div>

      {/* Typography */}
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
