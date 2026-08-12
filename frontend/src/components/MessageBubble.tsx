import { useState, useEffect, useRef } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { useTranslation } from 'react-i18next'
import type { Message } from '../types'
import { vocabularyAPI, grammarAPI } from '../services/api'
import GrammarPanel from './GrammarPanel'

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  nativeLanguage: string
  targetLanguage?: string
}

export default function MessageBubble({ message, isOwn, nativeLanguage, targetLanguage }: MessageBubbleProps) {
  const { t } = useTranslation()
  const [showActions, setShowActions] = useState(false)
  const [savedWord, setSavedWord] = useState<string | null>(null)
  const [showGrammar, setShowGrammar] = useState(false)
  const [grammarAnalysis, setGrammarAnalysis] = useState<any>(null)
  const [loadingGrammar, setLoadingGrammar] = useState(false)
  const [grammarProvider, setGrammarProvider] = useState<string | null>(null)
  const [showFallbackMsg, setShowFallbackMsg] = useState(false)

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

  // Show "Switching models..." only after the primary provider's realistic
  // worst-case latency has passed (Gemini can take ~12s), so a normally-working
  // request doesn't show a false "falling back" message.
  useEffect(() => {
    if (loadingGrammar) {
      const timer = setTimeout(() => setShowFallbackMsg(true), 20000)
      return () => clearTimeout(timer)
    }
    setShowFallbackMsg(false)
  }, [loadingGrammar])
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
    setGrammarProvider(null)
    try {
      // Always analyze the ORIGINAL message text.
      // Use the sender's native language as fallback when originalLanguage is not set.
      const sourceLang = message.originalLanguage || message.sender?.nativeLanguage || 'en'
      const response = await grammarAPI.analyzeAI(message.text, sourceLang, nativeLanguage)
      setGrammarAnalysis(response.analysis || response)
      setGrammarProvider(response.provider_used || null)
      if (!silent) setShowGrammar(true)
    } catch (err) {
      console.error('AI grammar analysis failed, falling back to regex:', err)
      // Fallback to regex analysis
      try {
        const sourceLang = message.originalLanguage || message.sender?.nativeLanguage || 'en'
        const response = await grammarAPI.analyze(message.text, sourceLang, nativeLanguage)
        setGrammarAnalysis(response.analysis || response)
        setGrammarProvider(null)
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
                🌐 {t('grammar.translating')}
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
                <span>🌐 {t('grammar.inYourLang')}</span>
                {!isOwn && !message.translationEnhanced && (
                  <span className="inline-flex items-center text-[10px] text-amber-500 animate-pulse" title={t('grammar.translating')}>
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
                📖 {t('grammar.learning', { lang: targetLanguage?.toUpperCase() })}
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
          <GrammarPanel
            analysis={grammarAnalysis}
            nativeLanguage={nativeLanguage}
            messageText={message.text}
            messageLanguage={message.originalLanguage || message.sender?.nativeLanguage || 'en'}
            onClose={() => setShowGrammar(false)}
            providerUsed={grammarProvider || undefined}
          />
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
                  {showFallbackMsg ? t('grammar.switchingModels') : t('grammar.analyzing')}
                </span>
              ) : (
                `📝 ${t('grammar.grammar')}`
              )}
            </button>
            {words.slice(0, 3).map((word: string) => (
              <button
                key={word}
                onClick={() => handleSaveWord(word)}
                className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 transition"
                title={t('grammar.saveWord', { word })}
              >
                {savedWord === word ? t('common.saved') : `+ ${word.substring(0, 12)}`}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
