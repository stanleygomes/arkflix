import React from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface LogoProps {
  size?: 'sm' | 'md' | 'lg' | 'xl'
  animated?: boolean
  withLink?: boolean
  className?: string
}

export const Logo: React.FC<LogoProps> = ({
  size = 'md',
  animated = true,
  withLink = true,
  className,
}) => {
  const sizeMap = {
    sm: {
      text: 'text-lg tracking-[-0.03em]',
      icon: 'w-6 h-6 text-xs',
      dot: 'w-1.5 h-1.5',
    },
    md: {
      text: 'text-xl md:text-2xl tracking-[-0.04em]',
      icon: 'w-7 h-7 text-xs',
      dot: 'w-2 h-2',
    },
    lg: {
      text: 'text-3xl md:text-4xl tracking-[-0.05em]',
      icon: 'w-10 h-10 text-base',
      dot: 'w-2.5 h-2.5',
    },
    xl: {
      text: 'text-4xl md:text-5xl tracking-[-0.05em]',
      icon: 'w-14 h-14 text-xl',
      dot: 'w-3 h-3',
    },
  }

  const content = (
    <motion.div
      whileHover={animated ? { scale: 1.03 } : undefined}
      whileTap={animated ? { scale: 0.97 } : undefined}
      transition={{ type: 'spring', stiffness: 400, damping: 25 }}
      className={cn('inline-flex items-center gap-2.5 select-none group cursor-pointer', className)}
    >
      {/* Brand Icon: Apple-styled Squircle with Arkflix Prism Wave */}
      <div
        className={cn(
          'rounded-squircle bg-gradient-to-tr from-[#0071E3] via-[#47A1FF] to-[#FFFFFF] flex items-center justify-center font-brand font-black text-black shadow-apple relative overflow-hidden transition-all duration-300 group-hover:shadow-[0_0_20px_rgba(41,151,255,0.5)]',
          sizeMap[size].icon
        )}
      >
        <span className="relative z-10 text-black font-black tracking-tighter">A</span>
        <div className="absolute inset-0 bg-white/20 opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>

      {/* Brand Typography in Syne font with Gradient Mask */}
      <div className="flex items-baseline font-brand">
        <span
          className={cn(
            'font-black bg-clip-text text-transparent bg-gradient-to-r from-white via-[#F5F5F7] to-[#86868B] transition-all',
            sizeMap[size].text
          )}
        >
          ARK<span className="bg-clip-text text-transparent bg-gradient-to-r from-[#2997FF] to-[#0071E3]">FLIX</span>
        </span>
      </div>
    </motion.div>
  )

  if (withLink) {
    return <Link to="/">{content}</Link>
  }

  return content
}
