import { useState, useEffect, useRef } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import type { Message, GrammarJobStatus } from '../types'
import { vocabularyAPI, grammarAPI } from '../services/api'
import { useStore } from '../store'
import GrammarPanel from './GrammarPanel'
import ReportModal from './ReportModal'

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
  const [grammarProvider, setGrammarProvider] = useState<string | null>(null)
  const [grammarError, setGrammarError] = useState<string | null>(null)
  const [showReport, setShowReport] = useState(false)

  const entitlements = useStore((s) => s.entitlements)
  const blocked = useStore((s) => s.blockedTranslations[message.id])
  const grammarJob = useStore((s) => s.grammarJobs[message.id])
  const setGrammarJob = useStore((s) => s.setGrammarJob)
  const resyncGrammarJob = useStore((s) => s.resyncGrammarJob)

  const autoGrammar = entitlements?.features?.autoGrammar ?? false

  const nativeTranslation = message.translations?.[nativeLanguage]
  const showNativeTranslation = nativeTranslation && nativeTranslation !== message.text

  // Derive the blocked state as well as relying on the live WebSocket signal:
  // a free-plan user's own message that is over the translation word limit and
  // has no translation is one the server intentionally left untranslated. This
  // keeps the premium nudge visible even after a reload (the in-memory blocked
  // map is lost on navigation), instead of the message silently looking sent.
  const wordLimit = entitlements?.features?.translationWordLimit
  const isOwnLongUntranslated = Boolean(
    isOwn &&
    entitlements?.effectivePlan === 'free' &&
    wordLimit != null &&
    !nativeTranslation &&
    (message.text || '').trim().split(/\s+/).filter(Boolean).length > wordLimit
  )

  // Translation was blocked by the free-plan word limit (server decided not to
  // translate a long message). Show the premium nudge instead of a spinner.
  const isTranslationBlocked = Boolean(blocked && blocked.reason === 'message_too_long') || isOwnLongUntranslated

  // Show translation pending indicator only for recent messages still awaiting translation
  const isTranslationPending = !isOwn && !nativeTranslation && !isTranslationBlocked && (
    !message.translations || Object.keys(message.translations).length === 0
  )

  // Show Learning section only when target language is DIFFERENT from both native AND original
  const targetTranslation = targetLanguage && targetLanguage !== nativeLanguage && targetLanguage !== message.originalLanguage
    ? message.translations?.[targetLanguage]
    : null
  const showTargetTranslation = targetTranslation && targetTranslation !== message.text

  // Source the completed analysis and failures from the job record (delivered
  // over the WebSocket and/or polled), keeping the panel honest even across
  // reconnects. A fresh job pending for a previously-analyzed message clears
  // the stale analysis while the new request runs.
  const lastJobId = useRef<string | null>(null)
  useEffect(() => {
    // A job started before a reload is re-fetched so the bubble shows its
    // actual outcome instead of silently showing nothing.
    if (!grammarJob) {
      resyncGrammarJob(message.id)
    }
  }, [message.id, grammarJob])

  useEffect(() => {
    if (!grammarJob) return
    if (lastJobId.current !== grammarJob.jobId) {
      lastJobId.current = grammarJob.jobId
      setGrammarAnalysis(null)
      setGrammarProvider(null)
      setGrammarError(null)
    }
    if (grammarJob.status === 'done') {
      setGrammarAnalysis(grammarJob.analysis || null)
      setGrammarProvider(grammarJob.providerUsed || 'cache')
      setGrammarError(null)
    } else if (grammarJob.status === 'failed') {
      setGrammarError(grammarJob.error || null)
    }
  }, [grammarJob])

  const jobBusy = grammarJob?.status === 'queued' || grammarJob?.status === 'processing'

  // Re-run AI grammar analysis when translations arrive (WebSocket update).
  // For premium users (autoGrammar), analysis also runs automatically the
  // first time a translation arrives — no manual button press needed.
  const prevNativeTranslation = useRef(nativeTranslation)
  const autoTriggered = useRef(false)
  useEffect(() => {
    if (showGrammar && grammarAnalysis && prevNativeTranslation.current !== nativeTranslation) {
      handleAnalyzeGrammar(true)
    } else if (
      autoGrammar &&
      !autoTriggered.current &&
      !isOwn &&
      nativeTranslation &&
      prevNativeTranslation.current !== nativeTranslation
    ) {
      autoTriggered.current = true
      handleAnalyzeGrammar(false)
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
    if (jobBusy) return
    // Always analyze the ORIGINAL message text.
    // Use the sender's native language as fallback when originalLanguage is not set.
    const sourceLang = message.originalLanguage || message.sender?.nativeLanguage || 'en'
    if (!silent) setShowGrammar(true)
    try {
      // Submit an async job; the result arrives over the WebSocket
      // ("grammar_analysis") and via GET /grammar/analyze/:jobId.
      const response = await grammarAPI.analyzeAI({
        text: message.text,
        language: sourceLang,
        nativeLanguage,
        messageId: message.id,
        chatId: message.chatId,
      })
      // Cache fast-path: the response already contains the finished analysis.
      if (response.status === 'done' && response.analysis) {
        setGrammarAnalysis(response.analysis)
        setGrammarProvider(response.providerUsed || 'cache')
        setGrammarError(null)
        return
      }
      setGrammarJob({
        jobId: response.jobId,
        status: (response.status as GrammarJobStatus) || 'queued',
        messageId: message.id,
      })
    } catch (err) {
      console.error('AI grammar analysis failed, falling back to regex:', err)
      // Fallback to regex analysis
      try {
        const response = await grammarAPI.analyze(message.text, sourceLang, nativeLanguage)
        setGrammarAnalysis(response.analysis || response)
        setGrammarProvider(null)
        setGrammarError(null)
      } catch (fallbackErr) {
        console.error('Grammar analysis failed:', fallbackErr)
        setGrammarError('failed')
      }
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

          {/* Translation blocked by free-plan char limit */}
          {isTranslationBlocked && !showNativeTranslation && (
            <div className="mt-2 pt-2 border-t border-gray-200 text-sm">
              <div className="text-xs mb-1 text-gray-500 flex items-center gap-1">
                <span>🔒 {t('grammar.translationBlocked')}</span>
              </div>
              <Link
                to="/pricing"
                onClick={(e) => e.stopPropagation()}
                className="inline-block text-xs mt-1 px-3 py-1 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-semibold hover:opacity-90 transition"
              >
                {t('plan.upgrade')}
              </Link>
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
        {showGrammar && jobBusy && !grammarError && (
          <div className="mt-1 bg-amber-50 border border-amber-200 rounded-lg shadow-sm max-w-md px-3 py-2.5">
            <div className="flex items-center gap-2">
              <span className="inline-block w-1.5 h-1.5 bg-amber-600 rounded-full animate-pulse" />
              <span className="text-xs font-medium text-amber-800">
                {grammarJob?.status === 'queued' ? t('grammar.queued') : t('grammar.analyzing')}
              </span>
            </div>
            <p className="text-[11px] text-amber-600 mt-1">{t('grammar.analysisInProgress')}</p>
          </div>
        )}

        {showGrammar && grammarError && !jobBusy && (
          <div className="mt-1 bg-red-50 border border-red-200 rounded-lg shadow-sm max-w-md px-3 py-2.5">
            <div className="text-xs font-medium text-red-700">{t('grammar.failed')}</div>
            <button
              onClick={() => handleAnalyzeGrammar()}
              className="text-[11px] mt-1 px-2 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 transition"
            >
              {t('grammar.retry')}
            </button>
          </div>
        )}

        {showGrammar && grammarAnalysis && !jobBusy && (
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
              disabled={jobBusy}
              className="text-xs px-2 py-1 bg-amber-100 text-amber-700 rounded hover:bg-amber-200 transition disabled:opacity-50"
            >
              {jobBusy ? (
                <span className="flex items-center gap-1">
                  <span className="inline-block w-1 h-1 bg-amber-600 rounded-full animate-pulse" />
                  {grammarJob?.status === 'queued' ? t('grammar.queued') : t('grammar.analyzing')}
                </span>
              ) : grammarError ? (
                `↻ ${t('grammar.retry')}`
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
            <button
              onClick={() => setShowReport(true)}
              className="text-xs px-2 py-1 bg-gray-100 text-gray-600 rounded hover:bg-gray-200 transition"
              title={t('report.reportMessage')}
            >
              🚩
            </button>
          </div>
        )}
      </div>

      {showReport && message.sender && (
        <ReportModal
          targetType="message"
          messageId={message.id}
          chatId={message.chatId}
          reportedUserName={message.sender.displayName}
          onClose={() => setShowReport(false)}
        />
      )}
    </div>
  )
}
