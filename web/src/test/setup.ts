import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach } from 'vitest'
import i18n from '@/lib/i18n'

// Vitest runs without globals, so Testing Library cannot register its own cleanup.
afterEach(() => cleanup())

// Pin the UI language so assertions on visible text do not depend on the host machine.
beforeEach(async () => {
  await i18n.changeLanguage('en')
})
