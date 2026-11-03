import { act, render, screen, fireEvent } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { FeatureBanner } from './FeatureBanner'

const tips = ['First tip', 'Second tip', 'Third tip']

function setReducedMotion(reduce: boolean) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: reduce && query.includes('prefers-reduced-motion'),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }))
}

beforeEach(() => {
  vi.useFakeTimers()
  localStorage.clear()
  setReducedMotion(false)
})
afterEach(() => {
  vi.useRealTimers()
})

const shown = () => screen.getByRole('complementary', { name: 'Feed guide' }).querySelector('p')?.textContent

describe('FeatureBanner rotation', () => {
  it('starts on the given tip and advances on its own', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    expect(shown()).toBe('First tip')
    act(() => vi.advanceTimersByTime(1000))
    expect(shown()).toBe('Second tip')
    act(() => vi.advanceTimersByTime(1000))
    expect(shown()).toBe('Third tip')
  })

  it('wraps around to the first tip', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={2} />)
    act(() => vi.advanceTimersByTime(1000))
    expect(shown()).toBe('First tip')
  })

  it('pauses while the pointer is over it and resumes after', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    const aside = screen.getByRole('complementary', { name: 'Feed guide' })
    fireEvent.mouseEnter(aside)
    act(() => vi.advanceTimersByTime(5000))
    expect(shown()).toBe('First tip')
    fireEvent.mouseLeave(aside)
    act(() => vi.advanceTimersByTime(1000))
    expect(shown()).toBe('Second tip')
  })

  it('pauses while keyboard focus is inside it', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    const next = screen.getByRole('button', { name: 'Next tip' })
    fireEvent.focus(next)
    act(() => vi.advanceTimersByTime(5000))
    expect(shown()).toBe('First tip')
  })

  it('does not rotate while the tab is hidden, but does when it is visible', () => {
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    // Sanity check first: the same timer does advance while visible, so the next step proves something.
    act(() => vi.advanceTimersByTime(1000))
    expect(shown()).toBe('Second tip')

    visibility.mockReturnValue('hidden')
    act(() => vi.advanceTimersByTime(5000))
    expect(shown()).toBe('Second tip')
    visibility.mockRestore()
  })

  it('never rotates on its own for people who prefer less motion', () => {
    setReducedMotion(true)
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    act(() => vi.advanceTimersByTime(10000))
    expect(shown()).toBe('First tip')
    expect(screen.queryByRole('button', { name: 'Pause tips' })).not.toBeInTheDocument()
  })

  it('lets the user pause and play, and moves manually in every mode', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    fireEvent.click(screen.getByRole('button', { name: 'Pause tips' }))
    act(() => vi.advanceTimersByTime(3000))
    expect(shown()).toBe('First tip')
    fireEvent.click(screen.getByRole('button', { name: 'Next tip' }))
    expect(shown()).toBe('Second tip')
    fireEvent.click(screen.getByRole('button', { name: 'Play tips' }))
    act(() => vi.advanceTimersByTime(1000))
    expect(shown()).toBe('Third tip')
  })

  it('jumps to a tip from its dot indicator', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    fireEvent.click(screen.getByRole('button', { name: 'Tip 3 of 3' }))
    expect(shown()).toBe('Third tip')
    expect(screen.getByRole('button', { name: 'Tip 3 of 3' })).toHaveAttribute('aria-current', 'true')
  })

  it('clears its timer on unmount', () => {
    const clear = vi.spyOn(window, 'clearInterval')
    const { unmount } = render(<FeatureBanner id="feed" title="Feed guide" tips={tips} intervalMs={1000} startIndex={0} />)
    unmount()
    expect(clear).toHaveBeenCalled()
  })

  it('renders nothing without tips and does not rotate a single tip', () => {
    const { unmount } = render(<FeatureBanner id="empty" title="Feed guide" tips={[]} />)
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    unmount()
    render(<FeatureBanner id="one" title="Feed guide" tips={['Only tip']} startIndex={0} />)
    expect(screen.queryByRole('button', { name: 'Pause tips' })).not.toBeInTheDocument()
  })
})

describe('FeatureBanner dismissal', () => {
  it('disappears when dismissed and stays hidden on the next visit', () => {
    const { unmount } = render(<FeatureBanner id="feed" title="Feed guide" tips={tips} startIndex={0} />)
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    unmount()
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} startIndex={0} />)
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
  })

  it('remembers dismissal per banner id', () => {
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} startIndex={0} />)
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    render(<FeatureBanner id="jobs" title="Feed guide" tips={tips} startIndex={0} />)
    expect(screen.getByRole('complementary', { name: 'Feed guide' })).toBeInTheDocument()
  })

  it('still shows when storage is blocked, and dismissing still hides it for the session', () => {
    const getItem = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    const setItem = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    render(<FeatureBanner id="feed" title="Feed guide" tips={tips} startIndex={0} />)
    expect(screen.getByRole('complementary', { name: 'Feed guide' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    getItem.mockRestore()
    setItem.mockRestore()
  })
})
