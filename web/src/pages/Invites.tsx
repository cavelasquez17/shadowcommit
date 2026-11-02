import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Copy, Ticket } from 'lucide-react'
import { api, ApiError, type Invite } from '@/lib/api'
import { relativeTime } from '@/lib/format'
import { useApp } from '@/store/app'
import { Avatar, Chip, EmptyState, SectionTitle, Spinner } from '@/components/ui'
import { DevLoginBox } from '@/components/DevLoginBox'

export function Invites() {
  const { t, i18n } = useTranslation()
  const { user, ready } = useApp()
  const navigate = useNavigate()

  const [invites, setInvites] = useState<Invite[]>([])
  const [baseURL, setBaseURL] = useState('')
  const [note, setNote] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    if (ready && !user) void navigate('/')
  }, [ready, user, navigate])

  useEffect(() => {
    if (!user) return
    void api
      .invites()
      .then((r) => {
        setInvites(r.items)
        setBaseURL(r.base_url)
      })
      .finally(() => setLoading(false))
  }, [user])

  async function mint() {
    setBusy(true)
    setError(null)
    try {
      const { invite } = await api.createInvite(note)
      setInvites((prev) => [invite, ...prev])
      setNote('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  function copy(code: string) {
    void navigator.clipboard.writeText(`${baseURL}${code}`)
    setCopied(code)
    setTimeout(() => setCopied(null), 1800)
  }

  function statusOf(inv: Invite) {
    if (inv.used_at) return { label: t('invites.used'), color: '#39ff88' }
    if (new Date(inv.expires_at) < new Date()) return { label: t('invites.expired'), color: '#71717a' }
    return { label: t('invites.unused'), color: '#ccff00' }
  }

  if (!ready || loading) return <Spinner label={t('common.loading')} />

  return (
    <div className="mx-auto max-w-3xl">
      <SectionTitle color="#a855f7" sub={t('invites.subtitle')}>
        {t('invites.title')}
      </SectionTitle>

      <div className="slab mb-6 border-[var(--color-grape)] p-4">
        <div className="flex flex-col gap-2 sm:flex-row">
          <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={t('invites.notePlaceholder')}
            maxLength={200}
            className="field"
          />
          <button type="button" onClick={() => void mint()} disabled={busy} className="btn btn-primary shrink-0">
            <Ticket size={14} strokeWidth={2.5} /> {busy ? '…' : t('invites.create')}
          </button>
        </div>
        {error && <p className="mt-2 text-[11px] text-[var(--color-blood)]">{error}</p>}
      </div>

      <h2 className="stamp mb-4 bg-[var(--color-butter)] text-black">{t('invites.yourCodes')}</h2>

      {invites.length === 0 ? (
        <EmptyState emoji="🎟">{t('invites.empty')}</EmptyState>
      ) : (
        <div className="space-y-2">
          {invites.map((inv) => {
            const status = statusOf(inv)
            return (
              <div
                key={inv.code}
                className="slab flex flex-wrap items-center gap-3 p-3"
                style={{ borderColor: status.color }}
              >
                <code className="border-2 border-[var(--color-edge)] bg-black px-2 py-1 text-sm font-black tracking-[0.2em] text-[var(--color-toxic)]">
                  {inv.code}
                </code>
                <Chip color={status.color}>{status.label}</Chip>
                {inv.note && <span className="text-[11px] text-zinc-500 italic">“{inv.note}”</span>}
                <span className="ml-auto text-[10px] tracking-wider text-zinc-600 uppercase">
                  {relativeTime(inv.created_at, i18n.language)}
                </span>
                <button
                  type="button"
                  onClick={() => copy(inv.code)}
                  className="btn btn-ghost px-2 py-1"
                >
                  <Copy size={11} strokeWidth={2.5} />
                  {copied === inv.code ? t('invites.copied') : t('invites.copy')}
                </button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

/** Join is the landing page an invite link points at. */
export function Join() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const code = params.get('code') ?? ''
  const { user, meta } = useApp()
  const navigate = useNavigate()

  const [state, setState] = useState<Awaited<ReturnType<typeof api.checkInvite>> | null>(null)

  useEffect(() => {
    if (!code) return
    void api.checkInvite(code).then(setState).catch(() => null)
  }, [code])

  useEffect(() => {
    if (user) void navigate('/feed')
  }, [user, navigate])

  if (!code) return <EmptyState emoji="🎟">{t('invites.joinInvalid')}</EmptyState>
  if (!state) return <Spinner />

  if (!state.valid) {
    return (
      <div className="mx-auto max-w-lg">
        <EmptyState emoji="🚫">{state.reason ?? t('invites.joinInvalid')}</EmptyState>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5">
      <div className="slab border-[var(--color-toxic)] p-5 text-center">
        <span className="text-4xl">🎟</span>
        <h1 className="mt-3 text-2xl font-black tracking-tight text-white uppercase">
          {t('invites.joinTitle')}
        </h1>
        {state.inviter && (
          <div className="mt-4 flex items-center justify-center gap-2">
            <Avatar
              user={{
                username: state.inviter.username,
                display_name: state.inviter.display_name,
                avatar_url: state.inviter.avatar_url,
              }}
              size={32}
            />
            <p className="text-xs text-zinc-400">
              {t('invites.joinBy', { username: state.inviter.username })}
            </p>
          </div>
        )}
        {state.note && (
          <p className="mt-4 border-l-4 border-[var(--color-grape)] pl-3 text-left text-sm text-zinc-300 italic">
            “{state.note}”
          </p>
        )}
      </div>

      {meta?.dev_login && <DevLoginBox showGithub={meta?.github_enabled} inviteCode={code} />}
    </div>
  )
}
