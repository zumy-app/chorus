import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, Pressable, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { PlacementQuestion, PlacementResult, User } from '@chorus/shared';

export default function PlacementScreen({ navigation }: any) {
  const [user, setUser] = useState<User | null>(null);
  const [attemptId, setAttemptId] = useState<string | null>(null);
  const [question, setQuestion] = useState<PlacementQuestion | null>(null);
  const [total, setTotal] = useState(0);
  const [answered, setAnswered] = useState(0);
  const [selected, setSelected] = useState<string | null>(null);
  const [result, setResult] = useState<PlacementResult | null>(null);
  const [loading, setLoading] = useState(true);

  const targetLanguage = user?.targetLanguages?.[0] ?? 'es';
  const nativeLanguage = user?.nativeLanguage ?? 'en';

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

  useEffect(() => {
    if (!user) return;
    setLoading(true);
    apiService
      .startPlacement(targetLanguage, nativeLanguage)
      .then((res) => {
        setAttemptId(res.attemptId);
        setQuestion(res.question);
        setTotal(res.totalQuestions);
      })
      .catch(() => Alert.alert('Error', 'Could not start placement.'))
      .finally(() => setLoading(false));
  }, [user, targetLanguage, nativeLanguage]);

  const submit = useCallback(async () => {
    if (!attemptId || !selected) return;
    setLoading(true);
    try {
      const res = await apiService.answerPlacement(attemptId, selected);
      setSelected(null);
      if ('estimatedCefr' in res && (res as PlacementResult).estimatedCefr) {
        setResult(res as PlacementResult);
      } else {
        const r = res as any;
        setQuestion(r.question);
        setAnswered((a) => a + 1);
        setAttemptId(r.attemptId);
      }
    } catch {
      // progress
    } finally {
      setLoading(false);
    }
  }, [attemptId, selected]);

  const skip = useCallback(async () => {
    try {
      const res = await apiService.skipPlacement(targetLanguage, nativeLanguage);
      setResult(res);
    } catch {
      setResult({ attemptId: '', estimatedCefr: 'A1', readinessScore: 0, activeUnitId: '' });
    }
  }, [targetLanguage, nativeLanguage]);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {result ? (
        <View style={styles.center}>
          <Text style={styles.doneIcon}>🚩</Text>
          <Text style={styles.title}>You are ready to learn!</Text>
          <Text style={styles.subtitle}>
            Your starting level is <Text style={styles.accent}>{result.estimatedCefr}</Text>
            {result.readinessScore > 0 ? ` — readiness ${result.readinessScore}/1000` : ''}
          </Text>
          <Pressable style={styles.primaryButton} onPress={() => navigation.navigate('Learn' as never)}>
            <Text style={styles.primaryButtonText}>Start Learning</Text>
          </Pressable>
        </View>
      ) : question ? (
        <View>
          <View style={styles.topBar}>
            <Text style={styles.counter}>Question {answered + 1} of {total}</Text>
            <Pressable onPress={skip}>
              <Text style={styles.skip}>Skip test</Text>
            </Pressable>
          </View>
          <View style={styles.progressTrack}>
            <View style={[styles.progressFill, { width: `${(answered / total) * 100}%` }]} />
          </View>
          <View style={styles.card}>
            <Text style={styles.level}>{question.cefrLevel} · {question.itemType}</Text>
            <Text style={styles.prompt}>{typeof question.prompt === 'object' ? (question.prompt as any).text : question.prompt}</Text>
            {(question.choices || []).map((choice, i) => (
              <Pressable
                key={i}
                style={[styles.choice, selected === choice && styles.choiceSel]}
                onPress={() => setSelected(choice)}>
                <Text style={selected === choice ? styles.choiceTextSel : styles.choiceText}>{choice}</Text>
              </Pressable>
            ))}
            <Pressable style={[styles.primaryButton, !selected && styles.disabled]} disabled={!selected} onPress={submit}>
              <Text style={styles.primaryButtonText}>Check</Text>
            </Pressable>
          </View>
        </View>
      ) : (
        <ActivityIndicator color={COLOR.primary} style={{ marginTop: 48 }} />
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  center: { alignItems: 'center', gap: SPACING.stackMd, marginTop: 40 },
  doneIcon: { fontSize: 48 },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onSurface, fontFamily: FONTS.headline, textAlign: 'center' },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center' },
  accent: { color: COLOR.primary, fontWeight: '700' },
  topBar: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: SPACING.stackSm },
  counter: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  skip: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, fontFamily: FONTS.label },
  progressTrack: { height: 8, borderRadius: 4, backgroundColor: COLOR.surfaceContainerHigh, overflow: 'hidden', marginBottom: SPACING.stackMd },
  progressFill: { height: 8, backgroundColor: COLOR.primary },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackLg, gap: SPACING.stackMd },
  level: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, fontFamily: FONTS.label },
  prompt: { ...TYPOGRAPHY.headlineMd, color: COLOR.onSurface, fontFamily: FONTS.headline },
  choice: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd },
  choiceSel: { backgroundColor: COLOR.primary },
  choiceText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  choiceTextSel: { ...TYPOGRAPHY.bodyMd, color: COLOR.onPrimary, fontFamily: FONTS.body },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 18, paddingVertical: 12, alignItems: 'center' },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  disabled: { opacity: 0.4 },
});
