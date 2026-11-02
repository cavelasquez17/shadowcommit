import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ReactionSummary } from '@/lib/api'
import { makeReactions, makeUser } from '@/test/factories'
import { ReactionBar } from './ReactionBar'

const app = vi.hoisted(() => ({
  state: { user: null as unknown, meta: null as unknown, copium: false },
}))
vi.mock('@/store/app', () => ({ useApp: () => app.state }))

const meta = {
  reactions: [
    { value: 'f', label: 'Press F', color: '#f87171', snark: 'Respects paid.' },
    { value: 'laugh', label: 'Laugh', color: '#facc15', snark: 'Nervous.' },
    { value: 'relate', label: 'Too real', color: '#22d3ee', snark: 'Same.' },
  ],
}

beforeEach(() => {
  app.state = { user: null, meta, copium: false }
})

const signIn = () => {
  app.state = { user: makeUser(), meta, copium: false }
}
const button = (name: RegExp) => screen.getByRole('button', { name })

describe('ReactionBar for anonymous visitors', () => {
  it('shows only reactions that have votes, all disabled', () => {
    render(<ReactionBar reactions={makeReactions({ f: { count: 4 }, laugh: { count: 1 } })} onReact={vi.fn()} />)
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(2)
    buttons.forEach((b) => expect(b).toBeDisabled())
    expect(button(/Press F: 4/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Too real/ })).not.toBeInTheDocument()
  })

  it('says so when nothing has been reacted to yet', () => {
    render(<ReactionBar reactions={makeReactions()} onReact={vi.fn()} />)
    expect(screen.getByText(/no reactions/i)).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('never calls onReact', async () => {
    const onReact = vi.fn()
    render(<ReactionBar reactions={makeReactions({ f: { count: 1 } })} onReact={onReact} />)
    await userEvent.click(button(/Press F/))
    expect(onReact).not.toHaveBeenCalled()
  })

  it('does not expose a pressed state it cannot honour', () => {
    render(<ReactionBar reactions={makeReactions({ f: { count: 1 } })} onReact={vi.fn()} />)
    expect(button(/Press F/)).not.toHaveAttribute('aria-pressed')
  })
})

describe('ReactionBar for signed-in members', () => {
  beforeEach(signIn)

  it('renders every reaction kind, even those with zero votes', () => {
    render(<ReactionBar reactions={makeReactions()} onReact={vi.fn()} />)
    expect(screen.getAllByRole('button')).toHaveLength(3)
    screen.getAllByRole('button').forEach((b) => expect(b).toBeEnabled())
  })

  it('reflects the server state through aria-pressed', () => {
    render(<ReactionBar reactions={makeReactions({ f: { count: 2, mine: true } })} onReact={vi.fn()} />)
    expect(button(/Press F/)).toHaveAttribute('aria-pressed', 'true')
    expect(button(/Laugh/)).toHaveAttribute('aria-pressed', 'false')
  })

  it('updates optimistically, then reconciles with the server response', async () => {
    let resolve!: (r: ReactionSummary[]) => void
    const onReact = vi.fn(() => new Promise<ReactionSummary[]>((r) => (resolve = r)))
    render(<ReactionBar reactions={makeReactions({ f: { count: 2 } })} onReact={onReact} />)

    await userEvent.click(button(/Press F/))
    expect(onReact).toHaveBeenCalledWith('f')
    // Server has not answered yet, but the UI already shows the new state.
    expect(button(/Press F: 3/)).toHaveAttribute('aria-pressed', 'true')

    resolve(makeReactions({ f: { count: 7, mine: true } })) // the server's truth wins
    await waitFor(() => expect(button(/Press F: 7/)).toBeInTheDocument())
  })

  it('un-reacts optimistically when you already reacted', async () => {
    const onReact = vi.fn(() => new Promise<ReactionSummary[]>(() => {}))
    render(<ReactionBar reactions={makeReactions({ f: { count: 3, mine: true } })} onReact={onReact} />)
    await userEvent.click(button(/Press F/))
    expect(button(/Press F: 2/)).toHaveAttribute('aria-pressed', 'false')
  })

  it('rolls back to the original state when the request fails', async () => {
    const onReact = vi.fn().mockRejectedValue(new Error('network'))
    render(<ReactionBar reactions={makeReactions({ f: { count: 2 } })} onReact={onReact} />)
    await userEvent.click(button(/Press F/))
    await waitFor(() => expect(button(/Press F: 2/)).toHaveAttribute('aria-pressed', 'false'))
  })

  it('ignores further clicks while a reaction is in flight', async () => {
    const onReact = vi.fn(() => new Promise<ReactionSummary[]>(() => {}))
    render(<ReactionBar reactions={makeReactions()} onReact={onReact} />)
    await userEvent.click(button(/Press F/))
    await userEvent.click(button(/Laugh/))
    await userEvent.click(button(/Press F/))
    expect(onReact).toHaveBeenCalledTimes(1)
  })

  it('swaps emoji in copium mode', () => {
    app.state = { user: makeUser(), meta, copium: true }
    render(<ReactionBar reactions={makeReactions({ f: { count: 1 } })} onReact={vi.fn()} />)
    expect(button(/Press F/)).not.toHaveTextContent('💀')
  })
})
