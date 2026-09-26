import React, { useState } from 'react';
import { View, Text, StyleSheet, Pressable, ActivityIndicator } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { COLOR, TYPOGRAPHY, FONTS, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';

export default function StreakRecoveryScreen() {
  const navigation = useNavigation<any>();
  const [recovering, setRecovering] = useState(false);
  const [error, setError] = useState(false);

  const recover = async (mode: string) => {
    setRecovering(true);
    setError(false);
    try {
      const raw = await storage.getItem('user');
      const user = raw ? JSON.parse(raw) : null;
      const target = user?.targetLanguages?.[0] ?? 'es';
      const native = user?.nativeLanguage ?? 'en';
      await apiService.recoverStreak(target, native);
      if (mode === 'scenario') navigation.navigate('Scenarios' as never);
      else navigation.navigate('LessonSession' as never, { mode: 'vocabulary' } as never);
    } catch {
      setError(true);
    } finally {
      setRecovering(false);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.hero}>
        <View style={styles.flameWrap}>
          <Text style={styles.flame}>🔥</Text>
          <View style={styles.badge}><Text style={styles.badgeText}>14</Text></View>
        </View>
        <Text style={styles.title}>Oh no, you missed a day!</Text>
        <Text style={styles.body}>Don&apos;t let your 14-day streak burn out completely. Complete one of these quick tasks right now to recover it.</Text>
      </View>
      {error && <Text style={styles.error}>Could not recover streak. Try again.</Text>}
      {recovering && <ActivityIndicator color={COLOR.primary} />}
      <View style={styles.cards}>
        <Pressable testID="streak-recovery-scenario" style={styles.card} onPress={() => recover('scenario')} disabled={recovering}>
          <View style={styles.iconSecondary}><Text style={styles.iconText}>🎭</Text></View>
          <View style={styles.cardBody}>
            <Text style={styles.cardTagSecondary}>The Challenge  •  5 min</Text>
            <Text style={styles.cardTitle}>Scenario Roleplay</Text>
            <Text style={styles.cardSub}>Grocery Checkout</Text>
          </View>
          <Text style={styles.chevron}>›</Text>
        </Pressable>
        <Pressable testID="streak-recovery-review" style={styles.card} onPress={() => recover('vocabulary')} disabled={recovering}>
          <View style={styles.iconTertiary}><Text style={styles.iconText}>📖</Text></View>
          <View style={styles.cardBody}>
            <Text style={styles.cardTagTertiary}>The Review</Text>
            <Text style={styles.cardTitle}>Clear 15 SRS Cards</Text>
            <Text style={styles.cardSub}>Review vocabulary due today</Text>
          </View>
          <Text style={styles.chevron}>›</Text>
        </Pressable>
      </View>
      <Pressable onPress={() => navigation.goBack()} style={styles.skipBtn}><Text style={styles.skipText}>No thanks, let it reset to 0</Text></Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background, padding: SPACING.marginMobile, gap: SPACING.stackMd },
  hero: { alignItems: 'center', gap: 8, marginTop: 24 },
  flameWrap: { width: 96, height: 96, borderRadius: 48, backgroundColor: COLOR.surfaceContainerLowest, alignItems: 'center', justifyContent: 'center', borderWidth: 1, borderColor: COLOR.outlineVariant, ...SHADOWS.elevation1 },
  flame: { fontSize: 42 },
  badge: { position: 'absolute', bottom: -6, right: -6, backgroundColor: COLOR.error, borderRadius: 12, width: 28, height: 28, alignItems: 'center', justifyContent: 'center' },
  badgeText: { color: COLOR.onError, fontWeight: '700' as const, fontSize: 12 },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, fontFamily: FONTS.headline, textAlign: 'center' as const },
  body: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center' as const },
  error: { color: COLOR.error, textAlign: 'center' as const },
  cards: { gap: SPACING.stackMd, marginTop: 8 },
  card: { flexDirection: 'row', alignItems: 'center', gap: SPACING.stackMd, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, ...SHADOWS.elevation1 },
  iconSecondary: { width: 48, height: 48, borderRadius: 24, backgroundColor: COLOR.secondaryContainer, alignItems: 'center', justifyContent: 'center' },
  iconTertiary: { width: 48, height: 48, borderRadius: 24, backgroundColor: COLOR.tertiaryContainer, alignItems: 'center', justifyContent: 'center' },
  iconText: { fontSize: 20 },
  cardBody: { flex: 1 },
  cardTagSecondary: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, fontFamily: FONTS.label },
  cardTagTertiary: { ...TYPOGRAPHY.labelSm, color: COLOR.tertiary, fontFamily: FONTS.label },
  cardTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 16 },
  cardSub: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  chevron: { fontSize: 24, color: COLOR.outlineVariant },
  skipBtn: { alignItems: 'center', padding: 12 },
  skipText: { ...TYPOGRAPHY.labelMd, color: COLOR.outlineVariant, fontFamily: FONTS.label },
});
