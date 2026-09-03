import { useEffect, useRef, useState, useCallback } from 'react'
import { useStore } from '../store'
import { wsService } from '../services/websocket'
import { api } from '../services/api'
import type { TranscriptSegment } from '@chorus/shared'

interface CallScreenProps {
  callId: string
  chatId: string
  chatName: string
  initialType?: 'audio' | 'video'
  onClose: () => void
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60).toString().padStart(2, '0')
  const s = (sec % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

export default function CallScreen({ callId, chatName, initialType = 'audio', onClose }: CallScreenProps) {
  const { user } = useStore()
  const [status, setStatus] = useState<'connecting' | 'active' | 'ended'>('active')
  const [muted, setMuted] = useState(false)
  const [cameraOn, setCameraOn] = useState(initialType === 'video')
  const [screenSharing, setScreenSharing] = useState(false)
  const [remoteScreenShare, setRemoteScreenShare] = useState(false)
  const [speakerOn, setSpeakerOn] = useState(true)
  const [captionsEnabled, setCaptionsEnabled] = useState(true)
  const [showTranslated, setShowTranslated] = useState(true)
  const [immersive, setImmersive] = useState(true)
  const [dualView, setDualView] = useState(false)
  const [transcribing, setTranscribing] = useState(false)
  const recognitionRef = useRef<unknown>(null)
  const [segments, setSegments] = useState<TranscriptSegment[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [captionInput, setCaptionInput] = useState('')
  const [sending, setSending] = useState(false)
  const [bookmarked, setBookmarked] = useState<Set<number>>(new Set())
  const [showTranscript, setShowTranscript] = useState(() => typeof window !== 'undefined' ? window.innerWidth >= 768 : true)
  const [duration, setDuration] = useState(0)
  const [callType, setCallType] = useState<'audio' | 'video'>(initialType)
  const listRef = useRef<HTMLDivElement>(null)
  const peerRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const screenStreamRef = useRef<MediaStream | null>(null)
  const remoteAudioRef = useRef<HTMLAudioElement | null>(null)
  const localVideoRef = useRef<HTMLVideoElement | null>(null)
  const remoteVideoRef = useRef<HTMLVideoElement | null>(null)
  const screenVideoRef = useRef<HTMLVideoElement | null>(null)

  const nativeLanguage = user?.nativeLanguage || 'en'
  const isVideo = callType === 'video'

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
    } catch {}
  }, [callId])

  useEffect(() => { loadCaptions(0) }, [loadCaptions])
  useEffect(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight }, [segments])

  useEffect(() => {
    const handler = (msg: { type: string; data: unknown }) => {
      const data = msg.data as Record<string, unknown>
      if (msg.type === 'live_caption' && (data.callId === callId || (data.segment as Record<string, unknown>) !== undefined)) {
        const seg = data.segment as TranscriptSegment
        if (seg) setSegments(prev => [...prev, seg])
      }
      if (msg.type === 'call_ended' && (data.callId === callId || data.callId === undefined)) setStatus('ended')
      if (msg.type === 'webrtc_signal' && data.callId === callId) {
        const sigType = data.type as string
        if (sigType === 'screen-share-start') setRemoteScreenShare(true)
        else if (sigType === 'screen-share-stop') setRemoteScreenShare(false)
        else if (sigType === 'video-toggle' && typeof data.enabled === 'boolean') {}
        else handleSignal(data as { type: string; sdp?: string; candidate?: string })
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
          if (remoteAudioRef.current) remoteAudioRef.current.srcObject = e.streams[0]
          if (remoteVideoRef.current) remoteVideoRef.current.srcObject = e.streams[0]
          if (e.streams[0].getVideoTracks().length > 0) setCallType('video')
        }
        pc.onicecandidate = (e) => {
          if (e.candidate) api.post(`/calls/${callId}/signal`, { type: 'ice-candidate', candidate: JSON.stringify(e.candidate) }).catch(() => {})
        }
        try {
          const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: isVideo })
          if (cancelled) { stream.getTracks().forEach(t => t.stop()); return }
          localStreamRef.current = stream
          stream.getTracks().forEach(track => pc.addTrack(track, stream))
          if (localVideoRef.current) localVideoRef.current.srcObject = stream
          if (!stream.getVideoTracks().length && isVideo) setCameraOn(false)
        } catch {
        }
        setStatus('active')
      } catch { setStatus('active') }
    }
    initWebRTC()
    return () => {
      cancelled = true
      peerRef.current?.close()
      peerRef.current = null
      localStreamRef.current?.getTracks().forEach(t => t.stop())
      screenStreamRef.current?.getTracks().forEach(t => t.stop())
    }
  }, [callId, isVideo])

  useEffect(() => {
    if (localVideoRef.current && localStreamRef.current) localVideoRef.current.srcObject = localStreamRef.current
  }, [cameraOn, isVideo])

  const toggleMute = () => {
    const next = !muted
    setMuted(next)
    localStreamRef.current?.getAudioTracks().forEach(t => { t.enabled = !next })
  }

  const toggleCamera = async () => {
    if (cameraOn) {
      localStreamRef.current?.getVideoTracks().forEach(t => { t.enabled = false; t.stop() })
      const audioOnly = localStreamRef.current?.getAudioTracks() || []
      const nextStream = new MediaStream(audioOnly)
      localStreamRef.current = nextStream
      if (localVideoRef.current) localVideoRef.current.srcObject = null
      setCameraOn(false)
      api.post(`/calls/${callId}/signal`, { type: 'video-toggle', data: { enabled: false } }).catch(() => {})
    } else {
      try {
        const vs = await navigator.mediaDevices.getUserMedia({ video: true })
        const vt = vs.getVideoTracks()[0]
        if (vt) {
          if (localStreamRef.current) localStreamRef.current.addTrack(vt)
          else { const s = new MediaStream([vt]); localStreamRef.current = s }
          const sender = peerRef.current?.getSenders().find(s => s.track?.kind === 'video')
          if (sender) await sender.replaceTrack(vt)
          else if (localStreamRef.current) localStreamRef.current.getTracks().forEach(track => {
            if (track === vt) peerRef.current?.addTrack(track, localStreamRef.current!)
          })
          if (localVideoRef.current) localVideoRef.current.srcObject = localStreamRef.current
          setCameraOn(true)
          setCallType('video')
          api.post(`/calls/${callId}/signal`, { type: 'video-toggle', data: { enabled: true } }).catch(() => {})
        }
      } catch {}
    }
  }

  const toggleScreenShare = async () => {
    if (screenSharing) {
      screenStreamRef.current?.getTracks().forEach(t => t.stop())
      screenStreamRef.current = null
      if (screenVideoRef.current) screenVideoRef.current.srcObject = null
      setScreenSharing(false)
      api.post(`/calls/${callId}/signal`, { type: 'screen-share-stop' }).catch(() => {})
      const camTrack = localStreamRef.current?.getVideoTracks()[0]
      const sender = peerRef.current?.getSenders().find(s => s.track?.kind === 'video')
      if (sender && camTrack) try { await sender.replaceTrack(camTrack) } catch {}
    } else {
      try {
        const disp = await (navigator.mediaDevices as unknown as { getDisplayMedia: (c: unknown) => Promise<MediaStream> }).getDisplayMedia({ video: true, audio: false })
        if (!disp) return
        screenStreamRef.current = disp
        if (screenVideoRef.current) screenVideoRef.current.srcObject = disp
        setScreenSharing(true)
        setCallType('video')
        api.post(`/calls/${callId}/signal`, { type: 'screen-share-start' }).catch(() => {})
        const screenTrack = disp.getVideoTracks()[0]
        const sender = peerRef.current?.getSenders().find(s => s.track?.kind === 'video')
        if (sender && screenTrack) {
          await sender.replaceTrack(screenTrack)
          screenTrack.onended = () => toggleScreenShare()
        } else if (screenTrack && peerRef.current) {
          peerRef.current.addTrack(screenTrack, disp)
        }
        disp.getVideoTracks()[0]?.addEventListener('ended', () => {
          setScreenSharing(false)
          api.post(`/calls/${callId}/signal`, { type: 'screen-share-stop' }).catch(() => {})
        })
      } catch {}
    }
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

  const toggleLiveTranscription = useCallback(() => {
    const w = window as unknown as Record<string, unknown>
    const RecCtor = (w.SpeechRecognition || w.webkitSpeechRecognition) as unknown as new () => {
      continuous: boolean; interimResults: boolean; lang: string
      onresult: ((e: { resultIndex: number; results: Array<{ isFinal: boolean; 0: { transcript: string } }> }) => void) | null
      onerror: (() => void) | null
      onend: (() => void) | null
      start: () => void; stop: () => void
    } | undefined
    if (!RecCtor) return
    if (transcribing) {
      try { (recognitionRef.current as { stop: () => void })?.stop() } catch {}
      setTranscribing(false)
      return
    }
    try {
      const rec: any = new (RecCtor as unknown as { new(): unknown })()
      rec.continuous = true
      rec.interimResults = true
      rec.lang = nativeLanguage || 'en-US'
      rec.onresult = (e: { resultIndex: number; results: Array<{ isFinal: boolean; 0: { transcript: string } }> }) => {
        let finalText = ''
        for (let i = e.resultIndex; i < e.results.length; i++) {
          if (e.results[i].isFinal) finalText += e.results[i][0].transcript + ' '
        }
        finalText = finalText.trim()
        if (finalText) {
          api.post<TranscriptSegment>(`/calls/${callId}/captions`, { text: finalText, language: nativeLanguage }).then(r => {
            if (r.data) setSegments(prev => [...prev, r.data])
          }).catch(() => {})
        }
      }
      rec.onerror = () => setTranscribing(false)
      rec.onend = () => setTranscribing(false)
      rec.start()
      recognitionRef.current = rec
      setTranscribing(true)
    } catch { setTranscribing(false) }
  }, [callId, nativeLanguage, transcribing])

  useEffect(() => () => {
    try { (recognitionRef.current as { stop: () => void })?.stop() } catch {}
  }, [])

  const [savedPhrases, setSavedPhrases] = useState<Set<string>>(new Set())
  const handleBookmark = async (idx: number, phrase?: string) => {
    const key = phrase ? `${idx}:${phrase}` : `${idx}:*`
    if (phrase ? savedPhrases.has(key) : bookmarked.has(idx)) return
    try {
      await api.post(`/calls/${callId}/captions/${idx}/bookmark`, phrase ? { phrase } : {})
      if (phrase) setSavedPhrases(prev => new Set(prev).add(key))
      else setBookmarked(prev => new Set(prev).add(idx))
    } catch {}
  }

  const handleLoadMore = async () => {
    if (!hasMore || loadingMore) return
    setLoadingMore(true)
    await loadCaptions(segments.length)
    setLoadingMore(false)
  }

  const latestImmersive = segments.slice(-3)

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
              {screenSharing && <span className="bg-emerald-500/20 text-emerald-300 px-2 py-0.5 rounded-full text-[10px] border border-emerald-500/30">Sharing screen</span>}
              {remoteScreenShare && <span className="bg-sky-500/20 text-sky-300 px-2 py-0.5 rounded-full text-[10px] border border-sky-500/30">Remote sharing</span>}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowTranscript(v => !v)} className={`w-9 h-9 rounded-full flex items-center justify-center ${showTranscript ? 'bg-indigo-600' : 'bg-white/10 hover:bg-white/20'}`} title={showTranscript ? 'Hide transcript' : 'Show transcript'} aria-label="Toggle transcript" data-testid="toggle-transcript-btn"><span className="material-symbols-outlined text-[18px]">forum</span></button>
          {isVideo && <button onClick={() => setDualView(v => !v)} className="w-9 h-9 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center" title={dualView ? 'PiP view' : 'Dual view'} aria-label="Toggle layout"><span className="material-symbols-outlined text-[18px]">{dualView ? 'picture_in_picture' : 'view_split'}</span></button>}
          <button onClick={onClose} className="w-9 h-9 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center" aria-label="Minimize call"><span className="material-symbols-outlined text-[20px]">close_fullscreen</span></button>
        </div>
      </div>

      <div className="flex-1 flex flex-col md:flex-row overflow-hidden relative">
        <div className="flex-1 relative flex flex-col overflow-hidden bg-gradient-to-b from-[#0B1C30] to-[#132a4a] min-h-[280px]">
          {isVideo ? (
            <div className={`flex-1 relative bg-black overflow-hidden ${dualView ? 'grid grid-cols-2 gap-1 p-1' : ''}`}>
              <div className={dualView ? 'relative bg-[#0B1C30] rounded-xl overflow-hidden' : 'absolute inset-0'}>
                <video ref={remoteVideoRef} autoPlay playsInline className="w-full h-full object-cover" data-testid="remote-video" />
                {!remoteVideoRef.current?.srcObject && <div className="absolute inset-0 flex flex-col items-center justify-center bg-[#0B1C30]"><div className="w-20 h-20 rounded-full bg-white/10 flex items-center justify-center text-2xl font-bold border border-white/15">{chatName.charAt(0).toUpperCase()}</div><div className="text-sm text-white/50 mt-3">{remoteScreenShare ? 'Presenting...' : 'Waiting for video...'}</div></div>}
                <div className="absolute bottom-2 left-2 bg-black/60 px-2 py-1 rounded-full text-xs">{chatName}</div>
              </div>
              <div className={dualView ? 'relative bg-[#0B1C30] rounded-xl overflow-hidden' : 'absolute bottom-3 right-3 w-28 h-36 sm:w-36 sm:h-48 rounded-xl overflow-hidden border-2 border-white/20 shadow-xl bg-[#0B1C30]'}>
                {screenSharing ? <video ref={screenVideoRef} autoPlay playsInline muted className="w-full h-full object-contain bg-black" data-testid="screen-video" /> : cameraOn ? <video ref={localVideoRef} autoPlay playsInline muted className="w-full h-full object-cover mirror" data-testid="local-video" /> : <div className="w-full h-full flex items-center justify-center bg-[#1a3354]"><span className="material-symbols-outlined text-white/40">videocam_off</span></div>}
                <div className="absolute bottom-1 left-1 bg-black/60 px-1.5 py-0.5 rounded-full text-[10px]">You {screenSharing ? '· Screen' : cameraOn ? '' : '· Cam off'}</div>
                {!dualView && cameraOn && !screenSharing && <div className="absolute top-1 right-1 w-2 h-2 bg-emerald-400 rounded-full animate-pulse" />}
              </div>
              {immersive && captionsEnabled && latestImmersive.length > 0 && (
                <div className="absolute bottom-3 left-3 right-3 sm:left-6 sm:right-20 pointer-events-none">
                  <div className="bg-black/70 backdrop-blur rounded-xl px-3 py-2 border border-white/10 space-y-1" data-testid="immersive-captions">
                    {latestImmersive.map((seg, i) => {
                      const tr = showTranslated ? (seg.translations?.[nativeLanguage] || Object.values(seg.translations || {})[0]) : undefined
                      return (
                        <div key={i} className="text-sm leading-5">
                          <span className="text-white">{seg.originalText}</span>
                          {tr && tr !== seg.originalText && <span className="text-indigo-200 italic"> · {tr}</span>}
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center p-6 relative">
              <div className="w-28 h-28 rounded-full bg-white/10 flex items-center justify-center text-4xl font-bold mb-4 border border-white/15">{chatName.charAt(0).toUpperCase()}</div>
              <div className="text-lg font-semibold">{chatName}</div>
              <div className="text-sm text-white/60 mt-1">{status === 'active' ? 'On call' : status === 'ended' ? 'Call ended' : 'Connecting'}</div>
              {status === 'active' && <div className="mt-6 flex items-center gap-2 text-xs text-white/50"><span className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse" />{formatDuration(duration)}</div>}
              {immersive && captionsEnabled && latestImmersive.length > 0 && (
                <div className="absolute bottom-4 left-4 right-4">
                  <div className="bg-black/50 backdrop-blur rounded-xl px-3 py-2 border border-white/10 space-y-1" data-testid="immersive-captions">
                    {latestImmersive.map((seg, i) => {
                      const tr = showTranslated ? (seg.translations?.[nativeLanguage] || Object.values(seg.translations || {})[0]) : undefined
                      return <div key={i} className="text-sm text-center"><span className="text-white">{seg.originalText}</span>{tr && tr !== seg.originalText && <span className="text-indigo-200 italic"> · {tr}</span>}</div>
                    })}
                  </div>
                </div>
              )}
            </div>
          )}
          <audio ref={remoteAudioRef} autoPlay playsInline className="hidden" />
        </div>

        <div id="transcript-panel" data-testid="transcript-panel" className={`${showTranscript ? 'flex' : 'hidden md:hidden'} w-full md:w-[380px] lg:w-96 border-t md:border-t-0 md:border-l border-white/10 flex-col bg-[#0F2440] shrink-0 absolute md:relative inset-x-0 bottom-0 md:inset-auto z-30 md:z-auto h-[50vh] md:h-auto md:flex-1 md:min-h-0 rounded-t-2xl md:rounded-none shadow-xl md:shadow-none transition-transform duration-300 ${showTranscript ? 'translate-y-0' : 'translate-y-full'}`}>
          <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 shrink-0">
            <h3 className="font-semibold text-sm flex items-center gap-2"><span className="material-symbols-outlined text-[18px]">closed_caption</span>Live captions</h3>
            <div className="flex items-center gap-3">
              <label className="hidden sm:flex items-center gap-1.5 text-xs text-white/70 cursor-pointer"><input type="checkbox" checked={showTranslated} onChange={e => setShowTranslated(e.target.checked)} className="accent-indigo-500" data-testid="translated-toggle" />Translated</label>
              <label className="hidden sm:flex items-center gap-1.5 text-xs text-white/70 cursor-pointer"><input type="checkbox" checked={immersive} onChange={e => setImmersive(e.target.checked)} className="accent-indigo-500" />Immersive</label>
              <label className="hidden sm:flex items-center gap-1.5 text-xs text-white/70 cursor-pointer"><input type="checkbox" checked={captionsEnabled} onChange={e => setCaptionsEnabled(e.target.checked)} className="accent-indigo-500" data-testid="captions-toggle" />Show</label>
              <button onClick={() => setShowTranscript(false)} className="md:hidden w-8 h-8 rounded-full bg-white/10 flex items-center justify-center" aria-label="Close transcript" id="close-transcript-btn" data-testid="close-transcript-btn"><span className="material-symbols-outlined text-[18px]">expand_more</span></button>
              <button onClick={() => setShowTranscript(false)} className="hidden md:flex w-8 h-8 rounded-full bg-white/10 hover:bg-white/20 items-center justify-center" aria-label="Close transcript"><span className="material-symbols-outlined text-[18px]">close</span></button>
            </div>
          </div>
          <div className="flex sm:hidden items-center gap-3 px-4 py-2 border-b border-white/5 shrink-0">
            <label className="flex items-center gap-1.5 text-xs text-white/70 cursor-pointer"><input type="checkbox" checked={showTranslated} onChange={e => setShowTranslated(e.target.checked)} className="accent-indigo-500" />Translated</label>
            <label className="flex items-center gap-1.5 text-xs text-white/70 cursor-pointer"><input type="checkbox" checked={immersive} onChange={e => setImmersive(e.target.checked)} className="accent-indigo-500" />Immersive</label>
            <label className="flex items-center gap-1.5 text-xs text-white/70 cursor-pointer"><input type="checkbox" checked={captionsEnabled} onChange={e => setCaptionsEnabled(e.target.checked)} className="accent-indigo-500" />Show</label>
          </div>
          <div ref={listRef} className="flex-1 overflow-y-auto custom-scrollbar px-4 py-3 space-y-3 min-h-0 overscroll-contain [scrollbar-width:thin] [scrollbar-color:rgba(255,255,255,0.2)_transparent] [&::-webkit-scrollbar]:w-1 [&::-webkit-scrollbar-thumb]:bg-white/20 [&::-webkit-scrollbar-thumb]:rounded-full" data-testid="transcript-scroll">
            {captionsEnabled ? (
              segments.length === 0 ? (
                <div className="text-center text-white/40 text-sm py-8"><div className="text-2xl mb-2">💬</div>Captions will appear here as you speak.<br /><span className="text-xs">Type below to add a caption manually.</span></div>
              ) : (
                <>
                  {hasMore && <button onClick={handleLoadMore} disabled={loadingMore} className="w-full text-xs text-white/50 hover:text-white/80 py-1">{loadingMore ? 'Loading...' : 'Load older captions'}</button>}
                  {segments.map((seg, idx) => {
                    const translation = showTranslated ? (seg.translations?.[nativeLanguage] || Object.values(seg.translations || {})[0]) : undefined
                    const isOwn = seg.speakerId === user?.id
                    const words = seg.originalText.split(/(\s+)/)
                    return (
                      <div key={`${seg.startTime}-${idx}`} className={`rounded-xl px-3 py-2.5 ${isOwn ? 'bg-indigo-600/30 border border-indigo-500/30' : 'bg-white/5 border border-white/10'}`}>
                        <div className="text-xs text-white/50 mb-1 flex items-center justify-between"><span>{isOwn ? 'You' : seg.speakerId.slice(0, 6)}</span><span>{new Date(seg.startTime * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span></div>
                        <div className="text-sm leading-5 flex flex-wrap gap-1">{words.map((w, wi) => {
                          if (/^\s+$/.test(w)) return <span key={wi}>{w}</span>
                          const clean = w.replace(/^[.,!?¿¡"'()]+|[.,!?¿¡"'()]+$/g, '')
                          if (!clean || clean.length < 2) return <span key={wi}>{w}</span>
                          const key = `${idx}:${clean}`
                          const saved = savedPhrases.has(key)
                          return <button key={wi} onClick={() => handleBookmark(idx, clean)} disabled={saved} title={saved ? 'Saved' : `Save "${clean}"`} className={`px-1 rounded ${saved ? 'bg-emerald-500/30 text-emerald-200' : 'hover:bg-white/15 hover:underline decoration-dotted underline-offset-2'}`}>{w}</button>
                        })}</div>
                        {translation && translation !== seg.originalText && <div className="mt-1.5 pt-1.5 border-t border-white/10 text-sm italic text-indigo-200" data-testid="caption-translation">{translation}</div>}
                        <div className="mt-2 flex gap-2"><button onClick={() => handleBookmark(idx)} disabled={bookmarked.has(idx)} className={`text-xs px-2 py-1 rounded-full border ${bookmarked.has(idx) ? 'bg-emerald-500/20 border-emerald-500/30 text-emerald-300' : 'bg-white/5 border-white/10 hover:bg-white/10'}`} data-testid={`save-phrase-${idx}`}>{bookmarked.has(idx) ? '✓ Saved to vocab' : '☆ Save phrase'}</button></div>
                      </div>
                    )
                  })}
                </>
              )
            ) : <div className="text-center text-white/30 text-sm py-8">Captions hidden</div>}
          </div>
          <div className="px-3 py-1 text-center text-[11px] text-white/30 shrink-0">Transcript auto-scrolls during conversation</div>
          <form onSubmit={handleSendCaption} className="p-3 border-t border-white/10 flex gap-2 shrink-0">
            <input value={captionInput} onChange={e => setCaptionInput(e.target.value)} placeholder="Type a caption..." className="flex-1 min-w-0 bg-white/10 border border-white/15 rounded-full px-4 py-2 text-sm placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-indigo-500" maxLength={500} />
            <button type="button" onClick={toggleLiveTranscription} className={`w-9 h-9 rounded-full flex items-center justify-center shrink-0 ${transcribing ? 'bg-red-600 animate-pulse' : 'bg-white/10 hover:bg-white/15'}`} aria-label="Live transcribe" title="Live transcribe" data-testid="live-transcribe-btn"><span className="material-symbols-outlined text-[18px]">{transcribing ? 'mic' : 'graphic_eq'}</span></button>
            <button type="submit" disabled={!captionInput.trim() || sending} className="w-9 h-9 rounded-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 flex items-center justify-center shrink-0" aria-label="Send caption"><span className="material-symbols-outlined text-[18px]">send</span></button>
          </form>
        </div>
      </div>

      <div className="px-4 py-4 border-t border-white/10 bg-[#081428] flex items-center justify-center gap-2 sm:gap-3 shrink-0 flex-wrap">
        <button onClick={() => setShowTranscript(v => !v)} aria-pressed={showTranscript} className={`w-12 h-12 rounded-full flex items-center justify-center transition md:hidden ${showTranscript ? 'bg-indigo-600 text-white' : 'bg-white/10 text-white/60'}`} title="Toggle transcript" aria-label="Toggle transcript" data-testid="toggle-transcript-btn-mobile"><span className="material-symbols-outlined">chat</span></button>
        <button onClick={toggleMute} aria-pressed={muted} className={`w-12 h-12 rounded-full flex items-center justify-center transition ${muted ? 'bg-amber-500 text-white' : 'bg-white/10 hover:bg-white/15 text-white'}`} title={muted ? 'Unmute' : 'Mute'} aria-label={muted ? 'Unmute microphone' : 'Mute microphone'}><span className="material-symbols-outlined">{muted ? 'mic_off' : 'mic'}</span></button>
        <button onClick={toggleCamera} aria-pressed={cameraOn} className={`w-12 h-12 rounded-full flex items-center justify-center transition ${cameraOn ? 'bg-white/10 hover:bg-white/15' : 'bg-white/5 opacity-60'}`} title={cameraOn ? 'Camera on' : 'Camera off'} aria-label="Toggle camera"><span className="material-symbols-outlined">{cameraOn ? 'videocam' : 'videocam_off'}</span></button>
        <button onClick={toggleScreenShare} aria-pressed={screenSharing} className={`w-12 h-12 rounded-full flex items-center justify-center transition ${screenSharing ? 'bg-emerald-600 text-white' : 'bg-white/10 hover:bg-white/15'}`} title={screenSharing ? 'Stop sharing' : 'Share screen'} aria-label="Toggle screen share"><span className="material-symbols-outlined">screen_share</span></button>
        <button onClick={() => setSpeakerOn(v => !v)} aria-pressed={speakerOn} className={`w-12 h-12 rounded-full flex items-center justify-center transition ${speakerOn ? 'bg-white/10 hover:bg-white/15' : 'bg-white/5 opacity-60'}`} title={speakerOn ? 'Speaker on' : 'Speaker off'} aria-label="Toggle speaker"><span className="material-symbols-outlined">{speakerOn ? 'volume_up' : 'volume_mute'}</span></button>
        <button onClick={() => setCaptionsEnabled(v => !v)} aria-pressed={captionsEnabled} className={`w-12 h-12 rounded-full flex items-center justify-center transition ${captionsEnabled ? 'bg-indigo-600 text-white' : 'bg-white/10 text-white/60'}`} title="Toggle captions" aria-label="Toggle captions"><span className="material-symbols-outlined">closed_caption</span></button>
        <button onClick={handleEnd} className="ml-2 px-6 h-12 rounded-full bg-red-600 hover:bg-red-500 text-white font-semibold flex items-center gap-2" aria-label="End call"><span className="material-symbols-outlined">call_end</span>End</button>
      </div>

      {status === 'ended' && (
        <div className="absolute inset-0 bg-[#0B1C30]/80 flex items-center justify-center">
          <div className="bg-white text-gray-900 rounded-2xl p-6 text-center max-w-sm mx-4">
            <div className="text-3xl mb-2">📞</div><div className="font-bold">Call ended</div><div className="text-sm text-gray-500 mt-1">Duration {formatDuration(duration)}</div>
            <button onClick={onClose} className="mt-4 px-6 py-2 bg-indigo-600 text-white rounded-full font-semibold">Close</button>
          </div>
        </div>
      )}
    </div>
  )
}
