import React, { useCallback, useEffect, useRef, useState } from 'react';
import { View, Text, StyleSheet, TextInput, Pressable, ScrollView, ActivityIndicator, RefreshControl } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import webSocketService from '../services/websocket';
import type { TeacherAssignment } from '@chorus/shared';

type GradingState =
  | { phase: 'idle' }
  | { phase: 'thinking'; jobId: string }
  | { phase: 'done'; score?: number; feedback: string }
  | { phase: 'failed'; message: string };

export default function AssignmentsScreen() {
  const [assignments, setAssignments] = useState<TeacherAssignment[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [draft, setDraft] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [grading, setGrading] = useState<GradingState>({ phase: 'idle' });
  const pollTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = useCallback(async () => {
    try {
      const list = await apiService.listStudentAssignments();
      setAssignments(list as TeacherAssignment[]);
    } catch {
      // keep the previous list on transient failures
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const applyGradePush = useCallback((data: any) => {
    if (pollTimer.current) {
      clearInterval(pollTimer.current);
      pollTimer.current = null;
    }
    if (data?.error) {
      setGrading({ phase: 'failed', message: 'Grading is taking longer than expected. Your work is saved.' });
      return;
    }
    setGrading({
      phase: 'done',
      score: data?.score,
      feedback: data?.feedback || 'AI feedback ready.',
    });
  }, []);

  // Async grading delivery: WebSocket push when connected, manual polling
  // fallback every 2s otherwise (plan V3 section D).
  useEffect(() => {
    webSocketService.connect();
    const off = (webSocketService as any).onMessage?.((msg: any) => {
      if (msg?.type === 'grade_result' && grading.phase === 'thinking' && msg?.data?.jobId === grading.jobId) {
        applyGradePush(msg.data);
      }
    });
    return () => {
      off?.();
      if (pollTimer.current) clearInterval(pollTimer.current);
    };
  }, [grading, applyGradePush]);

  const startPolling = useCallback((jobId: string) => {
    if (pollTimer.current) clearInterval(pollTimer.current);
    pollTimer.current = setInterval(async () => {
      try {
        const job = await apiService.getGradingJob(jobId);
        if (job?.status === 'done' && job?.result) {
          applyGradePush({ jobId, ...job.result });
        } else if (job?.status === 'failed') {
          applyGradePush({ jobId, error: job.lastError || 'failed' });
        }
      } catch {
        // transient poll failure: the next tick retries
      }
    }, 2000);
  }, [applyGradePush]);

  const submit = useCallback(async () => {
    const assignment = assignments.find((a) => a.id === selectedId);
    if (!assignment || !draft.trim() || submitting) return;
    setSubmitting(true);
    setGrading({ phase: 'idle' });
    try {
      const res = await apiService.submitAssignment(assignment.id, { text: draft.trim() });
      setGrading({ phase: 'thinking', jobId: res.gradingJobId });
      if (res.gradingJobId) startPolling(res.gradingJobId);
      load();
    } catch (e: any) {
      setGrading({ phase: 'failed', message: e?.message || 'Could not submit. Try again.' });
    } finally {
      setSubmitting(false);
    }
  }, [assignments, selectedId, draft, submitting, startPolling, load]);


  const statusColor = (status?: string) =>
    status === 'reviewed' ? '#007C55' : status === 'submitted' ? COLOR.primary : COLOR.onSurfaceVariant;

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => { setRefreshing(true); load(); }} />}
    >
      <Text style={styles.title}>Assignments</Text>
      <Text style={styles.subtitle}>Work your tutor assigned you</Text>

      {loading ? (
        <ActivityIndicator color={COLOR.primary} style={{ marginTop: 32 }} />
      ) : assignments.length === 0 ? (
        <Text style={styles.empty}>No assignments yet. Book a session with a tutor to get personalized work.</Text>
      ) : (
        assignments.map((a) => (
          <Pressable
            key={a.id}
            style={[styles.card, selectedId === a.id && styles.cardSel]}
            onPress={() => {
              setSelectedId(selectedId === a.id ? null : a.id);
              setGrading({ phase: 'idle' });
              setDraft('');
            }}>
            <View style={styles.cardTop}>
              <Text style={styles.cardTitle}>{a.title}</Text>
              <Text style={[styles.status, { color: statusColor(a.status) }]}>{a.status}</Text>
            </View>
            <Text style={styles.cardSub}>
              {a.type.replace('_', ' ')} · {a.targetLanguage.toUpperCase()}
              {a.dueDate ? ` · due ${new Date(a.dueDate).toLocaleDateString()}` : ''}
            </Text>
            {a.instructions ? <Text style={styles.cardInstructions} numberOfLines={selectedId === a.id ? 100 : 2}>{a.instructions}</Text> : null}

            {selectedId === a.id ? (
              a.status === 'pending' || a.status === 'in_progress' ? (
                <View style={styles.submitBox}>
                  <TextInput
                    style={styles.input}
                    value={draft}
                    onChangeText={setDraft}
                    placeholder="Write your answer here..."
                    placeholderTextColor={COLOR.outline}
                    multiline
                    editable={!submitting}
                  />
                  <Pressable
                    style={[styles.primaryButton, (!draft.trim() || submitting) && styles.disabled]}
                    disabled={!draft.trim() || submitting}
                    onPress={submit}>
                    <Text style={styles.primaryButtonText}>{submitting ? 'Submitting…' : 'Submit'}</Text>
                  </Pressable>
                </View>
              ) : (
                <View style={styles.feedbackBox}>
                  {a.submission?.score != null ? (
                    <Text style={styles.score}>Score: {a.submission.score}/10</Text>
                  ) : null}
                  {a.submission?.aiFeedback ? (
                    <Text style={styles.feedbackText}>
                      AI: {(a.submission.aiFeedback as any)?.feedback ?? JSON.stringify(a.submission.aiFeedback)}
                    </Text>
                  ) : null}
                  {a.submission?.teacherFeedback ? (
                    <Text style={styles.feedbackText}>
                      Tutor: {(a.submission.teacherFeedback as any)?.feedback ?? JSON.stringify(a.submission.teacherFeedback)}
                    </Text>
                  ) : null}
                </View>
              )
            ) : null}

            {selectedId === a.id && grading.phase !== 'idle' ? (
              <View style={styles.gradingBox}>
                {grading.phase === 'thinking' ? (
                  <View style={styles.thinkingRow}>
                    <ActivityIndicator size="small" color={COLOR.primary} />
                    <Text style={styles.thinkingText}>AI partner is thinking…</Text>
                  </View>
                ) : grading.phase === 'done' ? (
                  <View>
                    {grading.score != null ? <Text style={styles.score}>AI score: {grading.score}/10</Text> : null}
                    <Text style={styles.feedbackText}>{grading.feedback}</Text>
                  </View>
                ) : (
                  <Text style={styles.errorText}>{grading.message}</Text>
                )}
              </View>
            ) : null}
          </Pressable>
        ))
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onSurface, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, marginBottom: SPACING.stackLg },
  empty: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, marginTop: 24, textAlign: 'center' },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackLg, marginBottom: SPACING.stackMd, gap: SPACING.stackSm / 2 },
  cardSel: { borderWidth: 1, borderColor: COLOR.primary },
  cardTop: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  cardTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, flex: 1 },
  status: { ...TYPOGRAPHY.labelSm, fontFamily: FONTS.label, textTransform: 'capitalize' },
  cardSub: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  cardInstructions: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body, marginTop: SPACING.stackSm / 2 },
  submitBox: { marginTop: SPACING.stackMd, gap: SPACING.stackSm },
  input: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd, minHeight: 120, ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body, textAlignVertical: 'top' },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 24, paddingVertical: 12, alignItems: 'center' },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  disabled: { opacity: 0.4 },
  feedbackBox: { marginTop: SPACING.stackSm, gap: 4 },
  gradingBox: { marginTop: SPACING.stackSm, backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd },
  thinkingRow: { flexDirection: 'row', alignItems: 'center', gap: SPACING.stackSm },
  thinkingText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  score: { ...TYPOGRAPHY.headlineSm, color: COLOR.primary, fontFamily: FONTS.headline, marginBottom: 4 },
  feedbackText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  errorText: { ...TYPOGRAPHY.bodySm, color: '#BA1A1A', fontFamily: FONTS.body },
});
