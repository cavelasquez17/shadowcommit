/** Presentation helpers: flags, colours, relative dates, and a tiny markdown renderer. */

/**
 * flagEmoji converts an ISO 3166-1 alpha-2 code into its regional-indicator emoji.
 * Deriving it beats shipping 250 emoji strings that a linter will mangle.
 */
export function flagEmoji(code?: string): string {
  if (!code || code.length !== 2) return '🏴'
  const base = 0x1f1e6
  return String.fromCodePoint(
    ...[...code.toUpperCase()].map((c) => base + (c.charCodeAt(0) - 65)),
  )
}

/** A deterministic accent colour per string, so every user/tag gets a stable identity. */
const PALETTE = [
  '#ccff00',
  '#ff3355',
  '#00e5ff',
  '#39ff88',
  '#a855f7',
  '#ff8a1f',
  '#ff5ecf',
  '#ffe14d',
]

export function accentFor(seed: string): string {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) | 0
  return PALETTE[Math.abs(h) % PALETTE.length]
}

export function relativeTime(iso: string, locale = 'en'): string {
  const then = new Date(iso).getTime()
  const secs = Math.round((then - Date.now()) / 1000)
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })

  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ['year', 31536000],
    ['month', 2592000],
    ['week', 604800],
    ['day', 86400],
    ['hour', 3600],
    ['minute', 60],
  ]
  for (const [unit, secondsInUnit] of units) {
    if (Math.abs(secs) >= secondsInUnit) return rtf.format(Math.round(secs / secondsInUnit), unit)
  }
  return rtf.format(secs, 'second')
}

export function compactNumber(n: number, locale = 'en'): string {
  return new Intl.NumberFormat(locale, { notation: 'compact', maximumFractionDigits: 1 }).format(n)
}

const HTML_ESCAPES: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
}

function escapeHTML(s: string): string {
  return s.replace(/[&<>"']/g, (c) => HTML_ESCAPES[c])
}

/**
 * renderMarkdown handles the subset of markdown people actually use in forum posts:
 * fenced code, inline code, bold, italic, blockquotes, lists, links and @mentions.
 * Everything is escaped first, so this is safe to feed into dangerouslySetInnerHTML.
 * A full markdown dependency is not worth 40kb for seven constructs.
 */
export function renderMarkdown(src: string): string {
  const fences: string[] = []
  // \u0000 is reserved as the fence placeholder below; drop any the user typed so they
  // cannot forge a placeholder that points at a fence that does not exist.
  // eslint-disable-next-line no-control-regex
  src = src.replace(/\u0000/g, '')
  // Pull fenced blocks out first so their contents are never touched by inline rules.
  let text = escapeHTML(src).replace(/```(\w*)\n?([\s\S]*?)```/g, (_m, _lang, code) => {
    fences.push(`<pre><code>${code.replace(/\n$/, '')}</code></pre>`)
    return `\u0000FENCE${fences.length - 1}\u0000`
  })

  const lines = text.split('\n')
  const out: string[] = []
  let listType: 'ul' | 'ol' | null = null
  let inQuote = false

  const closeList = () => {
    if (listType) {
      out.push(`</${listType}>`)
      listType = null
    }
  }
  const closeQuote = () => {
    if (inQuote) {
      out.push('</blockquote>')
      inQuote = false
    }
  }

  for (const raw of lines) {
    const line = raw.trimEnd()

    // oxlint(no-control-regex): \u0000 is a deliberate sentinel marking an already-escaped
    // fenced code block, not user input — see the fences.push() above.
    // eslint-disable-next-line no-control-regex
    if (/^\u0000FENCE\d+\u0000$/.test(line)) {
      closeList()
      closeQuote()
      out.push(line)
      continue
    }
    if (line === '') {
      closeList()
      closeQuote()
      continue
    }

    const heading = line.match(/^(#{1,3})\s+(.*)$/)
    if (heading) {
      closeList()
      closeQuote()
      const level = heading[1].length
      out.push(`<h${level}>${inline(heading[2])}</h${level}>`)
      continue
    }

    const quote = line.match(/^&gt;\s?(.*)$/)
    if (quote) {
      closeList()
      if (!inQuote) {
        out.push('<blockquote>')
        inQuote = true
      }
      out.push(`<p>${inline(quote[1])}</p>`)
      continue
    }
    closeQuote()

    const ol = line.match(/^\s*\d+\.\s+(.*)$/)
    if (ol) {
      if (listType !== 'ol') {
        closeList()
        out.push('<ol>')
        listType = 'ol'
      }
      out.push(`<li>${inline(ol[1])}</li>`)
      continue
    }

    const ul = line.match(/^\s*[-*]\s+(.*)$/)
    if (ul) {
      if (listType !== 'ul') {
        closeList()
        out.push('<ul>')
        listType = 'ul'
      }
      out.push(`<li>${inline(ul[1])}</li>`)
      continue
    }

    closeList()
    out.push(`<p>${inline(line)}</p>`)
  }
  closeList()
  closeQuote()

  text = out.join('\n')
  // eslint-disable-next-line no-control-regex -- see sentinel note above
  return text.replace(/\u0000FENCE(\d+)\u0000/g, (_m, i) => fences[Number(i)])
}

function inline(s: string): string {
  return (
    s
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
      .replace(/(^|[\s(])\*([^*\n]+)\*/g, '$1<em>$2</em>')
      .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
      // Bare URLs, but not ones already inside an href we just produced.
      .replace(/(^|[\s>])((?:https?:\/\/|github\.com\/)[^\s<]+)/g, (m, pre, url) =>
        m.includes('href') ? m : `${pre}<a href="${url.startsWith('http') ? url : `https://${url}`}" target="_blank" rel="noreferrer">${url}</a>`,
      )
      // @mentions link into the profile route.
      .replace(
        /(^|[\s(])@([A-Za-z0-9][A-Za-z0-9_-]{0,38})/g,
        '$1<a href="/u/$2" class="mention">@$2</a>',
      )
  )
}

/** initials is the avatar fallback when someone has no picture and no shame. */
export function initials(name: string): string {
  return name
    .split(/[\s._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}
