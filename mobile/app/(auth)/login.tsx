import { Link, useRouter } from 'expo-router'
import { useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { validateEmail } from './form'
import { authApi, getApiErrorMessage } from '@/services/api'

export default function LoginScreen() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  async function submit() {
    const validationError = validateEmail(email) ?? (password ? undefined : 'Enter your password.')
    if (validationError) return setError(validationError)

    setSubmitting(true)
    setError(undefined)
    try {
      await authApi.login(email, password)
      router.replace('/(app)')
    } catch (cause) {
      setError(getApiErrorMessage(cause))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View style={styles.screen}>
      <Text style={styles.title}>Welcome back</Text>
      <Text style={styles.subtitle}>Sign in to continue your conversations.</Text>
      <TextInput autoCapitalize="none" autoComplete="email" keyboardType="email-address" onChangeText={setEmail} placeholder="Email address" style={styles.input} value={email} />
      <TextInput autoComplete="password" onChangeText={setPassword} placeholder="Password" secureTextEntry style={styles.input} value={password} />
      {error ? <Text accessibilityLiveRegion="polite" style={styles.error}>{error}</Text> : null}
      <Pressable accessibilityRole="button" disabled={submitting} onPress={submit} style={styles.button}>
        {submitting ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Sign in</Text>}
      </Pressable>
      <Link href="/(auth)/forgot-password" style={styles.link}>Forgot password?</Link>
      <Link href="/(auth)/register" style={styles.link}>New to Chorus? Create an account</Link>
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
