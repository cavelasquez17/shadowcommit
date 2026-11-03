import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

interface Props {
  /** Stable id; the dismissed state is remembered under this id. */
  id: string
  title: string
  /** Already-translated tips. The caller owns the copy; this component only rotates it. */
  tips: string[]
  /** Milliseconds between tips. */
  intervalMs?: number
  /** First tip to show. Defaults to a random one so returning visitors do not always see the same. */
  startIndex?: number
}

const STORAGE_VERSION = 'v1'

function storageKey(id: string) {
  return `sc_banner_${id}_${STORAGE_VERSION}`
}

// Reading or writing storage can throw (private mode, blocked site data). The banner must still show.
function readDismissed(id: string): boolean {
  try {
    return localStorage.getItem(storageKey(id)) === 'dismissed'
  } catch {
    return false
  }
}

function writeDismissed(id: string) {
  try {
    localStorage.setItem(storageKey(id), 'dismissed')
  } catch {
    /* the banner just reappears next visit */
  }
}

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
    : false
}

/**
 * A side banner that explains what a section can do. It rotates through tips, pauses on hover or
 * focus, and never moves on its own for people who ask the system for less motion or when the tab
 * is hidden. Dismissing it is remembered per id.
 */
export function FeatureBanner({ id, title, tips, intervalMs = 7000, startIndex }: Props) {
  const { t } = useTranslation()
  const count = tips.length
  const [index, setIndex] = useState(() =>
    startIndex ?? (count > 0 ? Math.floor(Math.random() * count) : 0),
  )
  const [dismissed, setDismissed] = useState(() => readDismissed(id))
  const [paused, setPaused] = useState(false)
  const [reduced] = useState(prefersReducedMotion)
  const [playing, setPlaying] = useState(!reduced)

  const autoRotate = playing && !paused && !reduced && count > 1 && !dismissed

  useEffect(() => {
    if (!autoRotate) return
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'hidden') return
      setIndex((i) => (i + 1) % count)
    }, intervalMs)
    return () => window.clearInterval(timer)
  }, [autoRotate, count, intervalMs])

  if (dismissed || count === 0) return null

  const current = ((index % count) + count) % count
  const go = (delta: number) => setIndex((i) => (((i + delta) % count) + count) % count)
  const dismiss = () => {
    writeDismissed(id)
    setDismissed(true)
  }

  return (
    <aside
      aria-label={title}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setPaused(false)
      }}
      className="slab flex flex-col gap-3 p-4"
    >
      <div className="flex items-start justify-between gap-2">
        <h2 className="text-[10px] font-black tracking-widest text-[var(--color-toxic)] uppercase">{title}</h2>
        <button
          type="button"
          onClick={dismiss}
          aria-label={t('banner.dismiss')}
          className="text-xs text-zinc-500 hover:text-white"
        >
          ✕
        </button>
      </div>

      <p aria-live="off" className="min-h-[3.5rem] text-xs leading-relaxed text-zinc-400">
        {tips[current]}
      </p>

      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1" role="group" aria-label={t('banner.tips')}>
          <button type="button" onClick={() => go(-1)} aria-label={t('banner.previous')} className="px-1 text-zinc-500 hover:text-white">
            ‹
          </button>
          {count > 1 && !reduced && (
            <button
              type="button"
              onClick={() => setPlaying((p) => !p)}
              aria-label={playing ? t('banner.pause') : t('banner.play')}
              aria-pressed={!playing}
              className="px-1 text-[10px] text-zinc-500 hover:text-white"
            >
              {playing ? '❚❚' : '▶'}
            </button>
          )}
          <button type="button" onClick={() => go(1)} aria-label={t('banner.next')} className="px-1 text-zinc-500 hover:text-white">
            ›
          </button>
        </div>
        {count > 1 && (
          <div className="flex items-center gap-1">
            {tips.map((_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => setIndex(i)}
                aria-label={t('banner.tipNumber', { n: i + 1, total: count })}
                aria-current={i === current ? 'true' : undefined}
                className={`h-1.5 w-1.5 border border-[var(--color-edge)] ${i === current ? 'bg-[var(--color-toxic)]' : ''}`}
              />
            ))}
          </div>
        )}
      </div>
    </aside>
  )
}
