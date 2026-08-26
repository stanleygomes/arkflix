import React from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface SelectOption {
  value: string
  label: string
}

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  options?: SelectOption[]
  error?: string
}

export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, label, options = [], children, error, id, ...props }, ref) => {
    const selectId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined)

    return (
      <div className="w-full space-y-1.5">
        {label && (
          <label htmlFor={selectId} className="block text-xs font-medium text-apple-subtext pl-1">
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          <select
            id={selectId}
            ref={ref}
            className={cn(
              'w-full appearance-none bg-[#1C1C1E]/80 text-[#F5F5F7] text-sm rounded-squircle px-3.5 py-2.5 pr-9 border border-white/10 backdrop-blur-md transition-all focus:outline-none focus:border-apple-accent/60 focus:ring-4 focus:ring-apple-accent/20 cursor-pointer disabled:opacity-50',
              error && 'border-red-500/80',
              className
            )}
            {...props}
          >
            {options.length > 0
              ? options.map((opt) => (
                  <option key={opt.value} value={opt.value} className="bg-[#1C1C1E] text-white">
                    {opt.label}
                  </option>
                ))
              : children}
          </select>
          <ChevronDown className="absolute right-3 w-4 h-4 text-apple-subtext pointer-events-none" />
        </div>
        {error && <p className="text-xs text-red-400 pl-1 mt-1 font-medium">{error}</p>}
      </div>
    )
  }
)

Select.displayName = 'Select'
