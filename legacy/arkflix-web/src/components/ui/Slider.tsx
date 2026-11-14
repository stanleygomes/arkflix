import React from 'react'
import { cn } from '@/lib/utils'

export interface SliderProps extends React.InputHTMLAttributes<HTMLInputElement> {
  value: number
  max?: number
  min?: number
  step?: number
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
}

export const Slider: React.FC<SliderProps> = ({
  className,
  value,
  max = 100,
  min = 0,
  step = 1,
  onChange,
  ...props
}) => {
  const percentage = ((value - min) / (max - min)) * 100

  return (
    <div className={cn('relative w-full flex items-center group py-2', className)}>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={onChange}
        className="w-full h-1.5 bg-white/20 accent-white rounded-full appearance-none cursor-pointer group-hover:h-2 transition-all focus:outline-none"
        style={{
          background: `linear-gradient(to right, #FFFFFF ${percentage}%, rgba(255, 255, 255, 0.15) ${percentage}%)`,
        }}
        {...props}
      />
    </div>
  )
}
