import { useEffect, useRef, useState, useCallback } from 'react'
import { useStore } from '../store'
import { wsService } from '../services/websocket'
import { api } from '../services/api'
import type { TranscriptSegment } from '@chorus/shared'

interface CallScreenProps {
  callId: string
  chatId: string
  chatName: string
  onClose: () => void
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60).toString().padStart(2, '0')
  const s = (sec % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

export default function CallScreen({ callId, chatName, onClose }: CallScreenProps) {
  const { user } = useStore()
  const [status, setStatus] = useState<'connecting' | 'active' | 'ended'>('active')
  const [muted, setMuted] = useState(false)
  const [speakerOn, setSpeakerOn] = useState(true)
  const [captionsEnabled, setCaptionsEnabled] = useState(true)
  const [segments, setSegments] = useState<TranscriptSegment[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [captionInput, setCaptionInput] = useState('')
  const [sending, setSending] = useState(false)
  const [bookmarked, setBookmarked] = useState<Set<number>>(new Set())
  const [duration, setDuration] = useState(0)
  const [callType] = useState<'audio' | 'video'>('audio')
  const listRef = useRef<HTMLDivElement>(null)
  const peerRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const remoteAudioRef = useRef<HTMLAudioElement | null>(null)

  const nativeLanguage = user?.nativeLanguage || 'en'

  useEffect(() => {
    const id = setInterval(() => setDuration(d => d + 1), 1000)
    return () => clearInterval(id)
  }, [])

  const loadCaptions = useCallback(async (offset = 0) => {
    try {
      const res = await api.get<{ segments: TranscriptSegment[]; total: number; hasMore: boolean }>(`/calls/${callId}/captions?limit=50&offset=${offset}`)
      const data = res.data
      if (offset === 0) setSegments(data.segments)
      else setSegments(prev => [...prev, ...data.segments])
      setHasMore(data.hasMore)
    } catch {
      // silent
    }
  }, [callId])

  useEffect(() => {
    loadCaptions(0)
  }, [loadCaptions])

  useEffect(() => {
    if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight
  }, [segments])

  useEffect(() => {
    const handler = (msg: { type: string; data: unknown }) => {
      const data = msg.data as Record<string, unknown>
      if (msg.type === 'live_caption' && (data.callId === callId || (data.segment as Record<string, unknown>) !== undefined)) {
        const seg = data.segment as TranscriptSegment
        if (seg) setSegments(prev => [...prev, seg])
      }
      if (msg.type === 'call_ended' && (data.callId === callId || data.callId === undefined)) {
        setStatus('ended')
      }
      if (msg.type === 'webrtc_signal' && data.callId === callId) {
        handleSignal(data as { type: string; sdp?: string; candidate?: string })
      }
    }
    const unsub = wsService.onMessage(handler as Parameters<typeof wsService.onMessage>[0])
    return () => unsub()
  }, [callId])

  const handleSignal = async (payload: { type: string; sdp?: string; candidate?: string }) => {
    try {
      if (!peerRef.current) return
      if (payload.type === 'offer' && payload.sdp) {
        await peerRef.current.setRemoteDescription({ type: 'offer', sdp: payload.sdp })
        const answer = await peerRef.current.createAnswer()
        await peerRef.current.setLocalDescription(answer)
        await api.post(`/calls/${callId}/signal`, { type: 'answer', sdp: answer.sdp })
      } else if (payload.type === 'answer' && payload.sdp) {
        await peerRef.current.setRemoteDescription({ type: 'answer', sdp: payload.sdp })
      } else if (payload.type === 'ice-candidate' && payload.candidate) {
        try { await peerRef.current.addIceCandidate(JSON.parse(payload.candidate)) } catch { try { await peerRef.current.addIceCandidate({ candidate: payload.candidate } as RTCIceCandidateInit) } catch {} }
      }
    } catch {}
  }

  useEffect(() => {
    let cancelled = false
    async function initWebRTC() {
      try {
        const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] })
        peerRef.current = pc
        pc.ontrack = (e) => {
          if (remoteAudioRef.current) {
            remoteAudioRef.current.srcObject = e.streams[0]
          }
        }
        pc.onicecandidate = (e) => {
          if (e.candidate) {
            api.post(`/calls/${callId}/signal`, { type: 'ice-candidate', candidate: JSON.stringify(e.candidate) }).catch(() => {})
          }
        }
        try {
          const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: callType === 'video' })
          if (cancelled) { stream.getTracks().forEach(t => t.stop()); return }
          localStreamRef.current = stream
          stream.getTracks().forEach(track => pc.addTrack(track, stream))
        } catch {
          // mic permission denied — still show UI without audio
        }
        setStatus('active')
      } catch {
        setStatus('active')
      }
    }
    initWebRTC()
    return () => {
      cancelled = true
      peerRef.current?.close()
      peerRef.current = null
      localStreamRef.current?.getTracks().forEach(t => t.stop())
    }
  }, [callId, callType])

  const toggleMute = () => {
    const next = !muted
    setMuted(next)
    localStreamRef.current?.getAudioTracks().forEach(t => { t.enabled = !next })
  }

  const handleEnd = async () => {
    try { await api.post(`/calls/${callId}/end`) } catch {}
    setStatus('ended')
    setTimeout(onClose, 600)
  }

  const handleSendCaption = async (e?: React.FormEvent) => {
    e?.preventDefault()
    const text = captionInput.trim()
    if (!text || sending) return
    setSending(true)
    try {
      const res = await api.post<TranscriptSegment>(`/calls/${callId}/captions`, { text, language: nativeLanguage })
      setSegments(prev => [...prev, res.data])
      setCaptionInput('')
    } catch {} finally { setSending(false) }
  }

  const handleBookmark = async (idx: number) => {
    if (bookmarked.has(idx)) return
    try {
      await api.post(`/calls/${callId}/captions/${idx}/bookmark`)
      setBookmarked(prev => new Set(prev).add(idx))
    } catch {}
  }

  const handleLoadMore = async () => {
    if (!hasMore || loadingMore) return
    setLoadingMore(true)
    await loadCaptions(segments.length)
    setLoadingMore(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-[#0B1C30] text-white" role="dialog" aria-modal="true" aria-label="Call screen">
      <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 shrink-0">
        <div className="flex items-center gap-3 min-w-0">
          <div className="w-9 h-9 rounded-full bg-white/15 flex items-center justify-center font-bold shrink-0">{chatName.charAt(0).toUpperCase()}</div>
          <div className="min-w-0">
            <div className="font-semibold truncate text-sm">{chatName}</div>
            <div className="text-xs text-white/60 flex items-center gap-2">
              <span className={`w-2 h-2 rounded-full ${status === 'active' ? 'bg-emerald-400 animate-pulse' : status === 'ended' ? 'bg-red-400' : 'bg-amber-400'}`} />
              {status === 'active' ? formatDuration(duration) : status === 'ended' ? 'Ended' : 'Connecting...'}
              <span className="hidden sm:inline">· {callType === 'video' ? 'Video' : 'Audio'} call</span>
            </div>
          </div>
        </div>
        <button onClick={onClose} className="w-9 h-9 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center" aria-label="Minimize call">
          <span className="material-symbols-outlined text-[20px]">close_fullscreen</span>
        </button>
      </div>

      <div className="flex-1 flex flex-col md:flex-row overflow-hidden">
        <div className="flex-1 flex flex-col items-center justify-center p-6 bg-gradient-to-b from-[#0B1C30] to-[#132a4a] relative overflow-hidden">
          <div className="w-28 h-28 rounded-full bg-white/10 flex items-center justify-center text-4xl font-bold mb-4 border border-white/15">
            {chatName.charAt(0).toUpperCase()}
          </div>
          <div className="text-lg font-semibold">{chatName}</div>
          <div className="text-sm text-white/60 mt-1">{status === 'active' ? 'On call' : status === 'ended' ? 'Call ended' : 'Connecting'}</div>
          {status === 'active' && (
            <div className="mt-6 flex items-center gap-2 text-xs text-white/50">
              <span className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse" />
              {formatDuration(duration)}
            </div>
          )}
          <audio ref={remoteAudioRef} autoPlay playsInline className="hidden" />
        </div>

        <div className="w-full md:w-[380px] border-t md:border-t-0 md:border-l border-white/10 flex flex-col bg-[#0F2440] shrink-0 max-h-[45vh] md:max-h-none md:min-h-0">
          <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 shrink-0">
            <h3 className="font-semibold text-sm flex items-center gap-2">
              <span className="material-symbols-outlined text-[18px]">closed_caption</span>
              Live captions
            </h3>
            <label className="flex items-center gap-2 text-xs text-white/70 cursor-pointer">
              <input type="checkbox" checked={captionsEnabled} onChange={e => setCaptionsEnabled(e.target.checked)} className="accent-indigo-500" />
              Show
            </label>
          </div>

          <div ref={listRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-3 min-h-0" data-testid="transcript-panel">
            {captionsEnabled ? (
              segments.length === 0 ? (
                <div className="text-center text-white/40 text-sm py-8">
                  <div className="text-2xl mb-2">💬</div>
                  Captions will appear here as you speak.<br />
                  <span className="text-xs">Type below to add a caption manually.</span>
                </div>
              ) : (
                <>
                  {hasMore && (
                    <button onClick={handleLoadMore} disabled={loadingMore} className="w-full text-xs text-white/50 hover:text-white/80 py-1">
                      {loadingMore ? 'Loading...' : 'Load older captions'}
                    </button>
                  )}
                  {segments.map((seg, idx) => {
                    const translation = seg.translations?.[nativeLanguage] || Object.values(seg.translations || {})[0]
                    const isOwn = seg.speakerId === user?.id
                    return (
                      <div key={`${seg.startTime}-${idx}`} className={`rounded-xl px-3 py-2.5 ${isOwn ? 'bg-indigo-600/30 border border-indigo-500/30' : 'bg-white/5 border border-white/10'}`}>
                        <div className="text-xs text-white/50 mb-1 flex items-center justify-between">
                          <span>{isOwn ? 'You' : seg.speakerId.slice(0, 6)}</span>
                          <span>{new Date(seg.startTime * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                        </div>
                        <div className="text-sm leading-5">{seg.originalText}</div>
                        {translation && translation !== seg.originalText && (
                          <div className="mt-1.5 pt-1.5 border-t border-white/10 text-sm italic text-indigo-200">{translation}</div>
                        )}
                        <div className="mt-2 flex gap-2">
                          <button
                            onClick={() => handleBookmark(idx)}
                            disabled={bookmarked.has(idx)}
                            className={`text-xs px-2 py-1 rounded-full border ${bookmarked.has(idx) ? 'bg-emerald-500/20 border-emerald-500/30 text-emerald-300' : 'bg-white/5 border-white/10 hover:bg-white/10'}`}
                          >
                            {bookmarked.has(idx) ? '✓ Saved to vocab' : '☆ Save phrase'}
                          </button>
                        </div>
                      </div>
                    )
                  })}
                </>
              )
            ) : (
              <div className="text-center text-white/30 text-sm py-8">Captions hidden</div>
            )}
          </div>

          <form onSubmit={handleSendCaption} className="p-3 border-t border-white/10 flex gap-2 shrink-0">
            <input
              value={captionInput}
              onChange={e => setCaptionInput(e.target.value)}
              placeholder="Type a caption..."
              className="flex-1 min-w-0 bg-white/10 border border-white/15 rounded-full px-4 py-2 text-sm placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              maxLength={500}
            />
            <button type="submit" disabled={!captionInput.trim() || sending} className="w-9 h-9 rounded-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 flex items-center justify-center shrink-0" aria-label="Send caption">
              <span className="material-symbols-outlined text-[18px]">send</span>
            </button>
          </form>
        </div>
      </div>

      <div className="px-4 py-4 border-t border-white/10 bg-[#081428] flex items-center justify-center gap-3 shrink-0">
        <button
          onClick={toggleMute}
          aria-pressed={muted}
          className={`w-12 h-12 rounded-full flex items-center justify-center transition ${muted ? 'bg-amber-500 text-white' : 'bg-white/10 hover:bg-white/15 text-white'}`}
          title={muted ? 'Unmute' : 'Mute'}
          aria-label={muted ? 'Unmute microphone' : 'Mute microphone'}
        >
          <span className="material-symbols-outlined">{muted ? 'mic_off' : 'mic'}</span>
        </button>
        <button
          onClick={() => setSpeakerOn(v => !v)}
          aria-pressed={speakerOn}
          className={`w-12 h-12 rounded-full flex items-center justify-center transition ${speakerOn ? 'bg-white/10 hover:bg-white/15' : 'bg-white/5 opacity-60'}`}
          title={speakerOn ? 'Speaker on' : 'Speaker off'}
          aria-label="Toggle speaker"
        >
          <span className="material-symbols-outlined">{speakerOn ? 'volume_up' : 'volume_mute'}</span>
        </button>
        <button
          onClick={() => setCaptionsEnabled(v => !v)}
          aria-pressed={captionsEnabled}
          className={`w-12 h-12 rounded-full flex items-center justify-center transition ${captionsEnabled ? 'bg-indigo-600 text-white' : 'bg-white/10 text-white/60'}`}
          title="Toggle captions"
          aria-label="Toggle captions"
        >
          <span className="material-symbols-outlined">closed_caption</span>
        </button>
        <button
          onClick={handleEnd}
          className="ml-2 px-6 h-12 rounded-full bg-red-600 hover:bg-red-500 text-white font-semibold flex items-center gap-2"
          aria-label="End call"
        >
          <span className="material-symbols-outlined">call_end</span>
          End
        </button>
      </div>

      {status === 'ended' && (
        <div className="absolute inset-0 bg-[#0B1C30]/80 flex items-center justify-center">
          <div className="bg-white text-gray-900 rounded-2xl p-6 text-center max-w-sm mx-4">
            <div className="text-3xl mb-2">📞</div>
            <div className="font-bold">Call ended</div>
            <div className="text-sm text-gray-500 mt-1">Duration {formatDuration(duration)}</div>
            <button onClick={onClose} className="mt-4 px-6 py-2 bg-indigo-600 text-white rounded-full font-semibold">Close</button>
          </div>
        </div>
      )}
    </div>
  )
}
