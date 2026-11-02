import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AtSign, Bell, CheckCheck, Heart, MessageSquare, UserPlus } from 'lucide-react'
import { api, type Notification } from '@/lib/api'
import { relativeTime } from '@/lib/format'
import { useApp } from '@/store/app'
import { Avatar, EmptyState, SectionTitle, Spinner } from '@/components/ui'

const KIND_STYLE: Record<string, { color: string; Icon: typeof Bell; key: string }> = {
  reply: { color: '#00e5ff', Icon: MessageSquare, key: 'notifications.kindReply' },
  mention: { color: '#a855f7', Icon: AtSign, key: 'notifications.kindMention' },
  reaction: { color: '#ff3355', Icon: Heart, key: 'notifications.kindReaction' },
  follow: { color: '#39ff88', Icon: UserPlus, key: 'notifications.kindFollow' },
}

export function Notifications() {
  const { t, i18n } = useTranslation()
  const { user, ready, setUnread } = useApp()
  const navigate = useNavigate()

  const [items, setItems] = useState<Notification[]>([])
  const [mentions, setMentions] = useState<
    { created_at: string; thread_id: string; thread_title: string; by_username: string; by_avatar: string }[]
  >([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (ready && !user) void navigate('/')
  }, [ready, user, navigate])

  useEffect(() => {
    if (!user) return
    void Promise.all([api.notifications(), api.mentions().catch(() => ({ items: [] }))])
      .then(([n, m]) => {
        setItems(n.items)
        setMentions(m.items)
      })
      .finally(() => setLoading(false))
  }, [user])

  async function markAll() {
    await api.markNotificationsRead()
    setItems((prev) => prev.map((n) => ({ ...n, is_read: true })))
    setUnread(0)
  }

  if (!ready || loading) return <Spinner label={t('common.loading')} />

  return (
    <div className="mx-auto max-w-3xl space-y-8">
      <SectionTitle
        color="#ffe14d"
        sub={t('notifications.subtitle')}
        right={
          items.some((n) => !n.is_read) && (
            <button type="button" onClick={() => void markAll()} className="btn btn-ghost px-3 py-1.5">
              <CheckCheck size={13} strokeWidth={2.5} /> {t('notifications.markRead')}
            </button>
          )
        }
      >
        {t('notifications.title')}
      </SectionTitle>

      {items.length === 0 ? (
        <EmptyState emoji="🔔">{t('notifications.empty')}</EmptyState>
      ) : (
        <div className="space-y-2">
          {items.map((n) => {
            const style = KIND_STYLE[n.kind] ?? { color: '#8b8ba7', Icon: Bell, key: 'notifications.title' }
            return (
              <Link
                key={n.id}
                to={n.link || '/feed'}
                className="slab slab-hover flex items-start gap-3 p-3"
                style={{
                  borderColor: n.is_read ? 'var(--color-edge)' : style.color,
                  opacity: n.is_read ? 0.6 : 1,
                }}
              >
                <span
                  className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center border-2 border-black"
                  style={{ background: style.color, color: '#000' }}
                >
                  <style.Icon size={13} strokeWidth={2.5} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-bold text-zinc-100">{n.title}</p>
                  {n.body && <p className="mt-0.5 line-clamp-2 text-[11px] text-zinc-500">{n.body}</p>}
                </div>
                <span className="shrink-0 text-[10px] tracking-wider text-zinc-600 uppercase">
                  {relativeTime(n.created_at, i18n.language)}
                </span>
              </Link>
            )
          })}
        </div>
      )}

      {mentions.length > 0 && (
        <section>
          <h2 className="stamp mb-4 bg-[var(--color-grape)] text-black">
            {t('notifications.mentions')}
          </h2>
          <div className="space-y-2">
            {mentions.map((m, i) => (
              <Link
                key={`${m.thread_id}-${i}`}
                to={`/t/${m.thread_id}`}
                className="slab slab-hover flex items-center gap-3 border-[var(--color-grape)] p-3"
              >
                <Avatar
                  user={{ username: m.by_username, display_name: m.by_username, avatar_url: m.by_avatar }}
                  size={28}
                />
                <div className="min-w-0 flex-1">
                  <p className="text-xs text-zinc-400">
                    <span className="font-bold text-[var(--color-grape)]">@{m.by_username}</span>{' '}
                    {t('notifications.mentions').toLowerCase()}
                  </p>
                  <p className="truncate text-[11px] text-zinc-500">{m.thread_title}</p>
                </div>
                <span className="shrink-0 text-[10px] tracking-wider text-zinc-600 uppercase">
                  {relativeTime(m.created_at, i18n.language)}
                </span>
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
