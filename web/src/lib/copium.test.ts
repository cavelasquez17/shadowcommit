import { describe, expect, it } from 'vitest'
import { applyCopium } from './copium'

describe('applyCopium', () => {
  it('rewrites English layoff language into corporate euphemisms', () => {
    const out = applyCopium('I was fired and my team was laid off')
    expect(out).not.toMatch(/\bfired\b/i)
    expect(out).toContain('liberated')
  })

  it('is case-insensitive', () => {
    expect(applyCopium('I WAS FIRED')).not.toMatch(/fired/i)
  })

  it('leaves text without trigger words untouched', () => {
    expect(applyCopium('Had a nice walk in the park')).toBe('Had a nice walk in the park')
    expect(applyCopium('')).toBe('')
  })

  it('does not rewrite words that merely contain a trigger word', () => {
    expect(applyCopium('misfired')).toBe('misfired')
  })

  it('rewrites Spanish and Portuguese too', () => {
    expect(applyCopium('me demitiram ontem')).not.toContain('demitiram')
    expect(applyCopium('un hachazo enorme')).not.toContain('hachazo')
  })

  it.each([
    ['murió', 'murió'],
    ['mató', 'mató'],
  ])('rewrites Spanish words that end in an accented vowel: %s', (word, trigger) => {
    const out = applyCopium(`ayer ${word} la empresa`)
    expect(out).not.toContain(trigger)
  })

  it('applies every replacement in a longer text', () => {
    const out = applyCopium('Fired after a layoff. Total misery.')
    expect(out).not.toMatch(/\bmisery\b/i)
    expect(out).not.toMatch(/\bfired\b/i)
  })
})
