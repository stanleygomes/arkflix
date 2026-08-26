import { useThemeStore, ThemeMode } from '@/stores/themeStore'

export function useTheme() {
  const { theme, effectiveTheme, setTheme } = useThemeStore()

  return {
    theme,
    effectiveTheme,
    setTheme: (newTheme: ThemeMode) => setTheme(newTheme),
    isDark: effectiveTheme === 'dark',
  }
}
