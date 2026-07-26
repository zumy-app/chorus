import { useState } from 'react'
import LearningPanel from './LearningPanel'

interface GrammarPanelProps {
  analysis: any
  nativeLanguage: string
  messageText: string
  messageLanguage: string
  onClose: () => void
}

const labels: Record<string, Record<string, string>> = {
  en: { overview: 'Overview', wordByWord: 'Word by Word', grammar: 'Grammar', aiTutor: 'AI Tutor', keyPhrases: 'Key Phrases', sentenceStructure: 'How it\'s built', noData: 'No data available', context: 'When to use' },
  es: { overview: 'Resumen', wordByWord: 'Palabra por Palabra', grammar: 'Gramática', aiTutor: 'Tutor IA', keyPhrases: 'Frases Clave', sentenceStructure: 'Cómo se construye', noData: 'Sin datos', context: 'Cuándo usar' },
  fr: { overview: 'Aperçu', wordByWord: 'Mot à Mot', grammar: 'Grammaire', aiTutor: 'Tuteur IA', keyPhrases: 'Phrases Clés', sentenceStructure: 'Comment c\'est construit', noData: 'Pas de données', context: 'Quand utiliser' },
  de: { overview: 'Übersicht', wordByWord: 'Wort für Wort', grammar: 'Grammatik', aiTutor: 'KI-Tutor', keyPhrases: 'Schlüsselphrasen', sentenceStructure: 'Wie es aufgebaut ist', noData: 'Keine Daten', context: 'Wann verwenden' },
}

function t(key: string, lang: string): string {
  return labels[lang]?.[key] || labels.en[key] || key
}

const typeColors: Record<string, string> = {
  verb: 'bg-blue-100 text-blue-700',
  noun: 'bg-green-100 text-green-700',
  pronoun: 'bg-pink-100 text-pink-700',
  preposition: 'bg-orange-100 text-orange-700',
  article: 'bg-teal-100 text-teal-700',
  adjective: 'bg-yellow-100 text-yellow-700',
  adverb: 'bg-indigo-100 text-indigo-700',
  conjunction: 'bg-red-100 text-red-700',
  phrase: 'bg-gray-100 text-gray-600',
}

export default function GrammarPanel({ analysis, nativeLanguage, messageText, messageLanguage, onClose }: GrammarPanelProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'wordByWord' | 'grammar'>('overview')
  const [showLearning, setShowLearning] = useState(false)
  const [expandedWord, setExpandedWord] = useState<number | null>(null)

  const tabs = [
    { id: 'overview' as const, label: t('overview', nativeLanguage), icon: '💡' },
    { id: 'wordByWord' as const, label: t('wordByWord', nativeLanguage), icon: '🔤' },
    { id: 'grammar' as const, label: t('grammar', nativeLanguage), icon: '📝' },
  ]

  return (
    <div className="mt-1 bg-gradient-to-br from-amber-50 to-yellow-50 border border-amber-200 rounded-lg shadow-sm max-w-md">
      {/* Header */}
      <div className="flex items-center justify-between px-3 pt-2.5 pb-1">
        <div className="flex items-center gap-1.5">
          <span className="text-sm">📖</span>
          {analysis.difficulty && analysis.difficulty !== 'N/A' && (
            <span className="text-[10px] bg-amber-200 text-amber-800 px-1.5 py-0.5 rounded-full font-bold">
              {analysis.difficulty}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setShowLearning(true)}
            className="text-[11px] px-2 py-0.5 bg-indigo-100 text-indigo-700 rounded hover:bg-indigo-200 transition font-medium"
          >
            🤖 {t('aiTutor', nativeLanguage)}
          </button>
          <button onClick={onClose} className="text-amber-600 hover:text-amber-800 text-lg leading-none ml-1">×</button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-amber-200 mx-2">
        {tabs.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 text-[11px] font-medium py-1.5 px-2 transition border-b-2 ${
              activeTab === tab.id
                ? 'border-amber-500 text-amber-800'
                : 'border-transparent text-amber-600 hover:text-amber-800'
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
              <p className="text-xs text-amber-900 leading-relaxed">{analysis.summary}</p>
            )}

            {/* Sentence Structure */}
            {analysis.sentenceStructure && (
              <div className="bg-white/70 rounded-lg p-2.5 border border-amber-100">
                <div className="text-[10px] font-semibold text-amber-700 uppercase tracking-wide mb-1">
                  {t('sentenceStructure', nativeLanguage)}
                </div>
                <p className="text-xs text-amber-900 leading-relaxed">{analysis.sentenceStructure}</p>
              </div>
            )}

            {/* Key Phrases */}
            {analysis.keyPhrases && analysis.keyPhrases.length > 0 && (
              <div>
                <div className="text-[10px] font-semibold text-amber-700 uppercase tracking-wide mb-1.5">
                  {t('keyPhrases', nativeLanguage)}
                </div>
                <div className="space-y-1.5">
                  {analysis.keyPhrases.map((kp: any, i: number) => (
                    <div key={i} className="bg-white/70 rounded-lg p-2 border border-amber-100">
                      <div className="flex items-baseline gap-2">
                        <span className="font-semibold text-xs text-gray-900">{kp.phrase}</span>
                        <span className="text-xs text-amber-700">→</span>
                        <span className="text-xs text-gray-700 italic">{kp.translation}</span>
                      </div>
                      {kp.context && (
                        <p className="text-[11px] text-gray-500 mt-0.5">{kp.context}</p>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Fallback: if no summary and no key phrases */}
            {!analysis.summary && (!analysis.keyPhrases || analysis.keyPhrases.length === 0) && (
              <p className="text-xs text-amber-600 italic">{t('noData', nativeLanguage)}</p>
            )}
          </div>
        )}

        {activeTab === 'wordByWord' && (
          <div className="space-y-1">
            {analysis.detailedBreakdown && analysis.detailedBreakdown.length > 0 ? (
              analysis.detailedBreakdown.map((item: any, i: number) => {
                const isExpanded = expandedWord === i
                const badgeClass = typeColors[item.type] || 'bg-gray-100 text-gray-600'
                return (
                  <div
                    key={i}
                    className="bg-white/70 rounded-lg border border-amber-100 overflow-hidden cursor-pointer hover:bg-white/90 transition"
                    onClick={() => setExpandedWord(isExpanded ? null : i)}
                  >
                    <div className="flex items-center gap-2 p-2">
                      <span className="font-semibold text-xs text-gray-900 min-w-[60px]">{item.text}</span>
                      <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${badgeClass}`}>
                        {item.role || item.type}
                      </span>
                      {item.translation && (
                        <span className="text-xs text-gray-600 italic ml-auto">{item.translation}</span>
                      )}
                      <span className={`text-amber-400 text-[10px] transition-transform ${isExpanded ? 'rotate-90' : ''}`}>▶</span>
                    </div>
                    {isExpanded && (
                      <div className="px-2 pb-2 border-t border-amber-100 pt-1.5">
                        {item.translation && (
                          <div className="text-xs text-gray-700 mb-1">
                            <span className="font-medium text-amber-700">Meaning:</span> {item.translation}
                          </div>
                        )}
                        {item.note && (
                          <div className="text-[11px] text-gray-600">
                            <span className="font-medium text-amber-700">Note:</span> {item.note}
                          </div>
                        )}
                        {item.explanation && !item.translation && (
                          <div className="text-xs text-gray-700">{item.explanation}</div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })
            ) : (
              <p className="text-xs text-amber-600 italic">{t('noData', nativeLanguage)}</p>
            )}
          </div>
        )}

        {activeTab === 'grammar' && (
          <div className="space-y-2">
            {analysis.grammarNotes && analysis.grammarNotes.length > 0 ? (
              analysis.grammarNotes.map((note: any, i: number) => (
                <div key={i} className="bg-white/70 rounded-lg p-2.5 border border-amber-100">
                  <div className="font-semibold text-xs text-amber-800 mb-1">{note.title}</div>
                  <p className="text-[11px] text-gray-700 leading-relaxed mb-1.5">{note.explanation}</p>
                  {note.examples && note.examples.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {note.examples.map((ex: string, j: number) => (
                        <span key={j} className="text-[10px] bg-amber-100 text-amber-800 px-1.5 py-0.5 rounded font-medium">
                          {ex}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))
            ) : (
              <p className="text-xs text-amber-600 italic">{t('noData', nativeLanguage)}</p>
            )}
          </div>
        )}
      </div>

      {/* Learning Panel */}
      {showLearning && (
        <div className="mt-1">
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
