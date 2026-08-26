import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface ModalProps {
  isOpen: boolean
  onClose: () => void
  children: React.ReactNode
  className?: string
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '4xl'
}

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  children,
  className,
  maxWidth = '4xl',
}) => {
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    if (isOpen) {
      document.body.style.overflow = 'hidden'
      window.addEventListener('keydown', handleKeyDown)
    }
    return () => {
      document.body.style.overflow = 'unset'
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [isOpen, onClose])

  const maxWidths = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '4xl': 'max-w-4xl',
  }

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.25, ease: 'easeOut' }}
          className="fixed inset-0 z-50 overflow-y-auto bg-black/60 flex justify-center items-start pt-12 pb-16 px-4 backdrop-blur-2xl"
          onClick={onClose}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.94, y: 24 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 16 }}
            transition={{
              type: 'spring',
              stiffness: 350,
              damping: 32,
              mass: 0.85,
            }}
            className={cn(
              'relative w-full bg-[#1C1C1E]/90 rounded-squircle-2xl overflow-hidden shadow-apple border border-white/15 text-white backdrop-blur-2xl will-change-transform',
              maxWidths[maxWidth],
              className
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <motion.button
              whileTap={{ scale: 0.9 }}
              whileHover={{ scale: 1.08 }}
              transition={{ type: 'spring', stiffness: 500, damping: 25 }}
              onClick={onClose}
              className="absolute top-4 right-4 z-20 w-8 h-8 rounded-full bg-white/15 text-white flex items-center justify-center hover:bg-white/25 transition-colors backdrop-blur-lg border border-white/10"
              aria-label="Fechar"
            >
              <X className="w-4 h-4" />
            </motion.button>

            {children}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
