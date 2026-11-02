import type { Category, ReactionSummary, Tag, Thread, User } from '@/lib/api'

export function makeUser(over: Partial<User> = {}): User {
  return {
    id: 'usr_1', username: 'carlos', display_name: 'Carlos', avatar_url: '', bio: '',
    country_code: 'CO', seniority: 'senior', years_exp: 8, cause_of_death: 'ceo_saw_a_demo',
    epitaph: '', current_role: 'Backend engineer', is_employed: false, fired_at: null,
    github_url: '', website_url: '', linkedin_url: '', ui_language: 'en', spoken_languages: ['en'],
    copium_mode: false, thread_count: 0, post_count: 0, reputation: 0, role: 'member',
    created_at: '2026-01-01T00:00:00Z', repos: [], badges: [], days_since_fired: null,
    follower_count: 0, ...over,
  }
}

export function makeReactions(counts: Record<string, { count: number; mine?: boolean }> = {}): ReactionSummary[] {
  const kinds: [string, string, string][] = [
    ['f', '💀', 'Press F'],
    ['laugh', '😂', 'Laugh'],
    ['relate', '🫠', 'Too real'],
  ]
  return kinds.map(([kind, emoji, label]) => ({
    kind, emoji, label, count: counts[kind]?.count ?? 0, mine: counts[kind]?.mine ?? false,
  }))
}

export function makeThread(over: Partial<Thread> = {}): Thread {
  const category: Category = {
    slug: 'the-incident', name: 'The Incident', description: '', emoji: '🔥', color: '#ff3355', thread_count: 1,
  }
  const tags: Tag[] = []
  return {
    id: 'thr_1', slug: 'a-thread', title: 'A thread about being replaced', body: 'The machine did my job.',
    language: 'en', sarcasm: 50, is_pinned: false, is_locked: false, nsfw_corporate: false,
    view_count: 1200, post_count: 3, reaction_count: 0, hot_score: 0,
    created_at: '2026-11-01T12:00:00Z', last_post_at: null, author: makeUser(), category, tags,
    reactions: makeReactions(), can_edit: false, ...over,
  }
}
