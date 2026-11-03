import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PenLine, SlidersHorizontal, X } from 'lucide-react'
import { api, type Category, type Tag, type Thread } from '@/lib/api'
import { flagEmoji } from '@/lib/format'
import { useApp } from '@/store/app'
import { Chip, EmptyState, SectionTitle, Spinner } from '@/components/ui'
import { ThreadCard } from '@/components/ThreadCard'
import { FeatureBanner } from '@/components/FeatureBanner'

const SORTS = ['hot', 'new', 'top', 'active'] as const
const FILTER_KEYS = ['category', 'tag', 'country', 'seniority', 'cause', 'language', 'q'] as const

export function Feed() {
  const { t } = useTranslation()
  const { user, meta } = useApp()
  const [params, setParams] = useSearchParams()

  const [threads, setThreads] = useState<Thread[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [categories, setCategories] = useState<Category[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [showFilters, setShowFilters] = useState(false)

  const sort = params.get('sort') ?? 'hot'
  // Serialised so the effect below re-runs on any filter change without deep-comparing objects.
  const filterKey = FILTER_KEYS.map((k) => `${k}=${params.get(k) ?? ''}`).join('&')

  const filters = useMemo(() => {
    const out: Record<string, string> = {}
    for (const k of FILTER_KEYS) {
      const v = params.get(k)
      if (v) out[k] = v
    }
    return out
  }, [filterKey]) // eslint-disable-line react-hooks/exhaustive-deps

  const activeCount = Object.keys(filters).length

  useEffect(() => {
    void api.categories().then((r) => setCategories(r.items)).catch(() => null)
    void api.tags().then((r) => setTags(r.items)).catch(() => null)
  }, [])

  const load = useCallback(
    async (targetPage: number, append: boolean) => {
      setLoading(true)
      try {
        const res = await api.threads({ ...filters, sort, page: targetPage, limit: 12 })
        setThreads((prev) => (append ? [...prev, ...res.items] : res.items))
        setTotal(res.total)
        setPage(targetPage)
      } finally {
        setLoading(false)
      }
    },
    [filters, sort],
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

  function clearFilters() {
    const next = new URLSearchParams()
    if (sort !== 'hot') next.set('sort', sort)
    setParams(next, { replace: true })
  }

  const hasMore = threads.length < total

  return (
    <div>
      <SectionTitle
        color="#ff8a1f"
        sub={t('feed.subtitle')}
        right={
          user && (
            <Link to="/new" className="btn btn-primary">
              <PenLine size={13} strokeWidth={2.5} /> {t('nav.new')}
            </Link>
          )
        }
      >
        {t('feed.title')}
      </SectionTitle>

      <div className="lg:flex lg:items-start lg:gap-6">
        <aside className="mb-6 lg:sticky lg:top-4 lg:mb-0 lg:w-64 lg:shrink-0">
          <FeatureBanner
            id="feed"
            title={t('guides.feed.title')}
            tips={[1, 2, 3, 4].map((n) => t(`guides.feed.tip${n}`))}
          />
        </aside>
        <div className="min-w-0 flex-1">
      <div className="mb-5 flex flex-wrap items-center gap-2">
        {SORTS.map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => setParam('sort', s)}
            className="btn px-3 py-1.5"
            style={
              sort === s
                ? { background: '#ff8a1f', borderColor: '#000', color: '#000' }
                : { background: 'var(--color-slab)', borderColor: 'var(--color-edge)', color: '#a1a1aa' }
            }
          >
            {t(`feed.sort${s[0].toUpperCase()}${s.slice(1)}`)}
          </button>
        ))}

        <button
          type="button"
          onClick={() => setShowFilters((v) => !v)}
          className="btn btn-ghost ml-auto px-3 py-1.5"
          style={activeCount ? { borderColor: '#00e5ff', color: '#00e5ff' } : undefined}
        >
          <SlidersHorizontal size={12} strokeWidth={2.5} />
          {t('feed.filters')}
          {activeCount > 0 && ` (${activeCount})`}
        </button>
      </div>

      {(showFilters || activeCount > 0) && (
        <div className="slab mb-6 grid gap-3 border-[var(--color-cyber)] p-4 sm:grid-cols-2 lg:grid-cols-3">
          <Select
            value={filters.category ?? ''}
            onChange={(v) => setParam('category', v)}
            placeholder={t('feed.allCategories')}
            options={categories.map((c) => ({ value: c.slug, label: `${c.emoji} ${c.name}` }))}
          />
          <Select
            value={filters.country ?? ''}
            onChange={(v) => setParam('country', v)}
            placeholder={t('feed.anyCountry')}
            options={(meta?.countries ?? []).map((c) => ({
              value: c.value,
              label: `${flagEmoji(c.value)} ${c.label}`,
            }))}
          />
          <Select
            value={filters.seniority ?? ''}
            onChange={(v) => setParam('seniority', v)}
            placeholder={t('feed.anySeniority')}
            options={(meta?.seniorities ?? []).map((s) => ({
              value: s.value,
              label: `${s.emoji} ${s.label}`,
            }))}
          />
          <Select
            value={filters.cause ?? ''}
            onChange={(v) => setParam('cause', v)}
            placeholder={t('feed.anyCause')}
            options={(meta?.causes_of_death ?? []).map((c) => ({
              value: c.value,
              label: `${c.emoji} ${c.label}`,
            }))}
          />
          <Select
            value={filters.language ?? ''}
            onChange={(v) => setParam('language', v)}
            placeholder={t('feed.anyLanguage')}
            options={(meta?.languages ?? []).map((l) => ({
              value: l.value,
              label: `${l.emoji} ${l.label}`,
            }))}
          />
          <input
            value={filters.q ?? ''}
            onChange={(e) => setParam('q', e.target.value)}
            placeholder={t('nav.search')}
            className="field"
          />

          {activeCount > 0 && (
            <button
              type="button"
              onClick={clearFilters}
              className="btn btn-danger justify-self-start px-3 py-1.5 sm:col-span-2 lg:col-span-3"
            >
              <X size={12} strokeWidth={3} /> {t('feed.clearFilters')}
            </button>
          )}
        </div>
      )}

      {tags.length > 0 && activeCount === 0 && (
        <div className="mb-6">
          <p className="mb-2 text-[10px] tracking-widest text-zinc-600 uppercase">
            {t('feed.trendingTags')}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {tags.slice(0, 14).map((tag) => (
              <Link key={tag.slug} to={`/feed?tag=${tag.slug}`}>
                <Chip color="#a855f7">
                  #{tag.label} <span className="opacity-60">{tag.usage_count}</span>
                </Chip>
              </Link>
            ))}
          </div>
        </div>
      )}

      {loading && threads.length === 0 ? (
        <Spinner label={t('common.loading')} />
      ) : threads.length === 0 ? (
        <EmptyState emoji="🕳️">
          {activeCount > 0 ? t('feed.emptyFiltered') : t('feed.empty')}
        </EmptyState>
      ) : (
        <>
          <p className="mb-4 text-[10px] tracking-widest text-zinc-600 uppercase">
            {t('feed.resultCount', { count: total })}
          </p>
          <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
            {threads.map((th) => (
              <ThreadCard key={th.id} thread={th} />
            ))}
          </div>
          {hasMore && (
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
      </div>
    </div>
  )
}

/** Select is a native select styled to match the brutalist form controls. */
export function Select({
  value,
  onChange,
  options,
  placeholder,
  label,
}: {
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
  placeholder?: string
  label?: string
}) {
  return (
    <label className="block">
      {label && (
        <span className="mb-1 block text-[10px] tracking-widest text-zinc-500 uppercase">
          {label}
        </span>
      )}
      <select value={value} onChange={(e) => onChange(e.target.value)} className="field cursor-pointer">
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  )
}
