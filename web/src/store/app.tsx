import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, type Meta, type Option, type User } from '@/lib/api'

/**
 * AppState is a single context for session + reference data. Two separate providers would
 * mean two loading states and two waterfalls for something every page needs anyway.
 */
interface AppState {
  user: User | null
  meta: Meta | null
  unread: number
  ready: boolean
  /** copium flips the site's emotional register. Persisted per-device, not per-account. */
  copium: boolean
  setCopium: (v: boolean) => void
  refresh: () => Promise<void>
  setUser: (u: User | null) => void
  setUnread: (n: number) => void
  logout: () => Promise<void>
  /** Lookup helpers so components never re-implement the enum → label mapping. */
  seniority: (value: string) => Option | undefined
  cause: (value: string) => Option | undefined
  language: (value: string) => Option | undefined
  country: (value: string) => Option | undefined
}

const AppContext = createContext<AppState | null>(null)

function findIn(opts: Option[] | undefined, value: string) {
  return opts?.find((o) => o.value === value)
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [meta, setMeta] = useState<Meta | null>(null)
  const [unread, setUnread] = useState(0)
  const [ready, setReady] = useState(false)
  const [copium, setCopiumState] = useState(
    () => localStorage.getItem('sc_copium') === 'true',
  )

  const setCopium = useCallback((v: boolean) => {
    setCopiumState(v)
    localStorage.setItem('sc_copium', String(v))
  }, [])

  const refresh = useCallback(async () => {
    // A 401 here is the normal anonymous case, not an error worth surfacing.
    const [me, m] = await Promise.all([
      api.me().catch(() => null),
      api.meta().catch(() => null),
    ])
    setUser(me?.user ?? null)
    setUnread(me?.unread_notifications ?? 0)
    if (m) setMeta(m)
    setReady(true)
  }, [])

  useEffect(() => {
    // Mount-time session hydration: genuinely synchronizing with the server, not
    // derivable during render.
    // eslint-disable-next-line react/set-state-in-effect
    void refresh()
  }, [refresh])

  // Copium mode tints the whole document; a data attribute keeps it out of React's way.
  useEffect(() => {
    document.documentElement.dataset.copium = String(copium)
  }, [copium])

  const logout = useCallback(async () => {
    await api.logout().catch(() => null)
    setUser(null)
    setUnread(0)
  }, [])

  const value = useMemo<AppState>(
    () => ({
      user,
      meta,
      unread,
      ready,
      copium,
      setCopium,
      refresh,
      setUser,
      setUnread,
      logout,
      seniority: (v) => findIn(meta?.seniorities, v),
      cause: (v) => findIn(meta?.causes_of_death, v),
      language: (v) => findIn(meta?.languages, v),
      country: (v) => findIn(meta?.countries, v),
    }),
    [user, meta, unread, ready, copium, setCopium, refresh, logout],
  )

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>
}

// useApp is intentionally co-located with AppProvider: they are one contract, and
// splitting them into a second file for Fast Refresh purism would hurt cohesion for
// what is otherwise a two-line accessor.
// eslint-disable-next-line react/only-export-components
export function useApp(): AppState {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp must be used inside <AppProvider>')
  return ctx
}
