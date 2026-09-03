import { useTranslation } from 'react-i18next'
import type { Chat, Message } from '@chorus/shared'

export default function ForwardDialog({ message, chats, currentChatId, onClose, onForward }: { message: Message; chats: Chat[]; currentChatId: string; onClose: () => void; onForward: (chatId: string) => Promise<void> }) {
  const { t } = useTranslation()
  const targets = chats.filter(c => c.id !== currentChatId)
  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div className="bg-surface rounded-2xl max-w-sm w-full max-h-[70vh] flex flex-col overflow-hidden" onClick={e=>e.stopPropagation()}>
        <div className="px-5 py-4 border-b border-outline-variant flex items-center justify-between">
          <h3 className="font-semibold">{t('chat.forwardTo')}</h3>
          <button onClick={onClose} className="w-8 h-8 rounded-full hover:bg-surface-variant flex items-center justify-center">✕</button>
        </div>
        <div className="px-4 py-2 text-sm text-on-surface-variant bg-surface-container-low border-b border-outline-variant truncate">"{message.text.slice(0,80)}"</div>
        <div className="overflow-y-auto flex-1 divide-y divide-outline-variant/20">
          {targets.length===0 ? <div className="p-6 text-center text-on-surface-variant text-sm">{t('chat.noChatsYet')}</div> : targets.map(c=>{
            const name = c.type==='group' ? (c.name||'Group') : (c.participants?.find(p=>p.user)?.user?.displayName || 'Chat')
            return <button key={c.id} onClick={()=>onForward(c.id)} className="w-full text-left px-4 py-3 hover:bg-surface-container-high flex items-center gap-3">
              <div className="w-9 h-9 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center text-sm font-bold">{name.charAt(0).toUpperCase()}</div>
              <span className="truncate text-sm font-medium">{name}</span>
            </button>
          })}
        </div>
      </div>
    </div>
  )
}
