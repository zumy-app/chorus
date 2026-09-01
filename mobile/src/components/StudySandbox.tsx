import React, { useState } from 'react';
import { View, Text, StyleSheet, Pressable } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import { useStudySandbox } from '../hooks/useStudySandbox';

export default function StudySandbox() {
  const { state, addActivity } = useStudySandbox();
  const [raised, setRaised] = useState(false);
  const [saved, setSaved] = useState(false);
  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Text style={styles.headerIcon}>🧩</Text>
          <Text style={styles.headerTitle}>{state.topic}</Text>
        </View>
        <View style={styles.badge}><Text style={styles.badgeText}>Sandbox</Text></View>
      </View>
      <View style={styles.card}>
        <Text style={styles.prompt}>“{state.prompt}”</Text>
        <Text style={styles.hint}>Focus on using past tense verbs. Group sandbox — everyone sees the same prompt.</Text>
      </View>
      <View style={styles.actions}>
        <Pressable
          onPress={() => { setRaised((v) => !v); addActivity('You', raised ? 'lowered hand' : 'raised hand'); }}
          style={[styles.buttonPrimary, raised && styles.buttonRaised]}>
          <Text style={styles.buttonPrimaryText}>{raised ? 'Hand raised' : 'Raise Hand'}</Text>
        </Pressable>
        <Pressable
          onPress={() => { setSaved(true); addActivity('You', 'saved a word'); setTimeout(() => setSaved(false), 1500); }}
          style={styles.buttonSecondary}>
          <Text style={styles.buttonSecondaryText}>{saved ? 'Saved!' : 'Save Word'}</Text>
        </Pressable>
      </View>
      {state.activityLog.length > 0 ? (
        <View style={styles.log}>
          {state.activityLog.slice(-3).map((e) => (
            <Text key={e.id} style={styles.logText}>{e.user}: {e.text}</Text>
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { backgroundColor: COLOR.surfaceContainerLow, borderRadius: RADIUS.xl, padding: SPACING.stackMd, gap: SPACING.stackSm, borderWidth: 1, borderColor: COLOR.outlineVariant, ...SHADOWS.elevation1 },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  headerLeft: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  headerIcon: { fontSize: 18 },
  headerTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 16 },
  badge: { backgroundColor: COLOR.secondaryContainer, borderRadius: 6, paddingHorizontal: 8, paddingVertical: 2 },
  badgeText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSecondaryContainer, fontFamily: FONTS.label },
  card: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.lg, padding: SPACING.stackMd, alignItems: 'center', gap: 6 },
  prompt: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, textAlign: 'center' },
  hint: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center' },
  actions: { flexDirection: 'row', justifyContent: 'space-between', gap: SPACING.stackSm },
  buttonPrimary: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 10 },
  buttonRaised: { backgroundColor: COLOR.secondaryContainer },
  buttonPrimaryText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  buttonSecondary: { backgroundColor: COLOR.surface, borderWidth: 1, borderColor: COLOR.outlineVariant, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 10 },
  buttonSecondaryText: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  log: { gap: 2 },
  logText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
});
