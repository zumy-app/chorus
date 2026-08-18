import { Stack } from 'expo-router'
import { SafeAreaView, StyleSheet } from 'react-native'

import { NewChatForm } from '@/components/new-chat-form'

export default function NewChatScreen() {
  return (
    <SafeAreaView style={styles.safe}>
      <Stack.Screen options={{ headerShown: true, title: 'New chat' }} />
      <NewChatForm />
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  safe: { backgroundColor: '#fff', flex: 1 },
})
