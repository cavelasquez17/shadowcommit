import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { ReactionSummary } from '@/lib/api'
import { COPIUM_REACTIONS } from '@/lib/copium'
import { useApp } from '@/store/app'

/**
 * ReactionBar renders every reaction kind so the button row never reflows, and updates
 * optimistically because waiting 200ms to see your own 💀 land feels broken.
 */
export function ReactionBar({
  reactions,
  onReact,
  compact = false,
}: {
  reactions: ReactionSummary[]
  onReact: (kind: string) => Promise<ReactionSummary[]>
  compact?: boolean
}) {
  const { t } = useTranslation()
  const { user, meta, copium } = useApp()
  const [local, setLocal] = useState(reactions)
  const [pending, setPending] = useState<string | null>(null)

  async function handle(kind: string) {
    if (!user || pending) return
    // Optimistic toggle, reconciled with the server response.
    setLocal((prev) =>
      prev.map((r) =>
        r.kind === kind ? { ...r, mine: !r.mine, count: r.count + (r.mine ? -1 : 1) } : r,
      ),
    )
    setPending(kind)
    try {
      setLocal(await onReact(kind))
    } catch {
      setLocal(reactions)
    } finally {
      setPending(null)
    }
  }

  // Anonymous visitors only see the tallies that exist; there is nothing to toggle.
  const shown = user ? local : local.filter((r) => r.count > 0)
  if (shown.length === 0) {
    return <span className="text-[10px] tracking-wider text-zinc-700 uppercase">no reactions</span>
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {shown.map((r) => {
        const color = meta?.reactions.find((x) => x.value === r.kind)?.color ?? 'var(--color-muted)'
        return (
          <button
            key={r.kind}
            type="button"
            disabled={!user}
            // Without these the tally and the toggled state only exist as colours/emoji.
            aria-label={`${r.label}: ${r.count}`}
            aria-pressed={user ? r.mine : undefined}
            onClick={() => void handle(r.kind)}
            title={
              user
                ? `${r.label} — ${meta?.reactions.find((x) => x.value === r.kind)?.snark ?? ''}`
                : t('auth.required')
            }
            className={`inline-flex items-center gap-1 border-2 font-bold tabular-nums transition-transform
              ${compact ? 'px-1.5 py-0.5 text-[11px]' : 'px-2 py-1 text-xs'}
              ${user ? 'cursor-pointer hover:-translate-y-0.5' : 'cursor-default'}
              ${pending === r.kind ? 'opacity-50' : ''}`}
            style={{
              borderColor: r.mine ? color : 'var(--color-edge)',
              color: r.mine ? 'var(--color-black)' : 'var(--color-zinc-400)',
              background: r.mine ? color : 'transparent',
            }}
          >
            <span className={compact ? 'text-xs' : 'text-sm'}>
              {copium ? (COPIUM_REACTIONS[r.kind]?.emoji ?? r.emoji) : r.emoji}
            </span>
          </button>
        )
      })}
    </div>
  )
}
