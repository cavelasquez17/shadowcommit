import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Briefcase, ExternalLink, MapPin, Sparkles, X, Zap } from 'lucide-react'
import { api, type JobPosting, type JobStats } from '@/lib/api'
import { compactNumber, flagEmoji, relativeTime } from '@/lib/format'
import { useApp } from '@/store/app'
import { Chip, EmptyState, SectionTitle, Spinner, StatBox } from '@/components/ui'

/**
 * Jobs is "Hire the Fallen" — a job board where companies pay to reach devs the machine
 * couldn't replace. Sponsored jobs float to the top. Anyone logged in can post; sponsorship
 * is toggled by admin after payment.
 */
export function Jobs() {
  const { t, i18n } = useTranslation()
  const { user, meta, seniority } = useApp()
  const [jobs, setJobs] = useState<JobPosting[]>([])
  const [stats, setStats] = useState<JobStats | null>(null)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [showPostForm, setShowPostForm] = useState(false)
  const [filterSeniority, setFilterSeniority] = useState('')
  const [filterRemote, setFilterRemote] = useState(false)

  const load = useCallback(async (targetPage: number) => {
    setLoading(true)
    try {
      const [jobRes, statRes] = await Promise.all([
        api.jobs({
          page: targetPage,
          limit: 20,
          seniority: filterSeniority || undefined,
          remote: filterRemote || undefined,
        }),
        api.jobStats(),
      ])
      setJobs(jobRes.items)
      setTotal(jobRes.total)
      setStats(statRes)
      setPage(targetPage)
    } catch {
      setJobs([])
    } finally {
      setLoading(false)
    }
  }, [filterSeniority, filterRemote])

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- network fetch, not derivable
    void load(1)
  }, [load])

  const hasMore = jobs.length < total
  const n = (v: number) => compactNumber(v, i18n.language)

  return (
    <div className="space-y-8">
      <SectionTitle color="var(--color-slime)" sub={t('jobs.subtitle')}>
        {t('jobs.title')}
      </SectionTitle>

      {/* Stats strip */}
      {stats && (
        <div className="grid grid-cols-3 gap-3">
          <StatBox value={n(stats.open)} label={t('jobs.open')} color="var(--color-slime)" />
          <StatBox value={n(stats.sponsored)} label={t('jobs.sponsored')} color="var(--color-butter)" />
          <StatBox value={n(total)} label={t('jobs.total')} color="var(--color-cyber)" />
        </div>
      )}

      {/* Filters + post button */}
      <div className="flex flex-wrap items-center gap-3">
        <select
          className="field max-w-40"
          value={filterSeniority}
          onChange={(e) => setFilterSeniority(e.target.value)}
        >
          <option value="">{t('jobs.allSeniorities')}</option>
          {meta?.seniorities.map((s) => (
            <option key={s.value} value={s.value}>{s.label}</option>
          ))}
        </select>
        <button
          className={`btn ${filterRemote ? 'btn-primary' : 'btn-ghost'}`}
          onClick={() => setFilterRemote(!filterRemote)}
        >
          <MapPin size={12} strokeWidth={3} /> {t('jobs.remoteOnly')}
        </button>
        {user && (
          <button
            className="btn btn-primary ml-auto"
            onClick={() => setShowPostForm(true)}
          >
            <Briefcase size={12} strokeWidth={3} /> {t('jobs.post')}
          </button>
        )}
      </div>

      {/* Job list */}
      {loading ? (
        <Spinner label={t('common.loading')} />
      ) : jobs.length === 0 ? (
        <EmptyState emoji="📭">{t('jobs.empty')}</EmptyState>
      ) : (
        <div className="space-y-4">
          {jobs.map((job) => (
            <JobCard key={job.id} job={job} seniorityLabel={seniority(job.seniority)?.label ?? job.seniority} />
          ))}
          {hasMore && (
            <button
              className="btn btn-ghost w-full"
              onClick={() => void load(page + 1)}
              disabled={loading}
            >
              {t('jobs.loadMore')} →
            </button>
          )}
        </div>
      )}

      {/* Post form modal */}
      {showPostForm && user && (
        <PostJobModal
          onClose={() => setShowPostForm(false)}
          onPosted={() => {
            setShowPostForm(false)
            void load(1)
          }}
        />
      )}
    </div>
  )
}

function JobCard({ job, seniorityLabel }: { job: JobPosting; seniorityLabel: string }) {
  const { t, i18n } = useTranslation()
  const salary = job.salary_min && job.salary_max
    ? `${compactNumber(job.salary_min, i18n.language)}–${compactNumber(job.salary_max, i18n.language)} ${job.currency}`
    : job.salary_min
      ? `${compactNumber(job.salary_min, i18n.language)}+ ${job.currency}`
      : t('jobs.salaryHidden')

  return (
    <div
      className={`slab slab-hover p-4 ${job.is_sponsored ? 'border-[var(--color-butter)]' : ''}`}
      style={job.is_sponsored ? { boxShadow: '4px 4px 0 0 color-mix(in srgb, var(--color-butter) 40%, transparent)' } : undefined}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {job.is_sponsored && (
              <span className="chip border-[var(--color-butter)] text-[var(--color-butter)]">
                <Sparkles size={10} /> {t('jobs.sponsored')}
              </span>
            )}
            <span className="text-xs font-bold tracking-wider text-zinc-500 uppercase">
              {job.company}
            </span>
          </div>
          <h3 className="mt-1 text-sm font-black tracking-tight text-white">
            {job.title}
          </h3>
        </div>
        <a
          href={job.apply_url}
          target="_blank"
          rel="noopener noreferrer"
          className="btn btn-primary shrink-0"
        >
          <ExternalLink size={12} strokeWidth={3} /> {t('jobs.apply')}
        </a>
      </div>

      <p className="mt-2 text-[12.5px] leading-relaxed text-zinc-400 line-clamp-3">
        {job.body}
      </p>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        {job.remote && <Chip color="var(--color-slime)">📡 {t('jobs.remote')}</Chip>}
        {job.location && <Chip color="var(--color-cyber)">{flagEmoji(job.country_code)} {job.location}</Chip>}
        <Chip color="var(--color-grape)">{seniorityLabel}</Chip>
        <Chip color="var(--color-butter)">{salary}</Chip>
        <span className="ml-auto text-[10px] text-zinc-600">
          {relativeTime(job.created_at, i18n.language)}
        </span>
      </div>
    </div>
  )
}

function PostJobModal({ onClose, onPosted }: { onClose: () => void; onPosted: () => void }) {
  const { t } = useTranslation()
  const { meta } = useApp()
  const [form, setForm] = useState(() => ({
    company: '',
    title: '',
    body: '',
    apply_url: '',
    location: '',
    remote: true,
    country_code: '',
    salary_min: '',
    salary_max: '',
    currency: 'USD',
    seniority: meta?.seniorities.find((s) => s.value === 'mid')?.value ?? meta?.seniorities[0]?.value ?? '',
    language: 'en',
  }))
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      await api.createJob({
        company: form.company,
        title: form.title,
        body: form.body,
        apply_url: form.apply_url,
        location: form.location || undefined,
        remote: form.remote,
        country_code: form.country_code || undefined,
        salary_min: form.salary_min ? Number(form.salary_min) : undefined,
        salary_max: form.salary_max ? Number(form.salary_max) : undefined,
        currency: form.currency,
        seniority: form.seniority,
        language: form.language,
      })
      onPosted()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  const set = (key: keyof typeof form, value: string | boolean) =>
    setForm((f) => ({ ...f, [key]: value }))

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4" onClick={onClose}>
      <div
        className="slab w-full max-w-2xl p-6 max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
        style={{ borderColor: 'var(--color-slime)' }}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-black tracking-wider uppercase text-[var(--color-slime)]">
            <Briefcase size={14} className="inline" /> {t('jobs.postTitle')}
          </h2>
          <button onClick={onClose} className="text-zinc-500 hover:text-white">
            <X size={18} />
          </button>
        </div>

        <form onSubmit={submit} className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fCompany')}</span>
              <input className="field mt-1" required maxLength={120}
                value={form.company} onChange={(e) => set('company', e.target.value)} />
            </label>
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fTitle')}</span>
              <input className="field mt-1" required maxLength={200}
                value={form.title} onChange={(e) => set('title', e.target.value)} />
            </label>
          </div>

          <label className="block">
            <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fBody')}</span>
            <textarea className="field mt-1 min-h-24" required maxLength={8000}
              value={form.body} onChange={(e) => set('body', e.target.value)} />
          </label>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fApplyURL')}</span>
              <input className="field mt-1" type="url" required
                placeholder="https://..."
                value={form.apply_url} onChange={(e) => set('apply_url', e.target.value)} />
            </label>
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fLocation')}</span>
              <input className="field mt-1" maxLength={120}
                value={form.location} onChange={(e) => set('location', e.target.value)} />
            </label>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fSalaryMin')}</span>
              <input className="field mt-1" type="number" min={0}
                value={form.salary_min} onChange={(e) => set('salary_min', e.target.value)} />
            </label>
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fSalaryMax')}</span>
              <input className="field mt-1" type="number" min={0}
                value={form.salary_max} onChange={(e) => set('salary_max', e.target.value)} />
            </label>
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fCurrency')}</span>
              <input className="field mt-1" maxLength={3}
                value={form.currency} onChange={(e) => set('currency', e.target.value.toUpperCase())} />
            </label>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fSeniority')}</span>
              <select className="field mt-1"
                value={form.seniority} onChange={(e) => set('seniority', e.target.value)}>
                {meta?.seniorities.map((s) => (
                  <option key={s.value} value={s.value}>{s.label}</option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="text-[10px] font-bold tracking-wider text-zinc-500 uppercase">{t('jobs.fLanguage')}</span>
              <select className="field mt-1"
                value={form.language} onChange={(e) => set('language', e.target.value)}>
                {meta?.languages.map((l) => (
                  <option key={l.value} value={l.value}>{l.label}</option>
                ))}
              </select>
            </label>
          </div>

          <label className="flex items-center gap-2">
            <input type="checkbox" checked={form.remote}
              onChange={(e) => set('remote', e.target.checked)}
              className="h-4 w-4 accent-[var(--color-slime)]" />
            <span className="text-xs text-zinc-300">{t('jobs.fRemote')}</span>
          </label>

          {error && (
            <div className="border-2 border-[var(--color-blood)] bg-[color-mix(in_srgb,var(--color-blood)_12%,transparent)] p-2 text-xs text-[var(--color-blood)]">
              {error}
            </div>
          )}

          <div className="flex items-center gap-3 pt-2">
            <button type="submit" className="btn btn-primary" disabled={submitting}>
              <Zap size={12} strokeWidth={3} /> {t('jobs.fSubmit')}
            </button>
            <p className="text-[10px] text-zinc-600">
              {t('jobs.fSponsorNote')}
            </p>
          </div>
        </form>
      </div>
    </div>
  )
}
