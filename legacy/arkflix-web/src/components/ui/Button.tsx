import React from 'react'
import { cn } from '@/lib/utils'

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'icon'
  size?: 'sm' | 'md' | 'lg'
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = 'primary', size = 'md', children, ...props }, ref) => {
    const baseStyles = 'inline-flex items-center justify-center font-semibold rounded transition-all duration-200 focus:outline-none disabled:opacity-50 disabled:pointer-events-none'
    
    const variants = {
      primary: 'bg-white text-black hover:bg-white/80 active:bg-white/60',
      secondary: 'bg-netflix-gray/50 text-white hover:bg-netflix-gray/30 active:bg-netflix-gray/20',
      ghost: 'bg-transparent text-white hover:bg-white/10',
      icon: 'bg-netflix-dark/60 text-white border border-white/20 hover:border-white hover:bg-netflix-dark/80 rounded-full',
    }

    const sizes = {
      sm: 'text-xs px-3 py-1.5 gap-1.5',
      md: 'text-sm px-5 py-2.5 gap-2',
      lg: 'text-base px-7 py-3 gap-2.5',
    }

    return (
      <button
        ref={ref}
        className={cn(baseStyles, variants[variant], sizes[size], className)}
        {...props}
      >
        {children}
      </button>
    )
  }
)

Button.displayName = 'Button'
