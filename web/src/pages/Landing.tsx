import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight, Briefcase, Coffee, Skull, Sparkles, TriangleAlert } from 'lucide-react'
import { GithubIcon } from '@/components/BrandIcons'
import { api, type Stats, type Thread } from '@/lib/api'
import { compactNumber } from '@/lib/format'
import { useApp } from '@/store/app'
import { StatBox } from '@/components/ui'
import { ThreadCard } from '@/components/ThreadCard'
import { DevLoginBox } from '@/components/DevLoginBox'

export function Landing() {
  const { t, i18n } = useTranslation()
  const { user, meta } = useApp()
  const [stats, setStats] = useState<Stats | null>(null)
  const [threads, setThreads] = useState<Thread[]>([])

  useEffect(() => {
    void api.stats().then(setStats).catch(() => null)
    void api
      .threads({ sort: 'hot', limit: 3 })
      .then((r) => setThreads(r.items))
      .catch(() => null)
  }, [])

  const n = (v: number | undefined) => compactNumber(v ?? 0, i18n.language)

  return (
    <div className="space-y-14">
      <section className="grid gap-8 lg:grid-cols-[1.35fr_1fr]">
        <div>
          <span className="chip border-[var(--color-grape)] text-[var(--color-grape)]">
            {t('landing.kicker')}
          </span>

          <h1 className="mt-4 text-5xl leading-[0.92] font-black tracking-tighter uppercase sm:text-6xl">
            <span className="text-white">{t('landing.headline')}</span>
            <br />
            <span
              className="mt-2 inline-block border-4 border-black bg-[var(--color-toxic)] px-3 text-black"
              style={{ boxShadow: '6px 6px 0 0 rgba(0,0,0,0.8)' }}
            >
              {t('landing.headline2')}
            </span>
          </h1>

          <p className="mt-6 max-w-xl text-sm leading-relaxed text-zinc-400">{t('landing.lede')}</p>

          <div className="mt-7 flex flex-wrap items-center gap-3">
            {user ? (
              <Link to="/feed" className="btn btn-primary px-5 py-3">
                {t('nav.feed')} <ArrowRight size={14} strokeWidth={3} />
              </Link>
            ) : (
              <a href={api.githubLoginURL()} className="btn btn-primary px-5 py-3">
                <GithubIcon size={15} /> {t('landing.ctaPrimary')}
              </a>
            )}
            <Link to="/wall" className="btn btn-ghost px-5 py-3">
              <Skull size={14} strokeWidth={2.5} /> {t('landing.ctaSecondary')}
            </Link>
          </div>

          <p className="mt-5 text-[10px] tracking-wider text-zinc-600 uppercase">
            {t('landing.disclaimer')}
          </p>
        </div>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <StatBox value={n(stats?.ways_documented)} label={t('landing.statsWays')} color="var(--color-toxic)" />
            <StatBox value={n(stats?.total_fired)} label={t('landing.statsFallen')} color="var(--color-blood)" />
            <StatBox
              value={n((stats?.total_users ?? 0) - (stats?.total_fired ?? 0))}
              label={t('landing.statsSurvivors')}
              color="var(--color-slime)"
            />
            <StatBox
              value={n(stats?.total_reactions)}
              label={t('landing.statsReactions')}
              color="var(--color-cyber)"
            />
          </div>
          {!user && meta?.dev_login && <DevLoginBox />}
        </div>
      </section>

      <section>
        <h2 className="stamp bg-[var(--color-cyber)] text-black">{t('landing.howTitle')}</h2>
        <div className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {(
            [
              ['how1', 'var(--color-blood)', '🪦'],
              ['how2', 'var(--color-grape)', '🌍'],
              ['how3', 'var(--color-butter)', '💀'],
              ['how4', 'var(--color-slime)', '📦'],
            ] as const
          ).map(([key, color, emoji]) => (
            <div key={key} className="slab slab-hover p-4" style={{ borderColor: color }}>
              <div className="text-2xl">{emoji}</div>
              <h3 className="mt-2 text-xs font-black tracking-wider uppercase" style={{ color }}>
                {t(`landing.${key}Title`)}
              </h3>
              <p className="mt-2 text-[11.5px] leading-relaxed text-zinc-500">
                {t(`landing.${key}Body`)}
              </p>
            </div>
          ))}
        </div>
      </section>

      {threads.length > 0 && (
        <section>
          <div className="mb-5 flex items-end justify-between gap-3">
            <h2 className="stamp bg-[var(--color-tangerine)] text-black">{t('feed.sortHot')}</h2>
            <Link
              to="/feed"
              className="text-[10px] font-bold tracking-widest text-zinc-500 uppercase hover:text-[var(--color-toxic)]"
            >
              {t('feed.loadMore')} →
            </Link>
          </div>
          <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
            {threads.map((th) => (
              <ThreadCard key={th.id} thread={th} />
            ))}
          </div>
        </section>
      )}

      {/* Risk calculator CTA — viral traffic driver */}
      <section>
        <div className="slab p-6 text-center" style={{ borderColor: 'var(--color-blood)', boxShadow: '6px 6px 0 0 color-mix(in srgb, var(--color-blood) 30%, transparent)' }}>
          <div className="text-4xl">🫠</div>
          <h2 className="mt-3 text-2xl font-black tracking-tight text-white uppercase">
            {t('risk.title')}
          </h2>
          <p className="mx-auto mt-2 max-w-md text-sm text-zinc-400">
            {t('risk.subtitle')}
          </p>
          <Link to="/risk" className="btn btn-danger mt-4 px-5 py-3">
            <TriangleAlert size={14} strokeWidth={3} /> {t('risk.calculate')}
          </Link>
        </div>
      </section>

      {/* Monetization: support the server */}
      <section>
        <h2 className="stamp bg-[var(--color-slime)] text-black">{t('support.title')}</h2>
        <p className="mt-3 max-w-2xl text-sm text-zinc-500">{t('support.subtitle')}</p>
        <div className="mt-5 grid gap-4 sm:grid-cols-3">
          <div className="slab slab-hover p-4" style={{ borderColor: 'var(--color-toxic)' }}>
            <Coffee size={20} className="text-[var(--color-toxic)]" />
            <h3 className="mt-2 text-xs font-black tracking-wider uppercase text-[var(--color-toxic)]">
              {t('support.donate')}
            </h3>
            <p className="mt-2 text-[11.5px] leading-relaxed text-zinc-500">
              {t('support.donateHint')}
            </p>
          </div>
          <div className="slab slab-hover p-4" style={{ borderColor: 'var(--color-grape)' }}>
            <Sparkles size={20} className="text-[var(--color-grape)]" />
            <h3 className="mt-2 text-xs font-black tracking-wider uppercase text-[var(--color-grape)]">
              {t('support.sponsor')}
            </h3>
            <p className="mt-2 text-[11.5px] leading-relaxed text-zinc-500">
              {t('support.sponsorHint')}
            </p>
          </div>
          <Link to="/jobs" className="slab slab-hover p-4" style={{ borderColor: 'var(--color-slime)' }}>
            <Briefcase size={20} className="text-[var(--color-slime)]" />
            <h3 className="mt-2 text-xs font-black tracking-wider uppercase text-[var(--color-slime)]">
              {t('support.hireBoard')}
            </h3>
            <p className="mt-2 text-[11.5px] leading-relaxed text-zinc-500">
              {t('support.hireHint')}
            </p>
          </Link>
        </div>
        <p className="mt-5 text-[10px] tracking-wider text-zinc-600 uppercase">
          {t('support.footer')}
        </p>
      </section>
    </div>
  )
}
