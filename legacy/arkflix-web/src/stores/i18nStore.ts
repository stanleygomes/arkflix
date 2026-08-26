import { create } from 'zustand'
import { ptBR, TranslationSchema } from '@/locales/pt-BR'

export type SupportedLanguage = 'pt-BR' | 'en-US'

// Detect language from browser
function getBrowserLanguage(): SupportedLanguage {
  if (typeof navigator !== 'undefined') {
    const lang = navigator.language || (navigator as any).userLanguage || ''
    if (lang.toLowerCase().startsWith('pt')) {
      return 'pt-BR'
    }
  }
  return 'pt-BR' // Default fallback
}

interface I18nState {
  language: SupportedLanguage
  t: TranslationSchema
  setLanguage: (lang: SupportedLanguage) => void
}

const translations: Record<SupportedLanguage, TranslationSchema> = {
  'pt-BR': ptBR,
  'en-US': ptBR, // Por enquanto fallback para ptBR conforme solicitado
}

const initialLang = (localStorage.getItem('arkflix_lang') as SupportedLanguage) || getBrowserLanguage()

export const useI18nStore = create<I18nState>((set) => ({
  language: initialLang,
  t: translations[initialLang] || ptBR,

  setLanguage: (lang: SupportedLanguage) => {
    localStorage.setItem('arkflix_lang', lang)
    set({
      language: lang,
      t: translations[lang] || ptBR,
    })
  },
}))
