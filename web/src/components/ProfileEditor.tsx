import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, ApiError, type ProfilePatch, type User } from '@/lib/api'
import { flagEmoji } from '@/lib/format'
import { useApp } from '@/store/app'
import { Select } from '@/pages/Feed'

/** ProfileEditor edits every declarable field in one form. Only changed keys are sent. */
export function ProfileEditor({
  user,
  onSaved,
}: {
  user: User
  onSaved: (u: User) => Promise<void> | void
}) {
  const { t } = useTranslation()
  const { meta } = useApp()
  const [form, setForm] = useState<ProfilePatch>({
    display_name: user.display_name,
    bio: user.bio,
    country_code: user.country_code,
    seniority: user.seniority,
    years_exp: user.years_exp,
    cause_of_death: user.cause_of_death,
    epitaph: user.epitaph,
    current_role: user.current_role,
    is_employed: user.is_employed,
    fired_at: user.fired_at ? user.fired_at.slice(0, 10) : '',
    website_url: user.website_url,
    linkedin_url: user.linkedin_url,
    spoken_languages: user.spoken_languages,
    copium_mode: user.copium_mode,
  })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const set = <K extends keyof ProfilePatch>(key: K, value: ProfilePatch[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  function toggleLanguage(code: string) {
    const current = form.spoken_languages ?? []
    set(
      'spoken_languages',
      current.includes(code) ? current.filter((c) => c !== code) : [...current, code],
    )
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const { user: updated } = await api.updateProfile(form)
      await onSaved(updated)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit} className="slab grid gap-4 border-[var(--color-cyber)] p-5 sm:grid-cols-2">
      <Field label={t('profile.edit')}>
        <input
          value={form.display_name ?? ''}
          onChange={(e) => set('display_name', e.target.value)}
          maxLength={80}
          className="field"
        />
      </Field>

      <Field label={t('onboarding.q2')}>
        <input
          value={form.current_role ?? ''}
          onChange={(e) => set('current_role', e.target.value)}
          maxLength={120}
          className="field"
        />
      </Field>

      <Select
        label={t('onboarding.q1')}
        value={form.country_code ?? ''}
        onChange={(v) => set('country_code', v)}
        placeholder="—"
        options={(meta?.countries ?? []).map((c) => ({
          value: c.value,
          label: `${flagEmoji(c.value)} ${c.label}`,
        }))}
      />

      <Select
        label={t('profile.seniority')}
        value={form.seniority ?? ''}
        onChange={(v) => set('seniority', v)}
        options={(meta?.seniorities ?? []).map((s) => ({
          value: s.value,
          label: `${s.emoji} ${s.label}`,
        }))}
      />

      <Field label={t('onboarding.q3')}>
        <input
          type="number"
          min={0}
          max={60}
          value={form.years_exp ?? 0}
          onChange={(e) => set('years_exp', Number(e.target.value))}
          className="field"
        />
      </Field>

      <Select
        label={t('profile.causeOfDeath')}
        value={form.cause_of_death ?? ''}
        onChange={(v) => set('cause_of_death', v)}
        options={(meta?.causes_of_death ?? []).map((c) => ({
          value: c.value,
          label: `${c.emoji} ${c.label}`,
        }))}
      />

      <Field label={t('profile.epitaph')} className="sm:col-span-2">
        <input
          value={form.epitaph ?? ''}
          onChange={(e) => set('epitaph', e.target.value)}
          placeholder={t('onboarding.q5Placeholder')}
          maxLength={180}
          className="field"
        />
      </Field>

      <Field label="Bio" className="sm:col-span-2">
        <textarea
          value={form.bio ?? ''}
          onChange={(e) => set('bio', e.target.value)}
          rows={3}
          maxLength={500}
          className="field resize-y"
        />
      </Field>

      <Field label="Website">
        <input
          value={form.website_url ?? ''}
          onChange={(e) => set('website_url', e.target.value)}
          placeholder="https://"
          className="field"
        />
      </Field>

      <Field label="LinkedIn">
        <input
          value={form.linkedin_url ?? ''}
          onChange={(e) => set('linkedin_url', e.target.value)}
          placeholder="https://linkedin.com/in/…"
          className="field"
        />
      </Field>

      <Field label={t('profile.speaks')} className="sm:col-span-2">
        <div className="flex flex-wrap gap-1.5">
          {(meta?.languages ?? []).map((l) => {
            const on = (form.spoken_languages ?? []).includes(l.value)
            return (
              <button
                key={l.value}
                type="button"
                onClick={() => toggleLanguage(l.value)}
                className="chip cursor-pointer"
                style={{
                  borderColor: on ? 'var(--color-cyber)' : 'var(--color-edge)',
                  background: on ? 'var(--color-cyber)' : 'transparent',
                  color: on ? 'var(--color-black)' : 'var(--color-zinc-500)',
                }}
              >
                {l.emoji} {l.label}
              </button>
            )
          })}
        </div>
      </Field>

      <Field label={t('profile.status')}>
        <div className="flex gap-2">
          {[true, false].map((employed) => (
            <button
              key={String(employed)}
              type="button"
              onClick={() => set('is_employed', employed)}
              className="btn flex-1 px-2 py-2"
              style={
                form.is_employed === employed
                  ? {
                      background: employed ? 'var(--color-slime)' : 'var(--color-blood)',
                      borderColor: 'var(--color-black)',
                      color: employed ? 'var(--color-black)' : 'var(--color-white)',
                    }
                  : { background: 'var(--color-slab)', borderColor: 'var(--color-edge)', color: 'var(--color-zinc-400)' }
              }
            >
              {employed ? `🫡 ${t('profile.employed')}` : `🪦 ${t('profile.unemployed')}`}
            </button>
          ))}
        </div>
      </Field>

      {!form.is_employed && (
        <Field label={t('onboarding.firedWhen')}>
          <input
            type="date"
            value={form.fired_at ?? ''}
            onChange={(e) => set('fired_at', e.target.value)}
            className="field"
          />
        </Field>
      )}

      <label className="flex cursor-pointer items-center gap-2 text-xs text-zinc-400 sm:col-span-2">
        <input
          type="checkbox"
          checked={form.copium_mode ?? false}
          onChange={(e) => set('copium_mode', e.target.checked)}
          className="h-4 w-4 accent-[var(--color-slime)]"
        />
        🫡 {t('tone.copium')} — it will be fine, actually
      </label>

      {error && (
        <p className="text-xs text-[var(--color-blood)] sm:col-span-2">{error}</p>
      )}

      <button type="submit" disabled={busy} className="btn btn-primary py-3 sm:col-span-2">
        {busy ? t('profile.saving') : t('profile.save')}
      </button>
    </form>
  )
}

function Field({
  label,
  children,
  className = '',
}: {
  label: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <label className={`block ${className}`}>
      <span className="mb-1 block text-[10px] tracking-widest text-zinc-500 uppercase">{label}</span>
      {children}
    </label>
  )
}
