import { useState } from 'react'
import { Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { useChatSnapshot } from './chat-store'

const QUICK_EMOJIS = ['👍', '😊', '🎉', '❤️']

export function MessageComposer({ chatId }: { chatId: string }) {
  const snapshot = useChatSnapshot()
  const [text, setText] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string>()
  const offline = (globalThis as { navigator?: { onLine?: boolean } }).navigator?.onLine === false

  const send = async (content = text) => {
    const message = content.trim()
    if (!message || sending) return
    if (offline) {
      setError('You are offline. Your message cannot be sent yet.')
      return
    }
    if (!snapshot.sendMessage) {
      setError('Messaging is still preparing. Please try again.')
      return
    }

    setSending(true)
    setError(undefined)
    try {
      await snapshot.sendMessage(chatId, message)
      if (content === text) setText('')
    } catch {
      setError('Message could not be sent. Try again.')
    } finally {
      setSending(false)
    }
  }

  return (
    <View style={styles.container}>
      {error && <Text accessibilityRole="alert" style={styles.error}>{error}</Text>}
      {offline && <Text accessibilityRole="alert" style={styles.offline}>Offline mode</Text>}
      <View accessibilityLabel="Quick emoji replies" style={styles.quickActions}>
        {QUICK_EMOJIS.map((emoji) => (
          <Pressable key={emoji} accessibilityRole="button" accessibilityLabel={`Send ${emoji}`} disabled={sending || offline} onPress={() => void send(emoji)} style={({ pressed }) => [styles.emoji, pressed && styles.pressed]}>
            <Text style={styles.emojiText}>{emoji}</Text>
          </Pressable>
        ))}
      </View>
      <View style={styles.composer}>
        <TextInput
          accessibilityLabel="Message"
          editable={!sending && !offline}
          multiline
          onChangeText={setText}
          placeholder="Write a message"
          placeholderTextColor="#827c91"
          style={styles.input}
          value={text}
        />
        <Pressable accessibilityRole="button" accessibilityLabel="Send message" disabled={!text.trim() || sending || offline} onPress={() => void send()} style={({ pressed }) => [styles.send, (!text.trim() || sending || offline) && styles.sendDisabled, pressed && styles.pressed]}>
          <Text style={styles.sendText}>{sending ? '…' : 'Send'}</Text>
        </Pressable>
      </View>
    </View>
  )
}

const styles = StyleSheet.create({
  container: { backgroundColor: '#fff', borderTopColor: '#eceaf4', borderTopWidth: StyleSheet.hairlineWidth, paddingBottom: 12, paddingHorizontal: 14, paddingTop: 8 },
  quickActions: { flexDirection: 'row', gap: 8, marginBottom: 8 },
  emoji: { alignItems: 'center', backgroundColor: '#f3f1fa', borderRadius: 16, height: 32, justifyContent: 'center', width: 38 },
  emojiText: { fontSize: 17 },
  composer: { alignItems: 'flex-end', backgroundColor: '#f3f1fa', borderRadius: 20, flexDirection: 'row', minHeight: 46, paddingLeft: 14, paddingRight: 5 },
  input: { color: '#201d2e', flex: 1, fontSize: 16, maxHeight: 110, paddingVertical: 10 },
  send: { alignItems: 'center', backgroundColor: '#5b49e6', borderRadius: 17, justifyContent: 'center', minHeight: 34, paddingHorizontal: 13 },
  sendDisabled: { backgroundColor: '#bdb7d8' },
  sendText: { color: '#fff', fontSize: 13, fontWeight: '700' },
  error: { color: '#b03030', fontSize: 13, marginBottom: 5 },
  offline: { color: '#9a6010', fontSize: 13, fontWeight: '600', marginBottom: 5 },
  pressed: { opacity: .72 },
})
