import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { View, Text, StyleSheet, TextInput, Pressable, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { useRoute } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import webSocketService from '../services/websocket';
import type { SessionQuestion, User } from '@chorus/shared';

export default function LessonSessionScreen({ navigation }: any) {
  const route = useRoute() as any;
  const mode: string = route.params?.mode ?? 'daily';
  const [user, setUser] = useState<User | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [items, setItems] = useState<SessionQuestion[]>([]);
  const [index, setIndex] = useState(0);
  const [answer, setAnswer] = useState('');
  const [built, setBuilt] = useState<string[]>([]); // reconstruction: tapped word order
  const [feedback, setFeedback] = useState<{ correct: boolean; message: string } | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);
  const [xp, setXp] = useState(0);
  // Live WebSocket connection state (V3: transient session state mirrors over
  // WS; grading results arrive as "grade_result" pushes).
  const [wsConnected, setWsConnected] = useState(false);
  const [gradeNotice, setGradeNotice] = useState<string | null>(null);

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

  // WebSocket live state: connected indicator + async grade_result pushes.
  useEffect(() => {
    webSocketService.connect();
    const offMsg = (webSocketService as any).onMessage?.((msg: any) => {
      if (msg?.type === 'grade_result' && msg?.data?.feedback) {
        setGradeNotice(`AI feedback ready: ${msg.data.feedback}`);
      }
    });
    const offRc = (webSocketService as any).onReconnect?.(() => setWsConnected(true));
    return () => {
      offMsg?.();
      offRc?.();
    };
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
    setBuilt([]);
    if (index + 1 < items.length) {
      setIndex(index + 1);
      return;
    }
    if (sessionId) await apiService.completeSession(sessionId);
    setDone(true);
  }, [index, items.length, sessionId]);

  const current = items[index];

  // Grammar drills: reconstruction items build the sentence from word chips;
  // cloze items type the missing form; mcq items keep the choice buttons.
  const isReconstruction = current?.drillType === 'reconstruction';
  const builtAnswer = useMemo(() => (built.length ? built.join(' ') : ''), [built]);
  useEffect(() => {
    // Keep the typed answer in sync when the item changes.
    setAnswer('');
    setBuilt([]);
  }, [index]);

  const tapWord = useCallback((w: string) => setBuilt((b) => [...b, w]), []);
  const untapWord = useCallback((i: number) => setBuilt((b) => b.filter((_, idx) => idx !== i)), []);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
      <View style={styles.topBar}>
        <Pressable onPress={() => navigation.goBack()}>
          <Text style={styles.back}>← Back</Text>
        </Pressable>
        <Text style={styles.counter}>
          {items.length > 0 ? `${Math.min(index + 1, items.length)} / ${items.length}` : ''}
        </Text>
        <View style={[styles.wsDot, wsConnected ? styles.wsDotOn : styles.wsDotOff]} />
      </View>

      {gradeNotice ? (
        <Pressable style={styles.gradeNotice} onPress={() => setGradeNotice(null)}>
          <Text style={styles.gradeNoticeText}>{gradeNotice}</Text>
        </Pressable>
      ) : null}

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
            {current.drillType ? <Text style={styles.badgeSub}>{current.drillType}</Text> : null}
          </View>
          <Text style={styles.promptText}>{current.prompt.text}</Text>
          {current.prompt.source ? <Text style={styles.promptSource}>{current.prompt.source}</Text> : null}

          {isReconstruction ? (
            <View>
              <View style={styles.builtRow}>
                {built.length === 0 ? (
                  <Text style={styles.builtPlaceholder}>Tap the words in order</Text>
                ) : (
                  built.map((w, i) => (
                    <Pressable key={`${w}-${i}`} style={styles.chipBuilt} onPress={() => untapWord(i)}>
                      <Text style={styles.chipBuiltText}>{w}</Text>
                    </Pressable>
                  ))
                )}
              </View>
              <View style={styles.chipRow}>
                {(current.prompt.choices ?? []).map((w, i) => {
                  const used = built.filter((b) => b === w).length;
                  const total = (current.prompt.choices ?? []).filter((x) => x === w).length;
                  const exhausted = used >= total;
                  return (
                    <Pressable
                      key={`${w}-${i}`}
                      style={[styles.chip, exhausted && styles.chipUsed]}
                      disabled={exhausted || !!feedback || submitting}
                      onPress={() => tapWord(w)}>
                      <Text style={exhausted ? styles.chipTextUsed : styles.chipText}>{w}</Text>
                    </Pressable>
                  );
                })}
              </View>
              <Pressable
                style={[styles.sendButtonWide, (!builtAnswer || submitting) && styles.disabled]}
                disabled={!builtAnswer || submitting || !!feedback}
                onPress={() => submit(builtAnswer)}>
                <Text style={styles.sendButtonText}>Check</Text>
              </Pressable>
            </View>
          ) : current.prompt.choices && current.prompt.choices.length > 0 ? (
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
  wsDot: { width: 8, height: 8, borderRadius: 4 },
  wsDotOn: { backgroundColor: '#007C55' },
  wsDotOff: { backgroundColor: COLOR.outline },
  gradeNotice: { backgroundColor: 'rgba(0,74,198,0.10)', borderRadius: RADIUS.lg, padding: SPACING.stackMd, marginBottom: SPACING.stackSm },
  gradeNoticeText: { ...TYPOGRAPHY.bodySm, color: COLOR.primary, fontFamily: FONTS.body },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackLg, gap: SPACING.stackMd },
  badgeRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  badge: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, backgroundColor: 'rgba(208,188,255,0.3)', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 4, overflow: 'hidden', fontFamily: FONTS.label },
  badgeSub: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  promptText: { ...TYPOGRAPHY.headlineMd, color: COLOR.onSurface, fontFamily: FONTS.headline },
  promptSource: { ...TYPOGRAPHY.bodyLg, color: COLOR.onSurfaceVariant, marginTop: 2, fontFamily: FONTS.body },
  choices: { gap: SPACING.stackSm },
  choice: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd },
  choiceText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  inputRow: { flexDirection: 'row', gap: SPACING.stackSm, alignItems: 'center' },
  input: { flex: 1, backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, paddingHorizontal: SPACING.stackMd, paddingVertical: 12, color: COLOR.onSurface, fontFamily: FONTS.body },
  sendButton: { backgroundColor: COLOR.primary, borderRadius: RADIUS.lg, paddingHorizontal: 18, paddingVertical: 12 },
  sendButtonWide: { backgroundColor: COLOR.primary, borderRadius: RADIUS.lg, paddingHorizontal: 18, paddingVertical: 12, alignItems: 'center' },
  sendButtonText: { color: COLOR.onPrimary, fontSize: 18, fontFamily: FONTS.label },
  disabled: { opacity: 0.4 },
  builtRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd, minHeight: 56 },
  builtPlaceholder: { ...TYPOGRAPHY.bodyMd, color: COLOR.outline, fontFamily: FONTS.body },
  chipBuilt: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  chipBuiltText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onPrimary, fontFamily: FONTS.body },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  chipUsed: { opacity: 0.3 },
  chipText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  chipTextUsed: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
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
