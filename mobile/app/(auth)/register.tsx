import { Link, useLocalSearchParams, useRouter } from 'expo-router'
import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { validateRegistration } from './form'
import { api, authApi, getApiErrorMessage } from '@/services/api'

type InviteResponse = { email: string }

export default function RegisterScreen() {
  const router = useRouter()
  const { inviteToken } = useLocalSearchParams<{ inviteToken?: string }>()
  const token = typeof inviteToken === 'string' ? inviteToken : undefined
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!token) return
    api
      .get<InviteResponse>('/auth/invite', { params: { token } })
      .then(({ data }) => setEmail(data.email))
      .catch(() => setError('This invitation is invalid, expired, or already used.'))
  }, [token])

  async function submit() {
    const validationError = validateRegistration({ email, password, confirmPassword, displayName })
    if (validationError) return setError(validationError)

    setSubmitting(true)
    setError(undefined)
    try {
      await authApi.register({
        email,
        password,
        displayName: displayName.trim(),
        nativeLanguage: 'en',
        targetLanguages: [],
        inviteToken: token,
      })
      router.replace('/(onboarding)/profile')
    } catch (cause) {
      setError(getApiErrorMessage(cause))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View style={styles.screen}>
      <Text style={styles.title}>Create your account</Text>
      <Text style={styles.subtitle}>{token ? 'Your invitation will be applied when you join.' : 'Start messaging across languages.'}</Text>
      <TextInput autoCapitalize="none" autoComplete="email" editable={!token} keyboardType="email-address" onChangeText={setEmail} placeholder="Email address" style={styles.input} value={email} />
      <TextInput autoComplete="name" onChangeText={setDisplayName} placeholder="Your name" style={styles.input} value={displayName} />
      <TextInput autoComplete="new-password" onChangeText={setPassword} placeholder="Password (8+ characters)" secureTextEntry style={styles.input} value={password} />
      <TextInput autoComplete="new-password" onChangeText={setConfirmPassword} placeholder="Confirm password" secureTextEntry style={styles.input} value={confirmPassword} />
      {error ? <Text accessibilityLiveRegion="polite" style={styles.error}>{error}</Text> : null}
      <Pressable accessibilityRole="button" disabled={submitting} onPress={submit} style={styles.button}>
        {submitting ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Create account</Text>}
      </Pressable>
      <Link href="/(auth)/login" style={styles.link}>Already have an account? Sign in</Link>
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
