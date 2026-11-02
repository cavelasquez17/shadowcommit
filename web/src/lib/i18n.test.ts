import { describe, expect, it } from 'vitest'
import en from '@/locales/en.json'
import es from '@/locales/es.json'
import pt from '@/locales/pt.json'
import i18n, { UI_LOCALES } from './i18n'

type Tree = { [key: string]: string | Tree }

/** Flattens nested locale JSON into dotted keys so leaves can be compared across languages. */
function flatten(tree: Tree, prefix = ''): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(tree)) {
    const path = prefix ? `${prefix}.${key}` : key
    if (typeof value === 'string') out[path] = value
    else Object.assign(out, flatten(value, path))
  }
  return out
}

const locales = { en: flatten(en as Tree), es: flatten(es as Tree), pt: flatten(pt as Tree) }
const placeholders = (s: string) => [...s.matchAll(/\{\{\s*(\w+)\s*\}\}/g)].map((m) => m[1]).sort()

describe('locale files', () => {
  it.each(['es', 'pt'] as const)('%s has exactly the same keys as en', (code) => {
    const base = Object.keys(locales.en).sort()
    const other = Object.keys(locales[code]).sort()
    expect(base.filter((k) => !(k in locales[code])), `missing in ${code}`).toEqual([])
    expect(other.filter((k) => !(k in locales.en)), `unexpected in ${code}`).toEqual([])
  })

  it.each(['es', 'pt'] as const)('%s keeps the same {{placeholders}} as en for every key', (code) => {
    const mismatched = Object.keys(locales.en)
      .filter((k) => k in locales[code])
      .filter((k) => placeholders(locales.en[k]).join() !== placeholders(locales[code][k]).join())
    expect(mismatched).toEqual([])
  })

  it.each(['en', 'es', 'pt'] as const)('%s has no empty strings', (code) => {
    const empty = Object.entries(locales[code]).filter(([, v]) => v.trim() === '').map(([k]) => k)
    expect(empty).toEqual([])
  })
})

describe('UI_LOCALES', () => {
  it('offers exactly the languages that have a locale file', () => {
    expect(UI_LOCALES.map((l) => l.code).sort()).toEqual(Object.keys(locales).sort())
  })
})

describe('i18n runtime', () => {
  const sample = Object.keys(locales.en)[0]

  it.each(UI_LOCALES.map((l) => l.code))('translates a real key in %s', async (code) => {
    await i18n.changeLanguage(code)
    expect(i18n.t(sample)).toBe(locales[code][sample])
    expect(i18n.t(sample)).not.toBe(sample)
  })

  it('maps regional variants such as es-AR and pt-BR to their base language', async () => {
    await i18n.changeLanguage('es-AR')
    expect(i18n.t(sample)).toBe(locales.es[sample])
    await i18n.changeLanguage('pt-BR')
    expect(i18n.t(sample)).toBe(locales.pt[sample])
  })

  it('falls back to English for unsupported languages', async () => {
    await i18n.changeLanguage('fr')
    expect(i18n.t(sample)).toBe(locales.en[sample])
  })

  it('interpolates variables without HTML-escaping them', async () => {
    const key = Object.keys(locales.en).find((k) => placeholders(locales.en[k]).length > 0)
    if (!key) return // nothing interpolated yet
    await i18n.changeLanguage('en')
    const [name] = placeholders(locales.en[key])
    expect(i18n.t(key, { [name]: 'A&B' })).toContain('A&B')
  })
})
