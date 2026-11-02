import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { TerminalSquare } from 'lucide-react'
import { GithubIcon } from '@/components/BrandIcons'
import { api, ApiError } from '@/lib/api'
import { useApp } from '@/store/app'

/**
 * DevLoginBox is the passwordless local entrance. The API only enables it when
 * DEV_LOGIN is on, so this component simply won't be rendered in production.
 */
export function DevLoginBox({
  showGithub = false,
  inviteCode,
}: {
  showGithub?: boolean
  inviteCode?: string
}) {
  const { t } = useTranslation()
  const { refresh, meta } = useApp()
  const navigate = useNavigate()
  const [handle, setHandle] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!handle.trim() || busy) return
    setBusy(true)
    setError(null)
    try {
      const { is_new } = await api.devLogin(handle.trim(), inviteCode)
      await refresh()
      void navigate(is_new ? '/onboarding' : '/feed')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="slab border-[var(--color-grape)] p-4">
      {showGithub && meta?.github_enabled && (
        <>
          <a href={api.githubLoginURL(inviteCode)} className="btn btn-primary w-full py-3">
            <GithubIcon size={15} /> {t('auth.githubButton')}
          </a>
          <p className="mt-2 text-[10px] leading-relaxed text-zinc-600">{t('auth.githubHint')}</p>
          <div className="my-4 h-px bg-[var(--color-edge)]" />
        </>
      )}

      <h3 className="flex items-center gap-2 text-[11px] font-black tracking-widest text-[var(--color-grape)] uppercase">
        <TerminalSquare size={13} strokeWidth={2.5} /> {t('auth.devTitle')}
      </h3>
      <p className="mt-1.5 text-[10px] leading-relaxed text-zinc-600">{t('auth.devHint')}</p>

      <form onSubmit={submit} className="mt-3 flex gap-2">
        <input
          value={handle}
          onChange={(e) => setHandle(e.target.value.replace(/[^a-zA-Z0-9_-]/g, ''))}
          placeholder={t('auth.devPlaceholder')}
          maxLength={39}
          className="field"
        />
        <button type="submit" disabled={busy || handle.trim().length < 2} className="btn btn-primary shrink-0">
          {busy ? '…' : t('auth.devButton')}
        </button>
      </form>

      {error && <p className="mt-2 text-[11px] text-[var(--color-blood)]">{error}</p>}
    </div>
  )
}
