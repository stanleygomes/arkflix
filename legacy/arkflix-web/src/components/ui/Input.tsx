import React from 'react'
import { cn } from '@/lib/utils'
import { AppleSpinner } from './AppleSpinner'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  icon?: React.ReactNode
  isLoading?: boolean
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, icon, isLoading, id, type = 'text', disabled, ...props }, ref) => {
    const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined)

    return (
      <div className="w-full space-y-1.5">
        {label && (
          <label htmlFor={inputId} className="block text-xs font-medium text-apple-subtext pl-1">
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          {icon && (
            <div className="absolute left-3.5 text-apple-subtext flex items-center pointer-events-none transition-colors">
              {icon}
            </div>
          )}

          <input
            id={inputId}
            ref={ref}
            type={type}
            disabled={disabled || isLoading}
            className={cn(
              'w-full bg-[#1C1C1E]/80 text-[#F5F5F7] placeholder-[#636366] text-sm rounded-squircle px-4 py-3 border border-white/10 backdrop-blur-md transition-all duration-200 focus:outline-none focus:border-apple-accent/60 focus:bg-[#2C2C2E] focus:ring-4 focus:ring-apple-accent/20 disabled:opacity-50 disabled:cursor-not-allowed',
              icon && 'pl-10',
              isLoading && 'pr-10',
              error && 'border-red-500/80 focus:border-red-500 focus:ring-red-500/20',
              className
            )}
            {...props}
          />

          {isLoading && (
            <div className="absolute right-3.5 flex items-center pointer-events-none">
              <AppleSpinner size="sm" color="gray" />
            </div>
          )}
        </div>

        {error && (
          <p className="text-xs text-red-400 pl-1 mt-1 font-medium animate-fadeIn">{error}</p>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
