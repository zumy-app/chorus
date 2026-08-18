import { useCallback, useEffect, useMemo, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, StyleSheet, Text, View } from 'react-native'

import { loadMessagePage, messagesFrom, useChatSnapshot } from './chat-store'
import { MessageComposer } from './message-composer'
import type { Chat, Message } from '@/types'

function senderName(chat: Chat, message: Message, currentUserId?: string) {
  if (message.senderId === currentUserId) return 'You'
  return message.sender?.displayName || chat.participants?.find((participant) => participant.userId === message.senderId)?.user?.displayName || 'Member'
}

function messageTime(timestamp: string) {
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
}

function MessageBubble({ chat, message, currentUserId }: { chat: Chat; message: Message; currentUserId?: string }) {
  const isMine = message.senderId === currentUserId
  const [showTranslation, setShowTranslation] = useState(false)
  const translation = useMemo(() => Object.values(message.translations ?? {})[0], [message.translations])

  return (
    <View style={[styles.messageRow, isMine ? styles.mineRow : styles.theirsRow]}>
      <View style={[styles.bubble, isMine ? styles.mineBubble : styles.theirsBubble]}>
        {!isMine && <Text style={styles.sender}>{senderName(chat, message, currentUserId)}</Text>}
        <Text selectable style={[styles.messageText, isMine && styles.mineText]}>{message.text}</Text>
        {translation && showTranslation && <Text selectable style={[styles.translation, isMine && styles.mineTranslation]}>{translation}</Text>}
        <View style={styles.metadata}>
          {translation && <Pressable accessibilityRole="button" accessibilityLabel={showTranslation ? 'Hide translation' : 'Show translation'} onPress={() => setShowTranslation((value) => !value)}><Text style={[styles.translationControl, isMine && styles.mineControl]}>{showTranslation ? 'Original' : 'Translate'}</Text></Pressable>}
          <Text style={[styles.time, isMine && styles.mineControl]}>{messageTime(message.timestamp)}{isMine && message.deliveryStatus === 'failed' ? ' · Not sent' : ''}</Text>
        </View>
      </View>
    </View>
  )
}

export function ChatThread({ chat }: { chat: Chat }) {
  const snapshot = useChatSnapshot()
  const messages = messagesFrom(snapshot, chat.id)
  const [initializing, setInitializing] = useState(messages.length === 0)
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [error, setError] = useState<string>()
  const hasMore = snapshot.hasMoreByChatId?.[chat.id] ?? true
  const typers = snapshot.typingUserIdsByChatId?.[chat.id] ?? []

  const loadFirstPage = useCallback(async () => {
    setError(undefined)
    setInitializing(true)
    try {
      await loadMessagePage(snapshot, chat.id)
    } catch {
      setError('Unable to load messages. Pull down to retry.')
    } finally {
      setInitializing(false)
    }
  }, [chat.id, snapshot])

  useEffect(() => {
    void loadFirstPage()
  }, [loadFirstPage])

  const loadOlder = async () => {
    if (!hasMore || loadingOlder || messages.length === 0) return
    setLoadingOlder(true)
    try {
      await loadMessagePage(snapshot, chat.id, messages[0].id)
    } catch {
      setError('Unable to load older messages.')
    } finally {
      setLoadingOlder(false)
    }
  }

  if (initializing && messages.length === 0) {
    return <View style={styles.center}><ActivityIndicator size="large" color="#5b49e6" /><Text style={styles.stateText}>Loading messages…</Text></View>
  }

  return (
    <View style={styles.container}>
      {error && <Pressable accessibilityRole="button" accessibilityLabel="Retry loading messages" onPress={() => void loadFirstPage()} style={styles.error}><Text style={styles.errorText}>{error} Tap to retry.</Text></Pressable>}
      <FlatList
        accessibilityLabel={`Messages in ${chat.name || 'conversation'}`}
        contentContainerStyle={messages.length ? styles.list : styles.emptyList}
        data={messages}
        keyExtractor={(message) => message.id}
        renderItem={({ item }) => <MessageBubble chat={chat} message={item} currentUserId={snapshot.currentUserId} />}
        onEndReached={() => void loadOlder()}
        onEndReachedThreshold={.25}
        ListFooterComponent={loadingOlder ? <ActivityIndicator color="#5b49e6" style={styles.olderLoader} /> : null}
        ListEmptyComponent={<View style={styles.center}><Text style={styles.stateText}>No messages yet. Say hello!</Text></View>}
      />
      {typers.length > 0 && <Text accessibilityLiveRegion="polite" style={styles.typing}>{typers.length === 1 ? 'Someone is typing…' : `${typers.length} people are typing…`}</Text>}
      <MessageComposer chatId={chat.id} />
    </View>
  )
}

const styles = StyleSheet.create({
  container: { backgroundColor: '#fbfaff', flex: 1 },
  list: { paddingHorizontal: 16, paddingVertical: 16 },
  emptyList: { flexGrow: 1 },
  center: { alignItems: 'center', flex: 1, gap: 12, justifyContent: 'center', padding: 28 },
  stateText: { color: '#746f84', fontSize: 15, textAlign: 'center' },
  messageRow: { marginBottom: 12 },
  mineRow: { alignItems: 'flex-end' },
  theirsRow: { alignItems: 'flex-start' },
  bubble: { borderRadius: 18, maxWidth: '84%', paddingHorizontal: 13, paddingVertical: 10 },
  mineBubble: { backgroundColor: '#5b49e6', borderBottomRightRadius: 4 },
  theirsBubble: { backgroundColor: '#fff', borderBottomLeftRadius: 4, shadowColor: '#1d1938', shadowOpacity: .06, shadowRadius: 5 },
  sender: { color: '#5b49e6', fontSize: 12, fontWeight: '700', marginBottom: 3 },
  messageText: { color: '#262335', fontSize: 16, lineHeight: 22 },
  mineText: { color: '#fff' },
  translation: { borderTopColor: '#e6e3f0', borderTopWidth: StyleSheet.hairlineWidth, color: '#5c5670', fontSize: 14, fontStyle: 'italic', lineHeight: 20, marginTop: 8, paddingTop: 7 },
  mineTranslation: { borderTopColor: '#9588ed', color: '#f1efff' },
  metadata: { alignItems: 'center', flexDirection: 'row', gap: 8, justifyContent: 'flex-end', marginTop: 5 },
  translationControl: { color: '#4d3fc9', fontSize: 12, fontWeight: '700' },
  time: { color: '#847e94', fontSize: 11 },
  mineControl: { color: '#e1ddff' },
  typing: { color: '#746f84', fontSize: 13, fontStyle: 'italic', paddingHorizontal: 20, paddingVertical: 5 },
  error: { backgroundColor: '#ffefef', padding: 10 },
  errorText: { color: '#a23030', textAlign: 'center' },
  olderLoader: { marginVertical: 12 },
})
