import { useEffect, useRef, useState } from 'react'
import { api } from '@/lib/api'
import { Avatar, CountryFlag } from '@/components/ui'

interface Suggestion {
  username: string
  display_name: string
  avatar_url: string
  country_code: string
  seniority: string
}

/**
 * MentionTextarea is a textarea with @handle autocomplete. It resolves the token under the
 * caret, queries the directory, and supports arrow-key selection — the interaction people
 * already expect from every other forum they have been fired from.
 */
export function MentionTextarea({
  value,
  onChange,
  placeholder,
  rows = 8,
  autoFocus = false,
  className = '',
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  rows?: number
  autoFocus?: boolean
  className?: string
}) {
  const ref = useRef<HTMLTextAreaElement>(null)
  const [query, setQuery] = useState<string | null>(null)
  const [items, setItems] = useState<Suggestion[]>([])
  const [active, setActive] = useState(0)

  // Debounced lookup: one request per 180ms of typing, not one per keystroke.
  useEffect(() => {
    if (query === null) {
      // eslint-disable-next-line react/set-state-in-effect -- clearing stale results, not derivable
      setItems([])
      return
    }
    const handle = setTimeout(() => {
      api
        .searchUsers(query)
        .then((r) => {
          setItems(r.items as Suggestion[])
          setActive(0)
        })
        .catch(() => setItems([]))
    }, 180)
    return () => clearTimeout(handle)
  }, [query])

  /** Returns the @token immediately before the caret, or null when there isn't one. */
  function tokenAtCaret(text: string, caret: number): { start: number; term: string } | null {
    const upToCaret = text.slice(0, caret)
    const match = upToCaret.match(/(?:^|\s)@([A-Za-z0-9_-]*)$/)
    if (!match) return null
    return { start: caret - match[1].length - 1, term: match[1] }
  }

  function handleChange(next: string) {
    onChange(next)
    const caret = ref.current?.selectionStart ?? next.length
    setQuery(tokenAtCaret(next, caret)?.term ?? null)
  }

  function insert(username: string) {
    const el = ref.current
    if (!el) return
    const caret = el.selectionStart
    const token = tokenAtCaret(value, caret)
    if (!token) return

    const next = `${value.slice(0, token.start)}@${username} ${value.slice(caret)}`
    onChange(next)
    setQuery(null)
    // Restore the caret after the inserted handle once React has flushed the new value.
    requestAnimationFrame(() => {
      const pos = token.start + username.length + 2
      el.focus()
      el.setSelectionRange(pos, pos)
    })
  }

  const open = query !== null && items.length > 0

  return (
    <div className="relative">
      <textarea
        ref={ref}
        rows={rows}
        autoFocus={autoFocus}
        value={value}
        placeholder={placeholder}
        onChange={(e) => handleChange(e.target.value)}
        onKeyDown={(e) => {
          if (!open) return
          if (e.key === 'ArrowDown') {
            e.preventDefault()
            setActive((i) => (i + 1) % items.length)
          } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setActive((i) => (i - 1 + items.length) % items.length)
          } else if (e.key === 'Enter' || e.key === 'Tab') {
            e.preventDefault()
            insert(items[active].username)
          } else if (e.key === 'Escape') {
            setQuery(null)
          }
        }}
        onBlur={() => setTimeout(() => setQuery(null), 120)}
        className={`field resize-y font-mono ${className}`}
      />

      {open && (
        <ul className="slab absolute z-30 mt-1 max-h-64 w-full max-w-sm overflow-y-auto border-[var(--color-grape)] p-1">
          {items.map((s, i) => (
            <li key={s.username}>
              <button
                type="button"
                onMouseDown={(e) => {
                  e.preventDefault()
                  insert(s.username)
                }}
                onMouseEnter={() => setActive(i)}
                className={`flex w-full items-center gap-2 px-2 py-1.5 text-left ${
                  i === active ? 'bg-[var(--color-grape)] text-black' : 'text-zinc-300'
                }`}
              >
                <Avatar user={s} size={22} />
                <span className="truncate text-xs font-bold">@{s.username}</span>
                {s.country_code && <CountryFlag code={s.country_code} />}
                <span className="ml-auto truncate text-[10px] opacity-70">{s.display_name}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
