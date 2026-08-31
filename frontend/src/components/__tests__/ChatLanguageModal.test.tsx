import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18n from '../../i18n'
import { useStore } from '../../store'
import ChatLanguageModal from '../ChatLanguageModal'

const makeUser = (over = {}) => ({
  id: 'u1',
  username: 'me',
  displayName: 'Me',
  email: 'me@example.com',
  nativeLanguage: 'en',
  targetLanguages: ['es'],
  ...over,
})

const makeOther = (over = {}) => ({
  id: 'u2',
  username: 'alice',
  displayName: 'Alice',
  email: 'alice@example.com',
  nativeLanguage: 'es',
  targetLanguages: ['en'],
  ...over,
})

function seedDirectChat() {
  useStore.setState({
    user: makeUser() as any,
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
  })
}

describe('ChatLanguageModal (FR-35: own language only)', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(i18n, 'changeLanguage').mockImplementation(() => Promise.resolve())
    seedDirectChat()
  })
  afterEach(() => vi.restoreAllMocks())

  it('shows a single own-language dropdown and preview, with no reference to the contact language', () => {
    render(<ChatLanguageModal onClose={vi.fn()} />)

    expect(screen.getByRole('heading', { name: /Chat Language Settings/i })).toBeInTheDocument()

    // Exactly one language select remains (the contact's dropdown was removed).
    expect(screen.getAllByRole('combobox')).toHaveLength(1)

    // No trace of the other participant or their language.
    expect(screen.queryByText(/Alice/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/contact/i)).not.toBeInTheDocument()

    // Preview reflects the user's own language.
    expect(screen.getByText(/^English$/)).toBeInTheDocument()
  })

  it('updates the preview when the own language is changed', async () => {
    const user = userEvent.setup()
    render(<ChatLanguageModal onClose={vi.fn()} />)

    await user.selectOptions(screen.getByRole('combobox'), 'es')

    expect(screen.getByText(/^Español$/)).toBeInTheDocument()
  })
})
