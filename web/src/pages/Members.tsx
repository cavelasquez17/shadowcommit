import { useCallback, useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api, type User } from '@/lib/api'
import { flagEmoji } from '@/lib/format'
import { useApp } from '@/store/app'
import { Avatar, BadgeChip, Chip, EmptyState, OptionChip, SectionTitle, Spinner } from '@/components/ui'
import { Select } from '@/pages/Feed'

export function Members() {
  const { t } = useTranslation()
  const { meta, seniority, cause } = useApp()
  const [params, setParams] = useSearchParams()

  const [users, setUsers] = useState<User[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)

  const query = {
    country: params.get('country') ?? '',
    seniority: params.get('seniority') ?? '',
    cause: params.get('cause') ?? '',
    employed: params.get('employed') ?? '',
    language: params.get('language') ?? '',
    sort: params.get('sort') ?? 'reputation',
    q: params.get('q') ?? '',
  }
  const queryKey = JSON.stringify(query)

  const load = useCallback(
    async (targetPage: number, append: boolean) => {
      setLoading(true)
      try {
        const res = await api.users({ ...JSON.parse(queryKey), page: targetPage, limit: 24 })
        setUsers((prev) => (append ? [...prev, ...res.items] : res.items))
        setTotal(res.total)
        setPage(targetPage)
      } finally {
        setLoading(false)
      }
    },
    [queryKey],
  )

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- network fetch, not derivable
    void load(1, false)
  }, [load])

  function setParam(key: string, value: string) {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  return (
    <div>
      <SectionTitle color="#00e5ff" sub={t('members.subtitle')}>
        {t('members.title')}
      </SectionTitle>

      <div className="slab mb-6 grid gap-3 border-[var(--color-cyber)] p-4 sm:grid-cols-2 lg:grid-cols-3">
        <Select
          value={query.country}
          onChange={(v) => setParam('country', v)}
          placeholder={t('feed.anyCountry')}
          options={(meta?.countries ?? []).map((c) => ({
            value: c.value,
            label: `${flagEmoji(c.value)} ${c.label}`,
          }))}
        />
        <Select
          value={query.seniority}
          onChange={(v) => setParam('seniority', v)}
          placeholder={t('feed.anySeniority')}
          options={(meta?.seniorities ?? []).map((s) => ({
            value: s.value,
            label: `${s.emoji} ${s.label}`,
          }))}
        />
        <Select
          value={query.cause}
          onChange={(v) => setParam('cause', v)}
          placeholder={t('feed.anyCause')}
          options={(meta?.causes_of_death ?? []).map((c) => ({
            value: c.value,
            label: `${c.emoji} ${c.label}`,
          }))}
        />
        <Select
          value={query.employed}
          onChange={(v) => setParam('employed', v)}
          placeholder={t('members.anyStatus')}
          options={[
            { value: 'true', label: `🫡 ${t('members.onlyEmployed')}` },
            { value: 'false', label: `🪦 ${t('members.onlyFired')}` },
          ]}
        />
        <Select
          value={query.language}
          onChange={(v) => setParam('language', v)}
          placeholder={t('feed.anyLanguage')}
          options={(meta?.languages ?? []).map((l) => ({
            value: l.value,
            label: `${l.emoji} ${l.label}`,
          }))}
        />
        <Select
          value={query.sort}
          onChange={(v) => setParam('sort', v)}
          options={[
            { value: 'reputation', label: t('members.sortRep') },
            { value: 'new', label: t('members.sortNew') },
            { value: 'fired', label: t('members.sortFired') },
            { value: 'loud', label: t('members.sortLoud') },
          ]}
        />
      </div>

      {loading && users.length === 0 ? (
        <Spinner label={t('common.loading')} />
      ) : users.length === 0 ? (
        <EmptyState emoji="🫥">{t('members.empty')}</EmptyState>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {users.map((u) => (
              <Link
                key={u.id}
                to={`/u/${u.username}`}
                className="slab slab-hover p-4"
                style={{ borderColor: u.is_employed ? '#39ff88' : '#ff3355' }}
              >
                <div className="flex items-start gap-3">
                  <Avatar user={u} size={44} />
                  <div className="min-w-0 flex-1">
                    <p className="flex items-center gap-1.5 truncate text-sm font-bold text-white">
                      {u.display_name || u.username}
                      {u.country_code && <span>{flagEmoji(u.country_code)}</span>}
                      {!u.is_employed && <span title={t('profile.unemployed')}>🪦</span>}
                    </p>
                    <p className="truncate text-[10px] text-zinc-500">
                      @{u.username} · {u.current_role || '—'}
                    </p>
                  </div>
                </div>

                {u.epitaph && (
                  <p className="mt-3 line-clamp-2 text-[11px] leading-relaxed text-zinc-500 italic">
                    “{u.epitaph}”
                  </p>
                )}

                <div className="mt-3 flex flex-wrap gap-1.5">
                  <OptionChip option={seniority(u.seniority)} />
                  <OptionChip option={cause(u.cause_of_death)} showSnark />
                  {u.years_exp > 0 && (
                    <Chip color="#67e8f9">{t('profile.years', { count: u.years_exp })}</Chip>
                  )}
                </div>

                {u.badges.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {u.badges.slice(0, 3).map((b) => (
                      <BadgeChip key={b.slug} badge={b} />
                    ))}
                  </div>
                )}

                <div className="mt-3 flex items-center gap-3 border-t border-[var(--color-edge)] pt-2 text-[10px] tracking-wider text-zinc-600 uppercase">
                  <span>💀 {u.reputation}</span>
                  <span>🧵 {u.thread_count}</span>
                  <span>💬 {u.post_count}</span>
                  {u.spoken_languages.length > 0 && (
                    <span className="ml-auto">{u.spoken_languages.length} 🗣</span>
                  )}
                </div>
              </Link>
            ))}
          </div>

          {users.length < total && (
            <div className="mt-8 flex justify-center">
              <button
                type="button"
                disabled={loading}
                onClick={() => void load(page + 1, true)}
                className="btn btn-ghost px-6 py-3"
              >
                {loading ? t('common.loading') : t('feed.loadMore')}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
