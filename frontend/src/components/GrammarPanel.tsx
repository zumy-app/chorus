import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import LearningPanel from './LearningPanel'

interface GrammarPanelProps {
  analysis: any
  nativeLanguage: string
  messageText: string
  messageLanguage: string
  onClose: () => void
  providerUsed?: string
}

const typeColors: Record<string, string> = {
  verb: 'bg-primary-fixed text-primary',
  noun: 'bg-tertiary-fixed/40 text-tertiary-container',
  pronoun: 'bg-secondary-fixed text-secondary',
  preposition: 'bg-surface-container-high text-on-surface',
  article: 'bg-surface-variant text-on-surface-variant',
  adjective: 'bg-tertiary-fixed/30 text-tertiary-container',
  adverb: 'bg-primary-container/20 text-primary',
  conjunction: 'bg-error-container/50 text-error',
  phrase: 'bg-surface-container text-on-surface-variant',
}

export default function GrammarPanel({ analysis, nativeLanguage, messageText, messageLanguage, onClose, providerUsed }: GrammarPanelProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<'overview' | 'wordByWord' | 'grammar'>('overview')
  const [showLearning, setShowLearning] = useState(false)
  const [expandedWord, setExpandedWord] = useState<number | null>(null)

  const tabs = [
    { id: 'overview' as const, label: t('grammar.overview'), icon: '💡' },
    { id: 'wordByWord' as const, label: t('grammar.wordByWord'), icon: '🔤' },
    { id: 'grammar' as const, label: t('grammar.grammar'), icon: '📝' },
  ]

  return (
    <div data-testid="grammar-panel" className="mt-1 bg-surface-container-lowest border border-outline-variant/30 rounded-xl shadow-[0px_4px_12px_rgba(0,0,0,0.05)] max-w-md overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 pt-3 pb-1">
        <div className="flex items-center gap-2">
          <div className="w-6 h-6 rounded-full bg-secondary-container text-on-secondary-container flex items-center justify-center insight-glow shrink-0">
            <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>smart_toy</span>
          </div>
          <span className="font-label-sm text-label-sm text-on-surface">{t('grammar.sparkyInsight')}</span>
          {analysis.difficulty && analysis.difficulty !== 'N/A' && (
            <span data-testid="grammar-difficulty-badge" className="text-[10px] bg-secondary-fixed text-secondary px-1.5 py-0.5 rounded-full font-bold">
              {analysis.difficulty}
            </span>
          )}
          {providerUsed && providerUsed !== 'cache' && (
            <span data-testid="grammar-provider-badge" className="text-[9px] bg-surface-variant text-on-surface-variant px-1.5 py-0.5 rounded-full">
              {providerUsed === 'regex-fallback' ? '📊 regex' : `⚡ ${providerUsed}`}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setShowLearning(true)}
            className="text-[11px] px-2 py-0.5 bg-primary-container/20 text-primary rounded-full hover:bg-primary-container/30 transition font-semibold"
          >
            🤖 {t('grammar.aiTutor')}
          </button>
          <button onClick={onClose} className="text-outline hover:text-on-surface text-lg leading-none ml-1" aria-label={t('common.close')}>×</button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-outline-variant/30 mx-2 mt-1">
        {tabs.map(tab => (
          <button
            key={tab.id}
            data-testid={`grammar-tab-${tab.id}`}
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 text-[11px] font-semibold py-1.5 px-2 transition border-b-2 ${
              activeTab === tab.id
                ? 'border-secondary text-secondary'
                : 'border-transparent text-on-surface-variant hover:text-primary'
            }`}
          >
            <span className="mr-1">{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className="p-3 max-h-80 overflow-y-auto">
        {activeTab === 'overview' && (
          <div className="space-y-3">
            {/* Summary */}
            {analysis.summary && (
              <div className="bg-secondary-fixed/40 border-l-[3px] border-secondary p-3 rounded-r-lg insight-glow">
                <p className="font-body-sm text-body-sm text-on-surface leading-relaxed">{analysis.summary}</p>
              </div>
            )}

            {/* Sentence Structure */}
            {analysis.sentenceStructure && (
              <div className="bg-surface-container-low rounded-lg p-2.5 border border-outline-variant/30">
                <div className="font-label-sm text-label-sm text-primary uppercase tracking-wide mb-1">
                  {t('grammar.howItsBuilt')}
                </div>
                <p className="font-body-sm text-body-sm text-on-surface leading-relaxed">{analysis.sentenceStructure}</p>
              </div>
            )}

            {/* Key Phrases */}
            {analysis.keyPhrases && analysis.keyPhrases.length > 0 && (
              <div>
                <div className="font-label-sm text-label-sm text-primary uppercase tracking-wide mb-1.5">
                  {t('grammar.keyPhrases')}
                </div>
                <div className="space-y-1.5">
                  {analysis.keyPhrases.map((kp: any, i: number) => (
                    <div key={i} className="bg-surface-container-low rounded-lg p-2 border border-outline-variant/30">
                      <div className="flex items-baseline gap-2">
                        <span className="font-semibold text-xs text-on-surface">{kp.phrase}</span>
                        <span className="text-xs text-secondary">→</span>
                        <span className="text-xs text-on-surface-variant italic">{kp.translation}</span>
                      </div>
                      {kp.context && (
                        <p className="text-[11px] text-on-surface-variant mt-0.5">{kp.context}</p>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Fallback: if no summary and no key phrases */}
            {!analysis.summary && (!analysis.keyPhrases || analysis.keyPhrases.length === 0) && (
              <p className="text-xs text-on-surface-variant italic">{t('grammar.noData')}</p>
            )}

            {/* Practice CTA */}
            <button className="w-full bg-primary text-on-primary py-3 px-4 rounded-full font-label-md text-label-md flex justify-center items-center gap-2 shadow-sm hover:bg-primary/90 transition active:scale-[0.98]">
              <span className="material-symbols-outlined text-[18px]">model_training</span>
              {t('grammar.practiceNow')}
            </button>
          </div>
        )}

        {activeTab === 'wordByWord' && (
          <div data-testid="grammar-wordbyword" className="space-y-1">
            {analysis.detailedBreakdown && analysis.detailedBreakdown.length > 0 ? (
              analysis.detailedBreakdown.map((item: any, i: number) => {
                const isExpanded = expandedWord === i
                const badgeClass = typeColors[item.type] || 'bg-surface-container text-on-surface-variant'
                return (
                  <div
                    key={i}
                    className="bg-surface-container-low rounded-lg border border-outline-variant/30 overflow-hidden cursor-pointer hover:bg-surface-container transition"
                    onClick={() => setExpandedWord(isExpanded ? null : i)}
                  >
                    <div className="flex items-center gap-2 p-2">
                      <span className="font-semibold text-xs text-on-surface min-w-[60px]">{item.text}</span>
                      <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${badgeClass}`}>
                        {item.role || item.type}
                      </span>
                      {item.translation && (
                        <span className="text-xs text-on-surface-variant italic ml-auto">{item.translation}</span>
                      )}
                      <span className={`text-secondary text-[10px] transition-transform ${isExpanded ? 'rotate-90' : ''}`}>▶</span>
                    </div>
                    {isExpanded && (
                      <div className="px-2 pb-2 border-t border-outline-variant/30 pt-1.5">
                        {item.translation && (
                          <div className="text-xs text-on-surface mb-1">
                            <span className="font-semibold text-primary">{t('grammar.meaning')}</span> {item.translation}
                          </div>
                        )}
                        {item.note && (
                          <div className="text-[11px] text-on-surface-variant">
                            <span className="font-semibold text-primary">{t('grammar.note')}</span> {item.note}
                          </div>
                        )}
                        {item.explanation && !item.translation && (
                          <div className="text-xs text-on-surface">{item.explanation}</div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })
            ) : (
              <p className="text-xs text-on-surface-variant italic">{t('grammar.noData')}</p>
            )}
          </div>
        )}

        {activeTab === 'grammar' && (
          <div className="space-y-2">
            {analysis.grammarNotes && analysis.grammarNotes.length > 0 ? (
              analysis.grammarNotes.map((note: any, i: number) => (
                <div key={i} className="bg-surface-container-low rounded-lg p-2.5 border border-outline-variant/30">
                  <div className="font-semibold text-xs text-secondary mb-1">{note.title}</div>
                  <p className="text-[11px] text-on-surface leading-relaxed mb-1.5">{note.explanation}</p>
                  {note.examples && note.examples.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {note.examples.map((ex: string, j: number) => (
                        <span key={j} className="text-[10px] bg-primary-fixed text-primary px-1.5 py-0.5 rounded font-semibold">
                          {ex}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))
            ) : (
              <p className="text-xs text-on-surface-variant italic">{t('grammar.noData')}</p>
            )}
          </div>
        )}
      </div>

      {/* Learning Panel */}
      {showLearning && (
        <div className="mt-1 border-t border-outline-variant/30">
          <LearningPanel
            text={messageText}
            language={messageLanguage}
            nativeLanguage={nativeLanguage}
            onClose={() => setShowLearning(false)}
          />
        </div>
      )}
    </div>
  )
}