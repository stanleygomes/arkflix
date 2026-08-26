import { create } from 'zustand'

export type ThemeMode = 'dark' | 'light' | 'auto'

interface ThemeState {
  theme: ThemeMode
  effectiveTheme: 'dark' | 'light'
  setTheme: (theme: ThemeMode) => void
}

function getSystemTheme(): 'dark' | 'light' {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return 'dark'
}

function applyThemeToDocument(theme: ThemeMode) {
  const root = document.documentElement
  const effective = theme === 'auto' ? getSystemTheme() : theme

  if (effective === 'dark') {
    root.classList.add('dark')
    root.classList.remove('light')
  } else {
    root.classList.add('light')
    root.classList.remove('dark')
  }
  return effective
}

const savedTheme = (localStorage.getItem('arkflix_theme') as ThemeMode) || 'auto'
const initialEffective = applyThemeToDocument(savedTheme)

export const useThemeStore = create<ThemeState>((set) => {
  // Listen to system theme changes if 'auto'
  if (typeof window !== 'undefined' && window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      const current = useThemeStore.getState().theme
      if (current === 'auto') {
        const effective = applyThemeToDocument('auto')
        set({ effectiveTheme: effective })
      }
    })
  }

  return {
    theme: savedTheme,
    effectiveTheme: initialEffective,

    setTheme: (theme: ThemeMode) => {
      localStorage.setItem('arkflix_theme', theme)
      const effective = applyThemeToDocument(theme)
      set({ theme, effectiveTheme: effective })
    },
  }
})
