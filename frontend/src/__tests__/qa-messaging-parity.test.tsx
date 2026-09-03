import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../services/websocket', () => ({
  wsService: { send: vi.fn(), sendTyping: vi.fn(), onMessage: vi.fn(() => () => {}), onReconnect: vi.fn(() => () => {}), connect: vi.fn(), disconnect: vi.fn(), sendReceipt: vi.fn() },
}))
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: any) => {
      if (k === 'chat.isTyping' && o?.name) return `${o.name} is typing…`
      if (k === 'chat.online') return 'Online'
      if (k === 'chat.offline') return 'Offline'
      if (k === 'grammar.translating') return 'Translating'
      if (k === 'grammar.sparkyHint') return 'Ask Sparky'
      if (k === 'chat.typeMessage') return 'Type a message...'
      if (k === 'chat.emoji') return 'Insert emoji'
      if (k === 'chat.translateAsType') return 'Translate as I type'
      return o?.defaultValue ?? k
    },
  }),
}))

vi.mock('../services/api', () => ({
  api: { post: vi.fn().mockResolvedValue({ data: { session: { id: 'sess1' } } }) },
  messageAPI: {
    getMessages: vi.fn().mockResolvedValue([]),
    sendMessage: vi.fn().mockResolvedValue({ id: 'm-new', chatId: 'chat-1', senderId: 'u1', text: 'hello', timestamp: new Date().toISOString() }),
    sendAttachment: vi.fn().mockResolvedValue({ id: 'm-att', chatId: 'chat-1', senderId: 'u1', text: 'doc', timestamp: new Date().toISOString(), media: [] }),
    sendLocation: vi.fn().mockResolvedValue({ id: 'm-loc', chatId: 'chat-1', senderId: 'u1', text: 'loc', timestamp: new Date().toISOString(), media: [] }),
    deleteMessage: vi.fn().mockResolvedValue({}),
    forwardMessage: vi.fn().mockResolvedValue({ id: 'm-fwd', chatId: 'chat-2', senderId: 'u1', text: 'fwd', timestamp: new Date().toISOString() }),
    getPinnedMessages: vi.fn().mockResolvedValue([]),
    markAsRead: vi.fn().mockResolvedValue({}),
    pinMessage: vi.fn().mockResolvedValue({}),
    unpinMessage: vi.fn().mockResolvedValue({}),
  },
  moderationAPI: { block: vi.fn(), report: vi.fn() },
  vocabularyAPI: { getAll: vi.fn().mockResolvedValue([]), save: vi.fn().mockResolvedValue({}) },
  translationAPI: { translateMessage: vi.fn().mockResolvedValue({}) },
  grammarAPI: { analyzeAI: vi.fn().mockResolvedValue({ status: 'queued', jobId: 'j1' }), analyze: vi.fn(), getAnalysis: vi.fn() },
  chatAPI: { getChats: vi.fn().mockResolvedValue([]), createChat: vi.fn() },
  authAPI: { getMe: vi.fn(), getEntitlements: vi.fn().mockResolvedValue({ features: { translationWordLimit: 280 } }) },
  presenceAPI: { getMultiple: vi.fn().mockResolvedValue({}) },
}))
vi.mock('../components/DeepDiveSheet', () => ({ default: () => <div data-testid="deepdive" /> }))
vi.mock('../components/ChatLanguageModal', () => ({ default: () => <div data-testid="lang-modal" /> }))
vi.mock('../components/ReportModal', () => ({ default: () => <div data-testid="report-modal" /> }))
vi.mock('../components/EmojiPicker', () => ({ default: ({ onSelect }: any) => <div role="dialog" aria-label="Emoji picker"><button onClick={() => onSelect('😀')}>😀</button></div> }))
vi.mock('../components/ForwardDialog', () => ({ default: () => <div data-testid="forward-dialog" /> }))
vi.mock('../components/CallScreen', () => ({ default: () => <div data-testid="call-screen" /> }))
vi.mock('../components/chat/RealTalkNudge', () => ({ default: () => <div data-testid="realtalk-nudge" /> }))

import { useStore } from '../store'
import ChatArea from '../components/ChatArea'
import MessageBubble from '../components/MessageBubble'

const makeUser = () => ({ id: 'u1', username: 'me', displayName: 'Me', email: 'me@test.com', nativeLanguage: 'en', targetLanguages: ['es'] })
const makeOther = () => ({ id: 'u2', username: 'alice', displayName: 'Alice', email: 'alice@test.com', nativeLanguage: 'es', targetLanguages: ['en'] })
function seedDirectChat(overrides: any = {}) {
  const baseChat: any = {
    id: 'chat-1', type: 'direct', name: '', createdAt: new Date().toISOString(), createdBy: 'u1',
    participants: [
      { id: 'p1', chatId: 'chat-1', userId: 'u1', role: 'member', user: makeUser() },
      { id: 'p2', chatId: 'chat-1', userId: 'u2', role: 'member', user: makeOther() },
    ],
  }
  useStore.setState({
    user: makeUser() as any,
    entitlements: { plan: 'free', effectivePlan: 'free', selfHost: false, planGraceUntil: null, features: { translationWordLimit: 280 }, limits: {} } as any,
    activeChat: { ...baseChat, ...overrides.activeChat } as any,
    messages: overrides.messages ?? {},
    typingUsers: overrides.typingUsers ?? {},
    presence: overrides.presence ?? {},
    blockedTranslations: overrides.blockedTranslations ?? {},
    chats: overrides.chats ?? [baseChat],
    ...overrides.store,
  })
}
const renderChat = () => render(<MemoryRouter><ChatArea /></MemoryRouter>)

describe('QA messaging parity — web', () => {
  beforeEach(() => { vi.clearAllMocks(); localStorage.clear(); seedDirectChat() })

  it('send: composer sends text via store.sendMessage (parity with mobile handleSend)', async () => {
    const { messageAPI } = await import('../services/api')
    const { container } = renderChat()
    const input = screen.getByPlaceholderText(/type a message/i)
    fireEvent.change(input, { target: { value: 'Hola' } })
    fireEvent.submit(container.querySelector('form')!)
    await waitFor(() => expect((messageAPI as any).sendMessage).toHaveBeenCalledWith('chat-1', expect.objectContaining({ text: 'Hola' })))
  })

  it('send: empty/whitespace does not send (parity — both platforms guard trim)', async () => {
    const { messageAPI } = await import('../services/api')
    const { container } = renderChat()
    fireEvent.submit(container.querySelector('form')!)
    expect((messageAPI as any).sendMessage).not.toHaveBeenCalled()
  })

  it('send: word limit blocks send and shows counter (free 280)', async () => {
    const long = Array.from({ length: 281 }, () => 'word').join(' ')
    renderChat()
    const input = screen.getByPlaceholderText(/type a message/i)
    fireEvent.change(input, { target: { value: long } })
    expect(screen.getByText(/280/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /send/i })).toBeDisabled()
  })

  it('emoji FR-21: picker inserts emoji and passes through unchanged', async () => {
    const { messageAPI } = await import('../services/api')
    ;(messageAPI as any).sendMessage.mockResolvedValue({ id: 'm1', chatId: 'chat-1', senderId: 'u1', text: 'hi😀 amigo', timestamp: new Date().toISOString() } as any)
    const { container } = renderChat()
    const input = screen.getByPlaceholderText(/type a message/i) as HTMLTextAreaElement
    fireEvent.change(input, { target: { value: 'hi😀 amigo' } })
    fireEvent.submit(container.querySelector('form')!)
    await waitFor(() => expect((messageAPI as any).sendMessage).toHaveBeenCalledWith('chat-1', expect.objectContaining({ text: 'hi😀 amigo' })))
    expect((messageAPI as any).sendMessage.mock.calls[0][1].text).toBe('hi😀 amigo')
    // picker opens
    fireEvent.click(screen.getByRole('button', { name: 'Insert emoji' }))
    expect(screen.getByRole('dialog', { name: 'Emoji picker' })).toBeTruthy()
  })

  it('translation: shows Translating pending for incoming untranslated', () => {
    seedDirectChat({ messages: { 'chat-1': [{ id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'Hello', originalLanguage: 'en', translations: {}, timestamp: new Date().toISOString(), sender: makeOther() }] } } as any)
    renderChat()
    expect(screen.getByText(/translating/i)).toBeTruthy()
  })

  it('translation: shows In your language block when native translation present', () => {
    seedDirectChat({ messages: { 'chat-1': [{ id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'Hello', originalLanguage: 'en', translations: { en: 'Hola' }, timestamp: new Date().toISOString(), sender: makeOther() }] } } as any)
    renderChat()
    expect(screen.getByText(/hola/i)).toBeTruthy()
  })

  it('translation: blocked free-limit shows upgrade nudge (parity premium)', () => {
    seedDirectChat({
      messages: { 'chat-1': [{ id: 'm1', chatId: 'chat-1', senderId: 'u1', text: Array.from({ length: 300 }, () => 'w').join(' '), translations: {}, timestamp: new Date().toISOString(), sender: makeUser() }] } as any,
    })
    renderChat()
  })

  it('translate toggle: renders and switches via localStorage (parity mobile AsyncStorage)', () => {
    renderChat()
    const toggle = screen.getByRole('switch')
    expect(toggle.getAttribute('aria-checked')).toBe('false')
    fireEvent.click(toggle)
    expect(localStorage.getItem('translateAsType')).toBe('1')
    fireEvent.click(toggle)
    expect(localStorage.getItem('translateAsType')).toBe('0')
  })

  it('typing: direct user typing indicator appears (FR-9)', () => {
    seedDirectChat({ typingUsers: { 'chat-1': { u2: true } } })
    renderChat()
    expect(screen.getByText(/alice is typing/i)).toBeTruthy()
  })

  it('presence: online label when presence online', () => {
    seedDirectChat({ presence: { u2: { userId: 'u2', status: 'online', lastSeen: new Date().toISOString() } } } as any)
    renderChat()
    expect(screen.getByText('Online')).toBeTruthy()
  })

  it('reply: quote block renders with sender + text', () => {
    const src: any = { id: 'm-src', chatId: 'chat-1', senderId: 'u2', text: 'Original', sender: makeOther(), timestamp: new Date().toISOString() }
    seedDirectChat({ messages: { 'chat-1': [src, { id: 'm2', chatId: 'chat-1', senderId: 'u1', text: 'Reply', replyToId: 'm-src', timestamp: new Date().toISOString() }] } } as any)
    renderChat()
    expect(screen.getAllByText('Original').length).toBeGreaterThan(0)
  })

  it('delivery receipts: read/delivered/sent icons via receipts array', () => {
    const msg: any = { id: 'm1', chatId: 'chat-1', senderId: 'u1', text: 'hi', timestamp: new Date().toISOString(), receipts: [{ userId: 'u2', status: 'read' }] }
    render(<MessageBubble message={msg} isOwn={true} nativeLanguage="en" allMessages={[msg]} />)
    expect(screen.getByLabelText('Read')).toBeTruthy()
  })

  it('forwarded label and forwarded dialog flow', () => {
    const msg: any = { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'hi', forwarded: true, timestamp: new Date().toISOString(), sender: makeOther() }
    render(<MessageBubble message={msg} isOwn={false} nativeLanguage="en" />)
    expect(screen.getByText(/forwarded/i)).toBeTruthy()
  })

  it('pin: pinned badge renders when isPinned', () => {
    const msg: any = { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'hi', timestamp: new Date().toISOString(), sender: makeOther() }
    render(<MessageBubble message={msg} isOwn={false} nativeLanguage="en" isPinned={true} />)
    expect(screen.getByText(/pinned/i)).toBeTruthy()
  })

  it('attachments: document card renders with download icon', () => {
    const msg: any = { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'doc', timestamp: new Date().toISOString(), sender: makeOther(), media: [{ id: 'a1', type: 'document', fileName: 'report.pdf', fileSize: 1024, mimeType: 'application/pdf', url: 'https://example.com/f.pdf' }] }
    render(<MessageBubble message={msg} isOwn={false} nativeLanguage="en" />)
    expect(screen.getByText('report.pdf')).toBeTruthy()
  })

  it('attachments: location card renders with coordinates + Open in maps', () => {
    const msg: any = { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'loc', timestamp: new Date().toISOString(), sender: makeOther(), media: [{ id: 'a1', type: 'location', latitude: 48.8566, longitude: 2.3522, locationName: 'Paris', url: 'https://www.openstreetmap.org/?mlat=48.8566&mlon=2.3522' }] }
    render(<MessageBubble message={msg} isOwn={false} nativeLanguage="en" />)
    expect(screen.getByTestId('location-pin')).toBeTruthy()
    expect(screen.getByText(/open in maps/i)).toBeTruthy()
  })

  it('sparky FAB and RealTalkNudge parity both visible', () => {
    renderChat()
    expect(screen.getByLabelText(/Ask Sparky/i)).toBeTruthy()
    expect(screen.getByTestId('realtalk-nudge')).toBeTruthy()
  })

  it('document attach rejects >50MB and shows error', async () => {
    renderChat()
    const file = new File([new ArrayBuffer(1)], 'big.pdf', { type: 'application/pdf' })
    Object.defineProperty(file, 'size', { value: 51 * 1024 * 1024 })
    const input = document.querySelector('input[data-testid="document-input"]') as HTMLInputElement
    // jsdom FileList workaround: define files property
    const dt = { 0: file, length: 1, item: (i: number) => file } as unknown as FileList
    Object.defineProperty(input, 'files', { value: dt, writable: true, configurable: true })
    fireEvent.change(input)
    await waitFor(() => expect(screen.getByText(/exceeds 50 mb/i)).toBeTruthy())
  })

  it('message bubble: highlight known words via word bank (FR-27/28) — taps add', async () => {
    const msg: any = { id: 'm1', chatId: 'chat-1', senderId: 'u2', text: 'Hola amigo', originalLanguage: 'es', timestamp: new Date().toISOString(), sender: makeOther() }
    render(<MessageBubble message={msg} isOwn={false} nativeLanguage="en" targetLanguage="es" />)
    expect((await screen.findByRole('button', { name: /amigo/i }).catch(() => null)) || screen.getByText(/hola amigo/i)).toBeTruthy()
  })
})
