import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, Pressable, ActivityIndicator } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { RealTalkPrompt, User } from '@chorus/shared';
import StudySandbox from '../components/StudySandbox';

const CATEGORIES = ['All', 'Icebreakers', 'Deep Dives', 'Task-Based'] as const;
type Tab = typeof CATEGORIES[number];

export default function RealTalkHubScreen({ navigation }: any) {
  const [user, setUser] = useState<User | null>(null);
  const [prompts, setPrompts] = useState<RealTalkPrompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<Tab>('All');
  const [dashboard, setDashboard] = useState<any>(null);

  useEffect(() => {
    storage.getItem('user').then((s) => { if (s) try { setUser(JSON.parse(s)); } catch {} });
  }, []);

  const targetLanguage = user?.targetLanguages?.[0] ?? 'es';
  const nativeLanguage = user?.nativeLanguage ?? 'en';

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      const [p, d] = await Promise.all([
        apiService.getRealTalkPrompts(targetLanguage, nativeLanguage).catch(() => [] as RealTalkPrompt[]),
        apiService.getLearningDashboard(targetLanguage, nativeLanguage).catch(() => null),
      ]);
      setPrompts(p as RealTalkPrompt[]);
      setDashboard(d);
    } finally { setLoading(false); }
  }, [user, targetLanguage, nativeLanguage]);

  useEffect(() => { load(); }, [load]);

  const filtered = useMemo(() => {
    if (tab === 'All') return prompts;
    return prompts.filter((p) => p.category === tab);
  }, [prompts, tab]);

  const useInChat = useCallback(async (prompt: RealTalkPrompt) => {
    try { await apiService.markRealTalkUsed(prompt.id); } catch {}
    await storage.setItem('realTalkDraft', prompt.text);
    navigation.navigate('ChatsTab' as never);
  }, [navigation]);

  const currentUnit = dashboard?.currentUnit;

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Real Talk Starters</Text>
      <Text style={styles.subtitle}>Tap to drop a prompt into chat</Text>

      {currentUnit ? (
        <View style={styles.goalCard}>
          <View style={styles.goalTop}>
            <View style={styles.goalLevel}><Text style={styles.goalLevelText}>{currentUnit.cefrLevel}</Text></View>
            <Text style={styles.goalLabel}>Current Goal</Text>
          </View>
          <Text style={styles.goalTitle}>{currentUnit.title}</Text>
          <Text style={styles.goalCanDo}>{currentUnit.canDoStatement}</Text>
        </View>
      ) : null}

      <View style={styles.tabs}>
        {CATEGORIES.map((c) => (
          <Pressable key={c} onPress={() => setTab(c)} style={[styles.tab, tab === c && styles.tabActive]}>
            <Text style={[styles.tabText, tab === c && styles.tabTextActive]}>{c}</Text>
          </Pressable>
        ))}
      </View>

      {loading ? <ActivityIndicator color={COLOR.primary} style={{ marginTop: 24 }} /> : null}
      {!loading && filtered.length === 0 ? <Text style={styles.empty}>No prompts in this category yet.</Text> : null}

      {filtered.map((prompt) => (
        <View key={prompt.id} style={styles.card}>
          <View style={styles.cardTop}>
            <View style={styles.category}><Text style={styles.categoryText}>{prompt.category}</Text></View>
          </View>
          <Text style={styles.prompt}>“{prompt.text}”</Text>
          <View style={styles.cardBottom}>
            <Text style={styles.hint}>Try using target language</Text>
            <Pressable onPress={() => useInChat(prompt)} style={styles.useButton}><Text style={styles.useButtonText}>Use in Chat →</Text></Pressable>
          </View>
        </View>
      ))}

      <StudySandbox />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48, gap: SPACING.stackSm },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, marginBottom: 4 },
  goalCard: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: RADIUS.xl, padding: SPACING.stackMd, borderLeftWidth: 4, borderLeftColor: COLOR.primary, gap: 4 },
  goalTop: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  goalLevel: { backgroundColor: 'rgba(0,74,198,0.1)', borderRadius: 999, paddingHorizontal: 8, paddingVertical: 2 },
  goalLevelText: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, fontFamily: FONTS.label },
  goalLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label, textTransform: 'uppercase' },
  goalTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.primary, fontFamily: FONTS.headline },
  goalCanDo: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  tabs: { flexDirection: 'row', gap: 8, marginVertical: 4 },
  tab: { paddingHorizontal: 14, paddingVertical: 8, borderRadius: 999, backgroundColor: COLOR.surfaceContainer },
  tabActive: { backgroundColor: COLOR.primary },
  tabText: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  tabTextActive: { color: COLOR.onPrimary },
  empty: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center', marginTop: 16 },
  card: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, gap: SPACING.stackSm, ...SHADOWS.elevation1, borderWidth: 1, borderColor: COLOR.outlineVariant },
  cardTop: { flexDirection: 'row', justifyContent: 'space-between' },
  category: { backgroundColor: COLOR.secondaryFixed, borderRadius: 999, paddingHorizontal: 8, paddingVertical: 2 },
  categoryText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSecondaryFixed, fontFamily: FONTS.label },
  prompt: { ...TYPOGRAPHY.bodyLg, color: COLOR.onSurface, fontFamily: FONTS.body, lineHeight: 24 },
  cardBottom: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', borderTopWidth: 1, borderTopColor: COLOR.outlineVariant, paddingTop: 8, marginTop: 4 },
  hint: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, fontFamily: FONTS.label },
  useButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 8 },
  useButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
});
