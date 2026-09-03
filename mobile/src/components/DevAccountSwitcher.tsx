import React from 'react'
import { View, Text, Pressable, StyleSheet } from 'react-native'
import { DEV_ACCOUNTS } from '@chorus/shared'
import { COLOR, TYPOGRAPHY, RADIUS, SPACING } from '../theme'

type Props = {
  onSelect: (a: { email: string; password: string; username: string }) => void
}

// Runtime gate: RN global `__DEV__` is true only in dev/metro, false in release
// builds — the component (and DEV_ACCOUNTS import after minify) is stripped
// from production APK/IPA. Also respects explicit env kill-switch.
export default function DevAccountSwitcher({ onSelect }: Props) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const isDev = typeof (globalThis as any).__DEV__ !== 'undefined' ? (globalThis as any).__DEV__ : (typeof __DEV__ !== 'undefined' ? (__DEV__ as unknown as boolean) : true)
  const disabled = !isDev || process.env.EXPO_PUBLIC_ENABLE_TEST_ACCOUNTS === 'false'
  if (disabled) return null

  return (
    <View style={styles.wrap}>
      <View style={styles.header}>
        <Text style={styles.badge}>DEV ONLY</Text>
        <Text style={styles.hint}>Test accounts — not in production build</Text>
      </View>
      <View style={styles.list}>
        {DEV_ACCOUNTS.map((a) => (
          <Pressable
            key={a.email}
            onPress={() => onSelect({ email: a.email, username: a.username, password: a.password })}
            style={({ pressed }) => [styles.item, pressed && styles.itemPressed]}>
            <View style={styles.itemText}>
              <Text style={styles.label}>{a.label}</Text>
              <Text style={styles.email}>{a.email}</Text>
            </View>
            <Text style={styles.fill}>Fill →</Text>
          </Pressable>
        ))}
      </View>
      <Text style={styles.foot}>Password for all: ChorusDev123! — seed via `go run ./cmd/server --seed-dev`</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: {
    borderWidth: 1,
    borderColor: '#FDE68A',
    backgroundColor: '#FFFBEB',
    borderRadius: RADIUS.lg,
    padding: SPACING.stackMd,
    gap: 8,
  },
  header: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  badge: { fontSize: 10, fontWeight: '700', letterSpacing: 0.8, color: '#92400E' },
  hint: { fontSize: 11, color: '#B45309', flex: 1 },
  list: { gap: 6 },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: 'white',
    borderWidth: 1,
    borderColor: '#FDE68A',
    borderRadius: RADIUS.lg,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  itemPressed: { backgroundColor: '#FEF3C7' },
  itemText: { flex: 1, gap: 2 },
  label: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurface, fontSize: 13 },
  email: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontSize: 11 },
  fill: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, marginLeft: 8 },
  foot: { fontSize: 10, color: '#B45309', opacity: 0.7 },
})
