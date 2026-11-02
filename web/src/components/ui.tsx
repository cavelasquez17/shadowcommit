import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { accentFor, flagEmoji, initials } from '@/lib/format'
import type { Badge as BadgeModel, Option, User } from '@/lib/api'

/** Avatar falls back to generated initials on a deterministic colour. */
export function Avatar({
  user,
  size = 36,
}: {
  user: Pick<User, 'username' | 'display_name' | 'avatar_url'> | null
  size?: number
}) {
  const name = user?.display_name || user?.username || '??'
  const color = accentFor(user?.username ?? 'anon')

  if (user?.avatar_url) {
    return (
      <img
        src={user.avatar_url}
        alt={name}
        width={size}
        height={size}
        loading="lazy"
        className="shrink-0 border-2 bg-black object-cover"
        style={{ width: size, height: size, borderColor: color }}
      />
    )
  }
  return (
    <div
      className="flex shrink-0 items-center justify-center border-2 font-black text-black"
      style={{
        width: size,
        height: size,
        borderColor: color,
        background: color,
        fontSize: size * 0.4,
      }}
    >
      {initials(name)}
    </div>
  )
}

/** CountryFlag renders the flag plus an accessible label. Unset country gets a pirate flag. */
export function CountryFlag({ code, title }: { code?: string; title?: string }) {
  return (
    <span title={title ?? code ?? 'Unknown'} aria-label={title ?? code} className="leading-none">
      {flagEmoji(code)}
    </span>
  )
}

/** Chip is the workhorse label: hard border, tinted background, uppercase. */
export function Chip({
  children,
  color = 'var(--color-muted)',
  title,
  className = '',
}: {
  children: ReactNode
  color?: string
  title?: string
  className?: string
}) {
  return (
    <span
      className={`chip ${className}`}
      title={title}
      style={{
        borderColor: color,
        color,
        background: `color-mix(in srgb, ${color} 12%, transparent)`,
      }}
    >
      {children}
    </span>
  )
}

/** OptionChip renders a server-provided enum option with its emoji, label and colour. */
export function OptionChip({
  option,
  fallback,
  showSnark = false,
}: {
  option?: Option
  fallback?: string
  showSnark?: boolean
}) {
  if (!option) {
    return fallback ? <Chip>{fallback}</Chip> : null
  }
  return (
    <Chip color={option.color ?? 'var(--color-muted)'} title={showSnark ? option.snark : option.label}>
      {option.emoji} {option.label}
    </Chip>
  )
}

export function BadgeChip({ badge }: { badge: BadgeModel }) {
  return (
    <span
      className="chip cursor-help"
      title={`${badge.label} — ${badge.description}`}
      style={{
        borderColor: badge.color,
        color: badge.color,
        background: `color-mix(in srgb, ${badge.color} 14%, transparent)`,
      }}
    >
      {badge.emoji} {badge.label}
    </span>
  )
}

/** UserChip is the inline byline used in feeds, threads and replies. */
export function UserChip({
  user,
  size = 28,
  showCountry = true,
  sub,
}: {
  user: User | null
  size?: number
  showCountry?: boolean
  sub?: ReactNode
}) {
  if (!user) return <span className="text-xs text-zinc-600">deleted account</span>
  return (
    <Link to={`/u/${user.username}`} className="group flex min-w-0 items-center gap-2">
      <Avatar user={user} size={size} />
      <span className="min-w-0">
        <span className="flex items-center gap-1.5">
          <span className="truncate text-xs font-bold text-zinc-100 group-hover:text-[var(--color-toxic)]">
            {user.display_name || user.username}
          </span>
          {showCountry && user.country_code && <CountryFlag code={user.country_code} />}
          {!user.is_employed && (
            <span title="No longer required by the business" className="text-[10px]">
              🪦
            </span>
          )}
        </span>
        {sub && <span className="block truncate text-[10px] text-zinc-500">{sub}</span>}
      </span>
    </Link>
  )
}

/** SectionTitle is the stamped heading used at the top of every page block. */
export function SectionTitle({
  children,
  color = 'var(--color-toxic)',
  sub,
  right,
}: {
  children: ReactNode
  color?: string
  sub?: ReactNode
  right?: ReactNode
}) {
  return (
    <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="stamp text-black" style={{ background: color }}>
          {children}
        </h1>
        {sub && <p className="mt-2 max-w-2xl text-xs text-zinc-500">{sub}</p>}
      </div>
      {right}
    </div>
  )
}

export function Spinner({ label }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-3 py-16 text-xs tracking-widest text-zinc-600 uppercase">
      <span className="inline-block h-3 w-3 animate-spin border-2 border-[var(--color-toxic)] border-t-transparent" />
      {label ?? 'loading'}
    </div>
  )
}

export function EmptyState({ children, emoji = '🕳️' }: { children: ReactNode; emoji?: string }) {
  return (
    <div className="slab flex flex-col items-center gap-3 border-dashed p-10 text-center">
      <span className="text-4xl grayscale">{emoji}</span>
      <p className="max-w-md text-xs text-zinc-500">{children}</p>
    </div>
  )
}

/** StatBox is the big loud number used on the landing page and the Wall. */
export function StatBox({
  value,
  label,
  color = 'var(--color-toxic)',
}: {
  value: string | number
  label: string
  color?: string
}) {
  return (
    <div className="slab slab-hover p-4" style={{ borderColor: color }}>
      <div className="font-black text-3xl leading-none tabular-nums" style={{ color }}>
        {value}
      </div>
      <div className="mt-1.5 text-[10px] leading-tight tracking-widest text-zinc-500 uppercase">
        {label}
      </div>
    </div>
  )
}

/** Bar is the horizontal distribution row used across the Wall of the Fallen. */
export function Bar({
  label,
  emoji,
  count,
  max,
  color = 'var(--color-toxic)',
  title,
}: {
  label: string
  emoji?: string
  count: number
  max: number
  color?: string
  title?: string
}) {
  const pct = max > 0 ? Math.max(3, Math.round((count / max) * 100)) : 0
  return (
    <div className="group" title={title}>
      <div className="mb-1 flex items-center justify-between gap-2 text-[11px]">
        <span className="truncate text-zinc-300">
          {emoji && <span className="mr-1">{emoji}</span>}
          {label}
        </span>
        <span className="font-bold tabular-nums" style={{ color }}>
          {count}
        </span>
      </div>
      <div className="h-3 border border-[var(--color-edge)] bg-black">
        <div
          className="h-full transition-all duration-500"
          style={{ width: `${pct}%`, background: color }}
        />
      </div>
    </div>
  )
}
