import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeReactions, makeThread, makeUser } from '@/test/factories'
import { ThreadCard } from './ThreadCard'

const app = vi.hoisted(() => ({ state: {} as Record<string, unknown> }))
vi.mock('@/store/app', () => ({ useApp: () => app.state }))

beforeEach(() => {
  app.state = {
    user: null,
    meta: { reactions: [] },
    copium: false,
    cause: (v: string) => ({ value: v, label: `cause:${v}`, emoji: '☠️' }),
    seniority: (v: string) => ({ value: v, label: `level:${v}`, emoji: '🎖' }),
    language: (v: string) => ({ value: v, label: v, emoji: '🌐' }),
  }
})

const renderCard = (over = {}) =>
  render(
    <MemoryRouter>
      <ThreadCard thread={makeThread(over)} />
    </MemoryRouter>,
  )

describe('ThreadCard', () => {
  it('links the title to the thread page', () => {
    renderCard({ id: 'thr_abc', title: 'My team is gone' })
    expect(screen.getByRole('link', { name: 'My team is gone' })).toHaveAttribute('href', '/t/thr_abc')
  })

  it('shows the author role, category and language', () => {
    renderCard({ language: 'es' })
    expect(screen.getByText(/Backend engineer/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /The Incident/ })).toHaveAttribute('href', '/feed?category=the-incident')
    expect(screen.getByText(/ES/)).toBeInTheDocument()
  })

  it('shows "deleted account" when the author no longer exists', () => {
    renderCard({ author: null })
    expect(screen.getByText('deleted account')).toBeInTheDocument()
  })

  it('labels an author with no current role as Anonymous', () => {
    renderCard({ author: makeUser({ current_role: '' }) })
    expect(screen.getByText(/Anonymous/)).toBeInTheDocument()
  })

  it('renders a link for every tag', () => {
    renderCard({ tags: [{ slug: 'ai', label: 'AI', usage_count: 3 }, { slug: 'go', label: 'Go', usage_count: 1 }] })
    expect(screen.getByRole('link', { name: '#AI' })).toHaveAttribute('href', '/feed?tag=ai')
    expect(screen.getByRole('link', { name: '#Go' })).toHaveAttribute('href', '/feed?tag=go')
  })

  it('compacts view and reply counts', () => {
    renderCard({ view_count: 1500, post_count: 12 })
    expect(screen.getByText('1.5K')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
  })

  it('marks pinned, NSFW and high-sarcasm threads, and only those', () => {
    const { unmount } = renderCard({ is_pinned: true, nsfw_corporate: true, sarcasm: 97 })
    expect(screen.getByText(/pinned/)).toBeInTheDocument()
    expect(screen.getByText(/97%/)).toBeInTheDocument()
    unmount()

    renderCard({ is_pinned: false, nsfw_corporate: false, sarcasm: 94 })
    expect(screen.queryByText(/pinned/)).not.toBeInTheDocument()
    expect(screen.queryByText(/94%/)).not.toBeInTheDocument()
  })

  describe('excerpt', () => {
    it('strips markdown noise and replaces code blocks', () => {
      renderCard({ body: '# Heading\n**bold** and `code`\n```js\nconst secret = 1\n```\nend' })
      const text = screen.getByText(/Heading/).textContent ?? ''
      expect(text).toContain('[code]')
      expect(text).not.toContain('const secret')
      expect(text).not.toMatch(/[#*`]/)
    })

    it('truncates long bodies with an ellipsis', () => {
      renderCard({ body: 'word '.repeat(200) })
      const text = screen.getByText(/^word/).textContent ?? ''
      expect(text.endsWith('…')).toBe(true)
      expect(text.length).toBeLessThanOrEqual(261)
    })

    it('leaves short bodies alone', () => {
      renderCard({ body: 'Short and sweet.' })
      expect(screen.getByText('Short and sweet.')).toBeInTheDocument()
    })
  })

  it('applies the copium filter to the title and excerpt when copium mode is on', () => {
    app.state = { ...app.state, copium: true }
    renderCard({ title: 'I was fired today', body: 'Total misery, a real layoff.' })
    expect(screen.queryByText(/fired/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/misery/i)).not.toBeInTheDocument()
  })

  it('exposes the reactions bar', () => {
    app.state = { ...app.state, user: makeUser() }
    renderCard({ reactions: makeReactions({ f: { count: 5 } }) })
    const article = screen.getByRole('article')
    expect(within(article).getByRole('button', { name: /Press F: 5/ })).toBeInTheDocument()
  })
})
