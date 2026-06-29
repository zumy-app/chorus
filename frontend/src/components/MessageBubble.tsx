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
        const response = await grammarAPI.analyze(message.text, sourceLang)
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
  const words = message.text
    ?.split(/\s+/)
    .filter((w: string) => w.length > 3)
    .slice(0, 5) || []

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
                🌐 Translating...
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
              <div className={`text-xs mb-1 ${isOwn ? 'text-white/75' : 'text-gray-500'}`}>
                🌐 In your language:
              </div>
              <div className={`italic font-medium ${isOwn ? 'text-white' : 'text-gray-800'}`}>
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
              <div className={`italic opacity-90 ${isOwn ? 'text-white/90' : 'text-gray-600'}`}>
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
                <span>📝</span> Grammar
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
                  Learn More →
                </button>
                <button onClick={() => setShowGrammar(false)} className="text-amber-600 hover:text-amber-800 text-lg leading-none ml-1">×</button>
              </div>
            </div>

            {grammarAnalysis.summary && (
              <p className="text-amber-900 text-xs mb-2 leading-relaxed">{grammarAnalysis.summary}</p>
            )}

            {grammarAnalysis.patterns?.length > 0 && (
              <div className="space-y-1.5 mb-2">
                <p className="text-[11px] font-semibold text-amber-700 uppercase tracking-wide">Patterns</p>
                <div className="flex flex-wrap gap-1">
                  {grammarAnalysis.patterns.map((p: any, i: number) => {
                    // String patterns = regex fallback — show as simple tags
                    if (typeof p === 'string') {
                      return (
                        <span key={i} className="text-xs bg-amber-100 text-amber-700 px-2 py-1 rounded-full capitalize">
                          {p.replace(/_/g, ' ')}
                        </span>
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
                <p className="text-[11px] font-semibold text-amber-700 uppercase tracking-wide">Word-by-Word</p>
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
                    // Truncate long explanations for horizontal display
                    const short = item.explanation?.length > 30
                      ? item.explanation.slice(0, 28) + '…'
                      : item.explanation
                    return (
                      <span
                        key={i}
                        className="group relative inline-flex items-baseline gap-0.5 text-xs leading-relaxed cursor-default"
                      >
                        <span className="font-semibold text-gray-900">{item.text}</span>
                        <span className={`text-[10px] px-1 rounded font-medium ${badgeClass}`}>
                          {item.type || 'w'}
                        </span>
                        {short && (
                          <span className="text-[11px] text-gray-500 hidden sm:inline">
                            {short}
                          </span>
                        )}
                        {/* Full explanation on hover tooltip */}
                        {item.explanation && item.explanation.length > 30 && (
                          <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block z-20 w-56">
                            <div className="bg-gray-900 text-white text-[10px] rounded-lg px-2.5 py-1.5 shadow-lg leading-relaxed">
                              {item.explanation}
                              <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-gray-900" />
                            </div>
                          </div>
                        )}
                      </span>
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
                  Analyzing...
                </span>
              ) : (
                '📝 Grammar'
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
