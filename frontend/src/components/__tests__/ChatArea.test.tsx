import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import i18n from '../../i18n'
import { moderationAPI, messageAPI, vocabularyAPI } from '../../services/api'
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

  it('opens the emoji picker and inserts an emoji into the composer at the cursor', async () => {
    const user = userEvent.setup()
    render(<ChatArea />)

    // Composer starts empty; the emoji button is labelled via chat.emoji.
    const composer = screen.getByPlaceholderText(/type a message/i)
    await user.click(composer)
    await user.type(composer, 'hola')

    // Open the picker.
    await user.click(screen.getByRole('button', { name: 'Insert emoji' }))
    expect(screen.getByRole('dialog', { name: 'Emoji picker' })).toBeInTheDocument()

    // Click the smiley emoji in the grid.
    await user.click(screen.getByRole('button', { name: '😀' }))

    // The emoji is inserted right where the caret was — text becomes "hola😀".
    expect(composer).toHaveValue('hola😀')

    // Picker stays open so the user can keep adding emoji.
    expect(screen.getByRole('dialog', { name: 'Emoji picker' })).toBeInTheDocument()

    // Clicking outside the picker (e.g. the empty chat state) closes it.
    await user.click(screen.getByText(/no messages yet/i))
    expect(screen.queryByRole('dialog', { name: 'Emoji picker' })).not.toBeInTheDocument()
  })

  it('passes emoji through to the backend unchanged when sending a message', async () => {
    const user = userEvent.setup()
    const send = vi.spyOn(messageAPI, 'sendMessage').mockResolvedValue({
      id: 'm1', chatId: 'chat-1', senderId: 'u1', text: 'hola😀 amigo',
      deliveryStatus: 'sent', timestamp: new Date().toISOString(),
    } as any)

    render(<ChatArea />)

    const composer = screen.getByPlaceholderText(/type a message/i)
    await user.click(composer)
    await user.type(composer, 'hola')

    // Insert an emoji, then continue typing.
    await user.click(screen.getByRole('button', { name: 'Insert emoji' }))
    await user.click(screen.getByRole('button', { name: '😀' }))
    await user.type(composer, ' amigo')

    await user.click(screen.getByRole('button', { name: /send/i }))

    // FR-21: emoji must arrive untouched — pre-serving any client-side mangling.
    await waitFor(() =>
      expect(send).toHaveBeenCalledWith('chat-1', { text: 'hola😀 amigo' })
    )
    expect(send.mock.calls[0][1].text).not.toContain('?')
  })

  it('renders emoji inside a message bubble unchanged (passthrough display)', () => {
    useStore.setState({
      messages: {
        'chat-1': [
          {
            id: 'm1', chatId: 'chat-1', senderId: 'u2', text: '¡Hola! 😀 🎉',
            deliveryStatus: 'sent', timestamp: new Date().toISOString(),
            sender: makeOther(),
          } as any,
        ],
      },
    })

    render(<ChatArea />)

    expect(screen.getByText('¡Hola! 😀 🎉')).toBeInTheDocument()
  })

  it('highlights new words in messages written in the learning language and saves on tap (FR-27/28)', async () => {
    const user = userEvent.setup()
    vi.spyOn(vocabularyAPI, 'getAll').mockResolvedValue([])
    const save = vi.spyOn(vocabularyAPI, 'save').mockResolvedValue({} as any)

    seedDirectChat()
    useStore.setState({
      messages: {
        'chat-1': [
          {
            id: 'm1',
            chatId: 'chat-1',
            senderId: 'u2',
            text: 'Hola amigo',
            originalLanguage: 'es',
            deliveryStatus: 'sent',
            timestamp: new Date().toISOString(),
            sender: makeOther(),
          } as any,
        ],
      },
    })

    render(
      <MemoryRouter>
        <ChatArea />
      </MemoryRouter>
    )

    const word = await screen.findByRole('button', { name: /amigo/i })

    await user.click(word)
    await user.click(screen.getByRole('button', { name: /add to word bank/i }))

    await waitFor(() => expect(save).toHaveBeenCalledWith('amigo', 'es', 'm1'))
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

  it('renders the other participant presence from the store (FR-9)', () => {
    seedDirectChat()
    useStore.setState({
      presence: {
        u2: { userId: 'u2', status: 'online', lastSeen: new Date().toISOString() },
      },
    })

    render(<ChatArea />)

    expect(screen.getByText('Online')).toBeInTheDocument()
  })

  it('renders an animated typing indicator when the other user is typing (FR-9)', () => {
    seedDirectChat()
    useStore.setState({
      typingUsers: { 'chat-1': { u2: true } },
    })

    render(<ChatArea />)

    expect(screen.getByText('Alice is typing…')).toBeInTheDocument()
  })

  it('does not show a typing indicator once the other user stops typing', () => {
    seedDirectChat()
    useStore.setState({
      typingUsers: { 'chat-1': { u2: false } },
    })

    render(<ChatArea />)

    expect(screen.queryByText('Alice is typing…')).not.toBeInTheDocument()
  })
})