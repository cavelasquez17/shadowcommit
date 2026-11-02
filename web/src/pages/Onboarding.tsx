import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, ArrowRight, Check } from 'lucide-react'
import { api, type ProfilePatch } from '@/lib/api'
import { flagEmoji } from '@/lib/format'
import { useApp } from '@/store/app'
import { Spinner } from '@/components/ui'

const TOTAL = 6

/**
 * Onboarding is the intake form new members see once. It writes straight into the profile
 * with a PATCH per completion, so abandoning halfway simply leaves the defaults in place.
 */
export function Onboarding() {
  const { t } = useTranslation()
  const { user, meta, refresh, ready } = useApp()
  const navigate = useNavigate()

  const [step, setStep] = useState(1)
  const [form, setForm] = useState<ProfilePatch>({})
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!user) return
    // eslint-disable-next-line react/set-state-in-effect -- hydrating from an async-loaded user
    setForm({
      country_code: user.country_code,
      seniority: user.seniority,
      years_exp: user.years_exp,
      cause_of_death: user.cause_of_death,
      epitaph: user.epitaph,
      current_role: user.current_role,
      is_employed: user.is_employed,
      spoken_languages: user.spoken_languages.length ? user.spoken_languages : ['en'],
    })
  }, [user])

  useEffect(() => {
    if (ready && !user) void navigate('/')
  }, [ready, user, navigate])

  if (!ready || !user || !meta) return <Spinner />

  const set = <K extends keyof ProfilePatch>(k: K, v: ProfilePatch[K]) =>
    setForm((f) => ({ ...f, [k]: v }))

  async function finish() {
    setBusy(true)
    try {
      await api.updateProfile(form)
      await refresh()
      void navigate('/feed')
    } finally {
      setBusy(false)
    }
  }

  const canAdvance = (() => {
    switch (step) {
      case 1:
        return !!form.country_code
      case 2:
        return !!form.seniority
      case 4:
        return !!form.cause_of_death
      case 6:
        return (form.spoken_languages ?? []).length > 0
      default:
        return true
    }
  })()

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-6">
        <h1 className="stamp bg-[var(--color-grape)] text-black">{t('onboarding.title')}</h1>
        <p className="mt-2 text-xs text-zinc-500">{t('onboarding.subtitle')}</p>
      </div>

      {/* Progress: hard blocks, not a rounded bar. */}
      <div className="mb-6 flex gap-1">
        {Array.from({ length: TOTAL }, (_, i) => (
          <div
            key={i}
            className="h-2 flex-1 border border-black"
            style={{ background: i < step ? '#a855f7' : 'var(--color-slab)' }}
          />
        ))}
      </div>
      <p className="mb-4 text-[10px] tracking-widest text-zinc-600 uppercase">
        {t('onboarding.step', { current: step, total: TOTAL })}
      </p>

      <div className="slab min-h-72 border-[var(--color-grape)] p-5">
        {step === 1 && (
          <Question title={t('onboarding.q1')} hint={t('onboarding.q1Hint')}>
            <div className="grid max-h-72 grid-cols-2 gap-1.5 overflow-y-auto sm:grid-cols-3">
              {meta.countries.map((c) => (
                <Pick
                  key={c.value}
                  active={form.country_code === c.value}
                  onClick={() => set('country_code', c.value)}
                  color="#00e5ff"
                >
                  {flagEmoji(c.value)} {c.label}
                </Pick>
              ))}
            </div>
          </Question>
        )}

        {step === 2 && (
          <Question title={t('onboarding.q2')} hint={t('onboarding.q2Hint')}>
            <div className="grid gap-1.5 sm:grid-cols-2">
              {meta.seniorities.map((s) => (
                <Pick
                  key={s.value}
                  active={form.seniority === s.value}
                  onClick={() => set('seniority', s.value)}
                  color={s.color ?? '#ccff00'}
                  sub={s.snark}
                >
                  {s.emoji} {s.label}
                </Pick>
              ))}
            </div>
            <input
              value={form.current_role ?? ''}
              onChange={(e) => set('current_role', e.target.value)}
              placeholder="Formerly: Senior Backend Engineer"
              className="field mt-3"
            />
          </Question>
        )}

        {step === 3 && (
          <Question title={t('onboarding.q3')} hint={t('onboarding.q3Hint')}>
            <div className="flex items-center gap-4">
              <input
                type="range"
                min={0}
                max={40}
                value={form.years_exp ?? 0}
                onChange={(e) => set('years_exp', Number(e.target.value))}
                className="flex-1 accent-[var(--color-toxic)]"
              />
              <span className="w-16 text-right text-3xl font-black tabular-nums text-[var(--color-toxic)]">
                {form.years_exp ?? 0}
              </span>
            </div>

            <div className="mt-6">
              <p className="mb-2 text-[10px] tracking-widest text-zinc-500 uppercase">
                {t('onboarding.employedQuestion')}
              </p>
              <div className="flex gap-2">
                <Pick
                  active={form.is_employed === true}
                  onClick={() => set('is_employed', true)}
                  color="#39ff88"
                >
                  🫡 {t('onboarding.employedYes')}
                </Pick>
                <Pick
                  active={form.is_employed === false}
                  onClick={() => set('is_employed', false)}
                  color="#ff3355"
                >
                  🪦 {t('onboarding.employedNo')}
                </Pick>
              </div>
              {form.is_employed === false && (
                <label className="mt-3 block">
                  <span className="mb-1 block text-[10px] tracking-widest text-zinc-500 uppercase">
                    {t('onboarding.firedWhen')}
                  </span>
                  <input
                    type="date"
                    value={form.fired_at ?? ''}
                    onChange={(e) => set('fired_at', e.target.value)}
                    className="field"
                  />
                </label>
              )}
            </div>
          </Question>
        )}

        {step === 4 && (
          <Question title={t('onboarding.q4')} hint={t('onboarding.q4Hint')}>
            <div className="grid gap-1.5">
              {meta.causes_of_death.map((c) => (
                <Pick
                  key={c.value}
                  active={form.cause_of_death === c.value}
                  onClick={() => set('cause_of_death', c.value)}
                  color={c.color ?? '#ff3355'}
                  sub={c.snark}
                >
                  {c.emoji} {c.label}
                </Pick>
              ))}
            </div>
          </Question>
        )}

        {step === 5 && (
          <Question title={t('onboarding.q5')} hint={t('onboarding.q5Hint')}>
            <input
              value={form.epitaph ?? ''}
              onChange={(e) => set('epitaph', e.target.value)}
              placeholder={t('onboarding.q5Placeholder')}
              maxLength={180}
              className="field text-base"
              autoFocus
            />
            <p className="mt-1 text-right text-[10px] text-zinc-600 tabular-nums">
              {(form.epitaph ?? '').length}/180
            </p>
            {form.epitaph && (
              <blockquote className="mt-5 border-l-4 border-[var(--color-blood)] bg-[var(--color-slab)] p-3">
                <p className="text-sm text-zinc-300 italic">“{form.epitaph}”</p>
                <cite className="text-[10px] tracking-widest text-zinc-600 uppercase not-italic">
                  — @{user.username}
                </cite>
              </blockquote>
            )}
          </Question>
        )}

        {step === 6 && (
          <Question title={t('onboarding.q6')} hint={t('onboarding.q6Hint')}>
            <div className="flex flex-wrap gap-1.5">
              {meta.languages.map((l) => {
                const on = (form.spoken_languages ?? []).includes(l.value)
                return (
                  <button
                    key={l.value}
                    type="button"
                    onClick={() => {
                      const cur = form.spoken_languages ?? []
                      set(
                        'spoken_languages',
                        on ? cur.filter((c) => c !== l.value) : [...cur, l.value],
                      )
                    }}
                    className="chip cursor-pointer px-2.5 py-1"
                    style={{
                      borderColor: on ? '#00e5ff' : 'var(--color-edge)',
                      background: on ? '#00e5ff' : 'transparent',
                      color: on ? '#000' : '#71717a',
                    }}
                  >
                    {l.emoji} {l.label}
                  </button>
                )
              })}
            </div>
          </Question>
        )}
      </div>

      <div className="mt-5 flex items-center justify-between gap-3">
        <button
          type="button"
          onClick={() => (step === 1 ? navigate('/feed') : setStep((s) => s - 1))}
          className="btn btn-ghost px-4 py-2.5"
        >
          <ArrowLeft size={13} strokeWidth={3} />
          {step === 1 ? t('onboarding.skip') : t('onboarding.back')}
        </button>

        {step < TOTAL ? (
          <button
            type="button"
            disabled={!canAdvance}
            onClick={() => setStep((s) => s + 1)}
            className="btn btn-primary px-5 py-2.5"
          >
            {t('onboarding.next')} <ArrowRight size={13} strokeWidth={3} />
          </button>
        ) : (
          <button
            type="button"
            disabled={busy || !canAdvance}
            onClick={() => void finish()}
            className="btn btn-primary px-5 py-2.5"
          >
            <Check size={14} strokeWidth={3} />
            {busy ? t('profile.saving') : t('onboarding.finish')}
          </button>
        )}
      </div>
    </div>
  )
}

function Question({
  title,
  hint,
  children,
}: {
  title: string
  hint: string
  children: React.ReactNode
}) {
  return (
    <div>
      <h2 className="text-xl font-black tracking-tight text-white">{title}</h2>
      <p className="mt-1 mb-4 text-[11px] text-zinc-500">{hint}</p>
      {children}
    </div>
  )
}

function Pick({
  active,
  onClick,
  color,
  children,
  sub,
}: {
  active: boolean
  onClick: () => void
  color: string
  children: React.ReactNode
  sub?: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="border-2 px-3 py-2 text-left text-xs font-bold transition-transform hover:-translate-y-0.5"
      style={{
        borderColor: active ? color : 'var(--color-edge)',
        background: active ? color : 'var(--color-slab)',
        color: active ? '#000' : '#d4d4d8',
      }}
    >
      <span className="block truncate">{children}</span>
      {sub && (
        <span
          className="mt-0.5 block truncate text-[10px] font-normal"
          style={{ color: active ? 'rgba(0,0,0,0.65)' : '#71717a' }}
        >
          {sub}
        </span>
      )}
    </button>
  )
}
