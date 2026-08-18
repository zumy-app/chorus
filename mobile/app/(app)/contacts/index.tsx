import * as Contacts from 'expo-contacts'
import * as Crypto from 'expo-crypto'
import { router } from 'expo-router'
import { useCallback, useState } from 'react'
import { Alert, FlatList, Pressable, SafeAreaView, StyleSheet, Text, View } from 'react-native'

import { createContactHashes } from '@/services/contacts'

type ContactPreview = { id: string; name: string }

export default function ContactsScreen() {
  const [contacts, setContacts] = useState<ContactPreview[]>([])
  const [loading, setLoading] = useState(false)

  const loadContacts = useCallback(async () => {
    setLoading(true)
    try {
      const permission = await Contacts.requestPermissionsAsync()
      if (permission.status !== 'granted') {
        Alert.alert('Contacts permission', 'Allow contact access to show people you may know. Your address book is not uploaded.')
        return
      }
      const response = await Contacts.getContactsAsync({
        fields: [Contacts.Fields.Emails, Contacts.Fields.PhoneNumbers],
      })
      // Compute derived values locally. A server-side hash-matching endpoint is
      // intentionally required before these values can be sent anywhere.
      await createContactHashes(
        response.data.map((contact) => ({
          emails: contact.emails?.map((item) => item.email ?? '').filter(Boolean),
          phones: contact.phoneNumbers?.map((item) => item.number ?? '').filter(Boolean),
        })),
        (value) => Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, value),
      )
      setContacts(response.data.map((contact) => ({ id: contact.id, name: contact.name || 'Unnamed contact' })))
    } finally {
      setLoading(false)
    }
  }, [])

  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.header}>
        <Text accessibilityRole="header" style={styles.title}>Contacts</Text>
        <Pressable accessibilityRole="button" onPress={() => router.back()}><Text style={styles.back}>Back</Text></Pressable>
      </View>
      <Text style={styles.description}>Discover contacts privately. Chorus keeps raw address-book details on your device.</Text>
      <Pressable accessibilityRole="button" disabled={loading} onPress={() => void loadContacts()} style={styles.button}>
        <Text style={styles.buttonText}>{loading ? 'Loading contacts…' : 'Load contacts'}</Text>
      </Pressable>
      <FlatList
        data={contacts}
        keyExtractor={(item) => item.id}
        ListEmptyComponent={<Text style={styles.empty}>Allow access to view your contacts. Matching will be available when the privacy-preserving server endpoint is configured.</Text>}
        renderItem={({ item }) => <Text style={styles.contact}>{item.name}</Text>}
      />
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  safe: { backgroundColor: '#fff', flex: 1, padding: 20 },
  header: { alignItems: 'center', flexDirection: 'row', justifyContent: 'space-between' },
  title: { color: '#1b182a', fontSize: 28, fontWeight: '800' },
  back: { color: '#5b49e6', fontWeight: '700' },
  description: { color: '#746f84', lineHeight: 20, marginTop: 16 },
  button: { alignSelf: 'flex-start', backgroundColor: '#5b49e6', borderRadius: 10, marginVertical: 18, paddingHorizontal: 14, paddingVertical: 11 },
  buttonText: { color: '#fff', fontWeight: '700' },
  empty: { color: '#746f84', lineHeight: 20, marginTop: 18 },
  contact: { borderBottomColor: '#eceaf4', borderBottomWidth: StyleSheet.hairlineWidth, color: '#1b182a', fontSize: 16, paddingVertical: 14 },
})
