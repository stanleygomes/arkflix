import { useI18nStore } from '@/stores/i18nStore'

export function useTranslation() {
  const { t, language, setLanguage } = useI18nStore()

  return {
    t,
    language,
    setLanguage,
  }
}
