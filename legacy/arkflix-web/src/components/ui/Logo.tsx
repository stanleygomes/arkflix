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
      text: 'text-base font-semibold tracking-tight',
      icon: 'w-6 h-6',
      iconInner: 'w-3 h-3',
    },
    md: {
      text: 'text-lg md:text-xl font-bold tracking-tight',
      icon: 'w-8 h-8',
      iconInner: 'w-4 h-4',
    },
    lg: {
      text: 'text-2xl md:text-3xl font-extrabold tracking-tight',
      icon: 'w-10 h-10',
      iconInner: 'w-5 h-5',
    },
    xl: {
      text: 'text-3xl md:text-4xl font-extrabold tracking-tight',
      icon: 'w-14 h-14',
      iconInner: 'w-6 h-6',
    },
  }

  const isLight = theme === 'light'

  const content = (
    <motion.div
      whileHover={animated ? { scale: 1.02 } : undefined}
      whileTap={animated ? { scale: 0.98 } : undefined}
      transition={{ type: 'spring', stiffness: 400, damping: 25 }}
      className={cn('inline-flex items-center gap-2.5 select-none cursor-pointer group', className)}
    >
      {/* Icon */}
      <div
        className={cn(
          'rounded-squircle flex items-center justify-center shadow-sm relative overflow-hidden transition-all duration-300',
          isLight ? 'bg-black text-white shadow-md' : 'bg-white text-black group-hover:shadow-[0_0_15px_rgba(255,255,255,0.35)]',
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

      {/* Typography with 100% Crisp Contrast */}
      <div className="flex items-center font-sans">
        <span className={cn(isLight ? 'text-[#1D1D1F] font-bold' : 'text-[#F5F5F7] font-bold', sizeMap[size].text)}>
          Ark<span className={isLight ? 'font-normal text-[#6E6E73]' : 'font-light text-[#86868B]'}>flix</span>
        </span>
      </div>
    </motion.div>
  )

  if (withLink) {
    return <Link to="/">{content}</Link>
  }

  return content
}
