import { router } from 'expo-router'
import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { chatApi, contactApi } from '@/services/api'
import { useChatSnapshot } from './chat-store'
import type { User } from '@/types'

export function NewChatForm() {
  const snapshot = useChatSnapshot()
  const [kind, setKind] = useState<'direct' | 'group'>('direct')
  const [query, setQuery] = useState('')
  const [contacts, setContacts] = useState<User[]>([])
  const [selected, setSelected] = useState<User[]>([])
  const [name, setName] = useState('')
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string>()

  useEffect(() => {
    const search = async () => {
      if (!query.trim()) {
        setContacts([])
        return
      }
      setLoading(true)
      try {
        setContacts(await contactApi.search(query.trim()))
        setError(undefined)
      } catch {
        setError('Unable to find people. Check your connection and try again.')
      } finally {
        setLoading(false)
      }
    }
    const timeout = setTimeout(() => void search(), 250)
    return () => clearTimeout(timeout)
  }, [query])

  const togglePerson = (person: User) => {
    setSelected((current) => {
      const exists = current.some((user) => user.id === person.id)
      if (kind === 'direct') return exists ? [] : [person]
      return exists ? current.filter((user) => user.id !== person.id) : [...current, person]
    })
  }

  const createChat = async () => {
    const valid = kind === 'direct' ? selected.length === 1 : selected.length >= 2
    if (!valid) {
      setError(kind === 'direct' ? 'Select one person for a direct chat.' : 'Select at least two people for a group chat.')
      return
    }
    if (kind === 'group' && !name.trim()) {
      setError('Enter a name for the group.')
      return
    }
    setSubmitting(true)
    setError(undefined)
    try {
      const chat = await chatApi.create({ type: kind, participants: selected.map((person) => person.id), name: kind === 'group' ? name.trim() : undefined })
      snapshot.applySnapshot?.([chat])
      router.replace(`/(app)/chat/${chat.id}`)
    } catch {
      setError('Unable to create the chat. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <View style={styles.container}>
      <View accessibilityRole="tablist" style={styles.tabs}>
        {(['direct', 'group'] as const).map((option) => (
          <Pressable key={option} accessibilityRole="tab" accessibilityState={{ selected: kind === option }} onPress={() => { setKind(option); setSelected([]); setError(undefined) }} style={[styles.tab, kind === option && styles.activeTab]}>
            <Text style={[styles.tabText, kind === option && styles.activeTabText]}>{option === 'direct' ? 'Direct' : 'Group'}</Text>
          </Pressable>
        ))}
      </View>
      {kind === 'group' && <TextInput accessibilityLabel="Group name" onChangeText={setName} placeholder="Group name" placeholderTextColor="#827c91" style={styles.field} value={name} />}
      <TextInput accessibilityLabel="Search people" autoCapitalize="none" onChangeText={setQuery} placeholder="Search by name or email" placeholderTextColor="#827c91" style={styles.field} value={query} />
      {loading && <ActivityIndicator accessibilityLabel="Searching people" color="#5b49e6" style={styles.loader} />}
      {selected.length > 0 && <Text style={styles.selected}>{selected.map((person) => person.displayName).join(', ')}</Text>}
      <View style={styles.results}>
        {contacts.map((person) => {
          const isSelected = selected.some((user) => user.id === person.id)
          return <Pressable key={person.id} accessibilityRole="checkbox" accessibilityState={{ checked: isSelected }} onPress={() => togglePerson(person)} style={[styles.person, isSelected && styles.selectedPerson]}><Text style={styles.personName}>{person.displayName}</Text><Text style={styles.personEmail}>{person.email}</Text></Pressable>
        })}
        {query.trim() && !loading && contacts.length === 0 && <Text style={styles.noResults}>No people found.</Text>}
      </View>
      {error && <Text accessibilityRole="alert" style={styles.error}>{error}</Text>}
      <Pressable accessibilityRole="button" accessibilityLabel={`Create ${kind} chat`} disabled={submitting} onPress={() => void createChat()} style={[styles.create, submitting && styles.disabled]}><Text style={styles.createText}>{submitting ? 'Creating…' : `Create ${kind === 'direct' ? 'chat' : 'group'}`}</Text></Pressable>
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, gap: 13, padding: 20 },
  tabs: { backgroundColor: '#f0eef7', borderRadius: 11, flexDirection: 'row', padding: 3 },
  tab: { alignItems: 'center', borderRadius: 8, flex: 1, paddingVertical: 9 },
  activeTab: { backgroundColor: '#fff', shadowColor: '#292239', shadowOpacity: .1, shadowRadius: 4 },
  tabText: { color: '#706b80', fontWeight: '600' },
  activeTabText: { color: '#4436b2' },
  field: { backgroundColor: '#f3f1fa', borderColor: '#dedbe8', borderRadius: 10, borderWidth: 1, color: '#201d2e', fontSize: 16, paddingHorizontal: 13, paddingVertical: 12 },
  loader: { marginVertical: 6 },
  selected: { color: '#514796', fontSize: 14, fontWeight: '600' },
  results: { flex: 1 },
  person: { borderBottomColor: '#eceaf4', borderBottomWidth: StyleSheet.hairlineWidth, paddingVertical: 12 },
  selectedPerson: { backgroundColor: '#f1efff', marginHorizontal: -8, paddingHorizontal: 8 },
  personName: { color: '#201d2e', fontSize: 16, fontWeight: '700' },
  personEmail: { color: '#746f84', fontSize: 13, marginTop: 2 },
  noResults: { color: '#746f84', paddingTop: 14, textAlign: 'center' },
  error: { color: '#ae3030', fontSize: 14 },
  create: { alignItems: 'center', backgroundColor: '#5b49e6', borderRadius: 10, paddingVertical: 14 },
  disabled: { backgroundColor: '#bdb7d8' },
  createText: { color: '#fff', fontSize: 16, fontWeight: '700' },
})
