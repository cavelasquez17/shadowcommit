import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { accentFor, compactNumber, flagEmoji, initials, relativeTime, renderMarkdown } from './format'

describe('flagEmoji', () => {
  it('turns an ISO code into its regional-indicator flag', () => {
    expect(flagEmoji('CO')).toBe('🇨🇴')
    expect(flagEmoji('us')).toBe('🇺🇸') // case-insensitive
  })

  it('falls back to a pirate flag for missing or malformed codes', () => {
    expect(flagEmoji()).toBe('🏴')
    expect(flagEmoji('')).toBe('🏴')
    expect(flagEmoji('USA')).toBe('🏴')
  })
})

describe('accentFor', () => {
  it('is deterministic for the same seed', () => {
    expect(accentFor('carlos')).toBe(accentFor('carlos'))
  })

  it('always returns a colour from the palette, even for odd seeds', () => {
    for (const seed of ['', 'a', 'ñandú', '🔥', 'x'.repeat(500)]) {
      expect(accentFor(seed)).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })

  it('spreads different seeds over more than one colour', () => {
    const colours = new Set(['alice', 'bob', 'carol', 'dave', 'erin', 'frank'].map(accentFor))
    expect(colours.size).toBeGreaterThan(1)
  })
})

describe('relativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-11-02T12:00:00Z'))
  })
  afterEach(() => vi.useRealTimers())

  const ago = (ms: number) => new Date(Date.now() - ms).toISOString()

  it('describes the past in the requested locale', () => {
    expect(relativeTime(ago(3 * 3600_000), 'en')).toBe('3 hours ago')
    expect(relativeTime(ago(2 * 86400_000), 'en')).toBe('2 days ago')
    expect(relativeTime(ago(3 * 3600_000), 'es')).toBe('hace 3 horas')
  })

  it('uses seconds for very recent times and handles the future', () => {
    expect(relativeTime(ago(10_000), 'en')).toBe('10 seconds ago')
    expect(relativeTime(new Date(Date.now() + 2 * 3600_000).toISOString(), 'en')).toBe('in 2 hours')
  })

  it('picks the largest fitting unit', () => {
    expect(relativeTime(ago(400 * 86400_000), 'en')).toBe('last year')
    expect(relativeTime(ago(45 * 86400_000), 'en')).toBe('last month')
  })
})

describe('compactNumber', () => {
  it('abbreviates large numbers and leaves small ones alone', () => {
    expect(compactNumber(999, 'en')).toBe('999')
    expect(compactNumber(1500, 'en')).toBe('1.5K')
    expect(compactNumber(2_300_000, 'en')).toBe('2.3M')
  })
})

describe('initials', () => {
  it('takes the first letters of up to two words', () => {
    expect(initials('Carlos Velasquez')).toBe('CV')
    expect(initials('ada lovelace byron')).toBe('AL')
  })

  it('splits on dots, dashes and underscores used in handles', () => {
    expect(initials('mention-target')).toBe('MT')
    expect(initials('first.last')).toBe('FL')
    expect(initials('snake_case_name')).toBe('SC')
  })

  it('copes with empty and single-word names', () => {
    expect(initials('')).toBe('')
    expect(initials('   ')).toBe('')
    expect(initials('solo')).toBe('S')
  })
})

describe('renderMarkdown: formatting', () => {
  it('renders bold, italic and inline code', () => {
    const html = renderMarkdown('a **bold** and *em* and `code`')
    expect(html).toContain('<strong>bold</strong>')
    expect(html).toContain('<em>em</em>')
    expect(html).toContain('<code>code</code>')
  })

  it('renders headings, quotes and both list kinds', () => {
    expect(renderMarkdown('## Title')).toBe('<h2>Title</h2>')
    expect(renderMarkdown('> quoted')).toContain('<blockquote>')
    expect(renderMarkdown('- a\n- b')).toBe('<ul>\n<li>a</li>\n<li>b</li>\n</ul>')
    expect(renderMarkdown('1. a\n2. b')).toBe('<ol>\n<li>a</li>\n<li>b</li>\n</ol>')
  })

  it('keeps fenced code verbatim, without applying inline rules inside', () => {
    const html = renderMarkdown('```js\nconst x = **not bold**\n```')
    expect(html).toContain('<pre><code>const x = **not bold**</code></pre>')
    expect(html).not.toContain('<strong>')
  })

  it('links markdown links, bare URLs and github.com shortcuts', () => {
    expect(renderMarkdown('[site](https://example.com)')).toContain(
      '<a href="https://example.com" target="_blank" rel="noreferrer">site</a>',
    )
    expect(renderMarkdown('see https://example.com/x now')).toContain('href="https://example.com/x"')
    expect(renderMarkdown('github.com/foo/bar')).toContain('href="https://github.com/foo/bar"')
  })

  it('turns @mentions into profile links', () => {
    expect(renderMarkdown('hi @carlos-v!')).toContain('<a href="/u/carlos-v" class="mention">@carlos-v</a>')
  })
})

/** Parses the rendered HTML like a browser would and asserts nothing active survived. */
function expectInert(html: string) {
  const doc = new DOMParser().parseFromString(`<body>${html}</body>`, 'text/html')
  const allowed = new Set(['p', 'a', 'strong', 'em', 'code', 'pre', 'ul', 'ol', 'li', 'blockquote', 'h1', 'h2', 'h3'])
  for (const el of doc.body.querySelectorAll('*')) {
    expect(allowed.has(el.tagName.toLowerCase()), `unexpected <${el.tagName.toLowerCase()}>`).toBe(true)
    for (const attr of el.getAttributeNames()) {
      expect(attr.startsWith('on'), `event handler attribute ${attr}`).toBe(false)
      expect(['href', 'target', 'rel', 'class']).toContain(attr)
    }
  }
  for (const a of doc.body.querySelectorAll('a')) {
    expect(a.getAttribute('href')).toMatch(/^(https?:\/\/|\/u\/)/)
  }
}

describe('renderMarkdown: safety (the output goes into dangerouslySetInnerHTML)', () => {
  const hostile = [
    '<script>alert(1)</script>',
    '<img src=x onerror=alert(1)>',
    '[x](https://a.com/"onmouseover="alert(1))',
    'https://a.com/"onmouseover="alert(1)',
    '[click](javascript:alert(1))',
    '[click](JaVaScRiPt:alert(1))',
    '[click](data:text/html,<b>x</b>)',
    'javascript:alert(1)',
    '```\n<script>alert(1)</script>\n```',
    '`<img src=x onerror=alert(1)>`',
    '# <u onclick=alert(1)>h</u>',
    '> <iframe src=//evil></iframe>',
    '- <svg onload=alert(1)>',
    '@"><img src=x onerror=alert(1)>',
    '**<b onmouseover=alert(1)>x</b>**',
  ]

  it.each(hostile)('produces inert output for %j', (src) => {
    expectInert(renderMarkdown(src))
  })

  it('escapes raw HTML tags to visible text', () => {
    const html = renderMarkdown('<script>alert(1)</script>')
    expect(html).not.toContain('<script')
    expect(html).toContain('&lt;script&gt;')
  })

  it('does not create links for javascript: or data: URLs', () => {
    expect(renderMarkdown('[click](javascript:alert(1))')).not.toContain('<a ')
    expect(renderMarkdown('[click](data:text/html,<b>x</b>)')).not.toContain('<a ')
    expect(renderMarkdown('javascript:alert(1)')).not.toContain('<a ')
  })

  it('escapes HTML inside code fences and inline code too', () => {
    expect(renderMarkdown('```\n<b>x</b>\n```')).toContain('&lt;b&gt;x&lt;/b&gt;')
    expect(renderMarkdown('`<i>x</i>`')).toContain('<code>&lt;i&gt;x&lt;/i&gt;</code>')
  })

  it('does not let user text forge the internal fence marker', () => {
    const html = renderMarkdown('\u0000FENCE0\u0000 and \u0000FENCE7\u0000')
    expect(html).not.toContain('undefined')
    expect(html).not.toContain('\u0000')
  })

  it.each(['Tom & Jerry', 'a &lt; b', '&amp;', '&#60;script&#62;', 'x < y > z', 'say "hi" & \'bye\''])(
    'shows plain text exactly as typed, entities included: %j',
    (src) => {
      const doc = new DOMParser().parseFromString(`<body>${renderMarkdown(src)}</body>`, 'text/html')
      expect(doc.body.textContent).toBe(src)
    },
  )

  it('handles empty input', () => {
    expect(renderMarkdown('')).toBe('')
  })
})
