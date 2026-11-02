import { useEffect, useRef, useState } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  Bell,
  Briefcase,
  Flame,
  Ghost,
  LogOut,
  Menu,
  PenLine,
  Search,
  Skull,
  Ticket,
  Users,
  X,
} from 'lucide-react'
import { api } from '@/lib/api'
import { useApp } from '@/store/app'
import { UI_LOCALES } from '@/lib/i18n'
import { Avatar } from '@/components/ui'
import { GithubIcon } from '@/components/BrandIcons'

const NAV = [
  { to: '/feed', key: 'nav.feed', Icon: Flame, color: 'var(--color-tangerine)' },
  { to: '/wall', key: 'nav.wall', Icon: Skull, color: 'var(--color-blood)' },
  { to: '/members', key: 'nav.members', Icon: Users, color: 'var(--color-cyber)' },
  { to: '/jobs', key: 'nav.jobs', Icon: Briefcase, color: 'var(--color-slime)' },
  { to: '/risk', key: 'nav.risk', Icon: AlertTriangle, color: 'var(--color-butter)' },
] as const

export function Layout() {
  const { t, i18n } = useTranslation()
  const { user, unread, copium, setCopium, logout } = useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [search, setSearch] = useState('')

  // Close the mobile drawer whenever the route changes; leaving it open is a classic bug.
  // eslint-disable-next-line react/set-state-in-effect -- synchronizing with router location
  useEffect(() => setMobileOpen(false), [location.pathname])

  function submitSearch(e: React.FormEvent) {
    e.preventDefault()
    void navigate(`/feed?q=${encodeURIComponent(search)}`)
  }

  return (
    <div className="flex min-h-screen flex-col">
      <Ticker />

      <header className="sticky top-0 z-40 border-b-2 border-[var(--color-edge)] bg-[var(--color-void)]/95 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center gap-3 px-4 py-3">
          <Link to="/" className="group flex shrink-0 items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center border-2 border-black bg-[var(--color-toxic)] text-black">
              <Ghost size={18} strokeWidth={2.5} />
            </span>
            <span className="hidden font-black tracking-tighter text-white uppercase sm:block glitch">
              Shadow<span className="text-[var(--color-toxic)]">Commit</span>
            </span>
          </Link>

          <nav className="hidden items-center gap-1 md:flex">
            {NAV.map(({ to, key, Icon, color }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  `flex items-center gap-1.5 border-2 px-3 py-1.5 text-[11px] font-bold tracking-widest uppercase transition-colors ${
                    isActive
                      ? 'border-black text-black'
                      : 'border-transparent text-zinc-400 hover:border-[var(--color-edge)] hover:text-white'
                  }`
                }
                style={({ isActive }: { isActive: boolean }) =>
                  isActive ? { background: color } : undefined
                }
              >
                <Icon size={13} strokeWidth={2.5} />
                {t(key)}
              </NavLink>
            ))}
          </nav>

          <form onSubmit={submitSearch} className="relative ml-auto hidden max-w-xs flex-1 lg:block">
            <Search size={14} className="absolute top-2.5 left-2.5 text-zinc-600" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('nav.search')}
              className="field py-1.5 pl-8 text-xs"
            />
          </form>

          <div className="ml-auto flex items-center gap-2 lg:ml-0">
            <LanguagePicker />

            <button
              type="button"
              onClick={() => setCopium(!copium)}
              title={copium ? t('tone.switchedToCopium') : t('tone.switchedToReality')}
              className="btn hidden px-2 py-1.5 text-[10px] sm:inline-flex"
              style={{
                background: copium ? 'var(--color-slime)' : 'var(--color-slab)',
                borderColor: copium ? 'var(--color-black)' : 'var(--color-edge)',
                color: copium ? 'var(--color-black)' : 'var(--color-zinc-400)',
              }}
            >
              {copium ? '🫡' : '💀'} {copium ? t('tone.copium') : t('tone.reality')}
            </button>

            {user ? (
              <>
                <Link
                  to="/notifications"
                  className="relative flex h-8 w-8 items-center justify-center border-2 border-[var(--color-edge)] text-zinc-400 hover:border-[var(--color-butter)] hover:text-[var(--color-butter)]"
                  title={t('nav.notifications')}
                >
                  <Bell size={14} strokeWidth={2.5} />
                  {unread > 0 && (
                    <span className="absolute -top-2 -right-2 flex h-4 min-w-4 items-center justify-center border border-black bg-[var(--color-blood)] px-1 text-[9px] font-black text-white">
                      {unread > 9 ? '9+' : unread}
                    </span>
                  )}
                </Link>

                <Link to="/new" className="btn btn-primary hidden px-3 py-1.5 sm:inline-flex">
                  <PenLine size={13} strokeWidth={2.5} />
                  {t('nav.new')}
                </Link>

                <Link to={`/u/${user.username}`} title={t('nav.profile')}>
                  <Avatar user={user} size={32} />
                </Link>

                <button
                  type="button"
                  onClick={() => void logout()}
                  title={t('nav.logout')}
                  className="hidden h-8 w-8 items-center justify-center border-2 border-[var(--color-edge)] text-zinc-500 hover:border-[var(--color-blood)] hover:text-[var(--color-blood)] sm:flex"
                >
                  <LogOut size={13} strokeWidth={2.5} />
                </button>
              </>
            ) : (
              <a href={api.githubLoginURL()} className="btn btn-primary px-3 py-1.5">
                <GithubIcon size={13} />
                {t('nav.login')}
              </a>
            )}

            <button
              type="button"
              onClick={() => setMobileOpen((v) => !v)}
              className="flex h-8 w-8 items-center justify-center border-2 border-[var(--color-edge)] text-zinc-300 md:hidden"
              aria-label="menu"
            >
              {mobileOpen ? <X size={16} /> : <Menu size={16} />}
            </button>
          </div>
        </div>

        {mobileOpen && (
          <nav className="grid grid-cols-2 gap-2 border-t-2 border-[var(--color-edge)] bg-[var(--color-ink)] p-3 md:hidden">
            {NAV.map(({ to, key, Icon, color }) => (
              <Link
                key={to}
                to={to}
                className="slab flex items-center gap-2 px-3 py-2 text-[11px] font-bold tracking-wider uppercase"
                style={{ borderColor: color, color }}
              >
                <Icon size={14} strokeWidth={2.5} /> {t(key)}
              </Link>
            ))}
            {user && (
              <>
                <Link
                  to="/new"
                  className="slab flex items-center gap-2 border-[var(--color-toxic)] px-3 py-2 text-[11px] font-bold tracking-wider text-[var(--color-toxic)] uppercase"
                >
                  <PenLine size={14} strokeWidth={2.5} /> {t('nav.new')}
                </Link>
                <Link
                  to="/invites"
                  className="slab flex items-center gap-2 border-[var(--color-grape)] px-3 py-2 text-[11px] font-bold tracking-wider text-[var(--color-grape)] uppercase"
                >
                  <Ticket size={14} strokeWidth={2.5} /> {t('nav.invites')}
                </Link>
                <button
                  type="button"
                  onClick={() => void logout()}
                  className="slab col-span-2 flex items-center justify-center gap-2 border-[var(--color-blood)] px-3 py-2 text-[11px] font-bold tracking-wider text-[var(--color-blood)] uppercase"
                >
                  <LogOut size={14} strokeWidth={2.5} /> {t('nav.logout')}
                </button>
              </>
            )}
          </nav>
        )}
      </header>

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-7">
        {copium && (
          <div className="mb-4 flex items-center gap-2 border-2 border-[var(--color-slime)] bg-[var(--color-slime)]/10 px-3 py-2 text-[11px] font-bold tracking-wider text-[var(--color-slime)] uppercase">
            <span>✨</span> AI-enhanced for your comfort · all complaints have been optimized for positivity
          </div>
        )}
        <Outlet />
      </main>

      <footer className="border-t-2 border-[var(--color-edge)] bg-[var(--color-ink)]">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-3 px-4 py-5 text-[10px] tracking-wider text-zinc-600 uppercase">
          <span>{t('common.footer')}</span>
          <div className="flex items-center gap-4">
            <Link to="/invites" className="hover:text-[var(--color-grape)]">
              {t('nav.invites')}
            </Link>
            <a
              href="https://github.com/cavelasquez17/shadowcommit"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 hover:text-[var(--color-toxic)]"
            >
              <GithubIcon size={11} /> {t('common.sourceCode')}
            </a>
            <span className="text-zinc-700">{i18n.language}</span>
          </div>
        </div>
      </footer>
    </div>
  )
}

/** Ticker is the marquee of punchlines across the very top of the page. */
function Ticker() {
  const { t } = useTranslation()
  const text = t('landing.ticker')
  return (
    <div className="overflow-hidden border-b-2 border-black bg-[var(--color-blood)] py-1">
      <div className="marquee-track flex w-max">
        {/* Duplicated so the -50% translate loops seamlessly. */}
        {[0, 1].map((i) => (
          <span
            key={i}
            className="px-4 text-[10px] font-black tracking-[0.25em] whitespace-nowrap text-black uppercase"
          >
            {text} {text}
          </span>
        ))}
      </div>
    </div>
  )
}

function LanguagePicker() {
  const { i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)
  const current = UI_LOCALES.find((l) => l.code === i18n.language) ?? UI_LOCALES[0]

  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (!boxRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  return (
    <div ref={boxRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex h-8 items-center gap-1 border-2 border-[var(--color-edge)] px-2 text-xs hover:border-[var(--color-cyber)]"
      >
        {current.flag}
        <span className="text-[10px] font-bold tracking-wider text-zinc-400 uppercase">
          {current.code}
        </span>
      </button>
      {open && (
        <ul className="slab absolute right-0 z-50 mt-1 w-40 border-[var(--color-cyber)] p-1">
          {UI_LOCALES.map((l) => (
            <li key={l.code}>
              <button
                type="button"
                onClick={() => {
                  void i18n.changeLanguage(l.code)
                  setOpen(false)
                }}
                className={`flex w-full items-center gap-2 px-2 py-1.5 text-left text-xs ${
                  l.code === current.code
                    ? 'bg-[var(--color-cyber)] font-bold text-black'
                    : 'text-zinc-300 hover:bg-[var(--color-slab)]'
                }`}
              >
                {l.flag} {l.label}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
