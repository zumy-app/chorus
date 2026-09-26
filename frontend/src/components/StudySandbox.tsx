import { useState } from 'react'
import { useStudySandbox } from '../hooks/useStudySandbox'

export default function StudySandbox() {
  const { state, addActivity } = useStudySandbox()
  const [raised, setRaised] = useState(false)
  const [saved, setSaved] = useState(false)

  return (
    <div className="bg-surface-container-low rounded-xl p-4 shadow-sm border border-surface-variant flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-secondary">extension</span>
          <h2 className="font-headline-sm text-headline-sm text-on-surface">{state.topic}</h2>
        </div>
        <span className="bg-secondary-container text-on-secondary-container font-label-sm text-label-sm px-2 py-1 rounded-md">Sandbox</span>
      </div>
      <div className="bg-surface rounded-lg p-6 shadow-sm flex flex-col items-center text-center gap-2 relative overflow-hidden">
        <div className="absolute inset-0 opacity-5" style={{ backgroundImage: 'radial-gradient(#004ac6 1px, transparent 1px)', backgroundSize: '20px 20px' }} />
        <p className="font-headline-md text-headline-md text-on-surface z-10">“{state.prompt}”</p>
        <p className="font-body-sm text-body-sm text-on-surface-variant z-10">Focus on using past tense verbs. Group sandbox — everyone sees the same prompt.</p>
      </div>
      <div className="flex items-center justify-between pt-2">
        <button
          onClick={() => { setRaised(v => !v); addActivity('You', raised ? 'lowered hand' : 'raised hand') }}
          className={`${raised ? 'bg-secondary-container text-on-secondary-container' : 'bg-primary text-on-primary'} font-label-md text-label-md px-4 py-2 rounded-full flex items-center gap-2 shadow-sm`}
        >
          <span className="material-symbols-outlined text-[18px]">front_hand</span> {raised ? 'Hand raised' : 'Raise Hand'}
        </button>
        <button
          onClick={() => { setSaved(true); addActivity('You', 'saved a word'); setTimeout(() => setSaved(false), 1500) }}
          className="bg-surface text-primary border border-outline-variant font-label-md text-label-md px-3 py-2 rounded-full flex items-center gap-1 shadow-sm"
        >
          <span className="material-symbols-outlined text-[18px]">bookmark_add</span> {saved ? 'Saved!' : 'Save Word'}
        </button>
      </div>
      {state.activityLog.length > 0 && (
        <div className="flex flex-col gap-1 max-h-20 overflow-y-auto">
          {state.activityLog.slice(-3).map(e => (
            <span key={e.id} className="font-label-sm text-label-sm text-on-surface-variant">{e.user}: {e.text}</span>
          ))}
        </div>
      )}
    </div>
  )
}
