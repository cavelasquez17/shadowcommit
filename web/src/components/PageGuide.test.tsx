import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import i18n from '@/lib/i18n'
import { PageGuide } from './PageGuide'

vi.mock('@/components/FeatureBanner', () => ({
  FeatureBanner: ({ title, tips }: { title: string; tips: string[] }) => (
    <aside aria-label={title} data-tips={tips.length}>
      {tips.join('|')}
    </aside>
  ),
}))

describe('PageGuide', () => {
  it('shows the guide beside the page for a page that has copy', () => {
    render(
      <PageGuide page="feed">
        <p>feed content</p>
      </PageGuide>,
    )
    expect(screen.getByRole('complementary', { name: 'How the feed works' })).toBeInTheDocument()
    expect(screen.getByText('feed content')).toBeInTheDocument()
  })

  it('passes every tip the page defines, in order, and no more than five', () => {
    render(
      <PageGuide page="feed">
        <p>x</p>
      </PageGuide>,
    )
    const banner = screen.getByRole('complementary', { name: 'How the feed works' })
    const tips = banner.textContent?.split('|') ?? []
    expect(tips.length).toBeGreaterThanOrEqual(3)
    expect(tips.length).toBeLessThanOrEqual(5)
    expect(tips[0]).toContain('Hot ranks')
  })

  it('renders the page unchanged when it has no guide', () => {
    render(
      <PageGuide page="no-such-page">
        <p>plain page</p>
      </PageGuide>,
    )
    expect(screen.getByText('plain page')).toBeInTheDocument()
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
  })

  it('never shows more than five tips, even if a page defines more', () => {
    i18n.addResource('en', 'translation', 'guides.crowded.title', 'Crowded page')
    for (let n = 1; n <= 7; n++) {
      i18n.addResource('en', 'translation', `guides.crowded.tip${n}`, `Tip number ${n}`)
    }
    render(
      <PageGuide page="crowded">
        <p>x</p>
      </PageGuide>,
    )
    const tips = screen.getByRole('complementary', { name: 'Crowded page' }).textContent?.split('|') ?? []
    expect(tips).toHaveLength(5)
    expect(tips[4]).toBe('Tip number 5')
  })
})
