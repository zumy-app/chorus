import { useCallback, useEffect, useState } from 'react'
import { learningAPI } from '../../services/api'
import { useStore } from '../../store'
import type { RealTalkPrompt } from '@chorus/shared'

type Props = {
  chatId?: string
  onSendToInput: (text: string) => void
}

export default function RealTalkNudge({ chatId, onSendToInput }: Props) {
  const user = useStore(s => s.user)
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'
  const [prompts, setPrompts] = useState<RealTalkPrompt[]>([])
  const [idx, setIdx] = useState(0)
  const [dismissed, setDismissed] = useState(false)
  const [dashboard, setDashboard] = useState<any>(null)

  useEffect(() => {
    let active = true
    learningAPI.getRealTalkPrompts(targetLanguage, nativeLanguage, chatId).then(p => { if (active) setPrompts(p) }).catch(() => {})
    learningAPI.getDashboard(targetLanguage, nativeLanguage).then(d => { if (active) setDashboard(d) }).catch(() => {})
    return () => { active = false }
  }, [targetLanguage, nativeLanguage, chatId])

  const current = prompts[idx % Math.max(1, prompts.length)]
  const unitTitle = dashboard?.currentUnit?.title

  const send = useCallback(async () => {
    if (!current) return
    try { await learningAPI.markRealTalkUsed(current.id) } catch {}
    onSendToInput(current.text)
    setDismissed(true)
  }, [current, onSendToInput])

  const next = useCallback(() => setIdx(i => (i + 1) % Math.max(1, prompts.length)), [prompts.length])

  if (dismissed || !current) return null

  return (
    <div className="bg-primary-container text-on-primary-container rounded-2xl p-4 relative border border-on-primary-container/20 shadow-[0_8px_32px_-4px_rgba(37,99,235,0.15)] mb-2">
      <button aria-label="Dismiss suggestion" onClick={() => setDismissed(true)} className="absolute top-2 right-2 p-1 text-on-primary-container/60 hover:text-on-primary-container hover:bg-on-primary-container/10 rounded-full">
        <span className="material-symbols-outlined text-[20px]">close</span>
      </button>
      <div className="flex items-start gap-3">
        <div className="bg-on-primary-container/10 p-2 rounded-full">
          <span className="material-symbols-outlined text-on-primary-container">auto_awesome</span>
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="font-label-md text-label-md text-on-primary-container">Sparky’s Nudge</span>
            <span className="w-1 h-1 rounded-full bg-on-primary-container/40" />
            <span className="font-label-sm text-label-sm text-on-primary-container/70">{unitTitle ? `Unit goal` : current.category}</span>
          </div>
          <p className="font-body-sm text-body-sm text-on-primary-container/90 mb-3">Try this in the chat:</p>
          <div className="bg-on-primary-container/10 p-3 rounded-xl border border-on-primary-container/10 mb-4">
            <p className="font-headline-sm text-headline-sm font-bold tracking-tight">“{current.text}”</p>
          </div>
          <div className="flex items-center gap-2">
            <button onClick={send} className="flex-1 bg-on-primary text-primary font-label-md text-label-md px-4 py-2.5 rounded-full shadow-sm hover:bg-surface-container-low transition flex items-center justify-center gap-2">
              Send to Input <span className="material-symbols-outlined text-[18px]">keyboard_return</span>
            </button>
            <button aria-label="Next suggestion" onClick={next} className="p-2.5 text-on-primary-container/80 hover:text-on-primary-container hover:bg-on-primary-container/10 rounded-full">
              <span className="material-symbols-outlined">shuffle</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
