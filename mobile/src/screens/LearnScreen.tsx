import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, Pressable, ActivityIndicator, Alert } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { LearnStackParamList } from '../components/MainTabs';
import type { LearningDashboard, User } from '@chorus/shared';

type LearnNav = NativeStackNavigationProp<LearnStackParamList, 'Learn'>;

export default function LearnScreen() {
  const navigation = useNavigation<LearnNav>();
  const [user, setUser] = useState<User | null>(null);
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es';
  const nativeLanguage = user?.nativeLanguage ?? 'en';
  const [dashboard, setDashboard] = useState<LearningDashboard | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    apiService
      .getLearningDashboard(targetLanguage, nativeLanguage)
      .then(setDashboard)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [targetLanguage, nativeLanguage]);

  useEffect(() => {
    storage.getItem('user').then((userStr) => {
      if (userStr) {
        try {
          setUser(JSON.parse(userStr));
        } catch {
          // ignore
        }
      }
    });
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const startSession = (mode: string) =>
    navigation.navigate('LessonSession', { sessionId: undefined, mode });

  const openPlacement = () => navigation.navigate('Placement' as never);

  if (loading) {
    return (
      <View style={styles.container}>
        <ActivityIndicator color={COLOR.primary} style={{ marginTop: 48 }} />
      </View>
    );
  }

  const d = dashboard;
  const pendingPlacement = d?.profile.placementStatus === 'not_started' || d?.profile.placementStatus === 'in_progress';

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      showsVerticalScrollIndicator={false}>
      <View style={styles.header}>
        <Text style={styles.title}>Your Fluency Journey</Text>
        <Text style={styles.subtitle}>Track your progress and practice smart.</Text>
      </View>

      {pendingPlacement && d?.capability.supportTier === 'full_course' && (
        <View style={[styles.card, styles.placementCard]}>
          <Text style={styles.cardTitle}>Find your starting level</Text>
          <Text style={styles.cardSubtitle}>Take a short placement test or start from the beginning.</Text>
          <View style={{ flexDirection: 'row', gap: SPACING.stackSm, marginTop: SPACING.stackMd }}>
            <Pressable style={styles.primaryButton} onPress={openPlacement}>
              <Text style={styles.primaryButtonText}>Start test</Text>
            </Pressable>
            <Pressable
              style={styles.softButton}
              onPress={() =>
                apiService.skipPlacement(targetLanguage, nativeLanguage).then(() => load()).catch(() => Alert.alert('Could not skip'))
              }>
              <Text style={styles.softButtonText}>Start from scratch</Text>
            </Pressable>
          </View>
        </View>
      )}

      {/* Daily Goal */}
      <View style={styles.dailyGoalCard}>
        <View style={styles.dailyGoalBody}>
          <Text style={styles.cardTitle}>Daily Goal</Text>
          <Text style={styles.cardSubtitle}>
            {d?.dailyGoal.completedItems ?? 0} / {d?.dailyGoal.targetItems ?? 10} completed
          </Text>
          <View style={styles.streakBadge}>
            <Text style={styles.streakIcon}>🔥</Text>
            <Text style={styles.streakText}>{d?.streak.days ?? 0} Day Streak</Text>
          </View>
        </View>
        <View style={styles.goalRight}>
          <Text style={styles.goalPct}>{d?.dailyGoal.percent ?? 0}%</Text>
          <View style={styles.progressTrack}>
            <View style={[styles.progressFill, { width: `${d?.dailyGoal.percent ?? 0}%` }]} />
          </View>
        </View>
      </View>

      {/* Stat cards */}
      <View style={styles.statRow}>
        <View style={styles.statCard}>
          <View style={[styles.statIconWrap, styles.statIconPrimaryWrap]}>
            <Text style={styles.statIcon}>📖</Text>
          </View>
          <Text style={styles.statValue}>{d?.vocabulary.total ?? 0}</Text>
          <Text style={styles.statLabel}>Words</Text>
        </View>
        <View style={styles.statCard}>
          <View style={[styles.statIconWrap, styles.statIconTertiaryWrap]}>
            <Text style={styles.statIcon}>🎯</Text>
          </View>
          <Text style={styles.statValue}>{d?.fluency.readinessScore ?? 0}</Text>
          <Text style={styles.statLabel}>Readiness</Text>
        </View>
      </View>

      {/* Fluency / current unit */}
      <View style={styles.grammarCard}>
        <View style={styles.grammarLeft}>
          <View style={[styles.grammarIconWrap, styles.grammarIconSecondaryWrap]}>
            <Text style={styles.grammarIcon}>⚡</Text>
          </View>
          <View>
            <Text style={styles.grammarLabel}>Fluency</Text>
            <Text style={styles.grammarValue}>{d?.profile.currentCefrLevel ?? 'A1'} · {d?.fluency.label ?? 'Building A1'}</Text>
          </View>
        </View>
        {d?.currentUnit && <Text style={styles.chevron}>›</Text>}
      </View>

      {/* Quick actions */}
      <View style={styles.quickRow}>
        {[
          { label: 'Drills', glyph: '⚡', onPress: () => startSession('quick_drill') },
          { label: 'Vocabulary', glyph: '📖', onPress: () => navigation.navigate('VocabularyReview' as never) },
          { label: 'Scenarios', glyph: '🎭', onPress: () => navigation.navigate('Scenarios' as never) },
          { label: 'Roadmap', glyph: '🗺️', onPress: () => navigation.navigate('LearningRoadmap' as never) },
        ].map((a) => (
          <Pressable key={a.label} style={styles.quickCard} onPress={a.onPress}>
            <Text style={styles.quickGlyph}>{a.glyph}</Text>
            <Text style={styles.quickLabel}>{a.label}</Text>
          </Pressable>
        ))}
      </View>

      {/* Recommended Activities */}
      <View style={styles.sectionHeader}>
        <Text style={styles.sectionTitle}>Recommended Activities</Text>
      </View>

      {(d?.recommendedActivities ?? []).map((activity) => (
        <Pressable
          key={activity.id}
          style={styles.activityCard}
          onPress={() =>
            activity.action === 'open_scenarios'
              ? navigation.navigate('Scenarios' as never)
              : startSession(activity.type === 'lesson' ? 'daily' : activity.id === 'vocabulary' ? 'vocabulary' : activity.type)
          }>
          <View style={styles.activityTitleRow}>
            <Text style={styles.activityTipIcon}>{activity.priority === 'high' ? '⚠️' : '💡'}</Text>
            <Text style={styles.activityTitle}>{activity.title}</Text>
          </View>
          <Text style={styles.activityDesc}>{activity.description}</Text>
          <View style={styles.activityFooter}>
            <Text style={styles.activityMeta}>{activity.estimatedMinutes} min</Text>
            <View style={styles.startButton}>
              <Text style={styles.startButtonText}>Start</Text>
            </View>
          </View>
        </Pressable>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  header: { marginBottom: SPACING.stackLg },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, marginBottom: SPACING.unit, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: 24, padding: SPACING.stackLg },
  placementCard: { marginBottom: SPACING.stackMd, gap: SPACING.stackSm },
  cardTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline },
  cardSubtitle: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, marginTop: 2, fontFamily: FONTS.body },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 10 },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  softButton: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 10 },
  softButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  dailyGoalCard: {
    ...SHADOWS.elevation1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 24,
    padding: SPACING.stackLg,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  dailyGoalBody: { flex: 1 },
  streakBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'flex-start',
    backgroundColor: 'rgba(208, 188, 255, 0.3)',
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 6,
    marginTop: 12,
  },
  streakIcon: { fontSize: 16, marginRight: 6 },
  streakText: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  goalRight: { alignItems: 'center', width: 88 },
  goalPct: { ...TYPOGRAPHY.headlineMd, color: COLOR.primary, fontFamily: FONTS.headline, marginBottom: 8 },
  progressTrack: { height: 10, borderRadius: 5, backgroundColor: COLOR.surfaceContainerHigh, width: 88, overflow: 'hidden' },
  progressFill: { height: 10, borderRadius: 5, backgroundColor: COLOR.primary },
  statRow: { flexDirection: 'row', gap: SPACING.stackMd, marginTop: SPACING.stackMd },
  statCard: { ...SHADOWS.elevation1, flex: 1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: 20, padding: SPACING.stackMd },
  statIconWrap: { width: 40, height: 40, borderRadius: 20, alignItems: 'center', justifyContent: 'center', marginBottom: SPACING.stackSm },
  statIconPrimaryWrap: { backgroundColor: 'rgba(37, 99, 235, 0.1)' },
  statIconTertiaryWrap: { backgroundColor: 'rgba(0, 124, 85, 0.1)' },
  statIcon: { fontSize: 20 },
  statValue: { ...TYPOGRAPHY.headlineLg, color: COLOR.onSurface, fontFamily: FONTS.headline },
  statLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, textTransform: 'uppercase', marginTop: 2, fontFamily: FONTS.label },
  grammarCard: {
    ...SHADOWS.elevation1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 20,
    padding: SPACING.stackMd,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: SPACING.stackMd,
  },
  grammarLeft: { flexDirection: 'row', alignItems: 'center' },
  grammarIconWrap: { width: 48, height: 48, borderRadius: 24, alignItems: 'center', justifyContent: 'center', marginRight: SPACING.stackMd },
  grammarIconSecondaryWrap: { backgroundColor: 'rgba(132, 85, 239, 0.1)' },
  grammarIcon: { fontSize: 22 },
  grammarLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, textTransform: 'uppercase', fontFamily: FONTS.label },
  grammarValue: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, marginTop: 2, fontFamily: FONTS.headline },
  chevron: { fontSize: 28, color: COLOR.outlineVariant },
  quickRow: { flexDirection: 'row', gap: SPACING.stackSm, marginTop: SPACING.stackMd },
  quickCard: { flex: 1, ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: 20, padding: SPACING.stackSm, alignItems: 'center' },
  quickGlyph: { fontSize: 22, marginBottom: 4 },
  quickLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  sectionHeader: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginTop: SPACING.stackLg, marginBottom: SPACING.stackSm },
  sectionTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline },
  activityCard: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, marginBottom: SPACING.stackSm },
  activityTitleRow: { flexDirection: 'row', alignItems: 'center', flexShrink: 1 },
  activityTipIcon: { fontSize: 18, marginRight: 6 },
  activityTitle: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurface, fontFamily: FONTS.label },
  activityDesc: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, marginTop: SPACING.stackSm, fontFamily: FONTS.body },
  activityFooter: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginTop: SPACING.stackSm },
  activityMeta: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, fontFamily: FONTS.label },
  startButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 6 },
  startButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
});
