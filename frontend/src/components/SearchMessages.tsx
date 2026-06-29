import { useState, useRef, useEffect } from 'react'
import { messageAPI } from '../services/api'
import type { Message } from '../types'
import { formatDistanceToNow } from 'date-fns'

interface SearchMessagesProps {
  chatId?: string
  onClose: () => void
  onSelectMessage?: (message: Message) => void
}

export default function SearchMessages({ chatId, onClose, onSelectMessage }: SearchMessagesProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Message[]>([])
  const [isSearching, setIsSearching] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleSearch = async () => {
    if (!query.trim()) return
    setIsSearching(true)
    setHasSearched(true)
    try {
      const messages = await messageAPI.searchMessages(query, chatId)
      setResults(messages)
    } catch (err) {
      console.error('Search failed:', err)
    } finally {
      setIsSearching(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-start justify-center pt-20 z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[70vh] flex flex-col">
        <div className="p-4 border-b border-gray-200 flex items-center gap-3">
          <div className="flex-1 relative">
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              placeholder="Search messages..."
              className="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 pl-10"
            />
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">🔍</span>
          </div>
          <button
            onClick={handleSearch}
            disabled={isSearching || !query.trim()}
            className="px-4 py-2.5 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 text-sm font-medium"
          >
            {isSearching ? 'Searching...' : 'Search'}
          </button>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-xl">×</button>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          {!hasSearched && (
            <div className="text-center text-gray-500 py-8">
              <p className="text-4xl mb-3">🔍</p>
              <p>Type a query and press Enter or click Search</p>
            </div>
          )}

          {hasSearched && !isSearching && results.length === 0 && (
            <div className="text-center text-gray-500 py-8">
              <p className="text-4xl mb-3">📭</p>
              <p>No messages found for "{query}"</p>
            </div>
          )}

          {results.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm text-gray-500 mb-3">{results.length} result{results.length !== 1 ? 's' : ''}</p>
              {results.map((msg) => (
                <div
                  key={msg.id}
                  onClick={() => onSelectMessage?.(msg)}
                  className="p-3 rounded-lg border border-gray-200 hover:bg-gray-50 cursor-pointer transition"
                >
                  <div className="flex items-start justify-between mb-1">
                    <span className="text-sm font-semibold text-gray-900">
                      {msg.sender?.displayName || 'Unknown'}
                    </span>
                    <span className="text-xs text-gray-500">
                      {formatDistanceToNow(new Date(msg.timestamp), { addSuffix: true })}
                    </span>
                  </div>
                  <p className="text-sm text-gray-700 line-clamp-2">{msg.text}</p>
                  {msg.translations && Object.keys(msg.translations).length > 0 && (
                    <div className="mt-1 flex gap-1 flex-wrap">
                      {Object.entries(msg.translations).slice(0, 2).map(([lang, text]) => (
                        <span key={lang} className="text-xs bg-gray-100 px-1.5 py-0.5 rounded text-gray-500">
                          {lang.toUpperCase()}: {(text as string).substring(0, 30)}...
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
