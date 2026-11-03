import { describe, expect, it } from 'vitest'
import en from './en.json'
import es from './es.json'
import pt from './pt.json'

// Guide copy rules from issue #35: 3 to 5 tips per page, each one short enough to read in one glance.
const locales = { en, es, pt } as Record<string, { guides?: Record<string, Record<string, string>> }>

describe('page guide copy', () => {
  for (const [code, dict] of Object.entries(locales)) {
    for (const [page, guide] of Object.entries(dict.guides ?? {})) {
      it(`${code}.guides.${page} has a title and 3 to 5 short tips`, () => {
        expect(guide.title, 'title').toBeTruthy()
        const tips = Object.keys(guide).filter((k) => /^tip\d+$/.test(k))
        expect(tips.length).toBeGreaterThanOrEqual(3)
        expect(tips.length).toBeLessThanOrEqual(5)
        for (const k of tips) {
          expect(guide[k].length, `${code}.guides.${page}.${k}`).toBeLessThanOrEqual(140)
        }
      })
    }
  }
})
