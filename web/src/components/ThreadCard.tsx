import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Eye, MessageSquare, Pin } from 'lucide-react'
import { api, type Thread } from '@/lib/api'
import { applyCopium } from '@/lib/copium'
import { compactNumber, relativeTime } from '@/lib/format'
import { useApp } from '@/store/app'
import { Chip, OptionChip, UserChip } from '@/components/ui'
import { ReactionBar } from '@/components/ReactionBar'

/** Strips markdown noise so the card preview reads as prose, not as source. */
function excerpt(body: string, chars = 260): string {
  const flat = body
    .replace(/```[\s\S]*?```/g, ' [code] ')
    .replace(/[#>*`_]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
  return flat.length > chars ? `${flat.slice(0, chars).trimEnd()}…` : flat
}

export function ThreadCard({ thread }: { thread: Thread }) {
  const { t, i18n } = useTranslation()
  const { cause, seniority, language, copium } = useApp()
  const accent = thread.category?.color ?? 'var(--color-toxic)'
  const title = copium ? applyCopium(thread.title) : thread.title
  const body = copium ? applyCopium(excerpt(thread.body)) : excerpt(thread.body)

  return (
    <article className="slab slab-hover p-4" style={{ borderColor: accent }}>
      <header className="mb-3 flex flex-wrap items-start justify-between gap-3">
        <UserChip
          user={thread.author}
          sub={
            <>
              {thread.author?.current_role || t('common.anonymous')} ·{' '}
              {relativeTime(thread.created_at, i18n.language)}
            </>
          }
        />
        <div className="flex flex-wrap items-center justify-end gap-1.5">
          {thread.is_pinned && (
            <Chip color="var(--color-butter)">
              <Pin size={9} /> pinned
            </Chip>
          )}
          <OptionChip option={seniority(thread.author?.seniority ?? '')} />
          <OptionChip option={cause(thread.author?.cause_of_death ?? '')} showSnark />
        </div>
      </header>

      <Link to={`/t/${thread.id}`} className="group block">
        <h2 className="text-lg leading-tight font-black tracking-tight text-white group-hover:text-[var(--color-toxic)]">
          {title}
        </h2>
      </Link>

      <p className="mt-2 text-[12.5px] leading-relaxed text-zinc-500">{body}</p>
      <div className="mt-3 flex flex-wrap items-center gap-1.5">
        {thread.category && (
          <Link to={`/feed?category=${thread.category.slug}`}>
            <Chip color={accent}>
              {thread.category.emoji} {thread.category.name}
            </Chip>
          </Link>
        )}
        {thread.tags.map((tag) => (
          <Link key={tag.slug} to={`/feed?tag=${tag.slug}`}>
            <Chip color="var(--color-muted)">#{tag.label}</Chip>
          </Link>
        ))}
        <Chip color="var(--color-cyan-300)">
          {language(thread.language)?.emoji} {thread.language.toUpperCase()}
        </Chip>
        {thread.nsfw_corporate && <Chip color="var(--color-blood)">⚠ {t('thread.nsfw')}</Chip>}
        {thread.sarcasm >= 95 && (
          <Chip color="var(--color-bubble)" title={t('thread.sarcasm', { value: thread.sarcasm })}>
            🌡 {thread.sarcasm}%
          </Chip>
        )}
      </div>

      <footer className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t-2 border-[var(--color-edge)] pt-3">
        <ReactionBar
          reactions={thread.reactions}
          compact
          onReact={async (kind) => (await api.reactThread(thread.id, kind)).reactions}
        />
        <div className="flex items-center gap-3 text-[10px] tracking-wider text-zinc-600 uppercase">
          <Link
            to={`/t/${thread.id}`}
            className="flex items-center gap-1 hover:text-[var(--color-cyber)]"
          >
            <MessageSquare size={11} /> {compactNumber(thread.post_count, i18n.language)}
          </Link>
          <span className="flex items-center gap-1">
            <Eye size={11} /> {compactNumber(thread.view_count, i18n.language)}
          </span>
        </div>
      </footer>
    </article>
  )
}
