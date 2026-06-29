import { useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import type { Message } from '../types'
import { vocabularyAPI } from '../services/api'

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

  // Show translation in native language (for comprehension)
  const nativeTranslation = message.translations?.[nativeLanguage]
  const showNativeTranslation = nativeTranslation && nativeTranslation !== message.text

  // Show translation in target language (for learning) if different from native
  const targetTranslation = targetLanguage && targetLanguage !== nativeLanguage
    ? message.translations?.[targetLanguage]
    : null
  const showTargetTranslation = targetTranslation && targetTranslation !== message.text

  const handleSaveWord = async (word: string) => {
    try {
      await vocabularyAPI.save(word, message.originalLanguage || nativeLanguage, message.id)
      setSavedWord(word)
      setTimeout(() => setSavedWord(null), 2000)
    } catch (err) {
      console.error('Failed to save word:', err)
    }
  }

  const handleAnalyzeGrammar = async () => {
    if (grammarAnalysis) {
      setShowGrammar(!showGrammar)
      return
    }
    setLoadingGrammar(true)
    try {
      const response = await fetch('/api/v1/grammar/analyze-text', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('accessToken')}` },
        body: JSON.stringify({ text: message.text, language: message.originalLanguage || 'en' })
      })
      const data = await response.json()
      setGrammarAnalysis(data.data?.analysis || data)
      setShowGrammar(true)
    } catch (err) {
      console.error('Grammar analysis failed:', err)
    } finally {
      setLoadingGrammar(false)
    }
  }

  // Extract words from the translated text for saving
  const displayText = nativeTranslation || targetTranslation || message.text
  const words = displayText
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

        {/* Grammar Analysis */}
        {showGrammar && grammarAnalysis && (
          <div className="mt-1 bg-yellow-50 border border-yellow-200 rounded-lg p-3 text-sm max-w-sm">
            <div className="flex items-center justify-between mb-1">
              <span className="font-semibold text-yellow-800">📝 Grammar</span>
              <button onClick={() => setShowGrammar(false)} className="text-yellow-600 hover:text-yellow-800">×</button>
            </div>
            <p className="text-yellow-700">Level: <strong>{grammarAnalysis.difficulty || 'N/A'}</strong></p>
            {grammarAnalysis.patterns?.length > 0 && (
              <div className="mt-1 flex flex-wrap gap-1">
                {grammarAnalysis.patterns.map((p: string, i: number) => (
                  <span key={i} className="text-xs bg-yellow-100 text-yellow-700 px-1.5 py-0.5 rounded">{p}</span>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Action Buttons */}
        {showActions && !isOwn && (
          <div className="flex gap-1 mt-1">
            <button
              onClick={handleAnalyzeGrammar}
              disabled={loadingGrammar}
              className="text-xs px-2 py-1 bg-yellow-100 text-yellow-700 rounded hover:bg-yellow-200 transition"
              title="Analyze grammar"
            >
              {loadingGrammar ? '...' : '📝 Grammar'}
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
