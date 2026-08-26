import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { AppleSpinner } from './AppleSpinner'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  icon?: React.ReactNode
  isLoading?: boolean
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, icon, isLoading, id, type = 'text', disabled, onFocus, onBlur, ...props }, ref) => {
    const [isFocused, setIsFocused] = useState(false)
    const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined)

    return (
      <div className="w-full space-y-1.5">
        {label && (
          <motion.label
            htmlFor={inputId}
            animate={{ color: isFocused ? '#2997FF' : '#86868B' }}
            transition={{ duration: 0.2 }}
            className="block text-xs font-medium pl-1"
          >
            {label}
          </motion.label>
        )}
        <motion.div
          animate={{
            scale: isFocused ? 1.008 : 1,
          }}
          transition={{ type: 'spring', stiffness: 400, damping: 30 }}
          className="relative flex items-center will-change-transform"
        >
          {icon && (
            <div
              className={cn(
                'absolute left-3.5 flex items-center pointer-events-none transition-colors duration-200',
                isFocused ? 'text-apple-accent' : 'text-apple-subtext'
              )}
            >
              {icon}
            </div>
          )}

          <input
            id={inputId}
            ref={ref}
            type={type}
            disabled={disabled || isLoading}
            onFocus={(e) => {
              setIsFocused(true)
              onFocus?.(e)
            }}
            onBlur={(e) => {
              setIsFocused(false)
              onBlur?.(e)
            }}
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
        </motion.div>

        {error && (
          <motion.p
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="text-xs text-red-400 pl-1 mt-1 font-medium"
          >
            {error}
          </motion.p>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
