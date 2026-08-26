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
      iconInner: 'w-3.5 h-3.5',
    },
    md: {
      text: 'text-2xl md:text-3xl font-extrabold tracking-tight',
      icon: 'w-10 h-10 md:w-11 md:h-11',
      iconInner: 'w-5 h-5 md:w-5.5 md:h-5.5',
    },
    lg: {
      text: 'text-3xl md:text-4xl font-extrabold tracking-tight',
      icon: 'w-12 h-12 md:w-14 md:h-14',
      iconInner: 'w-6 h-6',
    },
    xl: {
      text: 'text-4xl md:text-5xl font-extrabold tracking-tight',
      icon: 'w-16 h-16',
      iconInner: 'w-7 h-7',
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
      {/* Icon Squircle (Enlarged & Prominent) */}
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
          <path d="M4 4.5A2.5 2.5 0 0 1 6.5 2h11A2.5 2.5 0 0 1 20 4.5v15a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 19.5v-15ZM8 4H6.5A.5.5 0 0 0 6 4.5V6h2V4Zm0 4H6v2h2V8Zm0 4H6v2h2v-2Zm0 4H6v2h2v-2Zm0 4H6.5a.5.5 0 0 0 .5-.5V20H6v-.5h2v.5Zm10-16h-1.5V6H18V4.5a.5.5 0 0 0-.5-.5Zm0 4h-2v2h2V8Zm0 4h-2v2h2v-2Zm0 4h-2v2h2v-2Zm0 4h-2v2h1.5a.5.5 0 0 0 .5-.5V20Zm-7.5-14a.5.5 0 0 0-.5.5v13a.5.5 0 0 0 .75.43l10-6.5a.5.5 0 0 0 0-.86l-10-6.5a.5.5 0 0 0-.25-.07Z" />
        </svg>
      </div>

      {/* Typography with 100% Crisp High-Contrast & Accessibility */}
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
