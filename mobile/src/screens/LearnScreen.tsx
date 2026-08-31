import React, { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Pressable,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { LearnStackParamList } from '../components/MainTabs';
import type { LearningDashboard, MonthlyActivityPoint, User } from '@chorus/shared';

type LearnNav = NativeStackNavigationProp<LearnStackParamList, 'Learn'>;

export default function LearnScreen() {
  const navigation = useNavigation<LearnNav>();
  const [user, setUser] = useState<User | null>(null);
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es';
  const nativeLanguage = user?.nativeLanguage ?? 'en';
  const [d, setD] = useState<LearningDashboard | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    apiService
      .getLearningDashboard(targetLanguage, nativeLanguage)
      .then(setD)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [targetLanguage, nativeLanguage]);

  useEffect(() => {
    storage.getItem('user').then((s) => {
      if (s) {
        try {
          setUser(JSON.parse(s));
        } catch {
          // ignore
        }
      }
    });
  }, []);

  useEffect(() => load(), [load]);

  const startSession = (mode: string) =>
    navigation.navigate('LessonSession', { sessionId: undefined, mode });
  const openPlacement = () => navigation.navigate('Placement' as never);

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color={COLOR.primary} />
      </View>
    );
  }

  const pendingPlacement =
    d?.profile.placementStatus === 'not_started' || d?.profile.placementStatus === 'in_progress';
  const goalPct = d?.dailyGoal.percent ?? 0;
  const totalXp = Math.max(1, d?.weeklyActivity.reduce((s, w) => s + (w.xp || 0), 0) || 1);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
      {/* Header + streak chip */}
      <View style={styles.header}>
        <View style={{ flex: 1 }}>
          <Text style={styles.title}>Your Learning Path</Text>
          <Text style={styles.subtitle}>Stay consistent to reach fluency.</Text>
        </View>
        <View style={styles.streakChip}>
          <Text style={styles.streakFlame}>🔥</Text>
          <Text style={styles.streakText}>{d?.streak.days ?? 0} Days</Text>
        </View>
      </View>

      {pendingPlacement && d?.capability.supportTier === 'full_course' && (
        <View style={styles.placementCard}>
          <Text style={styles.cardTitle}>Find your starting level</Text>
          <Text style={styles.cardSubtitle}>Take a short placement test or start from the beginning.</Text>
          <View style={styles.placementActions}>
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

      {/* Quick Drills (full width) */}
      <View style={[styles.card, styles.quickCard]}>
        <View style={styles.decorCircle} />
        <View style={styles.quickTop}>
          <View style={styles.quickTitleRow}>
            <View style={styles.roundIconPrimary}>
              <Text style={styles.roundIconText}>⚡</Text>
            </View>
            <View>
              <Text style={styles.cardTitle}>Quick Drills</Text>
              <Text style={styles.cardSubtitle}>Focus on recent mistakes</Text>
            </View>
          </View>
          <Pressable style={styles.startPill} onPress={() => startSession('quick_drill')}>
            <Text style={styles.startPillText}>Start</Text>
          </Pressable>
        </View>
        <View>
          <View style={styles.progressLabels}>
            <Text style={styles.progressLabel}>Daily Goal</Text>
            <Text style={styles.progressLabel}>
              {d?.dailyGoal.completedItems ?? 0}/{d?.dailyGoal.targetItems ?? 10} completed
            </Text>
          </View>
          <View style={styles.progressBarTrack}>
            <View style={[styles.progressBarFill, { width: `${goalPct}%` }]} />
          </View>
        </View>
      </View>

      {/* Vocabulary + Scenarios (two-up) */}
      <View style={styles.twoCol}>
        <Pressable style={[styles.card, styles.halfCard]} onPress={() => navigation.navigate('VocabularyReview' as never)}>
          <View style={styles.roundIconTertiary}>
            <Text style={styles.roundIconText}>📖</Text>
          </View>
          <Text style={styles.halfTitle}>Vocabulary</Text>
          <Text style={styles.halfSub}>Spaced repetition</Text>
          <View style={styles.miniProgressLabels}>
            <Text style={styles.miniLabel}>Review</Text>
            <Text style={styles.miniLabel}>{d?.vocabulary.dueToday ?? 0} words</Text>
          </View>
          <View style={styles.miniTrack}>
            <View style={[styles.miniFill, styles.miniFillTertiary, { width: `${Math.min(100, d?.vocabulary.dueToday ?? 0) * 4}%` }]} />
          </View>
        </Pressable>

        <Pressable style={[styles.card, styles.halfCard, styles.aiCard]} onPress={() => navigation.navigate('Scenarios' as never)}>
          <View style={styles.aiGlowOverlay} />
          <View style={styles.roundIconSecondary}>
            <Text style={styles.roundIconText}>💬</Text>
          </View>
          <Text style={styles.halfTitle}>Scenarios</Text>
          <Text style={styles.aiTag}>✦ AI Roleplay</Text>
          <View style={styles.miniProgressLabels}>
            <Text style={styles.miniLabel}>{d?.scenario.title || 'Scenario'}</Text>
            <Text style={styles.miniLabel}>0/1</Text>
          </View>
          <View style={styles.miniTrack}>
            <View style={[styles.miniFill, styles.miniFillSecondary, { width: `${d?.scenario.progressPct ?? 0}%` }]} />
          </View>
        </Pressable>
      </View>

      {/* Grammar Deep Dive (full width) */}
      <View style={[styles.card, styles.grammarCard]}>
        <View style={styles.grammarBody}>
          <View style={styles.roundIconPurpleBox}>
            <Text style={styles.roundIconText}>🧩</Text>
          </View>
          <View>
            <Text style={styles.cardTitle}>Grammar Deep Dive</Text>
            <Text style={styles.cardSubtitle}>{d?.grammar.weakestPointTitle || 'Keep practicing'}</Text>
          </View>
        </View>
        <View style={styles.ringWrap}>
          <Ring percent={d?.grammar.confidencePct ?? 0} />
        </View>
      </View>

      {/* Quick actions */}
      <View style={styles.quickRow}>
        {[
          { label: 'Drills', glyph: '⚡', onPress: () => startSession('quick_drill') },
          { label: 'Vocabulary', glyph: '📖', onPress: () => navigation.navigate('VocabularyReview' as never) },
          { label: 'Scenarios', glyph: '🎭', onPress: () => navigation.navigate('Scenarios' as never) },
          { label: 'Roadmap', glyph: '🗺️', onPress: () => navigation.navigate('LearningRoadmap' as never) },
        ].map((a) => (
          <Pressable key={a.label} style={styles.actionCard} onPress={a.onPress}>
            <Text style={styles.quickGlyph}>{a.glyph}</Text>
            <Text style={styles.quickLabel}>{a.label}</Text>
          </Pressable>
        ))}
      </View>

      {/* Weekly Goal chart */}
      <View style={[styles.card, styles.weeklyCard]}>
        <Text style={styles.weeklyTitle}>Weekly Goal</Text>
        <View style={styles.weeklyRow}>
          <View style={styles.barChart}>
            {(d?.weeklyActivity ?? []).map((w, i) => {
              const h = Math.round(8 + (w.xp / totalXp) * 56);
              const today = i === (d?.weeklyActivity?.length ?? 0) - 1;
              return (
                <View key={w.date} style={styles.barCol}>
                  <View style={[styles.bar, { height: h }, today ? styles.barToday : styles.barMuted]} />
                </View>
              );
            })}
          </View>
          <View style={styles.xpBlock}>
            <Text style={styles.xpValue}>
              {d?.weeklyActivity.reduce((s, w) => s + (w.xp || 0), 0) ?? 0}
              <Text style={styles.xpUnit}>xp</Text>
            </Text>
            {d?.streak.days ? <Text style={styles.onTrack}>On track!</Text> : null}
          </View>
        </View>
      </View>

      {/* Monthly activity (FR-31): words learned / sentences understood per month */}
      <MonthlyActivityCard activity={d?.monthlyActivity ?? []} />
    </ScrollView>
  );
}

function Ring({ percent }: { percent: number }) {
  const size = 48;
  const stroke = 4;
  const fill = Math.min(100, Math.max(0, percent));
  return (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        borderWidth: stroke,
        borderColor: 'rgba(132,85,239,0.2)',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
      <Text style={styles.ringLabel}>{fill}%</Text>
    </View>
  );
}

const MONTH_LABELS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

function MonthlyActivityCard({ activity }: { activity: MonthlyActivityPoint[] }) {
  const [idx, setIdx] = useState(() => Math.max(0, activity.length - 1));
  const clamped = Math.min(idx, Math.max(0, activity.length - 1));
  const point = activity[clamped];

  const prev = () =>
    setIdx((i) => Math.max(0, Math.min(i, activity.length - 1) - 1));
  const next = () =>
    setIdx((i) => Math.min(activity.length - 1, Math.min(i, activity.length - 1) + 1));

  return (
    <View testID="learn-monthly" style={[styles.card, styles.monthlyCard]}>
      <View style={styles.monthlyHeader}>
        <View style={styles.roundIconPurpleBox}>
          <Text style={styles.roundIconText}>📈</Text>
        </View>
        <View style={styles.monthControls}>
          <Pressable
            onPress={prev}
            disabled={clamped <= 0}
            style={[styles.monthBtn, clamped <= 0 && styles.monthBtnDisabled]}>
            <Text style={styles.monthBtnText}>‹</Text>
          </Pressable>
          <Pressable
            onPress={next}
            disabled={clamped >= activity.length - 1}
            style={[styles.monthBtn, clamped >= activity.length - 1 && styles.monthBtnDisabled]}>
            <Text style={styles.monthBtnText}>›</Text>
          </Pressable>
        </View>
      </View>
      <Text style={styles.monthlyTitle}>Monthly Activity</Text>
      {point ? (
        <Text style={styles.monthlyMonth}>{formatMonth(point.month, MONTH_LABELS)}</Text>
      ) : (
        <Text style={styles.monthlyMonth}>No activity yet</Text>
      )}
      {point ? (
        <View style={styles.monthlyStatsRow}>
          <View style={[styles.monthlyStat, styles.monthlyStatSecondary]}>
            <Text style={styles.monthlyStatValue}>{point.wordsLearned}</Text>
            <Text style={styles.monthlyStatLabel}>Words learned</Text>
          </View>
          <View style={[styles.monthlyStat, styles.monthlyStatTertiary]}>
            <Text style={styles.monthlyStatValue}>{point.sentencesUnderstood}</Text>
            <Text style={styles.monthlyStatLabel}>Sentences understood</Text>
          </View>
        </View>
      ) : (
        <Text style={styles.monthlyEmpty}>No learning activity recorded for this month yet.</Text>
      )}
    </View>
  );
}

function formatMonth(month: string, labels: string[]): string {
  const [y, m] = month.split('-').map(Number);
  if (!y || !m || m < 1 || m > 12) return month;
  return `${labels[m - 1]} ${y}`;
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 56, gap: SPACING.stackMd },
  center: { flex: 1, backgroundColor: COLOR.background, alignItems: 'center', justifyContent: 'center' },
  header: { flexDirection: 'row', alignItems: 'flex-end', justifyContent: 'space-between', gap: SPACING.stackSm },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, marginTop: 2, fontFamily: FONTS.body },
  streakChip: { flexDirection: 'row', alignItems: 'center', gap: 4, backgroundColor: '#FFF7ED', borderRadius: RADIUS.full, paddingHorizontal: 10, paddingVertical: 4, shadowColor: '#EA580C', shadowOpacity: 0.2, shadowRadius: 6, elevation: 2 },
  streakFlame: { fontSize: 14 },
  streakText: { ...TYPOGRAPHY.labelMd, color: '#EA580C', fontFamily: FONTS.label },

  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, overflow: 'hidden' },
  cardTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 20 },
  cardSubtitle: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, marginTop: 2, fontFamily: FONTS.body },
  placementCard: { gap: SPACING.stackSm },
  placementActions: { flexDirection: 'row', gap: SPACING.stackSm, marginTop: SPACING.stackMd },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: RADIUS.full, paddingHorizontal: 16, paddingVertical: 10 },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  softButton: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: RADIUS.full, paddingHorizontal: 16, paddingVertical: 10 },
  softButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },

  quickCard: { gap: SPACING.stackMd },
  decorCircle: { position: 'absolute', right: -16, top: -16, width: 96, height: 96, borderRadius: 48, backgroundColor: COLOR.primaryFixed, opacity: 0.5 },
  quickTop: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', position: 'relative', zIndex: 1 },
  quickTitleRow: { flexDirection: 'row', alignItems: 'center', gap: SPACING.stackSm },
  roundIconPrimary: { width: 40, height: 40, borderRadius: 20, backgroundColor: COLOR.primary, alignItems: 'center', justifyContent: 'center' },
  roundIconText: { fontSize: 18 },
  startPill: { backgroundColor: COLOR.primary, borderRadius: RADIUS.full, paddingHorizontal: 16, paddingVertical: 8, shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.3, shadowRadius: 4, elevation: 3 },
  startPillText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  progressLabels: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 },
  progressLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  progressBarTrack: { height: 8, borderRadius: 4, backgroundColor: '#E2E8F0', overflow: 'hidden' },
  progressBarFill: { height: 8, borderRadius: 4, backgroundColor: COLOR.tertiaryFixedDim },

  twoCol: { flexDirection: 'row', gap: SPACING.stackMd },
  halfCard: { flex: 1, justifyContent: 'space-between', gap: SPACING.stackSm },
  roundIconTertiary: { width: 32, height: 32, borderRadius: 16, backgroundColor: COLOR.tertiaryContainer, alignItems: 'center', justifyContent: 'center' },
  roundIconSecondary: { width: 32, height: 32, borderRadius: 16, backgroundColor: COLOR.secondaryContainer, alignItems: 'center', justifyContent: 'center' },
  roundIconPurpleBox: { width: 40, height: 40, borderRadius: 12, backgroundColor: COLOR.surfaceVariant, alignItems: 'center', justifyContent: 'center' },
  halfTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 16 },
  halfSub: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  aiTag: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, fontFamily: FONTS.label },
  miniProgressLabels: { flexDirection: 'row', justifyContent: 'space-between', marginTop: 4 },
  miniLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label, fontSize: 10 },
  miniTrack: { height: 6, borderRadius: 3, backgroundColor: '#E2E8F0', overflow: 'hidden' },
  miniFill: { height: 6, borderRadius: 3 },
  miniFillTertiary: { backgroundColor: COLOR.tertiary },
  miniFillSecondary: { backgroundColor: COLOR.secondary },
  aiCard: { backgroundColor: 'rgba(233,221,255,0.5)', borderWidth: 1, borderColor: 'rgba(132,85,239,0.25)', shadowColor: COLOR.secondary, shadowOpacity: 0.15, shadowRadius: 12, elevation: 3 },
  aiGlowOverlay: { position: 'absolute', inset: 0, backgroundColor: 'rgba(255,255,255,0.35)' },

  grammarCard: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: COLOR.surfaceContainerLowest },
  grammarBody: { flexDirection: 'row', alignItems: 'center', gap: SPACING.stackMd, flex: 1 },
  ringWrap: { alignItems: 'center', justifyContent: 'center' },
  ringLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurface, fontFamily: FONTS.label },

  quickRow: { flexDirection: 'row', gap: SPACING.stackSm },
  actionCard: { flex: 1, alignItems: 'center', paddingVertical: SPACING.stackMd, ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: 20 },
  quickGlyph: { fontSize: 22, marginBottom: 4 },
  quickLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },

  weeklyCard: { gap: SPACING.stackMd },
  weeklyTitle: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, textTransform: 'uppercase', letterSpacing: 0.5, fontFamily: FONTS.label },
  weeklyRow: { flexDirection: 'row', alignItems: 'flex-end', justifyContent: 'space-between' },
  barChart: { flexDirection: 'row', alignItems: 'flex-end', gap: 4, height: 64 },
  barCol: { flex: 1, alignItems: 'center' },
  bar: { width: 22, borderRadius: 4 },
  barToday: { backgroundColor: COLOR.primary },
  barMuted: { backgroundColor: COLOR.surfaceVariant },
  xpBlock: { alignItems: 'flex-end' },
  xpValue: { ...TYPOGRAPHY.headlineMd, color: COLOR.onSurface, fontFamily: FONTS.headline },
  xpUnit: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  onTrack: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, marginTop: 2, fontFamily: FONTS.label },

  monthlyCard: { gap: SPACING.stackSm },
  monthlyHeader: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  monthControls: { flexDirection: 'row', gap: 8 },
  monthBtn: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: COLOR.surfaceContainerHigh,
    alignItems: 'center',
    justifyContent: 'center',
  },
  monthBtnDisabled: { opacity: 0.3 },
  monthBtnText: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.headline, lineHeight: 28 },
  monthlyTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 16 },
  monthlyMonth: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  monthlyStatsRow: { flexDirection: 'row', gap: SPACING.stackMd, marginTop: SPACING.stackSm },
  monthlyStat: { flex: 1, borderRadius: RADIUS.lg, padding: SPACING.stackMd },
  monthlyStatSecondary: { backgroundColor: 'rgba(255,247,237,0.8)' },
  monthlyStatTertiary: { backgroundColor: 'rgba(225,247,239,0.8)' },
  monthlyStatValue: { ...TYPOGRAPHY.headlineLg, color: COLOR.onSurface, fontFamily: FONTS.headline },
  monthlyStatLabel: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label, marginTop: 2 },
  monthlyEmpty: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, marginTop: 4 },
});
