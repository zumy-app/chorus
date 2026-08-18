import { useRouter } from 'expo-router'
import { useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { authApi, getApiErrorMessage } from '@/services/api'

const languages = [
  { code: 'en', label: 'English' },
  { code: 'es', label: 'Spanish' },
  { code: 'fr', label: 'French' },
  { code: 'de', label: 'German' },
  { code: 'hi', label: 'Hindi' },
  { code: 'ja', label: 'Japanese' },
]

export default function ProfileOnboardingScreen() {
  const router = useRouter()
  const [displayName, setDisplayName] = useState('')
  const [nativeLanguage, setNativeLanguage] = useState('en')
  const [targetLanguages, setTargetLanguages] = useState<string[]>([])
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  function toggleTarget(language: string) {
    setTargetLanguages((current) =>
      current.includes(language) ? current.filter((item) => item !== language) : [...current, language],
    )
  }

  async function submit() {
    if (!displayName.trim()) return setError('Enter the name your contacts should see.')
    if (!targetLanguages.length) return setError('Choose at least one language to learn.')

    setSubmitting(true)
    setError(undefined)
    try {
      await authApi.updateMe({ displayName: displayName.trim(), nativeLanguage, targetLanguages })
      router.replace('/(app)')
    } catch (cause) {
      setError(getApiErrorMessage(cause))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View style={styles.screen}>
      <Text style={styles.eyebrow}>ONE MORE STEP</Text>
      <Text style={styles.title}>Make Chorus yours</Text>
      <Text style={styles.subtitle}>Choose a name and the languages you want help with in every chat.</Text>
      <TextInput autoComplete="name" onChangeText={setDisplayName} placeholder="Your name" style={styles.input} value={displayName} />
      <Text style={styles.label}>I speak</Text>
      <View style={styles.chips}>
        {languages.map((language) => <Pressable key={language.code} onPress={() => setNativeLanguage(language.code)} style={[styles.chip, nativeLanguage === language.code && styles.selectedChip]}><Text style={nativeLanguage === language.code ? styles.selectedChipText : styles.chipText}>{language.label}</Text></Pressable>)}
      </View>
      <Text style={styles.label}>I want to learn</Text>
      <View style={styles.chips}>
        {languages.filter((language) => language.code !== nativeLanguage).map((language) => <Pressable key={language.code} onPress={() => toggleTarget(language.code)} style={[styles.chip, targetLanguages.includes(language.code) && styles.selectedChip]}><Text style={targetLanguages.includes(language.code) ? styles.selectedChipText : styles.chipText}>{language.label}</Text></Pressable>)}
      </View>
      {error ? <Text accessibilityLiveRegion="polite" style={styles.error}>{error}</Text> : null}
      <Pressable accessibilityRole="button" disabled={submitting} onPress={submit} style={styles.button}>
        {submitting ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Finish setup</Text>}
      </Pressable>
    </View>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', padding: 24, gap: 12, backgroundColor: '#fff' },
  eyebrow: { color: '#2563eb', fontSize: 12, fontWeight: '700', letterSpacing: 1 },
  title: { color: '#18212f', fontSize: 30, fontWeight: '700' },
  subtitle: { color: '#526170', fontSize: 16, lineHeight: 23, marginBottom: 8 },
  input: { borderWidth: 1, borderColor: '#c9d2dc', borderRadius: 10, padding: 14, fontSize: 16 },
  label: { color: '#18212f', fontSize: 16, fontWeight: '600', marginTop: 4 },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { borderColor: '#c9d2dc', borderRadius: 20, borderWidth: 1, paddingHorizontal: 12, paddingVertical: 8 },
  selectedChip: { backgroundColor: '#dbeafe', borderColor: '#2563eb' },
  chipText: { color: '#334155' },
  selectedChipText: { color: '#1d4ed8', fontWeight: '700' },
  button: { minHeight: 50, alignItems: 'center', justifyContent: 'center', borderRadius: 10, backgroundColor: '#2563eb', marginTop: 8 },
  buttonText: { color: '#fff', fontSize: 16, fontWeight: '700' },
  error: { color: '#b42318', fontSize: 14 },
})
