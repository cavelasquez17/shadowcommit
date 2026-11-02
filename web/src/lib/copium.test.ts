import { describe, expect, it } from 'vitest'
import { applyCopium, COPIUM_REACTIONS } from './copium'

// Mirrors reactionOptions in api/internal/handlers/vocab.go. If the API gains or renames a
// reaction, update this list and the copium table together.
const API_REACTION_KINDS = ['f', 'relate', 'laugh', 'robot', 'copium', 'brutal', 'lawyer']

describe('COPIUM_REACTIONS', () => {
  it('actually overrides reactions the API sends (otherwise copium mode does nothing here)', () => {
    expect(Object.keys(COPIUM_REACTIONS).length).toBeGreaterThan(0)
    expect(Object.keys(COPIUM_REACTIONS).every((k) => API_REACTION_KINDS.includes(k))).toBe(true)
  })

  it('only overrides reaction kinds the API actually sends', () => {
    const unknown = Object.keys(COPIUM_REACTIONS).filter((k) => !API_REACTION_KINDS.includes(k))
    expect(unknown).toEqual([])
  })

  it('gives each override an emoji and a label', () => {
    for (const [kind, v] of Object.entries(COPIUM_REACTIONS)) {
      expect(v.emoji, kind).not.toBe('')
      expect(v.label, kind).not.toBe('')
    }
  })
})

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
