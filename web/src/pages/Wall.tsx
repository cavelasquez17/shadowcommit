import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api, type Stats, type User } from '@/lib/api'
import { compactNumber, flagEmoji } from '@/lib/format'
import { useApp } from '@/store/app'
import { Bar, EmptyState, SectionTitle, Spinner, StatBox } from '@/components/ui'

/**
 * Wall is the aggregate dashboard: how many of us, by which indignity, from where.
 * All charts are CSS bars — a charting library for four bar lists is not a trade worth making.
 */
export function Wall() {
  const { t, i18n } = useTranslation()
  const { cause, seniority, language, country } = useApp()
  const [stats, setStats] = useState<Stats | null>(null)
  const [fallen, setFallen] = useState<User[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    void Promise.all([
      api.stats(),
      api.users({ employed: 'false', sort: 'fired', limit: 12 }).catch(() => null),
    ])
      .then(([s, f]) => {
        setStats(s)
        setFallen(f?.items ?? [])
      })
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <Spinner label={t('common.loading')} />
  if (!stats) return <EmptyState emoji="📉">{t('wall.empty')}</EmptyState>

  const n = (v: number) => compactNumber(v, i18n.language)
  const rate = stats.total_users > 0 ? Math.round((stats.total_fired / stats.total_users) * 100) : 0
  const max = (rows: { count: number }[]) => Math.max(1, ...rows.map((r) => r.count))
  const withEpitaph = fallen.filter((u) => u.epitaph)

  return (
    <div className="space-y-9">
      <SectionTitle color="#ff3355" sub={t('wall.subtitle')}>
        {t('wall.title')}
      </SectionTitle>

      {/* The headline counter. This is the whole thesis of the site. */}
      <div className="slab border-[var(--color-blood)] p-6 text-center">
        <p className="text-[10px] tracking-[0.3em] text-zinc-500 uppercase">
          {t('wall.waysCounter')}
        </p>
        <p className="mt-2 text-6xl leading-none font-black tabular-nums text-[var(--color-blood)] flicker sm:text-7xl">
          {n(stats.ways_documented)}
        </p>
        <p className="mt-3 text-[10px] tracking-widest text-zinc-600 uppercase">
          / 1000 &nbsp;·&nbsp; {n(stats.total_posts)} corroborating statements
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatBox value={n(stats.total_users)} label={t('members.title')} color="#00e5ff" />
        <StatBox value={n(stats.total_fired)} label={t('landing.statsFallen')} color="#ff3355" />
        <StatBox value={`${rate}%`} label={t('wall.fatalityRate')} color="#ff8a1f" />
        <StatBox value={n(stats.total_reactions)} label={t('landing.statsReactions')} color="#ffe14d" />
      </div>
      <p className="-mt-6 text-[10px] tracking-wider text-zinc-600 uppercase">
        {t('wall.fatalityHint')}
      </p>

      <div className="grid gap-5 lg:grid-cols-2">
        <Panel title={t('wall.byCause')} color="#a855f7">
          {stats.by_cause.map((row) => {
            const opt = cause(row.key)
            return (
              <Bar
                key={row.key}
                label={opt?.label ?? row.key}
                emoji={opt?.emoji}
                count={row.count}
                max={max(stats.by_cause)}
                color={opt?.color ?? '#a855f7'}
                title={opt?.snark}
              />
            )
          })}
        </Panel>

        <Panel title={t('wall.byCountry')} color="#00e5ff">
          {stats.by_country.map((row) => (
            <Bar
              key={row.key}
              label={country(row.key)?.label ?? row.key}
              emoji={flagEmoji(row.key)}
              count={row.count}
              max={max(stats.by_country)}
              color="#00e5ff"
            />
          ))}
        </Panel>

        <Panel title={t('wall.bySeniority')} color="#ccff00">
          {stats.by_seniority.map((row) => {
            const opt = seniority(row.key)
            return (
              <Bar
                key={row.key}
                label={opt?.label ?? row.key}
                emoji={opt?.emoji}
                count={row.count}
                max={max(stats.by_seniority)}
                color={opt?.color ?? '#ccff00'}
                title={opt?.snark}
              />
            )
          })}
        </Panel>

        <Panel title={t('wall.byLanguage')} color="#ff5ecf">
          {stats.by_language.map((row) => (
            <Bar
              key={row.key}
              label={language(row.key)?.label ?? row.key}
              emoji={language(row.key)?.emoji}
              count={row.count}
              max={max(stats.by_language)}
              color="#ff5ecf"
            />
          ))}
        </Panel>
      </div>

      {withEpitaph.length > 0 && (
        <section>
          <h2 className="stamp mb-4 bg-[var(--color-butter)] text-black">{t('wall.topEpitaphs')}</h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {withEpitaph.map((u) => (
              <Link
                key={u.id}
                to={`/u/${u.username}`}
                className="slab slab-hover border-[var(--color-edge)] p-4 hover:border-[var(--color-butter)]"
              >
                <p className="text-[13px] leading-relaxed text-zinc-300 italic">“{u.epitaph}”</p>
                <p className="mt-3 flex items-center gap-1.5 text-[10px] tracking-wider text-zinc-600 uppercase">
                  {flagEmoji(u.country_code)} @{u.username} · {cause(u.cause_of_death)?.emoji}{' '}
                  {cause(u.cause_of_death)?.label}
                </p>
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function Panel({
  title,
  color,
  children,
}: {
  title: string
  color: string
  children: React.ReactNode
}) {
  return (
    <div className="slab p-4" style={{ borderColor: color }}>
      <h3
        className="mb-4 text-[10px] font-black tracking-[0.2em] uppercase"
        style={{ color }}
      >
        {title}
      </h3>
      <div className="space-y-2.5">{children}</div>
    </div>
  )
}
