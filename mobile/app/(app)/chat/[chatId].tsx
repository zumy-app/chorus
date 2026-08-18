import { Stack, useLocalSearchParams } from 'expo-router'
import { useEffect, useState } from 'react'
import { ActivityIndicator, SafeAreaView, StyleSheet, Text, View } from 'react-native'

import { ChatThread } from '@/components/chat-thread'
import { chatsFrom, useChatSnapshot } from '@/components/chat-store'
import { chatApi } from '@/services/api'
import type { Chat } from '@/types'

export default function ChatScreen() {
  const { chatId } = useLocalSearchParams<{ chatId: string }>()
  const snapshot = useChatSnapshot()
  const storedChat = chatsFrom(snapshot).find((chat) => chat.id === chatId)
  const [chat, setChat] = useState<Chat | undefined>(storedChat)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (storedChat || !chatId) return
    chatApi.get(chatId)
      .then((result) => {
        snapshot.applySnapshot?.([result])
        setChat(result)
      })
      .catch(() => setError(true))
  }, [chatId, snapshot, storedChat])

  const activeChat = storedChat ?? chat
  if (!activeChat) {
    return (
      <SafeAreaView style={styles.safe}>
        <Stack.Screen options={{ headerShown: true, title: 'Chat' }} />
        <View style={styles.center}>
          {error ? <Text style={styles.error}>This chat is unavailable. Return to your conversations and try again.</Text> : <><ActivityIndicator size="large" color="#5b49e6" /><Text style={styles.loading}>Opening conversation…</Text></>}
        </View>
      </SafeAreaView>
    )
  }

  return (
    <SafeAreaView style={styles.safe}>
      <Stack.Screen options={{ headerShown: true, title: activeChat.name || (activeChat.type === 'group' ? 'Group chat' : 'Chat') }} />
      <ChatThread chat={activeChat} />
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  safe: { backgroundColor: '#fbfaff', flex: 1 },
  center: { alignItems: 'center', flex: 1, gap: 12, justifyContent: 'center', padding: 28 },
  loading: { color: '#746f84', fontSize: 15 },
  error: { color: '#a53030', fontSize: 16, lineHeight: 23, textAlign: 'center' },
})
