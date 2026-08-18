import { router } from 'expo-router'
import { Pressable, SafeAreaView, StyleSheet, Text, View } from 'react-native'

import { ChatList } from '@/components/chat-list'

export default function ChatsScreen() {
  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.header}>
        <View>
          <Text accessibilityRole="header" style={styles.title}>Chats</Text>
          <Text style={styles.subtitle}>Keep the conversation going</Text>
        </View>
        <Pressable accessibilityRole="button" accessibilityLabel="Start a new chat" onPress={() => router.push('/(app)/new-chat')} style={styles.newButton}>
          <Text style={styles.newButtonText}>New chat</Text>
        </Pressable>
      </View>
      <ChatList />
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  safe: { backgroundColor: '#fff', flex: 1 },
  header: { alignItems: 'center', borderBottomColor: '#eceaf4', borderBottomWidth: StyleSheet.hairlineWidth, flexDirection: 'row', justifyContent: 'space-between', padding: 20 },
  title: { color: '#1b182a', fontSize: 30, fontWeight: '800' },
  subtitle: { color: '#746f84', fontSize: 14, marginTop: 2 },
  newButton: { backgroundColor: '#5b49e6', borderRadius: 10, paddingHorizontal: 13, paddingVertical: 10 },
  newButtonText: { color: '#fff', fontSize: 14, fontWeight: '700' },
})
