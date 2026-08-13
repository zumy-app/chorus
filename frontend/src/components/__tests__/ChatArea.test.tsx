import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18n from '../../i18n'
import { moderationAPI } from '../../services/api'
import { useStore } from '../../store'
import ChatArea from '../ChatArea'

const makeUser = () => ({
  id: 'u1',
  username: 'me',
  displayName: 'Me',
  email: 'me@example.com',
  nativeLanguage: 'en',
  targetLanguages: ['es'],
})

const makeOther = () => ({
  id: 'u2',
  username: 'alice',
  displayName: 'Alice',
  email: 'alice@example.com',
  nativeLanguage: 'es',
  targetLanguages: ['en'],
})

function seedDirectChat() {
  useStore.setState({
    user: makeUser() as any,
    entitlements: {
      plan: 'free',
      effectivePlan: 'free',
      selfHost: false,
      planGraceUntil: null,
      features: { translationWordLimit: 280 },
      limits: {},
    } as any,
    activeChat: {
      id: 'chat-1',
      type: 'direct',
      name: '',
      createdAt: new Date().toISOString(),
      createdBy: 'u1',
      participants: [
        { id: 'p1', chatId: 'chat-1', userId: 'u1', role: 'member', user: makeUser() },
        { id: 'p2', chatId: 'chat-1', userId: 'u2', role: 'member', user: makeOther() },
      ],
    } as any,
    messages: {},
  })
}

describe('ChatArea block & report', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(i18n, 'changeLanguage').mockImplementation(() => Promise.resolve())
    seedDirectChat()
  })
  afterEach(() => vi.restoreAllMocks())

  it('opens the actions menu and blocks the other participant after confirmation', async () => {
    const user = userEvent.setup()
    const block = vi.spyOn(moderationAPI, 'block').mockResolvedValue({ message: 'User blocked' } as any)

    render(<ChatArea />)

    await user.click(screen.getByRole('button', { name: 'More actions' }))

    const blockButton = screen.getByRole('button', { name: /block alice/i })
    // First click asks for confirmation.
    await user.click(blockButton)
    expect(block).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: /block alice\?/i })).toBeInTheDocument()

    // Second click performs the block.
    await user.click(screen.getByRole('button', { name: /block alice\?/i }))

    await waitFor(() => expect(block).toHaveBeenCalledWith('u2'))
    await waitFor(() => expect(screen.getByText(/Alice has been blocked/i)).toBeInTheDocument())
  })

  it('shows the error banner when blocking fails', async () => {
    const user = userEvent.setup()
    vi.spyOn(moderationAPI, 'block').mockRejectedValue({
      response: { data: { error: 'Could not block this user.' } },
    })

    render(<ChatArea />)

    await user.click(screen.getByRole('button', { name: 'More actions' }))
    await user.click(screen.getByRole('button', { name: /block alice/i }))
    await user.click(screen.getByRole('button', { name: /block alice\?/i }))

    await waitFor(() => expect(screen.getByText('Could not block this user.')).toBeInTheDocument())
  })

  it('opens the report-user modal from the menu and submits a report', async () => {
    const user = userEvent.setup()
    const report = vi.spyOn(moderationAPI, 'report').mockResolvedValue({
      id: 'r1', type: 'user', reporterId: 'u1', reportedUserId: 'u2', reason: 'spam', status: 'open',
    } as any)

    render(<ChatArea />)

    await user.click(screen.getByRole('button', { name: 'More actions' }))
    await user.click(screen.getByRole('button', { name: /report user/i }))

    expect(screen.getByRole('heading', { name: /report user/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Submit report' }))

    await waitFor(() =>
      expect(report).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'user', reportedUserId: 'u2', reason: 'spam' })
      )
    )
    await waitFor(() => expect(screen.getByText(/Thanks for reporting/i)).toBeInTheDocument())
  })

  it('does not render the actions menu for group chats', () => {
    useStore.setState({
      activeChat: {
        id: 'group-1',
        type: 'group',
        name: 'Team',
        createdAt: new Date().toISOString(),
        createdBy: 'u1',
        participants: [
          { id: 'p1', chatId: 'group-1', userId: 'u1', role: 'member', user: makeUser() },
          { id: 'p2', chatId: 'group-1', userId: 'u2', role: 'member', user: makeOther() },
        ],
      } as any,
    })

    render(<ChatArea />)

    expect(screen.queryByRole('button', { name: 'More actions' })).not.toBeInTheDocument()
  })
})