import { Link } from 'expo-router'
import { useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { validateEmail } from './form'
import { authApi, getApiErrorMessage } from '@/services/api'

export default function ForgotPasswordScreen() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string>()
  const [sent, setSent] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  async function submit() {
    const validationError = validateEmail(email)
    if (validationError) return setError(validationError)

    setSubmitting(true)
    setError(undefined)
    try {
      await authApi.forgotPassword(email)
      setSent(true)
    } catch (cause) {
      setError(getApiErrorMessage(cause))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View style={styles.screen}>
      <Text style={styles.title}>Reset your password</Text>
      <Text style={styles.subtitle}>Enter your email and we’ll send a reset link if an account exists.</Text>
      {sent ? <Text style={styles.success}>Check your inbox for password reset instructions.</Text> : <>
        <TextInput autoCapitalize="none" autoComplete="email" keyboardType="email-address" onChangeText={setEmail} placeholder="Email address" style={styles.input} value={email} />
        {error ? <Text accessibilityLiveRegion="polite" style={styles.error}>{error}</Text> : null}
        <Pressable accessibilityRole="button" disabled={submitting} onPress={submit} style={styles.button}>
          {submitting ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Send reset link</Text>}
        </Pressable>
      </>}
      <Link href="/(auth)/login" style={styles.link}>Back to sign in</Link>
    </View>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', padding: 24, gap: 14, backgroundColor: '#fff' },
  title: { fontSize: 32, fontWeight: '700', color: '#18212f' },
  subtitle: { fontSize: 16, color: '#526170', marginBottom: 12 },
  input: { borderWidth: 1, borderColor: '#c9d2dc', borderRadius: 10, padding: 14, fontSize: 16 },
  button: { minHeight: 50, alignItems: 'center', justifyContent: 'center', borderRadius: 10, backgroundColor: '#2563eb' },
  buttonText: { color: '#fff', fontSize: 16, fontWeight: '700' },
  link: { color: '#1d4ed8', fontSize: 15, textAlign: 'center', paddingVertical: 4 },
  error: { color: '#b42318', fontSize: 14 },
  success: { color: '#166534', fontSize: 16, lineHeight: 24 },
})
