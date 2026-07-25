import { useState, useEffect, useRef } from 'react'
import { formatDistanceToNow } from 'date-fns'
import type { Message } from '../types'
import { vocabularyAPI, grammarAPI } from '../services/api'
import LearningPanel from './LearningPanel'

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  nativeLanguage: string
  targetLanguage?: string
}

// Localized labels for grammar UI based on user's native language
const grammarLabels: Record<string, Record<string, string>> = {
  en: { grammar: 'Grammar', patterns: 'Patterns', wordByWord: 'Word-by-Word', aiTutor: 'AI Tutor', analyzing: 'Analyzing...', inYourLang: 'In your language:', translating: 'Translating...' },
  es: { grammar: 'Gramática', patterns: 'Patrones', wordByWord: 'Palabra por Palabra', aiTutor: 'Tutor IA', analyzing: 'Analizando...', inYourLang: 'En tu idioma:', translating: 'Traduciendo...' },
  fr: { grammar: 'Grammaire', patterns: 'Règles', wordByWord: 'Mot à Mot', aiTutor: 'Tuteur IA', analyzing: 'Analyse...', inYourLang: 'Dans votre langue:', translating: 'Traduction...' },
  de: { grammar: 'Grammatik', patterns: 'Muster', wordByWord: 'Wort für Wort', aiTutor: 'KI-Tutor', analyzing: 'Analysiere...', inYourLang: 'In Ihrer Sprache:', translating: 'Übersetzen...' },
  pt: { grammar: 'Gramática', patterns: 'Padrões', wordByWord: 'Palavra por Palavra', aiTutor: 'Tutor IA', analyzing: 'Analisando...', inYourLang: 'No seu idioma:', translating: 'Traduzindo...' },
  it: { grammar: 'Grammatica', patterns: 'Schemi', wordByWord: 'Parola per Parola', aiTutor: 'Tutor IA', analyzing: 'Analizzando...', inYourLang: 'Nella tua lingua:', translating: 'Traducendo...' },
}

function getLabel(key: string, lang: string): string {
  return grammarLabels[lang]?.[key] || grammarLabels.en[key] || key
}

// Human-readable pattern descriptions in each language (for regex fallback patterns)
const patternDescriptions: Record<string, Record<string, string>> = {
  present_continuous: {
    en: 'Actions happening now or temporary situations. Formed with "am/is/are + verb-ing".',
    es: 'Acciones que ocurren ahora o situaciones temporales. Se forma con "am/is/are + verbo-ing".',
  },
  past_tense: {
    en: 'Completed actions in the past. Regular verbs add "-ed".',
    es: 'Acciones completadas en el pasado. Los verbos regulares añaden "-ed".',
  },
  future_tense: {
    en: 'Actions that will happen. Uses "will + verb" or "going to + verb".',
    es: 'Acciones que sucederán. Usa "will + verbo" o "going to + verbo".',
  },
  conditional: {
    en: 'Hypothetical situations. Often uses "if" with "would/could/should".',
    es: 'Situaciones hipotéticas. Usa "if" con "would/could/should".',
  },
  passive_voice: {
    en: 'Emphasizes the action rather than the doer. Formed with "be + past participle".',
    es: 'Enfatiza la acción en lugar de quien la realiza. Se forma con "be + participio pasado".',
  },
  question: {
    en: 'Inverts subject and verb, or uses question words (who, what, when, etc.).',
    es: 'Invierte el sujeto y el verbo, o usa palabras interrogativas (quién, qué, cuándo, etc.).',
  },
  present_perfect: {
    en: 'Connects past action to present. Uses "have/has + past participle".',
    es: 'Conecta una acción pasada con el presente. Usa "have/has + participio pasado".',
  },
  comparison: {
    en: 'Compares qualities. Uses "-er/-est" or "more/most".',
    es: 'Compara cualidades. Usa "-er/-est" o "more/most".',
  },
  presente: {
    en: 'Verb tense for current or habitual actions.',
    es: 'Tiempo verbal para acciones actuales o habituales.',
  },
  preterito: {
    en: 'Verb tense for completed past actions.',
    es: 'Tiempo verbal para acciones completadas en el pasado.',
  },
  subjuntivo: {
    en: 'Verb mood to express wishes, doubts, or hypothetical situations.',
    es: 'Modo verbal para expresar deseos, dudas o situaciones hipotéticas.',
  },
  verbos_reflexivos: {
    en: 'Verbs where subject and object are the same person. Uses "me, te, se, nos, os".',
    es: 'Verbos donde el sujeto y el objeto son la misma persona. Usan "me, te, se, nos, os".',
  },
}

function getPatternDesc(pattern: string, lang: string): string {
  return patternDescriptions[pattern]?.[lang] || patternDescriptions[pattern]?.en || ''
}

export default function MessageBubble({ message, isOwn, nativeLanguage, targetLanguage }: MessageBubbleProps) {
  const [showActions, setShowActions] = useState(false)
  const [savedWord, setSavedWord] = useState<string | null>(null)
  const [showGrammar, setShowGrammar] = useState(false)
  const [grammarAnalysis, setGrammarAnalysis] = useState<any>(null)
  const [loadingGrammar, setLoadingGrammar] = useState(false)
  const [showLearning, setShowLearning] = useState(false)

  const nativeTranslation = message.translations?.[nativeLanguage]
  const showNativeTranslation = nativeTranslation && nativeTranslation !== message.text

  // Show translation pending indicator only for recent messages still awaiting translation
  const isTranslationPending = !isOwn && !nativeTranslation && (
    !message.translations || Object.keys(message.translations).length === 0
  )

  // Show Learning section only when target language is DIFFERENT from both native AND original
  const targetTranslation = targetLanguage && targetLanguage !== nativeLanguage && targetLanguage !== message.originalLanguage
    ? message.translations?.[targetLanguage]
    : null
  const showTargetTranslation = targetTranslation && targetTranslation !== message.text

  // Re-run AI grammar analysis when translations arrive (WebSocket update)
  const prevNativeTranslation = useRef(nativeTranslation)
  useEffect(() => {
    if (showGrammar && grammarAnalysis && prevNativeTranslation.current !== nativeTranslation) {
      handleAnalyzeGrammar(true)
    }
    prevNativeTranslation.current = nativeTranslation
  }, [nativeTranslation])

  const handleSaveWord = async (word: string) => {
    try {
      await vocabularyAPI.save(word, message.originalLanguage || nativeLanguage, message.id)
      setSavedWord(word)
      setTimeout(() => setSavedWord(null), 2000)
    } catch (err) {
      console.error('Failed to save word:', err)
    }
  }

  const handleAnalyzeGrammar = async (silent = false) => {
    if (grammarAnalysis && !silent) {
      setShowGrammar(!showGrammar)
      return
    }
    if (!silent) setLoadingGrammar(true)
    try {
      // Always analyze the ORIGINAL message text.
      // Use the sender's native language as fallback when originalLanguage is not set.
      const sourceLang = message.originalLanguage || message.sender?.nativeLanguage || 'en'
      const response = await grammarAPI.analyzeAI(message.text, sourceLang, nativeLanguage)
      setGrammarAnalysis(response.analysis || response)
      if (!silent) setShowGrammar(true)
    } catch (err) {
      console.error('AI grammar analysis failed, falling back to regex:', err)
      // Fallback to regex analysis
      try {
        const sourceLang = message.originalLanguage || message.sender?.nativeLanguage || 'en'
        const response = await grammarAPI.analyze(message.text, sourceLang, nativeLanguage)
        setGrammarAnalysis(response.analysis || response)
        if (!silent) setShowGrammar(true)
      } catch (fallbackErr) {
        console.error('Grammar analysis failed:', fallbackErr)
      }
    } finally {
      if (!silent) setLoadingGrammar(false)
    }
  }

    // Use original message text for word extraction (language being learned)
  const words = (message.text || '')
    .split(/\s+/)
    .filter((w: string) => w.length > 3)
    .slice(0, 5)

  return (
    <div
      className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => setShowActions(false)}
    >
      <div className={`max-w-[70%] ${isOwn ? 'items-end' : 'items-start'}`}>
        <div
          className={`rounded-lg px-4 py-2 ${
            isOwn
              ? 'bg-primary text-white'
              : 'bg-white text-gray-900 border border-gray-200'
          }`}
        >
          {!isOwn && message.sender && (
            <div className="text-xs font-semibold mb-1 opacity-75">
              {message.sender.displayName}
            </div>
          )}
          
          <div className="break-words whitespace-pre-wrap">
            {message.text}
          </div>

          {/* Translation loading indicator */}
          {isTranslationPending && (
            <div className={`mt-2 pt-2 border-t ${isOwn ? 'border-white/30' : 'border-gray-200'} text-sm`}>
              <div className={`text-xs mb-1 ${isOwn ? 'text-white/75' : 'text-gray-500'}`}>
                🌐 {getLabel('translating', nativeLanguage)}
              </div>
              <div className="flex items-center gap-1.5">
                <span className={`inline-block w-1.5 h-1.5 rounded-full ${isOwn ? 'bg-white/60' : 'bg-gray-400'} animate-pulse`}
                      style={{ animationDelay: '0ms' }} />
                <span className={`inline-block w-1.5 h-1.5 rounded-full ${isOwn ? 'bg-white/60' : 'bg-gray-400'} animate-pulse`}
                      style={{ animationDelay: '300ms' }} />
                <span className={`inline-block w-1.5 h-1.5 rounded-full ${isOwn ? 'bg-white/60' : 'bg-gray-400'} animate-pulse`}
                      style={{ animationDelay: '600ms' }} />
              </div>
            </div>
          )}

          {/* Translation in native language (for comprehension) */}
          {showNativeTranslation && (
            <div className={`mt-2 pt-2 border-t ${isOwn ? 'border-white/30' : 'border-gray-200'} text-sm`}>
              <div className={`text-xs mb-1 flex items-center gap-1 ${isOwn ? 'text-white/75' : 'text-gray-500'}`}>
                <span>🌐 {getLabel('inYourLang', nativeLanguage)}</span>
                {!isOwn && !message.translationEnhanced && (
                  <span className="inline-flex items-center text-[10px] text-amber-500 animate-pulse" title="AI-enhanced translation pending">
                    ✨
                  </span>
                )}
              </div>
              <div className={`italic font-medium whitespace-pre-wrap break-words ${isOwn ? 'text-white' : 'text-gray-800'}`}>
                {nativeTranslation}
              </div>
            </div>
          )}

          {/* Translation in target language (for learning) */}
          {showTargetTranslation && (
            <div className={`mt-2 pt-2 border-t ${isOwn ? 'border-white/30' : 'border-gray-200'} text-sm`}>
              <div className={`text-xs mb-1 ${isOwn ? 'text-white/75' : 'text-gray-500'}`}>
                📖 Learning ({targetLanguage?.toUpperCase()}):
              </div>
              <div className={`italic opacity-90 whitespace-pre-wrap break-words ${isOwn ? 'text-white/90' : 'text-gray-600'}`}>
                {targetTranslation}
              </div>
            </div>
          )}

          <div className={`text-xs mt-1 ${isOwn ? 'text-white/75' : 'text-gray-500'}`}>
            {formatDistanceToNow(new Date(message.timestamp), { addSuffix: true })}
          </div>
        </div>

        {/* AI-Powered Grammar Analysis */}
        {showGrammar && grammarAnalysis && (
          <div className="mt-1 bg-gradient-to-br from-amber-50 to-yellow-50 border border-amber-200 rounded-lg p-3 text-sm max-w-sm shadow-sm">
            <div className="flex items-center justify-between mb-2">
              <span className="font-semibold text-amber-800 flex items-center gap-1.5">
                <span>📝</span> {getLabel('grammar', nativeLanguage)}
                {grammarAnalysis.difficulty && grammarAnalysis.difficulty !== 'N/A' && (
                  <span className="text-[10px] bg-amber-200 text-amber-800 px-1.5 py-0.5 rounded-full font-bold">
                    {grammarAnalysis.difficulty}
                  </span>
                )}
              </span>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => setShowLearning(true)}
                  className="text-[11px] px-2 py-0.5 bg-indigo-100 text-indigo-700 rounded hover:bg-indigo-200 transition font-medium"
                >
                  🤖 {getLabel('aiTutor', nativeLanguage)}
                </button>
                <button onClick={() => setShowGrammar(false)} className="text-amber-600 hover:text-amber-800 text-lg leading-none ml-1">×</button>
              </div>
            </div>

            {grammarAnalysis.summary && (
              <p className="text-amber-900 text-xs mb-2 leading-relaxed">{grammarAnalysis.summary}</p>
            )}

            {grammarAnalysis.patterns?.length > 0 && (
              <div className="space-y-1.5 mb-2">
                <p className="text-[11px] font-semibold text-amber-700 uppercase tracking-wide">{getLabel('patterns', nativeLanguage)}</p>
                <div className="flex flex-wrap gap-1">
                  {grammarAnalysis.patterns.map((p: any, i: number) => {
                    // String patterns = regex fallback — show with description
                    if (typeof p === 'string') {
                      const desc = getPatternDesc(p, nativeLanguage)
                      return (
                        <div key={i} className="w-full bg-white/80 rounded-lg p-2 border border-amber-100">
                          <div className="flex items-center justify-between">
                            <span className="font-semibold text-amber-900 text-xs capitalize">{p.replace(/_/g, ' ')}</span>
                          </div>
                          {desc && (
                            <p className="text-[11px] text-amber-800 mt-0.5">{desc}</p>
                          )}
                        </div>
                      )
                    }
                    // Object patterns = AI analysis — show with description
                    return (
                      <div key={i} className="w-full bg-white/80 rounded-lg p-2 border border-amber-100">
                        <div className="flex items-center justify-between">
                          <span className="font-semibold text-amber-900 text-xs capitalize">{p.name}</span>
                          {p.example && (
                            <span className="text-[10px] text-amber-600 italic ml-2">e.g. "{p.example}"</span>
                          )}
                        </div>
                        {p.description && (
                          <p className="text-[11px] text-amber-800 mt-0.5">{p.description}</p>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            {grammarAnalysis.detailedBreakdown?.length > 0 && (
              <div className="space-y-1">
                <p className="text-[11px] font-semibold text-amber-700 uppercase tracking-wide">{getLabel('wordByWord', nativeLanguage)}</p>
                <div className="flex flex-wrap gap-x-2 gap-y-1">
                  {grammarAnalysis.detailedBreakdown.map((item: any, i: number) => {
                    const badgeColorMap: Record<string, string> = {
                      verb: 'bg-blue-100 text-blue-700',
                      tense: 'bg-purple-100 text-purple-700',
                      noun: 'bg-green-100 text-green-700',
                      pronoun: 'bg-pink-100 text-pink-700',
                      preposition: 'bg-orange-100 text-orange-700',
                      article: 'bg-teal-100 text-teal-700',
                      adjective: 'bg-yellow-100 text-yellow-700',
                      adverb: 'bg-indigo-100 text-indigo-700',
                      conjunction: 'bg-red-100 text-red-700',
                      phrase: 'bg-gray-100 text-gray-600',
                    }
                    const badgeClass = badgeColorMap[item.type] || 'bg-gray-100 text-gray-600'
                    return (
                      <div
                        key={i}
                        className="w-full bg-white/80 rounded p-1.5 border border-amber-100"
                      >
                        <div className="flex items-baseline gap-1 flex-wrap">
                          <span className="font-semibold text-gray-900 text-xs">{item.text}</span>
                          <span className={`text-[10px] px-1 rounded font-medium ${badgeClass}`}>
                            {item.type || 'w'}
                          </span>
                        </div>
                        {item.explanation && (
                          <p className="text-[11px] text-gray-600 mt-0.5 leading-relaxed">
                            {item.explanation}
                          </p>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Inline Learning Panel — always uses original text, AI explains in native language */}
        {showLearning && (
          <div className="mt-1">
            <LearningPanel
              text={message.text}
              language={message.originalLanguage || message.sender?.nativeLanguage || 'en'}
              nativeLanguage={nativeLanguage}
              onClose={() => setShowLearning(false)}
            />
          </div>
        )}

        {/* Action Buttons */}
        {showActions && !isOwn && (
          <div className="flex flex-wrap gap-1 mt-1">
            <button
              onClick={() => handleAnalyzeGrammar()}
              disabled={loadingGrammar}
              className="text-xs px-2 py-1 bg-amber-100 text-amber-700 rounded hover:bg-amber-200 transition disabled:opacity-50"
            >
              {loadingGrammar ? (
                <span className="flex items-center gap-1">
                  <span className="inline-block w-1 h-1 bg-amber-600 rounded-full animate-pulse" />
                  {getLabel('analyzing', nativeLanguage)}
                </span>
              ) : (
                `📝 ${getLabel('grammar', nativeLanguage)}`
              )}
            </button>
            {words.slice(0, 3).map((word: string) => (
              <button
                key={word}
                onClick={() => handleSaveWord(word)}
                className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 transition"
                title={`Save "${word}" to vocabulary`}
              >
                {savedWord === word ? '✅ Saved' : `+ ${word.substring(0, 12)}`}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
