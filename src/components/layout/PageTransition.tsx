import React from 'react'
import { motion, AnimatePresence, Transition } from 'framer-motion'
import { useLocation } from 'react-router-dom'

// Apple iOS Smooth Spring Curve Constants
export const iosSpringTransition: Transition = {
  type: 'spring',
  stiffness: 320,
  damping: 30,
  mass: 0.8,
}

export const iosEaseTransition: Transition = {
  duration: 0.35,
  ease: [0.25, 1, 0.5, 1] as [number, number, number, number],
}

export const PageTransition: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation()

  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.div
        key={location.pathname}
        initial={{ opacity: 0, y: 8, scale: 0.995 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: -8, scale: 0.995 }}
        transition={iosEaseTransition}
        className="w-full flex-grow will-change-transform"
      >
        {children}
      </motion.div>
    </AnimatePresence>
  )
}
