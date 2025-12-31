import { useState } from 'react'
import { useStore } from '../store'
import { authAPI } from '../services/api'
import type { User } from '../types'

interface NewChatModalProps {
  onClose: () => void
}

export default function NewChatModal({ onClose }: NewChatModalProps) {
  const { createChat, setActiveChat } = useStore()
  const [chatType, setChatType] = useState<'direct' | 'group'>('direct')
  const [groupName, setGroupName] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<User[]>([])
  const [selectedUsers, setSelectedUsers] = useState<User[]>([])
  const [isLoading, setIsLoading] = useState(false)

  const handleSearch = async (query: string) => {
    setSearchQuery(query)
    if (query.length < 2) {
      setSearchResults([])
      return
    }

    try {
      const users = await authAPI.searchUsers(query)
      setSearchResults(users)
    } catch (error) {
      console.error('Search failed:', error)
    }
  }

  const toggleUser = (user: User) => {
    if (selectedUsers.find((u) => u.id === user.id)) {
      setSelectedUsers(selectedUsers.filter((u) => u.id !== user.id))
    } else {
      setSelectedUsers([...selectedUsers, user])
    }
  }

  const handleCreate = async () => {
    if (selectedUsers.length === 0) return

    setIsLoading(true)
    try {
      const chat = await createChat(
        chatType,
        selectedUsers.map((u) => u.id),
        chatType === 'group' ? groupName : undefined
      )
      setActiveChat(chat)
      onClose()
    } catch (error) {
      console.error('Failed to create chat:', error)
      alert('Failed to create chat')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md max-h-[80vh] flex flex-col">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-2xl font-bold text-gray-900">New Chat</h2>
            <button
              onClick={onClose}
              className="text-gray-500 hover:text-gray-700 text-2xl"
            >
              ×
            </button>
          </div>

          <div className="flex space-x-4 mb-4">
            <button
              onClick={() => setChatType('direct')}
              className={`flex-1 py-2 px-4 rounded-lg transition ${
                chatType === 'direct'
                  ? 'bg-primary text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Direct Chat
            </button>
            <button
              onClick={() => setChatType('group')}
              className={`flex-1 py-2 px-4 rounded-lg transition ${
                chatType === 'group'
                  ? 'bg-primary text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Group Chat
            </button>
          </div>

          {chatType === 'group' && (
            <input
              type="text"
              value={groupName}
              onChange={(e) => setGroupName(e.target.value)}
              placeholder="Group name..."
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary mb-4"
            />
          )}

          <input
            type="text"
            value={searchQuery}
            onChange={(e) => handleSearch(e.target.value)}
            placeholder="Search users..."
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          {selectedUsers.length > 0 && (
            <div className="mb-4">
              <div className="text-sm font-semibold text-gray-700 mb-2">
                Selected ({selectedUsers.length})
              </div>
              <div className="flex flex-wrap gap-2">
                {selectedUsers.map((user) => (
                  <div
                    key={user.id}
                    className="bg-primary/10 text-primary px-3 py-1 rounded-full text-sm flex items-center space-x-1"
                  >
                    <span>{user.displayName}</span>
                    <button
                      onClick={() => toggleUser(user)}
                      className="text-primary hover:text-primary/70"
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {searchResults.length > 0 && (
            <div>
              <div className="text-sm font-semibold text-gray-700 mb-2">
                Search Results
              </div>
              <div className="space-y-2">
                {searchResults.map((user) => {
                  const isSelected = selectedUsers.find((u) => u.id === user.id)
                  return (
                    <div
                      key={user.id}
                      onClick={() => toggleUser(user)}
                      className={`p-3 rounded-lg border cursor-pointer transition ${
                        isSelected
                          ? 'border-primary bg-primary/5'
                          : 'border-gray-200 hover:bg-gray-50'
                      }`}
                    >
                      <div className="font-semibold text-gray-900">
                        {user.displayName}
                      </div>
                      <div className="text-sm text-gray-500">
                        @{user.username}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>

        <div className="p-6 border-t border-gray-200">
          <button
            onClick={handleCreate}
            disabled={selectedUsers.length === 0 || isLoading}
            className="w-full bg-primary text-white py-2 px-4 rounded-lg hover:bg-primary/90 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? 'Creating...' : 'Create Chat'}
          </button>
        </div>
      </div>
    </div>
  )
}
