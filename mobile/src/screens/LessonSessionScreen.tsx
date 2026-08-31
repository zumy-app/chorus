import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, TextInput, Pressable, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { useRoute } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { SessionQuestion, User } from '@chorus/shared';

export default function LessonSessionScreen({ navigation }: any) {
  const route = useRoute() as any;
  const mode: string = route.params?.mode ?? 'daily';
  const [user, setUser] = useState<User | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [items, setItems] = useState<SessionQuestion[]>([]);
  const [index, setIndex] = useState(0);
  const [answer, setAnswer] = useState('');
  const [feedback, setFeedback] = useState<{ correct: boolean; message: string } | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);
  const [xp, setXp] = useState(0);

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
    apiService
      .startSession({ targetLanguage, nativeLanguage, mode, source: 'learn' })
      .then((res) => {
        setSessionId(res.session.id);
        setItems(res.items);
        if (res.items.length === 0) setDone(true);
      })
      .catch(() => Alert.alert('Error', 'Could not start a session.'))
      .finally(() => {});
  }, [user, targetLanguage, nativeLanguage, mode]);

  const submit = useCallback(async (value: string) => {
    if (!sessionId || !items[index] || submitting) return;
    setFeedback(null);
    setSubmitting(true);
    const it = items[index];
    try {
      const res = await apiService.answerSessionItem(sessionId, it.id, { text: value, choice: value }, 800);
      setFeedback({ correct: res.correct, message: res.feedback.message });
      if (res.correct) setXp((x) => x + 10);
    } catch {
      Alert.alert('Error', 'Could not submit your answer. Try again.');
    } finally {
      setSubmitting(false);
    }
  }, [sessionId, items, index, submitting]);

  const next = useCallback(async () => {
    setFeedback(null);
    setAnswer('');
    if (index + 1 < items.length) {
      setIndex(index + 1);
      return;
    }
    if (sessionId) await apiService.completeSession(sessionId);
    setDone(true);
  }, [index, items.length, sessionId]);

  const current = items[index];

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
      <View style={styles.topBar}>
        <Pressable onPress={() => navigation.goBack()}>
          <Text style={styles.back}>← Back</Text>
        </Pressable>
        <Text style={styles.counter}>
          {items.length > 0 ? `${Math.min(index + 1, items.length)} / ${items.length}` : ''}
        </Text>
      </View>

      {done ? (
        <View style={styles.doneWrap}>
          <Text style={styles.doneIcon}>✅</Text>
          <Text style={styles.doneTitle}>Session complete!</Text>
          <Text style={styles.doneSub}>You earned {xp} XP.</Text>
          <Pressable style={styles.primaryButton} onPress={() => navigation.navigate('Learn' as never)}>
            <Text style={styles.primaryButtonText}>Back to Learn</Text>
          </Pressable>
        </View>
      ) : current ? (
        <View style={styles.card}>
          <View style={styles.badgeRow}>
            <Text style={styles.badge}>{current.activityType}</Text>
          </View>
          <Text style={styles.promptText}>{current.prompt.text}</Text>
          {current.prompt.source ? <Text style={styles.promptSource}>{current.prompt.source}</Text> : null}
          {current.prompt.choices && current.prompt.choices.length > 0 ? (
            <View style={styles.choices}>
              {current.prompt.choices.map((choice, i) => (
                <Pressable key={i} style={styles.choice} onPress={() => submit(choice)} disabled={!!feedback || submitting}>
                  <Text style={styles.choiceText}>{choice}</Text>
                </Pressable>
              ))}
            </View>
          ) : (
            <View style={styles.inputRow}>
              <TextInput
                value={answer}
                onChangeText={setAnswer}
                onSubmitEditing={() => answer.trim() && submit(answer.trim())}
                placeholder="Escribe aquí..."
                placeholderTextColor={COLOR.outline}
                style={styles.input}
                editable={!submitting}
              />
              <Pressable style={styles.sendButton} onPress={() => answer.trim() && submit(answer.trim())} disabled={submitting}>
                <Text style={styles.sendButtonText}>→</Text>
              </Pressable>
            </View>
          )}
          {submitting && (
            <View style={styles.submittingRow}>
              <ActivityIndicator size="small" color={COLOR.primary} />
              <Text style={styles.submittingText}>Checking…</Text>
            </View>
          )}
          {feedback && (
            <View style={[styles.feedback, feedback.correct ? styles.feedbackOk : styles.feedbackErr]}>
              <Text style={styles.feedbackText}>{feedback.message}</Text>
              <Pressable style={styles.continueButton} onPress={next}>
                <Text style={styles.continueButtonText}>Continue</Text>
              </Pressable>
            </View>
          )}
        </View>
      ) : (
        <ActivityIndicator color={COLOR.primary} style={{ marginTop: 40 }} />
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  topBar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: SPACING.stackMd },
  back: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  counter: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackLg, gap: SPACING.stackMd },
  badgeRow: { flexDirection: 'row' },
  badge: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, backgroundColor: 'rgba(208,188,255,0.3)', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 4, overflow: 'hidden', fontFamily: FONTS.label },
  promptText: { ...TYPOGRAPHY.headlineMd, color: COLOR.onSurface, fontFamily: FONTS.headline },
  promptSource: { ...TYPOGRAPHY.bodyLg, color: COLOR.onSurfaceVariant, marginTop: 2, fontFamily: FONTS.body },
  choices: { gap: SPACING.stackSm },
  choice: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd },
  choiceText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  inputRow: { flexDirection: 'row', gap: SPACING.stackSm, alignItems: 'center' },
  input: { flex: 1, backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, paddingHorizontal: SPACING.stackMd, paddingVertical: 12, color: COLOR.onSurface, fontFamily: FONTS.body },
  sendButton: { backgroundColor: COLOR.primary, borderRadius: RADIUS.lg, paddingHorizontal: 18, paddingVertical: 12 },
  sendButtonText: { color: COLOR.onPrimary, fontSize: 18, fontFamily: FONTS.label },
  submittingRow: { flexDirection: 'row', alignItems: 'center', gap: SPACING.stackSm },
  submittingText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  feedback: { borderRadius: RADIUS.lg, padding: SPACING.stackMd, gap: SPACING.stackSm },
  feedbackOk: { backgroundColor: 'rgba(0,124,85,0.12)' },
  feedbackErr: { backgroundColor: 'rgba(186,26,26,0.12)' },
  feedbackText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  continueButton: { backgroundColor: 'rgba(0,74,198,0.1)', borderRadius: 999, paddingHorizontal: 16, paddingVertical: 8, alignSelf: 'flex-start' },
  continueButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  doneWrap: { alignItems: 'center', gap: SPACING.stackMd, marginTop: 40 },
  doneIcon: { fontSize: 48 },
  doneTitle: { ...TYPOGRAPHY.headlineLg, color: COLOR.onSurface, fontFamily: FONTS.headline },
  doneSub: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 24, paddingVertical: 12, marginTop: SPACING.stackMd },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
});
