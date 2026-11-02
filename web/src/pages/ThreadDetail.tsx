import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Eye, Link2, Lock, Reply, Trash2 } from 'lucide-react'
import { api, type Post, type Thread } from '@/lib/api'
import { applyCopium } from '@/lib/copium'
import { relativeTime } from '@/lib/format'
import { useApp } from '@/store/app'
import { BadgeChip, Chip, EmptyState, OptionChip, Spinner, UserChip } from '@/components/ui'
import { ReactionBar } from '@/components/ReactionBar'
import { Markdown } from '@/components/Markdown'
import { MentionTextarea } from '@/components/MentionTextarea'

export function ThreadDetail() {
  const { id = '' } = useParams()
  const { t, i18n } = useTranslation()
  const { user, cause, seniority, language, copium } = useApp()
  const navigate = useNavigate()

  const [thread, setThread] = useState<Thread | null>(null)
  const [posts, setPosts] = useState<Post[]>([])
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)
  const [replyTo, setReplyTo] = useState<Post | null>(null)
  const [body, setBody] = useState('')
  const [posting, setPosting] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.thread(id)
      setThread(res.thread)
      setPosts(res.posts)
    } catch {
      setNotFound(true)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- network fetch, not derivable
    void load()
  }, [load])

  // Auto-dismiss the confirmation toasts.
  useEffect(() => {
    if (!toast) return
    const h = setTimeout(() => setToast(null), 2200)
    return () => clearTimeout(h)
  }, [toast])

  async function submitReply(e: React.FormEvent) {
    e.preventDefault()
    if (body.trim().length < 2 || posting) return
    setPosting(true)
    try {
      const { post } = await api.reply(id, {
        body: body.trim(),
        parent_id: replyTo?.id ?? null,
        language: i18n.language,
      })
      setPosts((prev) => [...prev, post])
      setThread((prev) => (prev ? { ...prev, post_count: prev.post_count + 1 } : prev))
      setBody('')
      setReplyTo(null)
    } finally {
      setPosting(false)
    }
  }

  async function removePost(post: Post) {
    if (!confirm(t('thread.deleteConfirm'))) return
    await api.deletePost(post.id)
    setPosts((prev) => prev.filter((p) => p.id !== post.id))
  }

  async function removeThread() {
    if (!thread || !confirm(t('thread.deleteConfirm'))) return
    await api.deleteThread(thread.id)
    void navigate('/feed')
  }

  if (loading) return <Spinner label={t('common.loading')} />
  if (notFound || !thread) return <EmptyState emoji="🪦">{t('thread.notFound')}</EmptyState>

  const accent = thread.category?.color ?? 'var(--color-toxic)'
  // Group replies under their parent so a reply chain reads as one visual block.
  const roots = posts.filter((p) => !p.parent_id)
  const childrenOf = (parentId: string) => posts.filter((p) => p.parent_id === parentId)

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-4 flex items-center justify-between gap-3">
        <Link
          to="/feed"
          className="flex items-center gap-1.5 text-[10px] font-bold tracking-widest text-zinc-500 uppercase hover:text-[var(--color-toxic)]"
        >
          <ArrowLeft size={12} strokeWidth={3} /> {t('nav.feed')}
        </Link>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => {
              void navigator.clipboard.writeText(window.location.href)
              setToast(t('thread.copied'))
            }}
            className="btn btn-ghost px-2 py-1"
          >
            <Link2 size={12} strokeWidth={2.5} /> {t('thread.share')}
          </button>
          {thread.can_edit && (
            <button type="button" onClick={() => void removeThread()} className="btn btn-danger px-2 py-1">
              <Trash2 size={12} strokeWidth={2.5} /> {t('thread.delete')}
            </button>
          )}
        </div>
      </div>

      {toast && (
        <div className="slab mb-4 border-[var(--color-slime)] px-3 py-2 text-xs text-[var(--color-slime)]">
          {toast}
        </div>
      )}

      <article className="slab p-5" style={{ borderColor: accent }}>
        <span className="stamp mb-4 bg-black" style={{ color: accent, borderColor: accent }}>
          {t('thread.originalPost')}
        </span>

        <h1 className="text-2xl leading-tight font-black tracking-tight text-white sm:text-3xl">
          {copium ? applyCopium(thread.title) : thread.title}
        </h1>

        <div className="mt-4 flex flex-wrap items-center gap-2">
          {thread.category && (
            <Link to={`/feed?category=${thread.category.slug}`}>
              <Chip color={accent}>
                {thread.category.emoji} {thread.category.name}
              </Chip>
            </Link>
          )}
          <Chip color="var(--color-cyan-300)">
            {language(thread.language)?.emoji} {thread.language.toUpperCase()}
          </Chip>
          <Chip color="var(--color-bubble)">🌡 {t('thread.sarcasm', { value: thread.sarcasm })}</Chip>
          {thread.nsfw_corporate && <Chip color="var(--color-blood)">⚠ {t('thread.nsfw')}</Chip>}
          {thread.tags.map((tag) => (
            <Link key={tag.slug} to={`/feed?tag=${tag.slug}`}>
              <Chip>#{tag.label}</Chip>
            </Link>
          ))}
        </div>

        <div className="my-5 flex flex-wrap items-center justify-between gap-3 border-y-2 border-[var(--color-edge)] py-3">
          <UserChip
            user={thread.author}
            size={40}
            sub={
              <>
                {thread.author?.current_role} · {relativeTime(thread.created_at, i18n.language)}
              </>
            }
          />
          <div className="flex flex-wrap items-center justify-end gap-1.5">
            <OptionChip option={seniority(thread.author?.seniority ?? '')} showSnark />
            <OptionChip option={cause(thread.author?.cause_of_death ?? '')} showSnark />
            {thread.author?.badges.slice(0, 2).map((b) => (
              <BadgeChip key={b.slug} badge={b} />
            ))}
          </div>
        </div>

        {thread.author?.epitaph && (
          <p className="mb-5 border-l-4 border-[var(--color-blood)] pl-3 text-xs text-zinc-500 italic">
            “{thread.author.epitaph}”
          </p>
        )}

        <Markdown source={thread.body} />

        <footer className="mt-6 flex flex-wrap items-center justify-between gap-3 border-t-2 border-[var(--color-edge)] pt-4">
          <ReactionBar
            reactions={thread.reactions}
            onReact={async (kind) => (await api.reactThread(thread.id, kind)).reactions}
          />
          <span className="flex items-center gap-1 text-[10px] tracking-wider text-zinc-600 uppercase">
            <Eye size={11} /> {t('thread.views', { count: thread.view_count })}
          </span>
        </footer>
      </article>

      <div className="mt-8">
        <h2 className="stamp bg-[var(--color-cyber)] text-black">
          {t('thread.replies', { count: thread.post_count })}
        </h2>

        <div className="mt-5 space-y-4">
          {roots.map((post) => (
            <div key={post.id} className="space-y-3">
              <PostCard post={post} onReply={setReplyTo} onDelete={removePost} />
              {childrenOf(post.id).length > 0 && (
                <div className="ml-4 space-y-3 border-l-2 border-[var(--color-edge)] pl-4 sm:ml-8">
                  {childrenOf(post.id).map((child) => (
                    <PostCard
                      key={child.id}
                      post={child}
                      onReply={setReplyTo}
                      onDelete={removePost}
                      nested
                    />
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      <div className="mt-8">
        {thread.is_locked ? (
          <div className="slab flex items-center gap-2 border-[var(--color-blood)] p-4 text-xs text-[var(--color-blood)]">
            <Lock size={14} strokeWidth={2.5} /> {t('thread.locked')}
          </div>
        ) : user ? (
          <form onSubmit={submitReply} className="slab border-[var(--color-toxic)] p-4">
            {replyTo && (
              <div className="mb-3 flex items-center justify-between gap-2 border-2 border-[var(--color-grape)] bg-[var(--color-slab)] px-3 py-1.5">
                <span className="text-[11px] text-[var(--color-grape)]">
                  {t('thread.replyingTo', { username: replyTo.author?.username ?? '?' })}
                </span>
                <button
                  type="button"
                  onClick={() => setReplyTo(null)}
                  className="text-[10px] tracking-wider text-zinc-500 uppercase hover:text-white"
                >
                  {t('thread.cancelReply')}
                </button>
              </div>
            )}
            <MentionTextarea
              value={body}
              onChange={setBody}
              rows={5}
              placeholder={t('thread.replyPlaceholder')}
            />
            <div className="mt-3 flex items-center justify-between gap-3">
              <span className="text-[10px] tracking-wider text-zinc-600 uppercase">
                {t('compose.mentionHint')} · markdown
              </span>
              <button
                type="submit"
                disabled={posting || body.trim().length < 2}
                className="btn btn-primary"
              >
                <Reply size={13} strokeWidth={2.5} />
                {posting ? t('compose.submitting') : t('thread.replyButton')}
              </button>
            </div>
          </form>
        ) : (
          <EmptyState emoji="🔒">{t('thread.loginToReply')}</EmptyState>
        )}
      </div>
    </div>
  )
}

function PostCard({
  post,
  onReply,
  onDelete,
  nested = false,
}: {
  post: Post
  onReply: (p: Post) => void
  onDelete: (p: Post) => Promise<void>
  nested?: boolean
}) {
  const { t, i18n } = useTranslation()
  const { user, cause, country } = useApp()
  const accent = nested ? 'var(--color-edge)' : 'var(--color-edge-light)'

  return (
    <article className="slab p-4" style={{ borderColor: accent }}>
      <header className="mb-3 flex flex-wrap items-start justify-between gap-2">
        <UserChip
          user={post.author}
          sub={
            <>
              {country(post.author?.country_code ?? '')?.label}
              {post.author?.country_code && ' · '}
              {relativeTime(post.created_at, i18n.language)}
            </>
          }
        />
        <div className="flex flex-wrap items-center justify-end gap-1.5">
          <OptionChip option={cause(post.author?.cause_of_death ?? '')} showSnark />
          {post.edited_by_ai && <Chip color="var(--color-cyber)">🤖 rewritten by a machine</Chip>}
          {post.language !== 'en' && <Chip>{post.language.toUpperCase()}</Chip>}
        </div>
      </header>

      <Markdown source={post.body} />

      <footer className="mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--color-edge)] pt-3">
        <ReactionBar
          reactions={post.reactions}
          compact
          onReact={async (kind) => (await api.reactPost(post.id, kind)).reactions}
        />
        <div className="flex items-center gap-3">
          {user && (
            <button
              type="button"
              onClick={() => onReply(post)}
              className="flex items-center gap-1 text-[10px] font-bold tracking-wider text-zinc-500 uppercase hover:text-[var(--color-toxic)]"
            >
              <Reply size={11} strokeWidth={3} /> {t('thread.replyButton')}
            </button>
          )}
          {post.can_edit && (
            <button
              type="button"
              onClick={() => void onDelete(post)}
              className="flex items-center gap-1 text-[10px] font-bold tracking-wider text-zinc-600 uppercase hover:text-[var(--color-blood)]"
            >
              <Trash2 size={11} strokeWidth={3} /> {t('thread.delete')}
            </button>
          )}
        </div>
      </footer>
    </article>
  )
}
