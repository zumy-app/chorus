import { formatDistanceToNow } from 'date-fns'
import type { Message } from '../types'

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  userLanguage: string
}

export default function MessageBubble({ message, isOwn, userLanguage }: MessageBubbleProps) {
  const translatedText = message.translations?.[userLanguage]
  const showTranslation = translatedText && translatedText !== message.text

  return (
    <div className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[70%] rounded-lg px-4 py-2 ${
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

        {showTranslation && (
          <div className={`mt-2 pt-2 border-t ${isOwn ? 'border-white/30' : 'border-gray-200'} text-sm opacity-90`}>
            <div className="text-xs mb-1 opacity-75">
              Translation ({userLanguage.toUpperCase()}):
            </div>
            <div className="italic">
              {translatedText}
            </div>
          </div>
        )}

        <div className={`text-xs mt-1 ${isOwn ? 'text-white/75' : 'text-gray-500'}`}>
          {formatDistanceToNow(new Date(message.timestamp), { addSuffix: true })}
        </div>
      </div>
    </div>
  )
}
