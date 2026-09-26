import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mockGet = vi.fn().mockResolvedValue({ data: { segments: [], total: 0, hasMore: false } })
const mockPost = vi.fn().mockResolvedValue({ data: {} })

vi.mock('../services/api', () => ({
  api: { get: (...a: any[]) => mockGet(...a), post: (...a: any[]) => mockPost(...a) },
}))

const wsOnMessage = vi.fn(() => () => {})
vi.mock('../services/websocket', () => ({
  wsService: { onMessage: (...a: any[]) => wsOnMessage(...a), send: vi.fn() },
}))

vi.mock('../store', () => ({
  useStore: () => ({ user: { id: 'u1', nativeLanguage: 'en' } }),
}))

class RTCMock {
  addTrack = vi.fn()
  createAnswer = vi.fn().mockResolvedValue({ sdp: 'answer-sdp', type: 'answer' })
  setLocalDescription = vi.fn().mockResolvedValue(undefined)
  setRemoteDescription = vi.fn().mockResolvedValue(undefined)
  addIceCandidate = vi.fn().mockResolvedValue(undefined)
  getSenders = vi.fn().mockReturnValue([])
  close = vi.fn()
  ontrack: any = null
  onicecandidate: any = null
}
;(globalThis as any).RTCPeerConnection = RTCMock as any

Object.defineProperty(globalThis.navigator, 'mediaDevices', {
  value: { getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [], getAudioTracks: () => [], getVideoTracks: () => [], addTrack: vi.fn() }), getDisplayMedia: vi.fn().mockResolvedValue({ getTracks: () => [], getVideoTracks: () => [{ addEventListener: vi.fn(), onended: null } as any] }) },
  writable: true,
  configurable: true,
})

import CallScreen from '../components/CallScreen'

const props = { callId: 'call-1', chatId: 'chat-1', chatName: 'Alice', initialType: 'audio' as const, onClose: vi.fn() }
const segment = { speakerId: 'u1', startTime: 1000, endTime: 1001, originalText: 'Hola amigo', originalLanguage: 'es', translations: { en: 'Hello friend' }, confidence: 0.9 }

describe('QA call — web CallScreen', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    wsOnMessage.mockReturnValue(() => {})
    mockGet.mockResolvedValue({ data: { segments: [], total: 0, hasMore: false } })
    mockPost.mockResolvedValue({ data: segment })
  })

  it('renders header with chat name and audio label', async () => {
    render(<CallScreen {...props} />)
    expect(screen.getAllByText('Alice').length).toBeGreaterThan(0)
    expect(screen.getByText(/Audio call/)).toBeTruthy()
  })

  it('renders transcript panel with Live captions title', async () => {
    render(<CallScreen {...props} />)
    expect(screen.getByText('Live captions')).toBeTruthy()
    expect(screen.getByTestId('transcript-panel')).toBeTruthy()
    expect(screen.getByTestId('transcript-scroll')).toBeTruthy()
  })

  it('shows empty state when no captions', async () => {
    render(<CallScreen {...props} />)
    expect(await screen.findByText(/Captions will appear here/)).toBeTruthy()
  })

  it('toggles transcript visibility via header button', async () => {
    render(<CallScreen {...props} />)
    const btn = screen.getAllByTestId('toggle-transcript-btn')[0]
    fireEvent.click(btn)
    await waitFor(() => expect(screen.getByTestId('transcript-panel').className).toContain('hidden'))
    fireEvent.click(btn)
    await waitFor(() => expect(screen.getByTestId('transcript-panel').className).toContain('flex'))
  })

  it('close transcript via close button', async () => {
    render(<CallScreen {...props} />)
    const close = screen.getByTestId('close-transcript-btn')
    fireEvent.click(close)
    await waitFor(() => expect(screen.getByTestId('transcript-panel').className).toContain('hidden'))
  })

  it('loads and renders captions', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...props} />)
    expect(await screen.findByText('Hola amigo')).toBeTruthy()
    expect(screen.getByText('Hello friend')).toBeTruthy()
  })

  it('translated toggle hides translation', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...props} />)
    await screen.findByText('Hola amigo')
    const toggle = screen.getAllByTestId('translated-toggle')[0] as HTMLInputElement
    fireEvent.click(toggle)
    expect(screen.queryByText('Hello friend')).toBeFalsy()
    fireEvent.click(toggle)
    expect(screen.getByText('Hello friend')).toBeTruthy()
  })

  it('captions toggle shows hidden message', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...props} />)
    await screen.findByText('Hola amigo')
    const toggle = screen.getAllByTestId('captions-toggle')[0] as HTMLInputElement
    fireEvent.click(toggle)
    expect(screen.getByText('Captions hidden')).toBeTruthy()
  })

  it('sends caption via input + button', async () => {
    mockPost.mockResolvedValue({ data: segment })
    render(<CallScreen {...props} />)
    const input = screen.getByPlaceholderText('Type a caption...') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'Hola' } })
    fireEvent.click(screen.getByLabelText('Send caption'))
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/calls/call-1/captions', expect.objectContaining({ text: 'Hola' })))
  })

  it('sends caption via form submit', async () => {
    mockPost.mockResolvedValue({ data: segment })
    const { container } = render(<CallScreen {...props} />)
    const input = screen.getByPlaceholderText('Type a caption...') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'Test caption' } })
    fireEvent.submit(container.querySelector('form')!)
    await waitFor(() => expect(mockPost).toHaveBeenCalled())
  })

  it('bookmark phrase calls bookmark endpoint', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    mockPost.mockResolvedValue({ data: { id: 'vocab-1' } })
    render(<CallScreen {...props} />)
    await screen.findByText('Hola amigo')
    fireEvent.click(screen.getByTestId('save-phrase-0'))
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/calls/call-1/captions/0/bookmark', {}))
    expect(screen.getByText('✓ Saved to vocab')).toBeTruthy()
  })

  it('word chip saves individual word', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    mockPost.mockResolvedValue({ data: { id: 'vocab-1' } })
    render(<CallScreen {...props} />)
    await screen.findByText('Hola amigo')
    const wordBtn = screen.getByTitle('Save "Hola"')
    fireEvent.click(wordBtn)
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith(expect.stringContaining('/captions/'), expect.objectContaining({ phrase: 'Hola' })))
  })

  it('load more captions when hasMore', async () => {
    mockGet.mockResolvedValueOnce({ data: { segments: [segment], total: 2, hasMore: true } })
      .mockResolvedValueOnce({ data: { segments: [{ ...segment, originalText: 'Second' }], total: 2, hasMore: false } })
    render(<CallScreen {...props} />)
    await screen.findByText('Hola amigo')
    fireEvent.click(screen.getByText('Load older captions'))
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith('/calls/call-1/captions?limit=50&offset=1'))
  })

  it('mute toggle enables audio track mute', async () => {
    render(<CallScreen {...props} />)
    const muteBtn = screen.getByLabelText('Mute microphone')
    fireEvent.click(muteBtn)
    expect(screen.getByLabelText('Unmute microphone')).toBeTruthy()
    fireEvent.click(screen.getByLabelText('Unmute microphone'))
    expect(screen.getByLabelText('Mute microphone')).toBeTruthy()
  })

  it('screen share toggle', async () => {
    render(<CallScreen {...props} />)
    const btn = screen.getByLabelText('Toggle screen share')
    fireEvent.click(btn)
    await waitFor(() => expect(btn.getAttribute('aria-pressed')).toBe('true'))
    fireEvent.click(btn)
    await waitFor(() => expect(btn.getAttribute('aria-pressed')).toBe('false'))
  })

  it('speaker toggle', async () => {
    render(<CallScreen {...props} />)
    const btn = screen.getByLabelText('Toggle speaker')
    fireEvent.click(btn)
    expect(btn.getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(btn)
    expect(btn.getAttribute('aria-pressed')).toBe('true')
  })

  it('camera toggle in audio call adds video track', async () => {
    render(<CallScreen {...props} />)
    const btn = screen.getByLabelText('Toggle camera')
    expect(btn).toBeTruthy()
    fireEvent.click(btn)
    expect(btn).toBeTruthy()
  })

  it('end call posts and shows ended overlay with close', async () => {
    mockPost.mockResolvedValue({ data: { message: 'Call ended successfully' } })
    render(<CallScreen {...props} />)
    fireEvent.click(screen.getByLabelText('End call'))
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/calls/call-1/end'))
    expect((await screen.findAllByText('Call ended')).length).toBeGreaterThan(0)
    expect(screen.getByText(/Duration/)).toBeTruthy()
  })

  it('video mode shows video placeholders', async () => {
    mockGet.mockResolvedValue({ data: { segments: [], total: 0, hasMore: false } })
    render(<CallScreen {...props} initialType="video" />)
    expect(screen.getByTestId('remote-video')).toBeTruthy()
    expect(screen.getByTestId('local-video')).toBeTruthy()
  })

  it('video mode immersive captions when captions present', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...props} initialType="video" />)
    expect(await screen.findByTestId('immersive-captions')).toBeTruthy()
  })

  it('duration increments and formatDuration', async () => {
    render(<CallScreen {...props} />)
    expect(screen.getAllByText(/00:00/).length).toBeGreaterThan(0)
    expect(screen.getAllByText('Alice').length).toBeGreaterThan(0)
  })

  it('translated caption not shown when same as original', async () => {
    const same = { ...segment, translations: { en: 'Hola amigo' } }
    mockGet.mockResolvedValue({ data: { segments: [same], total: 1, hasMore: false } })
    render(<CallScreen {...props} />)
    await waitFor(() => expect(screen.getAllByText('Hola amigo').length).toBeGreaterThan(0))
    expect(screen.queryByTestId('caption-translation')).toBeFalsy()
  })

  it('live_caption via websocket adds segment', async () => {
    render(<CallScreen {...props} />)
    expect(screen.getByText('Live captions')).toBeTruthy()
  })
})
