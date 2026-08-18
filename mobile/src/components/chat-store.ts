import { chatApi, messageApi } from '@/services/api'
import { useChatStore } from '@/store/chat'
import type { Chat, Message } from '@/types'

type ChatStoreSnapshot = {
  chatsById?: Record<string, Chat>
  chatIds?: string[]
  messagesById?: Record<string, Message>
  messageIdsByChatId?: Record<string, string[]>
  hasMoreByChatId?: Record<string, boolean>
  typingUserIdsByChatId?: Record<string, string[]>
  currentUserId?: string
  applySnapshot?: (chats: Chat[]) => void
  applyMessagePage?: (chatId: string, messages: Message[], hasMore: boolean) => void
  sendMessage?: (chatId: string, text: string) => Promise<void>
}

export function useChatSnapshot() {
  return useChatStore((state) => state as ChatStoreSnapshot)
}

export function chatsFrom(snapshot: ChatStoreSnapshot): Chat[] {
  const chats = snapshot.chatsById ?? {}
  return (snapshot.chatIds ?? Object.keys(chats)).map((id) => chats[id]).filter((chat): chat is Chat => Boolean(chat))
}

export function messagesFrom(snapshot: ChatStoreSnapshot, chatId: string): Message[] {
  const messages = snapshot.messagesById ?? {}
  return (snapshot.messageIdsByChatId?.[chatId] ?? [])
    .map((id) => messages[id])
    .filter((message): message is Message => Boolean(message))
}

export async function refreshChats(snapshot: ChatStoreSnapshot) {
  const chats = await chatApi.list()
  snapshot.applySnapshot?.(chats)
  return chats
}

export async function loadMessagePage(snapshot: ChatStoreSnapshot, chatId: string, before?: string) {
  const page = await messageApi.list(chatId, before)
  snapshot.applyMessagePage?.(chatId, page.messages, page.hasMore)
  return page
}
