import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18n from '../../i18n'
import { moderationAPI } from '../../services/api'
import { useStore } from '../../store'
import Settings from '../Settings'

const makeUser = () => ({
  id: 'u1',
  username: 'me',
  displayName: 'Me',
  email: 'me@example.com',
  nativeLanguage: 'en',
  targetLanguages: ['es'],
})

const makeBlock = (over = {}) => ({
  id: 'b1',
  blockerId: 'u1',
  blockedId: 'u2',
  reason: 'spam',
  createdAt: new Date().toISOString(),
  blocked: {
    id: 'u2',
    username: 'alice',
    displayName: 'Alice',
    email: 'alice@example.com',
  },
  ...over,
})

describe('Settings blocked users', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(i18n, 'changeLanguage').mockImplementation(() => Promise.resolve())
    useStore.setState({ user: makeUser() as any })
  })
  afterEach(() => vi.restoreAllMocks())

  it('shows the empty state when there are no blocked users', async () => {
    vi.spyOn(moderationAPI, 'getBlocked').mockResolvedValue([])
    render(<Settings onClose={vi.fn()} />)

    expect(await screen.findByText('No blocked users.')).toBeInTheDocument()
  })

  it('lists blocked users and unblocks them', async () => {
    const user = userEvent.setup()
    const getBlocked = vi.spyOn(moderationAPI, 'getBlocked').mockResolvedValue([makeBlock() as any])
    const unblock = vi.spyOn(moderationAPI, 'unblock').mockResolvedValue(undefined)

    render(<Settings onClose={vi.fn()} />)

    expect(await screen.findByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('@alice')).toBeInTheDocument()
    expect(getBlocked).toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Unblock' }))

    await waitFor(() => expect(unblock).toHaveBeenCalledWith('u2'))
    await waitFor(() => expect(screen.queryByText('Alice')).not.toBeInTheDocument())
    expect(screen.getByText('No blocked users.')).toBeInTheDocument()
  })

  it('renders a fallback name when the blocked user has no profile', async () => {
    vi.spyOn(moderationAPI, 'getBlocked').mockResolvedValue([
      makeBlock({ blocked: null }) as any,
    ])
    render(<Settings onClose={vi.fn()} />)

    expect(await screen.findByText('Unknown')).toBeInTheDocument()
  })
})