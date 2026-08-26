import React from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { AppleSpinner } from './AppleSpinner'

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'glass' | 'ghost' | 'icon' | 'apple-blue'
  size?: 'sm' | 'md' | 'lg'
  isLoading?: boolean
  loadingText?: string
  icon?: React.ReactNode
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant = 'primary',
      size = 'md',
      isLoading = false,
      loadingText,
      icon,
      children,
      disabled,
      type = 'button',
      onClick,
      ...props
    },
    ref
  ) => {
    const baseStyles =
      'relative inline-flex items-center justify-center font-medium rounded-squircle transition-colors duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-apple-accent/50 disabled:opacity-45 disabled:pointer-events-none select-none will-change-transform'

    const variants = {
      // Apple Solid Primary
      primary:
        'bg-[#000000] text-white dark:bg-[#FFFFFF] dark:text-[#000000] font-semibold hover:opacity-90 shadow-sm',
      // Apple Tinted Blue
      'apple-blue':
        'bg-apple-blue text-white font-semibold hover:bg-apple-blueHover shadow-sm',
      // iOS Glass Material
      glass:
        'bg-black/5 dark:bg-white/10 text-apple-text backdrop-blur-xl border border-black/10 dark:border-white/15 hover:bg-black/10 dark:hover:bg-white/20 shadow-glass',
      // Apple Subtle Secondary Fill
      secondary:
        'bg-black/5 dark:bg-white/10 text-apple-text hover:bg-black/10 dark:hover:bg-white/15',
      // Clean Ghost
      ghost:
        'bg-transparent text-apple-subtext hover:text-apple-text hover:bg-black/5 dark:hover:bg-white/5',
      // Squircle Icon Button
      icon:
        'bg-black/5 dark:bg-white/10 text-apple-text backdrop-blur-lg border border-black/10 dark:border-white/10 hover:bg-black/10 dark:hover:bg-white/20 rounded-full p-2.5',
    }

    const sizes = {
      sm: 'text-xs px-3.5 py-1.5 gap-1.5 rounded-squircle-sm',
      md: 'text-sm px-5 py-2.5 gap-2 rounded-squircle',
      lg: 'text-base px-6 py-3.5 gap-2.5 rounded-squircle-lg',
    }

    const spinnerColor = variant === 'primary' ? 'gray' : 'blue'

    return (
      <motion.button
        ref={ref}
        type={type}
        disabled={disabled || isLoading}
        onClick={onClick}
        whileTap={{ scale: 0.96 }}
        whileHover={{ scale: 1.015 }}
        transition={{ type: 'spring', stiffness: 450, damping: 25 }}
        className={cn(
          baseStyles,
          variants[variant],
          variant !== 'icon' && sizes[size],
          isLoading && 'cursor-wait',
          className
        )}
        {...(props as any)}
      >
        {isLoading ? (
          <>
            <AppleSpinner size={size === 'lg' ? 'md' : 'sm'} color={spinnerColor} className="mr-2" />
            <span>{loadingText || children}</span>
          </>
        ) : (
          <>
            {icon && <span className="flex items-center">{icon}</span>}
            {children}
          </>
        )}
      </motion.button>
    )
  }
)

Button.displayName = 'Button'
