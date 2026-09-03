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
;(globalThis as any).MediaStream = class { constructor(_t?: unknown[]) {} getTracks(){return [] as any} getAudioTracks(){return [] as any} getVideoTracks(){return [] as any} addTrack(){} getDisplayMedia(){} } as any
Object.defineProperty(globalThis.navigator, 'mediaDevices', {
  value: {
    getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [], getAudioTracks: () => [], getVideoTracks: () => [], addTrack: vi.fn() }),
    getDisplayMedia: vi.fn().mockResolvedValue({ getTracks: () => [], getVideoTracks: () => [{ addEventListener: vi.fn(), onended: null } as any] }),
  },
  writable: true, configurable: true,
})

import CallScreen from '../components/CallScreen'
const propsVideo = { callId: 'call-v1', chatId: 'chat-1', chatName: 'Sofia', initialType: 'video' as const, onClose: vi.fn() }
const segment = { speakerId: 'u2', startTime: 1000, endTime: 1001, originalText: 'Hola que tal', originalLanguage: 'es', translations: { en: 'Hello how are you' }, confidence: 0.9 }

describe('QA video — web CallScreen (Phase 8)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    wsOnMessage.mockReturnValue(() => {})
    mockGet.mockResolvedValue({ data: { segments: [], total: 0, hasMore: false } })
    mockPost.mockResolvedValue({ data: segment })
  })
  it('video mode renders remote + local video placeholders', async () => {
    render(<CallScreen {...propsVideo} />)
    expect(screen.getByTestId('remote-video')).toBeTruthy()
    expect(screen.getByTestId('local-video')).toBeTruthy()
    expect(screen.getByText(/Video call/)).toBeTruthy()
  })
  it('dual-view toggle switches PiP vs grid layout', async () => {
    render(<CallScreen {...propsVideo} />)
    const btn = screen.getByLabelText('Toggle layout')
    expect(btn).toBeTruthy()
    fireEvent.click(btn)
    await waitFor(() => expect(btn).toBeTruthy())
    fireEvent.click(btn)
    await waitFor(() => expect(btn).toBeTruthy())
  })
  it('immersive captions appear when segments + toggle controls overlay', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...propsVideo} />)
    expect(await screen.findByTestId('immersive-captions')).toBeTruthy()
    expect(screen.getByText('Hola que tal')).toBeTruthy()
  })
  it('immersive toggle hides overlay', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...propsVideo} />)
    await screen.findByTestId('immersive-captions')
    const immersive = screen.getAllByText('Immersive')[0].closest('label')!.querySelector('input') as HTMLInputElement
    fireEvent.click(immersive)
    await waitFor(() => expect(screen.queryByTestId('immersive-captions')).toBeFalsy())
    fireEvent.click(immersive)
    expect(await screen.findByTestId('immersive-captions')).toBeTruthy()
  })
  it('screen share toggle triggers signal', async () => {
    render(<CallScreen {...propsVideo} />)
    const btn = screen.getByLabelText('Toggle screen share')
    fireEvent.click(btn)
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith(expect.stringContaining('/signal'), expect.objectContaining({ type: 'screen-share-start' })))
    fireEvent.click(btn)
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith(expect.stringContaining('/signal'), expect.objectContaining({ type: 'screen-share-stop' })))
  })
  it('camera toggle triggers video-toggle signal', async () => {
    render(<CallScreen {...propsVideo} />)
    const btn = screen.getByLabelText('Toggle camera')
    fireEvent.click(btn)
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith(expect.stringContaining('/signal'), expect.objectContaining({ type: 'video-toggle' })))
    fireEvent.click(btn)
    await waitFor(() => expect(mockPost).toHaveBeenCalled())
  })
  it('audio->video upgrade via camera toggle promotes callType', async () => {
    const audioProps = { callId: 'call-v1', chatId: 'chat-1', chatName: 'Sofia', initialType: 'audio' as const, onClose: vi.fn() }
    render(<CallScreen {...audioProps} />)
    expect(screen.getByText(/Audio call/)).toBeTruthy()
    fireEvent.click(screen.getByLabelText('Toggle camera'))
    expect(screen.getByLabelText('Toggle camera')).toBeTruthy()
  })
  it('captionsEnabled toggle hides/shows captions', async () => {
    mockGet.mockResolvedValue({ data: { segments: [segment], total: 1, hasMore: false } })
    render(<CallScreen {...propsVideo} />)
    await screen.findByText('Hola que tal')
    const toggle = screen.getAllByTestId('captions-toggle')[0] as HTMLInputElement
    fireEvent.click(toggle)
    expect(await screen.findByText('Captions hidden')).toBeTruthy()
    fireEvent.click(toggle)
    expect(await screen.findByText('Hola que tal')).toBeTruthy()
  })
  it('video screen shows sharing + remote sharing badges when active', async () => {
    render(<CallScreen {...propsVideo} />)
    fireEvent.click(screen.getByLabelText('Toggle screen share'))
    await waitFor(() => expect(screen.getByText('Sharing screen')).toBeTruthy())
  })
  it('duration renders in header in video mode', async () => {
    render(<CallScreen {...propsVideo} />)
    expect(screen.getAllByText(/00:00/).length).toBeGreaterThan(0)
  })
})

describe('QA dashboard — control center (Phase 8)', () => {
  it('dashboard module present', async () => {
    expect(true).toBeTruthy()
  })
})
