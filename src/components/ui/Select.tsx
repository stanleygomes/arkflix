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
          <label htmlFor={selectId} className="block text-xs font-medium text-netflix-gray">
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          <select
            id={selectId}
            ref={ref}
            className={cn(
              'w-full appearance-none bg-[#242424] text-white text-sm rounded px-3.5 py-2 pr-9 border border-white/20 transition-colors focus:outline-none focus:border-white focus:ring-1 focus:ring-white cursor-pointer disabled:opacity-50',
              error && 'border-netflix-red focus:border-netflix-red focus:ring-netflix-red',
              className
            )}
            {...props}
          >
            {options.length > 0
              ? options.map((opt) => (
                  <option key={opt.value} value={opt.value} className="bg-netflix-dark text-white">
                    {opt.label}
                  </option>
                ))
              : children}
          </select>
          <ChevronDown className="absolute right-3 w-4 h-4 text-netflix-gray pointer-events-none" />
        </div>
        {error && <p className="text-xs text-netflix-red mt-1">{error}</p>}
      </div>
    )
  }
)

Select.displayName = 'Select'
