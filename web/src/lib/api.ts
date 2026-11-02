// Typed client for the ShadowCommit API. Cookies carry the session, so every
// request is credentialed and there is no token to babysit in localStorage.

export interface Option {
  value: string
  label: string
  emoji?: string
  snark?: string
  color?: string
}

export interface Repo {
  id: string
  name: string
  url: string
  description: string
  language: string
  stars: number
  written_by_ai: boolean
}

export interface Badge {
  slug: string
  label: string
  description: string
  emoji: string
  color: string
}

export interface User {
  id: string
  username: string
  display_name: string
  avatar_url: string
  bio: string
  country_code: string
  seniority: string
  years_exp: number
  cause_of_death: string
  epitaph: string
  current_role: string
  is_employed: boolean
  fired_at: string | null
  github_url: string
  website_url: string
  linkedin_url: string
  ui_language: string
  spoken_languages: string[]
  copium_mode: boolean
  thread_count: number
  post_count: number
  reputation: number
  role: string
  created_at: string
  repos: Repo[]
  badges: Badge[]
  days_since_fired: number | null
  is_following?: boolean
  follower_count: number
}

export interface Category {
  slug: string
  name: string
  description: string
  emoji: string
  color: string
  thread_count: number
}

export interface Tag {
  slug: string
  label: string
  usage_count: number
}

export interface ReactionSummary {
  kind: string
  emoji: string
  label: string
  count: number
  mine: boolean
}

export interface Thread {
  id: string
  slug: string
  title: string
  body: string
  language: string
  sarcasm: number
  is_pinned: boolean
  is_locked: boolean
  nsfw_corporate: boolean
  view_count: number
  post_count: number
  reaction_count: number
  hot_score: number
  created_at: string
  last_post_at: string | null
  author: User | null
  category: Category | null
  tags: Tag[]
  reactions: ReactionSummary[]
  can_edit: boolean
}

export interface Post {
  id: string
  body: string
  language: string
  created_at: string
  updated_at: string
  parent_id: string | null
  is_answer: boolean
  edited_by_ai: boolean
  reaction_count: number
  author: User | null
  reactions: ReactionSummary[]
  can_edit: boolean
}

export interface Paginated<T> {
  items: T[]
  page: number
  limit: number
  total: number
}

export interface Meta {
  seniorities: Option[]
  causes_of_death: Option[]
  reactions: Option[]
  countries: Option[]
  languages: Option[]
  invite_only: boolean
  dev_login: boolean
  github_enabled: boolean
}

export interface Stats {
  total_users: number
  total_threads: number
  total_posts: number
  total_fired: number
  total_reactions: number
  ways_documented: number
  distinct_causes: number
  by_cause: { key: string; count: number }[]
  by_country: { key: string; count: number }[]
  by_seniority: { key: string; count: number }[]
  by_language: { key: string; count: number }[]
}

export interface Notification {
  id: string
  kind: string
  title: string
  body: string
  link: string
  is_read: boolean
  created_at: string
}

export interface Invite {
  code: string
  note: string
  used_at: string | null
  expires_at: string
  created_at: string
}

export interface JobPosting {
  id: string
  created_at: string
  company: string
  title: string
  body: string
  apply_url: string
  location: string
  remote: boolean
  country_code: string
  salary_min: number | null
  salary_max: number | null
  currency: string
  seniority: string
  language: string
  is_sponsored: boolean
  is_filled: boolean
  view_count: number
  posted_by?: User
}

export interface JobStats {
  open: number
  sponsored: number
  by_seniority: { seniority: string; count: number }[]
}

export interface JobFilters {
  seniority?: string
  remote?: boolean
  country?: string
  language?: string
  page?: number
  limit?: number
}

export interface JobPayload {
  company: string
  title: string
  body: string
  apply_url: string
  location?: string
  remote?: boolean
  country_code?: string
  salary_min?: number
  salary_max?: number
  currency?: string
  seniority?: string
  language?: string
}

/** ApiError carries the server's in-character message so the UI can just render it. */
export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

const BASE = import.meta.env.VITE_API_BASE ?? '/api'

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include',
    headers: init.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })

  if (res.status === 204) return undefined as T

  const payload = await res.json().catch(() => null)
  if (!res.ok) {
    throw new ApiError(
      res.status,
      payload?.error ?? 'unknown',
      payload?.message ?? `Request failed with ${res.status}`,
    )
  }
  return payload as T
}

type QueryValue = string | number | boolean | undefined | null

function qs(params: Record<string, QueryValue>): string {
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') sp.set(k, String(v))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

/** ThreadFilters carries an index signature so it can be passed straight to qs(). */
export interface ThreadFilters {
  [key: string]: QueryValue
  sort?: string
  category?: string
  tag?: string
  country?: string
  seniority?: string
  cause?: string
  language?: string
  author?: string
  q?: string
  page?: number
  limit?: number
}

export interface ProfilePatch {
  display_name?: string
  bio?: string
  country_code?: string
  timezone?: string
  seniority?: string
  years_exp?: number
  cause_of_death?: string
  epitaph?: string
  current_role?: string
  is_employed?: boolean
  fired_at?: string
  website_url?: string
  linkedin_url?: string
  ui_language?: string
  spoken_languages?: string[]
  copium_mode?: boolean
}

export const api = {
  meta: () => request<Meta>('/meta'),
  stats: () => request<Stats>('/stats'),
  categories: () => request<{ items: Category[] }>('/categories'),
  tags: () => request<{ items: Tag[] }>('/tags'),

  me: () => request<{ user: User; unread_notifications: number }>('/auth/me'),
  devLogin: (username: string, inviteCode?: string) =>
    request<{ user: User; is_new: boolean }>('/auth/dev-login', {
      method: 'POST',
      body: JSON.stringify({ username, invite_code: inviteCode || undefined }),
    }),
  logout: () => request<{ message: string }>('/auth/logout', { method: 'POST' }),
  githubLoginURL: (inviteCode?: string) =>
    `${BASE}/auth/github${inviteCode ? `?invite=${encodeURIComponent(inviteCode)}` : ''}`,

  threads: (f: ThreadFilters = {}) => request<Paginated<Thread>>(`/threads${qs(f)}`),
  thread: (id: string) => request<{ thread: Thread; posts: Post[] }>(`/threads/${id}`),
  createThread: (body: {
    title: string
    body: string
    category: string
    language?: string
    sarcasm?: number
    tags?: string[]
    nsfw_corporate?: boolean
  }) => request<{ thread: Thread }>('/threads', { method: 'POST', body: JSON.stringify(body) }),
  deleteThread: (id: string) => request<{ message: string }>(`/threads/${id}`, { method: 'DELETE' }),
  reactThread: (id: string, kind: string) =>
    request<{ reactions: ReactionSummary[] }>(`/threads/${id}/react`, {
      method: 'POST',
      body: JSON.stringify({ kind }),
    }),

  reply: (threadId: string, body: { body: string; parent_id?: string | null; language?: string }) =>
    request<{ post: Post }>(`/threads/${threadId}/posts`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  deletePost: (id: string) => request<{ message: string }>(`/posts/${id}`, { method: 'DELETE' }),
  reactPost: (id: string, kind: string) =>
    request<{ reactions: ReactionSummary[] }>(`/posts/${id}/react`, {
      method: 'POST',
      body: JSON.stringify({ kind }),
    }),

  users: (f: Record<string, string | number | undefined> = {}) =>
    request<Paginated<User>>(`/users${qs(f)}`),
  user: (username: string) => request<{ user: User; threads: Thread[] }>(`/users/${username}`),
  searchUsers: (q: string) =>
    request<{ items: Pick<User, 'username' | 'display_name' | 'avatar_url' | 'country_code' | 'seniority'>[] }>(
      `/users/search${qs({ q })}`,
    ),
  follow: (username: string) =>
    request<{ following: boolean }>(`/users/${username}/follow`, { method: 'POST' }),

  updateProfile: (patch: ProfilePatch) =>
    request<{ user: User }>('/profile', { method: 'PATCH', body: JSON.stringify(patch) }),
  addRepo: (body: { name: string; url: string; description?: string; language?: string; written_by_ai?: boolean }) =>
    request<{ repo: Repo }>('/profile/repos', { method: 'POST', body: JSON.stringify(body) }),
  deleteRepo: (id: string) => request<{ message: string }>(`/profile/repos/${id}`, { method: 'DELETE' }),

  notifications: () => request<{ items: Notification[]; unread: number }>('/notifications'),
  markNotificationsRead: () => request<{ message: string }>('/notifications/read', { method: 'POST' }),
  mentions: () =>
    request<{
      items: { created_at: string; thread_id: string; thread_title: string; by_username: string; by_avatar: string }[]
    }>('/mentions'),

  invites: () => request<{ items: Invite[]; base_url: string }>('/invites'),
  createInvite: (note: string) =>
    request<{ invite: Invite; url: string }>('/invites', { method: 'POST', body: JSON.stringify({ note }) }),
  checkInvite: (code: string) =>
    request<{
      valid: boolean
      reason?: string
      note?: string
      inviter?: { username: string; display_name: string; avatar_url: string }
    }>(`/invites/check${qs({ code })}`),

  // --- Jobs: "Hire the Fallen" ---
  jobs: (f: JobFilters = {}) => request<Paginated<JobPosting>>(`/jobs${qs(f as Record<string, QueryValue>)}`),
  job: (id: string) => request<{ job: JobPosting }>(`/jobs/${id}`),
  jobStats: () => request<JobStats>('/jobs/stats'),
  createJob: (body: JobPayload) =>
    request<{ job: JobPosting }>('/jobs', { method: 'POST', body: JSON.stringify(body) }),
  markJobFilled: (id: string) =>
    request<{ job: JobPosting }>(`/jobs/${id}/filled`, { method: 'POST' }),
  deleteJob: (id: string) => request<{ message: string }>(`/jobs/${id}`, { method: 'DELETE' }),
}
