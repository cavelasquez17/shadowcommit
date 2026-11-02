import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

import en from '@/locales/en.json'
import es from '@/locales/es.json'
import pt from '@/locales/pt.json'

/**
 * UI_LOCALES are the languages the interface itself is translated into.
 * Forum *posts* accept far more languages (see api.meta().languages) — this list is only
 * about chrome, and it exists so the picker can't offer a locale that falls back to English.
 */
export const UI_LOCALES = [
  { code: 'en', label: 'English', flag: '🇬🇧' },
  { code: 'es', label: 'Español', flag: '🇪🇸' },
  { code: 'pt', label: 'Português', flag: '🇧🇷' },
] as const

export type UILocale = (typeof UI_LOCALES)[number]['code']

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      es: { translation: es },
      pt: { translation: pt },
    },
    fallbackLng: 'en',
    supportedLngs: UI_LOCALES.map((l) => l.code),
    // Treat "es-AR" and "pt-BR" as "es"/"pt" rather than falling back to English.
    load: 'languageOnly',
    nonExplicitSupportedLngs: true,
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'sc_lang',
      caches: ['localStorage'],
    },
  })

export default i18n
