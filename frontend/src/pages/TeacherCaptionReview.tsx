import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { CaptionReviewQueueItem, CaptionQualityStats, CaptionReview } from '@chorus/shared'

export default function TeacherCaptionReview() {
  const [items, setItems] = useState<CaptionReviewQueueItem[]>([])
  const [stats, setStats] = useState<CaptionQualityStats | null>(null)
  const [selected, setSelected] = useState<CaptionReviewQueueItem | null>(null)
  const [reviews, setReviews] = useState<CaptionReview[]>([])
  const [rating, setRating] = useState(5)
  const [corrected, setCorrected] = useState('')
  const [feedback, setFeedback] = useState('')
  const [msg, setMsg] = useState('')

  const load = async () => {
    try {
      const q = await api.get<{ items: CaptionReviewQueueItem[]; total: number }>('/captions/review-queue?limit=20')
      setItems(q.data.items)
      const s = await api.get<CaptionQualityStats>('/captions/quality-stats')
      setStats(s.data)
    } catch {}
  }
  useEffect(() => { load() }, [])

  const open = async (it: CaptionReviewQueueItem) => {
    setSelected(it); setMsg('')
    try {
      const r = await api.get<{ reviews: CaptionReview[] }>(`/calls/${it.callId}/captions/${it.segmentIndex}/reviews`)
      setReviews(r.data.reviews)
      setCorrected(Object.values(it.translations)[0] || '')
    } catch { setReviews([]) }
  }

  const submit = async () => {
    if (!selected) return
    try {
      await api.post(`/calls/${selected.callId}/captions/${selected.segmentIndex}/review`, { rating, correctedText: corrected, feedback })
      setMsg('Review submitted')
      load()
      const r = await api.get<{ reviews: CaptionReview[] }>(`/calls/${selected.callId}/captions/${selected.segmentIndex}/reviews`)
      setReviews(r.data.reviews)
    } catch (e: any) { setMsg(e?.response?.data?.error || 'Failed') }
  }

  return (
    <div className="p-4 max-w-4xl mx-auto space-y-4">
      <h1 className="text-xl font-bold">Caption Translation Review</h1>
      {stats && <div className="text-sm text-gray-600">Total: {stats.totalCaptions} | Reviewed: {stats.reviewedCount} | Avg: {stats.avgRating.toFixed(2)} | Pending: {stats.pendingCount}</div>}
      <div className="grid md:grid-cols-2 gap-4">
        <div className="space-y-2 max-h-[70vh] overflow-auto">
          {items.map((it, i) => (
            <button key={i} onClick={() => open(it)} className={`w-full text-left border p-3 rounded ${selected?.callId===it.callId&&selected?.segmentIndex===it.segmentIndex?'bg-blue-50 border-blue-400':''}`}>
              <div className="text-sm font-medium truncate">{it.originalText}</div>
              <div className="text-xs text-gray-500">{it.originalLanguage} → {Object.keys(it.translations).join(',') || 'pending'} {it.avgRating!==undefined?`★${it.avgRating.toFixed(1)}`:''} ({it.reviewCount})</div>
              <div className="text-xs truncate">{Object.values(it.translations)[0]||''}</div>
            </button>
          ))}
          {items.length===0 && <div className="text-sm text-gray-500">No captions yet. Make a call and post captions.</div>}
        </div>
        <div className="border rounded p-3 space-y-3">
          {!selected ? <div className="text-sm text-gray-500">Select a caption to review</div> : (
            <>
              <div className="text-sm"><span className="font-semibold">Original ({selected.originalLanguage}):</span> {selected.originalText}</div>
              <div className="text-sm"><span className="font-semibold">Translation:</span> {Object.entries(selected.translations).map(([k,v])=>`${k}: ${v}`).join(' | ')||'—'}</div>
              <div className="flex items-center gap-1">
                <span className="text-sm">Rating:</span>
                {[1,2,3,4,5].map(n=><button key={n} onClick={()=>setRating(n)} className={`w-8 h-8 rounded ${rating>=n?'bg-yellow-400':'bg-gray-200'}`}>{n}</button>)}
              </div>
              <input value={corrected} onChange={e=>setCorrected(e.target.value)} placeholder="Corrected translation" className="w-full border rounded px-2 py-1 text-sm" />
              <textarea value={feedback} onChange={e=>setFeedback(e.target.value)} placeholder="Feedback" className="w-full border rounded px-2 py-1 text-sm" rows={2} />
              <button onClick={submit} className="bg-blue-600 text-white px-4 py-1 rounded text-sm">Submit Review</button>
              {msg && <div className="text-sm text-green-600">{msg}</div>}
              <div className="border-t pt-2">
                <div className="text-xs font-semibold">Previous reviews ({reviews.length})</div>
                {reviews.map(r=><div key={r.id} className="text-xs border-b py-1"><span className="font-medium">★{r.rating}</span> {r.correctedText||r.translatedText} <span className="text-gray-500">{r.feedback}</span></div>)}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
