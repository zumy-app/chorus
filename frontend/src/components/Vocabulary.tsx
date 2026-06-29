import { useState, useEffect } from 'react'
import { vocabularyAPI } from '../services/api'
import { formatDistanceToNow } from 'date-fns'

interface VocabularyEntry {
  id: string
  term: string
  language: string
  translation: string
  definition?: string
  context?: { sentence?: string }
  learningData?: {
    reviewCount: number
    correctCount: number
    nextReview: string
    interval: number
  }
  createdAt: string
}

interface VocabularyProps {
  onClose: () => void
}

export default function Vocabulary({ onClose }: VocabularyProps) {
  const [entries, setEntries] = useState<VocabularyEntry[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [tab, setTab] = useState<'all' | 'due'>('due')
  const [stats, setStats] = useState<any>(null)

  useEffect(() => {
    loadData()
  }, [tab])

  const loadData = async () => {
    setIsLoading(true)
    try {
      if (tab === 'due') {
        const due = await vocabularyAPI.getDue()
        setEntries(due)
      } else {
        const all = await vocabularyAPI.getAll()
        setEntries(all)
      }
      const progress = await vocabularyAPI.getProgress()
      setStats(progress)
    } catch (err) {
      console.error('Failed to load vocabulary:', err)
    } finally {
      setIsLoading(false)
    }
  }

  const handlePractice = async (id: string, correct: boolean) => {
    try {
      await vocabularyAPI.practice(id, correct)
      // Remove from due list and update stats
      setEntries(prev => prev.filter(e => e.id !== id))
      loadData()
    } catch (err) {
      console.error('Practice failed:', err)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-2xl font-bold text-gray-900">📚 Vocabulary</h2>
            <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">×</button>
          </div>

          {/* Stats */}
          {stats && (
            <div className="grid grid-cols-4 gap-3 mb-4">
              <div className="bg-indigo-50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-indigo-600">{stats.totalVocabulary || 0}</div>
                <div className="text-xs text-gray-600">Total Words</div>
              </div>
              <div className="bg-green-50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-green-600">{stats.masteredCount || 0}</div>
                <div className="text-xs text-gray-600">Mastered</div>
              </div>
              <div className="bg-yellow-50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-yellow-600">{stats.dueToday || 0}</div>
                <div className="text-xs text-gray-600">Due Today</div>
              </div>
              <div className="bg-blue-50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-blue-600">{Math.round((stats.accuracy || 0) * 100)}%</div>
                <div className="text-xs text-gray-600">Accuracy</div>
              </div>
            </div>
          )}

          {/* Tabs */}
          <div className="flex gap-2">
            <button
              onClick={() => setTab('due')}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition ${
                tab === 'due' ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Due for Review {stats?.dueToday ? `(${stats.dueToday})` : ''}
            </button>
            <button
              onClick={() => setTab('all')}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition ${
                tab === 'all' ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              All Words ({stats?.totalVocabulary || 0})
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          {isLoading ? (
            <div className="text-center text-gray-500 py-8">Loading...</div>
          ) : entries.length === 0 ? (
            <div className="text-center text-gray-500 py-12">
              <p className="text-5xl mb-4">🎉</p>
              <p className="text-lg font-medium mb-2">
                {tab === 'due' ? 'No words due for review!' : 'No vocabulary saved yet'}
              </p>
              <p className="text-sm">
                {tab === 'due'
                  ? 'Great job keeping up with your studies!'
                  : 'Save words from your conversations to build your vocabulary.'}
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {entries.map((entry) => (
                <div key={entry.id} className="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-lg font-bold text-gray-900">{entry.term}</span>
                        <span className="text-xs bg-gray-100 px-2 py-0.5 rounded text-gray-500 uppercase">
                          {entry.language}
                        </span>
                      </div>
                      <p className="text-gray-600">{entry.translation}</p>
                      {entry.definition && (
                        <p className="text-sm text-gray-500 mt-1 italic">{entry.definition}</p>
                      )}
                      {entry.context?.sentence && (
                        <p className="text-xs text-gray-400 mt-1">
                          Context: "{entry.context.sentence}"
                        </p>
                      )}
                      {entry.learningData && (
                        <div className="flex items-center gap-3 mt-2 text-xs text-gray-500">
                          <span>Reviewed {entry.learningData.reviewCount}x</span>
                          <span>Interval: {entry.learningData.interval}d</span>
                          {entry.learningData.nextReview && (
                            <span>Next: {formatDistanceToNow(new Date(entry.learningData.nextReview), { addSuffix: true })}</span>
                          )}
                        </div>
                      )}
                    </div>
                    {tab === 'due' && (
                      <div className="flex gap-1 ml-3">
                        <button
                          onClick={() => handlePractice(entry.id, true)}
                          className="px-3 py-1.5 bg-green-100 text-green-700 rounded-lg text-sm hover:bg-green-200 transition"
                          title="I knew this"
                        >
                          ✓
                        </button>
                        <button
                          onClick={() => handlePractice(entry.id, false)}
                          className="px-3 py-1.5 bg-red-100 text-red-700 rounded-lg text-sm hover:bg-red-200 transition"
                          title="I didn't know this"
                        >
                          ✗
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
