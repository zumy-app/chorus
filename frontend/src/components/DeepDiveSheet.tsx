import { useState } from 'react'
import { useTranslation } from 'react-i18next'

interface DeepDiveSheetProps {
  message?: {
    text: string
    sender?: { displayName?: string } | null
    analysis?: any
  } | null
  onClose: () => void
}

export default function DeepDiveSheet({ message, onClose }: DeepDiveSheetProps) {
  const { t } = useTranslation()
  const [sparkyInput, setSparkyInput] = useState('')
  const analysis = message?.analysis

  return (
    <div className="fixed inset-0 z-50 flex flex-col justify-end md:justify-center items-center bg-on-background/40 backdrop-blur-[2px] p-0 md:p-4">
      <div className="bg-surface w-full md:max-w-lg h-[85vh] md:h-auto md:max-h-[85vh] rounded-t-[2rem] md:rounded-[2rem] shadow-[0_-8px_32px_rgba(0,0,0,0.1)] overflow-hidden flex flex-col">
        {/* Drag handle */}
        <div className="w-full flex flex-col items-center pt-3 pb-2 px-4 border-b border-outline-variant">
          <div className="w-12 h-1.5 bg-outline-variant rounded-full mb-3" />
          <div className="w-full flex justify-between items-center">
            <div className="flex items-center gap-2 text-secondary">
              <span className="material-symbols-outlined">auto_awesome</span>
              <h2 className="font-headline-sm text-headline-sm text-on-surface">{t('grammar.deepDive')}</h2>
            </div>
            <button
              onClick={onClose}
              className="text-outline p-2 hover:bg-surface-container rounded-full transition"
              aria-label={t('common.close')}
            >
              <span className="material-symbols-outlined">close</span>
            </button>
          </div>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto bg-background p-4 flex flex-col gap-6">
          {/* Message context */}
          {message && message.text ? (
            <div className="bg-surface-container-lowest p-4 rounded-xl border border-outline-variant/50 shadow-sm">
              <div className="flex items-center gap-2 mb-2 text-secondary">
                <span className="material-symbols-outlined text-[18px]">auto_awesome</span>
                <span className="font-label-md text-label-md">{t('grammar.sentenceAnalysis')}</span>
              </div>
              <p className="font-body-md text-body-md text-on-surface italic">
                "{message.text}"
              </p>
            </div>
          ) : (
            <div className="flex flex-col items-center text-center gap-3 py-6">
              <div className="w-14 h-14 rounded-full bg-secondary-container text-on-secondary-container flex items-center justify-center insight-glow">
                <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>smart_toy</span>
              </div>
              <div>
                <h2 className="font-headline-sm text-headline-sm text-on-surface mb-1">{t('grammar.sparkyInsight')}</h2>
                <p className="font-body-sm text-body-sm text-on-surface-variant">{t('grammar.sparkySubtitle')}</p>
              </div>
            </div>
          )}

          {/* Grammar breakdown */}
          <div className="bg-surface-container-lowest rounded-2xl border border-primary-fixed overflow-hidden shadow-sm">
            <div className="bg-primary-fixed p-3 flex items-center gap-2">
              <span className="material-symbols-outlined text-primary text-[20px]">school</span>
              <h3 className="font-label-md text-label-md text-primary">{t('grammar.grammar')}</h3>
            </div>
            <div className="p-4 flex flex-col gap-4">
              {analysis?.summary ? (
                <>
                  <p className="font-body-sm text-body-sm text-on-surface leading-relaxed">{analysis.summary}</p>
                  {analysis.grammarNotes && analysis.grammarNotes.length > 0 && (
                    <div className="space-y-3">
                      {analysis.grammarNotes.map((note: any, i: number) => (
                        <div key={i}>
                          <p className="font-label-md text-label-md text-on-surface mb-1">{note.title}</p>
                          <div className="bg-surface-container-low p-3 rounded-lg flex flex-col gap-2">
                            <div className="flex items-start gap-2">
                              <span className="text-tertiary-container font-bold mt-0.5">✓</span>
                              <p className="font-body-sm text-body-sm text-on-surface">{note.explanation}</p>
                            </div>
                          </div>
                          {note.examples && note.examples.length > 0 && (
                            <p className="font-label-sm text-label-sm text-outline mt-1.5">
                              {t('grammar.examples')}: {note.examples.join(' · ')}
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </>
              ) : (
                <p className="font-body-sm text-body-sm text-on-surface-variant">
                  {t('grammar.askSparkyAboutThis')}
                </p>
              )}
            </div>
          </div>

          {/* Contextual tutor chat */}
          <div className="flex flex-col gap-3">
            <h3 className="font-label-md text-label-md text-on-surface-variant">{t('grammar.askSparkyAboutThis')}</h3>
            <div className="bg-surface-container-low p-3 rounded-2xl rounded-bl-sm self-start max-w-[85%]">
              <p className="font-body-sm text-body-sm text-on-surface">{t('grammar.sparkySubtitle')}</p>
            </div>
          </div>
        </div>

        {/* Tutor input */}
        <div className="p-4 bg-surface-container-lowest border-t border-outline-variant">
          <div className="flex items-end gap-2">
            <div className="relative flex-1">
              <textarea
                value={sparkyInput}
                onChange={(e) => setSparkyInput(e.target.value)}
                placeholder={t('grammar.askSparky')}
                rows={1}
                className="w-full bg-surface-container-lowest border border-outline-variant rounded-xl py-2.5 pl-4 pr-3 font-body-sm text-body-sm text-on-surface focus:outline-none focus:border-secondary focus:ring-1 focus:ring-secondary resize-none"
              />
            </div>
            <button
              disabled={!sparkyInput.trim()}
              className="bg-secondary text-white w-10 h-10 rounded-full flex items-center justify-center shadow-sm hover:bg-secondary-container transition-colors disabled:opacity-40 shrink-0"
              aria-label={t('common.send')}
            >
              <span className="material-symbols-outlined text-[20px]">send</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}