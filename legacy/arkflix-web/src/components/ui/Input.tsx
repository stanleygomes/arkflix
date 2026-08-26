import React from 'react'
import { cn } from '@/lib/utils'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  icon?: React.ReactNode
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, icon, id, type = 'text', ...props }, ref) => {
    const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined)

    return (
      <div className="w-full space-y-1.5">
        {label && (
          <label htmlFor={inputId} className="block text-xs font-medium text-netflix-gray">
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          {icon && (
            <div className="absolute left-3 text-netflix-gray flex items-center pointer-events-none">
              {icon}
            </div>
          )}
          <input
            id={inputId}
            ref={ref}
            type={type}
            className={cn(
              'w-full bg-[#333333] text-white placeholder-netflix-gray text-sm rounded px-4 py-3 border border-transparent transition-all duration-200 focus:outline-none focus:bg-[#454545] focus:border-white/40 focus:ring-1 focus:ring-white/40 disabled:opacity-50 disabled:cursor-not-allowed',
              icon && 'pl-10',
              error && 'border-netflix-red focus:border-netflix-red focus:ring-netflix-red',
              className
            )}
            {...props}
          />
        </div>
        {error && <p className="text-xs text-netflix-red mt-1">{error}</p>}
      </div>
    )
  }
)

Input.displayName = 'Input'
