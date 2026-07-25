import { useState, useRef, useEffect } from 'react'
import { grammarAPI } from '../services/api'

interface LearningPanelProps {
  text: string
  language: string
  nativeLanguage: string
  onClose: () => void
}

type ChatMessage = {
  role: 'assistant' | 'user'
  content: string
  details?: string[]
  suggestedActions?: string[]
}

const actionLabels: Record<string, Record<string, { icon: string; label: string }>> = {
  en: { breakdown: { icon: '🔍', label: 'Break Down' }, examples: { icon: '📝', label: 'Examples' }, flashcards: { icon: '🃏', label: 'Flashcards' }, conjugation: { icon: '📊', label: 'Conjugation' }, custom: { icon: '💬', label: 'Ask More' } },
  es: { breakdown: { icon: '🔍', label: 'Analizar' }, examples: { icon: '📝', label: 'Ejemplos' }, flashcards: { icon: '🃏', label: 'Tarjetas' }, conjugation: { icon: '📊', label: 'Conjugación' }, custom: { icon: '💬', label: 'Preguntar' } },
  fr: { breakdown: { icon: '🔍', label: 'Analyser' }, examples: { icon: '📝', label: 'Exemples' }, flashcards: { icon: '🃏', label: 'Cartes' }, conjugation: { icon: '📊', label: 'Conjugaison' }, custom: { icon: '💬', label: 'Demander' } },
  de: { breakdown: { icon: '🔍', label: 'Analysieren' }, examples: { icon: '📝', label: 'Beispiele' }, flashcards: { icon: '🃏', label: 'Karten' }, conjugation: { icon: '📊', label: 'Konjugation' }, custom: { icon: '💬', label: 'Fragen' } },
}

function getActionLabel(action: string, lang: string): { icon: string; label: string } {
  return actionLabels[lang]?.[action] || actionLabels.en[action] || { icon: '📚', label: action }
}

const localizedTexts: Record<string, Record<string, string>> = {
  en: { title: 'AI Tutor', analyzing: 'Analyzing...', error: 'Sorry, I couldn\'t generate learning content right now. Please try again.', placeholder: 'Ask a question...', ask: 'Ask', grammarBreakdown: '📖 Grammar Breakdown' },
  es: { title: 'Tutor IA', analyzing: 'Analizando...', error: 'Lo siento, no pude generar contenido educativo ahora. Intenta de nuevo.', placeholder: 'Haz una pregunta...', ask: 'Preguntar', grammarBreakdown: '📖 Análisis Gramatical' },
  fr: { title: 'Tuteur IA', analyzing: 'Analyse...', error: 'Désolé, je n\'ai pas pu générer de contenu pour le moment. Réessayez.', placeholder: 'Posez une question...', ask: 'Demander', grammarBreakdown: '📖 Analyse Grammaticale' },
  de: { title: 'KI-Tutor', analyzing: 'Analysiere...', error: 'Entschuldigung, ich konnte gerade keine Lerninhalte erstellen. Bitte versuchen Sie es erneut.', placeholder: 'Stelle eine Frage...', ask: 'Fragen', grammarBreakdown: '📖 Grammatikanalyse' },
}

function getText(key: string, lang: string): string {
  return localizedTexts[lang]?.[key] || localizedTexts.en[key] || key
}

export default function LearningPanel({ text, language, nativeLanguage, onClose }: LearningPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [customQuery, setCustomQuery] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // Start with a breakdown on mount
  useEffect(() => {
    handleAction('breakdown')
  }, [])

  const handleAction = async (action: string, query?: string) => {
    setIsLoading(true)
    try {
      const result = await grammarAPI.learn(text, language, nativeLanguage, action, query)
      
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: result.content || getText('error', nativeLanguage),
        details: result.details || [],
        suggestedActions: result.suggestedActions || [],
      }])
    } catch (err) {
      console.error('Learning action failed:', err)
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: getText('error', nativeLanguage),
        suggestedActions: ['breakdown', 'examples', 'flashcards'],
      }])
    } finally {
      setIsLoading(false)
    }
  }

  const handleCustomSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!customQuery.trim()) return

    setMessages(prev => [...prev, { role: 'user', content: customQuery }])
    setCustomQuery('')
    await handleAction('custom', customQuery)
  }

  return (
    <div className="bg-white border border-indigo-200 rounded-lg shadow-md overflow-hidden max-w-sm">
      {/* Header */}
      <div className="bg-gradient-to-r from-indigo-600 to-purple-600 px-3 py-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">🤖</span>
          <span className="text-white font-semibold text-sm">{getText('title', nativeLanguage)}</span>
        </div>
        <button onClick={onClose} className="text-white/80 hover:text-white text-lg leading-none">×</button>
      </div>

      {/* Messages */}
      <div className="max-h-80 overflow-y-auto p-3 space-y-3 bg-gradient-to-b from-indigo-50/50 to-white">
        {/* Initial state */}
        {messages.length === 0 && isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500 py-4 justify-center">
            <span className="inline-block w-2 h-2 bg-indigo-500 rounded-full animate-pulse" 
                  style={{ animationDelay: '0ms' }} />
            <span className="inline-block w-2 h-2 bg-indigo-500 rounded-full animate-pulse" 
                  style={{ animationDelay: '300ms' }} />
            <span className="inline-block w-2 h-2 bg-indigo-500 rounded-full animate-pulse" 
                  style={{ animationDelay: '600ms' }} />
            <span className="ml-1">{getText('analyzing', nativeLanguage)}</span>
          </div>
        )}

        {messages.map((msg, i) => (
          <div key={i}>
            {/* Assistant message */}
            <div className="flex items-start gap-2">
              <div className="w-6 h-6 rounded-full bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center text-white text-[10px] font-bold shrink-0 mt-0.5">
                AI
              </div>
              <div className="flex-1 min-w-0">
                <div className="bg-white border border-indigo-100 rounded-lg p-2.5 text-xs text-gray-800 leading-relaxed shadow-sm">
                  {/* Action label */}
                  {i === 0 && (
                    <div className="text-[10px] font-semibold text-indigo-600 uppercase tracking-wide mb-1">
                      {getText('grammarBreakdown', nativeLanguage)}
                    </div>
                  )}
                  
                  {/* Content */}
                  <div className="whitespace-pre-wrap">{msg.content}</div>

                  {/* Details list */}
                  {msg.details && msg.details.length > 0 && (
                    <ul className="mt-2 space-y-1">
                      {msg.details.map((detail, j) => (
                        <li key={j} className="flex items-start gap-1.5 text-[11px] text-gray-600">
                          <span className="text-indigo-400 mt-0.5">•</span>
                          <span>{detail}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>

                {/* Suggested action buttons */}
                {msg.suggestedActions && msg.suggestedActions.length > 0 && (
                  <div className="flex flex-wrap gap-1 mt-1.5">
                    {msg.suggestedActions.map(action => {
                      const btn = getActionLabel(action, nativeLanguage)
                      return (
                        <button
                          key={action}
                          onClick={() => handleAction(action)}
                          disabled={isLoading}
                          className="text-[10px] px-2 py-1 bg-indigo-50 text-indigo-700 rounded-full border border-indigo-200 hover:bg-indigo-100 transition disabled:opacity-50"
                        >
                          {btn.icon} {btn.label}
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>

            {/* Loading indicator after this message if next is loading */}
            {i === messages.length - 1 && isLoading && (
              <div className="flex items-center gap-1.5 text-xs text-gray-400 mt-2 ml-8">
                <span className="inline-block w-1.5 h-1.5 bg-indigo-400 rounded-full animate-pulse"
                      style={{ animationDelay: '0ms' }} />
                <span className="inline-block w-1.5 h-1.5 bg-indigo-400 rounded-full animate-pulse"
                      style={{ animationDelay: '300ms' }} />
                <span className="inline-block w-1.5 h-1.5 bg-indigo-400 rounded-full animate-pulse"
                      style={{ animationDelay: '600ms' }} />
              </div>
            )}
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Custom question input */}
      <form onSubmit={handleCustomSubmit} className="border-t border-indigo-100 p-2 flex gap-2 bg-white">
        <input
          type="text"
          value={customQuery}
          onChange={e => setCustomQuery(e.target.value)}
          placeholder={getText('placeholder', nativeLanguage)}
          className="flex-1 text-xs px-2.5 py-1.5 border border-gray-300 rounded-lg focus:outline-none focus:ring-1 focus:ring-indigo-500"
          disabled={isLoading}
        />
        <button
          type="submit"
          disabled={!customQuery.trim() || isLoading}
          className="px-3 py-1.5 bg-gradient-to-r from-indigo-600 to-purple-600 text-white text-xs rounded-lg hover:opacity-90 transition disabled:opacity-50 font-medium"
        >
          {getText('ask', nativeLanguage)}
        </button>
      </form>
    </div>
  )
}
