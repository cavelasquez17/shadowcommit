import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Link } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ErrorBoundary, PageBoundary } from './ErrorBoundary'

let consoleError: ReturnType<typeof vi.spyOn>
beforeEach(() => {
  // React logs every caught render error; silence it and assert on it instead.
  consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
})
afterEach(() => consoleError.mockRestore())

function Bomb({ explode }: { explode: boolean }) {
  if (explode) throw new TypeError("Cannot read properties of null (reading 'map')")
  return <p>page content</p>
}

const renderAt = (ui: React.ReactNode, path = '/') =>
  render(<MemoryRouter initialEntries={[path]}>{ui}</MemoryRouter>)

describe('ErrorBoundary', () => {
  it('renders its children when nothing fails', () => {
    renderAt(
      <ErrorBoundary>
        <Bomb explode={false} />
      </ErrorBoundary>,
    )
    expect(screen.getByText('page content')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('replaces a crashed subtree with a message instead of a blank screen', () => {
    renderAt(
      <ErrorBoundary>
        <Bomb explode />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /died on the job/i })).toBeInTheDocument()
    expect(screen.queryByText('page content')).not.toBeInTheDocument()
  })

  it('keeps everything outside the boundary alive', () => {
    renderAt(
      <>
        <nav>site menu</nav>
        <ErrorBoundary>
          <Bomb explode />
        </ErrorBoundary>
      </>,
    )
    expect(screen.getByText('site menu')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('reports the error to the console so it is not swallowed', () => {
    renderAt(
      <ErrorBoundary>
        <Bomb explode />
      </ErrorBoundary>,
    )
    const logged = consoleError.mock.calls.flat().map(String).join(' ')
    expect(logged).toContain("Cannot read properties of null (reading 'map')")
  })

  it('offers a way home', () => {
    renderAt(
      <ErrorBoundary>
        <Bomb explode />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('link', { name: /go home/i })).toHaveAttribute('href', '/')
  })

  it('recovers when the user retries and the cause is gone', async () => {
    let explode = true
    function Flaky() {
      return <Bomb explode={explode} />
    }
    renderAt(
      <ErrorBoundary>
        <Flaky />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('alert')).toBeInTheDocument()
    explode = false
    await userEvent.click(screen.getByRole('button', { name: /try again/i }))
    expect(screen.getByText('page content')).toBeInTheDocument()
  })

  it('stays on the error message when retrying does not fix it', async () => {
    renderAt(
      <ErrorBoundary>
        <Bomb explode />
      </ErrorBoundary>,
    )
    await userEvent.click(screen.getByRole('button', { name: /try again/i }))
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })
})

describe('PageBoundary', () => {
  it('clears the error when the user navigates to another page', async () => {
    let broken = true
    function Page() {
      return <Bomb explode={broken} />
    }
    renderAt(
      <>
        <Link to="/ok">ok page</Link>
        <PageBoundary>
          <Page />
        </PageBoundary>
      </>,
      '/broken',
    )
    expect(screen.getByRole('alert')).toBeInTheDocument()
    broken = false
    await userEvent.click(screen.getByRole('link', { name: 'ok page' }))
    expect(screen.getByText('page content')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
