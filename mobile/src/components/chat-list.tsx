import { router } from 'expo-router'
import { useCallback, useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from 'react-native'

import { chatsFrom, refreshChats, useChatSnapshot } from './chat-store'
import type { Chat } from '@/types'

function chatTitle(chat: Chat) {
  if (chat.name) return chat.name
  return chat.participants?.map((participant) => participant.user?.displayName).filter(Boolean).join(', ') || 'Conversation'
}

function ChatRow({ chat }: { chat: Chat }) {
  const preview = chat.lastMessage?.text || 'No messages yet'
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`Open conversation with ${chatTitle(chat)}`}
      onPress={() => router.push(`/(app)/chat/${chat.id}`)}
      style={({ pressed }) => [styles.row, pressed && styles.pressed]}
    >
      <View style={styles.avatar} accessibilityElementsHidden>
        <Text style={styles.avatarText}>{chatTitle(chat).slice(0, 1).toUpperCase()}</Text>
      </View>
      <View style={styles.content}>
        <View style={styles.titleLine}>
          <Text numberOfLines={1} style={styles.title}>{chatTitle(chat)}</Text>
          {chat.type === 'group' && <Text style={styles.groupLabel}>GROUP</Text>}
        </View>
        <Text numberOfLines={1} style={styles.preview}>{preview}</Text>
      </View>
      {!!chat.unreadCount && (
        <View accessibilityLabel={`${chat.unreadCount} unread messages`} style={styles.unread}>
          <Text style={styles.unreadText}>{chat.unreadCount > 99 ? '99+' : chat.unreadCount}</Text>
        </View>
      )}
    </Pressable>
  )
}

export function ChatList() {
  const snapshot = useChatSnapshot()
  const chats = chatsFrom(snapshot)
  const [loading, setLoading] = useState(chats.length === 0)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string>()

  const load = useCallback(async (isRefresh = false) => {
    setError(undefined)
    isRefresh ? setRefreshing(true) : setLoading(true)
    try {
      await refreshChats(snapshot)
    } catch {
      setError('Unable to load conversations. Check your connection and try again.')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [snapshot])

  useEffect(() => {
    void load()
  }, [load])

  if (loading && chats.length === 0) {
    return <View style={styles.center}><ActivityIndicator size="large" color="#5b49e6" /><Text style={styles.stateText}>Loading conversations…</Text></View>
  }

  if (error && chats.length === 0) {
    return <StatePanel message={error} actionLabel="Try again" onAction={() => void load()} />
  }

  return (
    <FlatList
      accessibilityLabel="Conversations"
      contentContainerStyle={chats.length ? styles.list : styles.emptyList}
      data={chats}
      keyExtractor={(chat) => chat.id}
      renderItem={({ item }) => <ChatRow chat={item} />}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void load(true)} tintColor="#5b49e6" />}
      ListEmptyComponent={<StatePanel message="No conversations yet. Start one to practice together." actionLabel="New chat" onAction={() => router.push('/(app)/new-chat')} />}
      ListHeaderComponent={error ? <Text accessibilityRole="alert" style={styles.warning}>{error}</Text> : null}
    />
  )
}

function StatePanel({ message, actionLabel, onAction }: { message: string; actionLabel: string; onAction: () => void }) {
  return (
    <View accessibilityRole="summary" style={styles.center}>
      <Text style={styles.stateText}>{message}</Text>
      <Pressable accessibilityRole="button" onPress={onAction} style={styles.primaryButton}>
        <Text style={styles.primaryButtonText}>{actionLabel}</Text>
      </Pressable>
    </View>
  )
}

const styles = StyleSheet.create({
  list: { paddingBottom: 24 },
  emptyList: { flexGrow: 1 },
  center: { alignItems: 'center', flex: 1, gap: 14, justifyContent: 'center', padding: 28 },
  stateText: { color: '#5d5d68', fontSize: 16, lineHeight: 23, textAlign: 'center' },
  row: { alignItems: 'center', borderBottomColor: '#eceaf4', borderBottomWidth: StyleSheet.hairlineWidth, flexDirection: 'row', gap: 12, paddingHorizontal: 20, paddingVertical: 15 },
  pressed: { backgroundColor: '#f5f3ff' },
  avatar: { alignItems: 'center', backgroundColor: '#ded9ff', borderRadius: 24, height: 48, justifyContent: 'center', width: 48 },
  avatarText: { color: '#382d8c', fontSize: 18, fontWeight: '700' },
  content: { flex: 1, gap: 4 },
  titleLine: { alignItems: 'center', flexDirection: 'row', gap: 7 },
  title: { color: '#171523', flexShrink: 1, fontSize: 16, fontWeight: '700' },
  groupLabel: { color: '#746f84', fontSize: 10, fontWeight: '700', letterSpacing: .7 },
  preview: { color: '#746f84', fontSize: 14 },
  unread: { alignItems: 'center', backgroundColor: '#5b49e6', borderRadius: 12, justifyContent: 'center', minWidth: 23, paddingHorizontal: 6, paddingVertical: 3 },
  unreadText: { color: '#fff', fontSize: 11, fontWeight: '700' },
  warning: { backgroundColor: '#fff5df', color: '#765400', margin: 12, padding: 10 },
  primaryButton: { backgroundColor: '#5b49e6', borderRadius: 10, paddingHorizontal: 18, paddingVertical: 11 },
  primaryButtonText: { color: '#fff', fontWeight: '700' },
})
