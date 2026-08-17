import React from 'react';
import { View, Text, StyleSheet, ScrollView } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

const GOAL_PCT = 75;

export default function LearnScreen() {
  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      showsVerticalScrollIndicator={false}>
      <View style={styles.header}>
        <Text style={styles.title}>Your Fluency Journey</Text>
        <Text style={styles.subtitle}>Track your progress and practice smart.</Text>
      </View>

      {/* Daily Goal */}
      <View style={styles.dailyGoalCard}>
        <View style={styles.dailyGoalBody}>
          <Text style={styles.cardTitle}>Daily Goal</Text>
          <Text style={styles.cardSubtitle}>You're almost there!</Text>
          <View style={styles.streakBadge}>
            <Text style={styles.streakIcon}>🔥</Text>
            <Text style={styles.streakText}>12 Day Streak</Text>
          </View>
        </View>
        <View style={styles.goalRight}>
          <Text style={styles.goalPct}>{GOAL_PCT}%</Text>
          <View style={styles.progressTrack}>
            <View style={[styles.progressFill, { width: `${GOAL_PCT}%` }]} />
          </View>
        </View>
      </View>

      {/* Stat cards */}
      <View style={styles.statRow}>
        <View style={styles.statCard}>
          <View style={[styles.statIconWrap, styles.statIconTertiaryWrap]}>
            <Text style={styles.statIconTertiary}>💬</Text>
          </View>
          <Text style={styles.statValue}>1.2k</Text>
          <Text style={styles.statLabel}>Messages Translated</Text>
        </View>
        <View style={styles.statCard}>
          <View style={[styles.statIconWrap, styles.statIconPrimaryWrap]}>
            <Text style={styles.statIconPrimary}>📖</Text>
          </View>
          <Text style={styles.statValue}>342</Text>
          <Text style={styles.statLabel}>New Words Learned</Text>
        </View>
      </View>

      {/* Grammar mastered */}
      <View style={styles.grammarCard}>
        <View style={styles.grammarLeft}>
          <View style={[styles.grammarIconWrap, styles.grammarIconSecondaryWrap]}>
            <Text style={styles.grammarIcon}>⚖️</Text>
          </View>
          <View>
            <Text style={styles.grammarLabel}>Grammar Mastered</Text>
            <Text style={styles.grammarValue}>48 Concepts</Text>
          </View>
        </View>
        <Text style={styles.chevron}>›</Text>
      </View>

      {/* Recommended Activities */}
      <View style={styles.sectionHeader}>
        <Text style={styles.sectionTitle}>Recommended Activities</Text>
        <Text style={styles.viewAll}>View all</Text>
      </View>

      <View style={[styles.activityCard, styles.activityHighlighted]}>
        <View style={styles.activityTopRow}>
          <View style={styles.activityTitleRow}>
            <Text style={styles.activityErrorIcon}>⚠️</Text>
            <Text style={styles.activityTitle}>Past Tense Verbs</Text>
          </View>
          <View style={styles.priorityBadge}>
            <Text style={styles.priorityBadgeText}>High Priority</Text>
          </View>
        </View>
        <Text style={styles.activityDesc}>
          You made mistakes with irregular past tense verbs in recent chats.
        </Text>
        <View style={styles.activityFooter}>
          <Text style={styles.activityMeta}>5 min exercise</Text>
          <View style={styles.practiceButtonSoft}>
            <Text style={styles.practiceButtonSoftText}>Practice</Text>
          </View>
        </View>
      </View>

      <View style={[styles.activityCard, styles.activityHighlighted]}>
        <View style={styles.activityTitleRow}>
          <Text style={styles.activityTipIcon}>💡</Text>
          <Text style={styles.activityTitle}>Vocabulary Review</Text>
        </View>
        <Text style={styles.activityDesc}>
          Review newly translated words from recent conversations.
        </Text>
        <View style={styles.activityFooter}>
          <Text style={styles.activityMeta}>3 min review</Text>
          <View style={styles.startButton}>
            <Text style={styles.startButtonText}>Start</Text>
          </View>
        </View>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  content: {
    padding: SPACING.marginMobile,
    paddingBottom: 48,
  },
  header: {
    marginBottom: SPACING.stackLg,
  },
  title: {
    ...TYPOGRAPHY.headlineLg,
    color: COLOR.onBackground,
    marginBottom: SPACING.unit,
    fontFamily: FONTS.headline,
  },
  subtitle: {
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurfaceVariant,
    fontFamily: FONTS.body,
  },
  dailyGoalCard: {
    ...SHADOWS.elevation1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 24,
    padding: SPACING.stackLg,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  dailyGoalBody: {
    flex: 1,
  },
  cardTitle: {
    ...TYPOGRAPHY.headlineSm,
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
  },
  cardSubtitle: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
    marginTop: 2,
    fontFamily: FONTS.body,
  },
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
  streakIcon: {
    fontSize: 16,
    marginRight: 6,
  },
  streakText: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onSurfaceVariant,
    fontFamily: FONTS.label,
  },
  goalRight: {
    alignItems: 'center',
    width: 88,
  },
  goalPct: {
    ...TYPOGRAPHY.headlineMd,
    color: COLOR.primary,
    fontFamily: FONTS.headline,
    marginBottom: 8,
  },
  progressTrack: {
    height: 10,
    borderRadius: 5,
    backgroundColor: COLOR.surfaceContainerHigh,
    width: 88,
    overflow: 'hidden',
  },
  progressFill: {
    height: 10,
    borderRadius: 5,
    backgroundColor: COLOR.primary,
  },
  statRow: {
    flexDirection: 'row',
    gap: SPACING.stackMd,
    marginTop: SPACING.stackMd,
  },
  statCard: {
    ...SHADOWS.elevation1,
    flex: 1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 20,
    padding: SPACING.stackMd,
  },
  statIconWrap: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: SPACING.stackSm,
  },
  statIconTertiaryWrap: {
    backgroundColor: 'rgba(0, 124, 85, 0.1)',
  },
  statIconPrimaryWrap: {
    backgroundColor: 'rgba(37, 99, 235, 0.1)',
  },
  statIconTertiary: {
    fontSize: 20,
  },
  statIconPrimary: {
    fontSize: 20,
  },
  statValue: {
    ...TYPOGRAPHY.headlineLg,
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
  },
  statLabel: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.onSurfaceVariant,
    textTransform: 'uppercase',
    marginTop: 2,
    fontFamily: FONTS.label,
  },
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
  grammarLeft: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  grammarIconWrap: {
    width: 48,
    height: 48,
    borderRadius: 24,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: SPACING.stackMd,
  },
  grammarIconSecondaryWrap: {
    backgroundColor: 'rgba(132, 85, 239, 0.1)',
  },
  grammarIcon: {
    fontSize: 22,
  },
  grammarLabel: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.onSurfaceVariant,
    textTransform: 'uppercase',
    fontFamily: FONTS.label,
  },
  grammarValue: {
    ...TYPOGRAPHY.headlineSm,
    color: COLOR.onSurface,
    marginTop: 2,
    fontFamily: FONTS.headline,
  },
  chevron: {
    fontSize: 28,
    color: COLOR.outlineVariant,
  },
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: SPACING.stackLg,
    marginBottom: SPACING.stackSm,
  },
  sectionTitle: {
    ...TYPOGRAPHY.headlineSm,
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
  },
  viewAll: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.primary,
    fontFamily: FONTS.label,
  },
  activityCard: {
    ...SHADOWS.elevation1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: RADIUS.xl,
    padding: SPACING.stackMd,
    marginBottom: SPACING.stackSm,
  },
  activityHighlighted: {
    borderLeftWidth: 4,
    borderLeftColor: COLOR.secondaryFixed,
  },
  activityTopRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
  },
  activityTitleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    flexShrink: 1,
  },
  activityErrorIcon: {
    fontSize: 18,
    marginRight: 6,
  },
  activityTipIcon: {
    fontSize: 18,
    marginRight: 6,
  },
  activityTitle: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onSurface,
    fontFamily: FONTS.label,
  },
  priorityBadge: {
    backgroundColor: COLOR.errorContainer,
    borderRadius: 999,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
  priorityBadgeText: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.onErrorContainer,
    fontFamily: FONTS.label,
  },
  activityDesc: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
    marginTop: SPACING.stackSm,
    fontFamily: FONTS.body,
  },
  activityFooter: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: SPACING.stackSm,
  },
  activityMeta: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.outline,
    fontFamily: FONTS.label,
  },
  practiceButtonSoft: {
    backgroundColor: 'rgba(0, 74, 198, 0.1)',
    borderRadius: 999,
    paddingHorizontal: 16,
    paddingVertical: 6,
  },
  practiceButtonSoftText: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.primary,
    fontFamily: FONTS.label,
  },
  startButton: {
    backgroundColor: COLOR.primary,
    borderRadius: 999,
    paddingHorizontal: 16,
    paddingVertical: 6,
  },
  startButtonText: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onPrimary,
    fontFamily: FONTS.label,
  },
});