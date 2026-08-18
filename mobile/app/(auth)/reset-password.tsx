import { Link, useLocalSearchParams, useRouter } from 'expo-router'
import { useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { validatePassword } from './form'
import { authApi, getApiErrorMessage } from '@/services/api'

export default function ResetPasswordScreen() {
  const router = useRouter()
  const { token } = useLocalSearchParams<{ token?: string }>()
  const resetToken = typeof token === 'string' ? token : undefined
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  async function submit() {
    const validationError =
      validatePassword(password) ?? (password === confirmPassword ? undefined : 'Passwords do not match.')
    if (!resetToken) return setError('This reset link is invalid or incomplete.')
    if (validationError) return setError(validationError)

    setSubmitting(true)
    setError(undefined)
    try {
      await authApi.resetPassword(resetToken, password)
      router.replace('/(auth)/login')
    } catch (cause) {
      setError(getApiErrorMessage(cause))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View style={styles.screen}>
      <Text style={styles.title}>Choose a new password</Text>
      <Text style={styles.subtitle}>Use at least 8 characters to secure your account.</Text>
      <TextInput autoComplete="new-password" onChangeText={setPassword} placeholder="New password" secureTextEntry style={styles.input} value={password} />
      <TextInput autoComplete="new-password" onChangeText={setConfirmPassword} placeholder="Confirm new password" secureTextEntry style={styles.input} value={confirmPassword} />
      {error ? <Text accessibilityLiveRegion="polite" style={styles.error}>{error}</Text> : null}
      <Pressable accessibilityRole="button" disabled={submitting} onPress={submit} style={styles.button}>
        {submitting ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Reset password</Text>}
      </Pressable>
      <Link href="/(auth)/login" style={styles.link}>Back to sign in</Link>
    </View>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', padding: 24, gap: 14, backgroundColor: '#fff' },
  title: { fontSize: 32, fontWeight: '700', color: '#18212f' },
  subtitle: { fontSize: 16, color: '#526170', marginBottom: 12 },
  input: { borderWidth: 1, borderColor: '#c9d2dc', borderRadius: 10, padding: 14, fontSize: 16 },
  button: { minHeight: 50, alignItems: 'center', justifyContent: 'center', borderRadius: 10, backgroundColor: '#2563eb', marginTop: 6 },
  buttonText: { color: '#fff', fontSize: 16, fontWeight: '700' },
  link: { color: '#1d4ed8', fontSize: 15, textAlign: 'center', paddingVertical: 4 },
  error: { color: '#b42318', fontSize: 14 },
})
