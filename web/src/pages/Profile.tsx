import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, Globe, Plus, Star, Trash2 } from 'lucide-react'
import { GithubIcon, LinkedinIcon } from '@/components/BrandIcons'
import { api, type Thread, type User } from '@/lib/api'
import { accentFor, flagEmoji, relativeTime } from '@/lib/format'
import { useApp } from '@/store/app'
import {
  Avatar,
  BadgeChip,
  Chip,
  EmptyState,
  OptionChip,
  Spinner,
  StatBox,
} from '@/components/ui'
import { ThreadCard } from '@/components/ThreadCard'
import { ProfileEditor } from '@/components/ProfileEditor'

export function Profile() {
  const { username = '' } = useParams()
  const { t, i18n } = useTranslation()
  const { user: me, seniority, cause, country, language, refresh } = useApp()

  const [profile, setProfile] = useState<User | null>(null)
  const [threads, setThreads] = useState<Thread[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [repoForm, setRepoForm] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.user(username)
      setProfile(res.user)
      setThreads(res.threads)
    } catch {
      setProfile(null)
    } finally {
      setLoading(false)
    }
  }, [username])

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- network fetch, not derivable
    void load()
  }, [load])

  if (loading) return <Spinner label={t('common.loading')} />
  if (!profile) return <EmptyState emoji="👻">No such user.</EmptyState>

  const isMe = me?.username === profile.username
  const accent = accentFor(profile.username)
  const causeOpt = cause(profile.cause_of_death)

  async function toggleFollow() {
    const { following } = await api.follow(profile!.username)
    setProfile((p) =>
      p
        ? { ...p, is_following: following, follower_count: p.follower_count + (following ? 1 : -1) }
        : p,
    )
  }

  async function removeRepo(id: string) {
    await api.deleteRepo(id)
    setProfile((p) => (p ? { ...p, repos: p.repos.filter((r) => r.id !== id) } : p))
  }

  return (
    <div className="space-y-8">
      {/* Header slab: the tombstone. */}
      <header className="slab p-5" style={{ borderColor: accent }}>
        <div className="flex flex-wrap items-start gap-5">
          <Avatar user={profile} size={96} />

          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-2xl font-black tracking-tight text-white">
                {profile.display_name || profile.username}
              </h1>
              {profile.country_code && (
                <span className="text-xl" title={country(profile.country_code)?.label}>
                  {flagEmoji(profile.country_code)}
                </span>
              )}
              {!profile.is_employed && <Chip color="var(--color-blood)">🪦 {t('profile.unemployed')}</Chip>}
              {profile.is_employed && <Chip color="var(--color-slime)">🫡 {t('profile.employed')}</Chip>}
            </div>

            <p className="mt-1 text-xs text-zinc-500">
              @{profile.username} · {profile.current_role || '—'} ·{' '}
              {t('profile.joined', { when: relativeTime(profile.created_at, i18n.language) })}
            </p>

            <div className="mt-3 flex flex-wrap items-center gap-1.5">
              <OptionChip option={seniority(profile.seniority)} showSnark />
              <OptionChip option={causeOpt} showSnark />
              <Chip color="var(--color-cyan-300)">
                {t('profile.years', { count: profile.years_exp })} {t('profile.experience')}
              </Chip>
              {profile.days_since_fired !== null && (
                <Chip color="var(--color-tangerine)">
                  ⏱ {t('profile.daysSince', { count: profile.days_since_fired })}
                </Chip>
              )}
            </div>

            {profile.epitaph && (
              <blockquote className="mt-4 border-l-4 border-[var(--color-blood)] pl-3">
                <p className="text-sm text-zinc-300 italic">“{profile.epitaph}”</p>
                <cite className="text-[10px] tracking-widest text-zinc-600 uppercase not-italic">
                  — {t('profile.epitaph')}
                </cite>
              </blockquote>
            )}

            {profile.bio && (
              <p className="mt-3 max-w-2xl text-[12.5px] leading-relaxed text-zinc-400">
                {profile.bio}
              </p>
            )}

            <div className="mt-4 flex flex-wrap items-center gap-3">
              {profile.github_url && (
                <a
                  href={profile.github_url}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-[var(--color-toxic)]"
                >
                  <GithubIcon size={12} /> github
                </a>
              )}
              {profile.website_url && (
                <a
                  href={profile.website_url}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-[var(--color-cyber)]"
                >
                  <Globe size={12} /> site
                </a>
              )}
              {profile.linkedin_url && (
                <a
                  href={profile.linkedin_url}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-[var(--color-cyber)]"
                >
                  <LinkedinIcon size={12} /> linkedin
                </a>
              )}
              {profile.spoken_languages.length > 0 && (
                <span className="flex items-center gap-1 text-[11px] text-zinc-500">
                  {t('profile.speaks')}:{' '}
                  {profile.spoken_languages.map((l) => language(l)?.emoji ?? l).join(' ')}
                </span>
              )}
            </div>
          </div>

          <div className="flex flex-col gap-2">
            {isMe ? (
              <button type="button" onClick={() => setEditing((v) => !v)} className="btn btn-primary">
                {editing ? t('common.close') : t('profile.edit')}
              </button>
            ) : (
              me && (
                <button
                  type="button"
                  onClick={() => void toggleFollow()}
                  className={profile.is_following ? 'btn btn-ghost' : 'btn btn-primary'}
                >
                  {profile.is_following ? (
                    <>
                      <EyeOff size={13} strokeWidth={2.5} /> {t('profile.unfollow')}
                    </>
                  ) : (
                    <>
                      <Eye size={13} strokeWidth={2.5} /> {t('profile.follow')}
                    </>
                  )}
                </button>
              )
            )}
          </div>
        </div>

        {profile.badges.length > 0 && (
          <div className="mt-5 border-t-2 border-[var(--color-edge)] pt-4">
            <p className="mb-2 text-[10px] tracking-widest text-zinc-600 uppercase">
              {t('profile.badges')}
            </p>
            <div className="flex flex-wrap gap-1.5">
              {profile.badges.map((b) => (
                <BadgeChip key={b.slug} badge={b} />
              ))}
            </div>
          </div>
        )}
      </header>

      {editing && isMe && (
        <ProfileEditor
          user={profile}
          onSaved={async (updated) => {
            setProfile(updated)
            setEditing(false)
            await refresh()
          }}
        />
      )}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatBox value={profile.reputation} label={t('profile.reputation')} color="var(--color-blood)" />
        <StatBox value={profile.thread_count} label={t('profile.threads')} color="var(--color-toxic)" />
        <StatBox value={profile.post_count} label={t('profile.posts')} color="var(--color-cyber)" />
        <StatBox value={profile.follower_count} label={t('profile.followers')} color="var(--color-grape)" />
      </div>

      <section>
        <div className="mb-4 flex items-end justify-between gap-3">
          <h2 className="stamp bg-[var(--color-slime)] text-black">{t('profile.repos')}</h2>
          {isMe && (
            <button
              type="button"
              onClick={() => setRepoForm((v) => !v)}
              className="btn btn-ghost px-3 py-1.5"
            >
              <Plus size={12} strokeWidth={3} /> {t('profile.addRepo')}
            </button>
          )}
        </div>

        {repoForm && isMe && (
          <RepoForm
            onAdded={(repo) => {
              setProfile((p) => (p ? { ...p, repos: [...p.repos, repo] } : p))
              setRepoForm(false)
            }}
          />
        )}

        {profile.repos.length === 0 ? (
          <EmptyState emoji="📦">{t('profile.reposEmpty')}</EmptyState>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {profile.repos.map((repo) => (
              <div key={repo.id} className="slab slab-hover border-[var(--color-slime)] p-3">
                <div className="flex items-start justify-between gap-2">
                  <a
                    href={repo.url}
                    target="_blank"
                    rel="noreferrer"
                    className="truncate text-sm font-bold text-[var(--color-slime)] hover:underline"
                  >
                    {repo.name}
                  </a>
                  {isMe && (
                    <button
                      type="button"
                      onClick={() => void removeRepo(repo.id)}
                      className="shrink-0 text-zinc-600 hover:text-[var(--color-blood)]"
                    >
                      <Trash2 size={12} />
                    </button>
                  )}
                </div>
                {repo.description && (
                  <p className="mt-1.5 text-[11px] leading-relaxed text-zinc-500">
                    {repo.description}
                  </p>
                )}
                <div className="mt-3 flex flex-wrap items-center gap-1.5">
                  {repo.language && <Chip color="var(--color-cyan-300)">{repo.language}</Chip>}
                  {repo.stars > 0 && (
                    <Chip color="var(--color-butter)">
                      <Star size={9} /> {repo.stars}
                    </Chip>
                  )}
                  {repo.written_by_ai && <Chip color="var(--color-grape)">🤖 {t('profile.writtenByAI')}</Chip>}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="stamp mb-4 bg-[var(--color-tangerine)] text-black">{t('profile.threads')}</h2>
        {threads.length === 0 ? (
          <EmptyState emoji="🤐">{t('profile.noThreads')}</EmptyState>
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            {threads.map((th) => (
              <ThreadCard key={th.id} thread={th} />
            ))}
          </div>
        )}
      </section>

      {isMe && (
        <div className="flex justify-center">
          <Link to="/invites" className="btn btn-ghost px-5 py-2.5">
            🎟 {t('nav.invites')}
          </Link>
        </div>
      )}
    </div>
  )
}

function RepoForm({ onAdded }: { onAdded: (repo: import('@/lib/api').Repo) => void }) {
  const { t } = useTranslation()
  const [form, setForm] = useState({ name: '', url: '', description: '', language: '' })
  const [ai, setAi] = useState(false)
  const [busy, setBusy] = useState(false)

  return (
    <form
      onSubmit={async (e) => {
        e.preventDefault()
        setBusy(true)
        try {
          const { repo } = await api.addRepo({ ...form, written_by_ai: ai })
          onAdded(repo)
        } finally {
          setBusy(false)
        }
      }}
      className="slab mb-4 grid gap-3 border-[var(--color-slime)] p-4 sm:grid-cols-2"
    >
      <input
        required
        value={form.name}
        onChange={(e) => setForm({ ...form, name: e.target.value })}
        placeholder={t('profile.repoName')}
        className="field"
      />
      <input
        required
        type="url"
        value={form.url}
        onChange={(e) => setForm({ ...form, url: e.target.value })}
        placeholder="https://github.com/…"
        className="field"
      />
      <input
        value={form.description}
        onChange={(e) => setForm({ ...form, description: e.target.value })}
        placeholder={t('profile.repoDesc')}
        className="field sm:col-span-2"
      />
      <input
        value={form.language}
        onChange={(e) => setForm({ ...form, language: e.target.value })}
        placeholder={t('profile.repoLang')}
        className="field"
      />
      <label className="flex cursor-pointer items-center gap-2 text-xs text-zinc-400">
        <input
          type="checkbox"
          checked={ai}
          onChange={(e) => setAi(e.target.checked)}
          className="h-4 w-4 accent-[var(--color-grape)]"
        />
        🤖 {t('profile.writtenByAI')}
      </label>
      <button type="submit" disabled={busy} className="btn btn-primary sm:col-span-2">
        {busy ? t('profile.saving') : t('profile.save')}
      </button>
    </form>
  )
}
