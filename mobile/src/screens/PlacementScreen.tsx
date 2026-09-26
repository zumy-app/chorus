import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { View, Text, StyleSheet, Pressable, ScrollView, ActivityIndicator, Alert, TextInput } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { PlacementQuestion, PlacementResult, User } from '@chorus/shared';

type PromptPayload = {
  text?: string;
  sentence?: string;
  target_word?: string;
  context_translation?: string;
  sentence_with_blank?: string;
  verb?: string;
  translation?: string;
  instruction?: string;
  words?: string[];
  passage?: string;
  question?: string;
};

export default function PlacementScreen({ navigation }: any) {
  const [user, setUser] = useState<User | null>(null);
  const [attemptId, setAttemptId] = useState<string | null>(null);
  const [question, setQuestion] = useState<PlacementQuestion | null>(null);
  const [total, setTotal] = useState(0);
  const [answered, setAnswered] = useState(0);
  const [answer, setAnswer] = useState<string | null>(null); // MCQ choice or typed text
  const [typed, setTyped] = useState('');
  const [built, setBuilt] = useState<string[]>([]); // sentence reconstruction: tapped word order
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

  const prompt = useMemo<PromptPayload>(() => {
    if (!question) return {};
    if (typeof question.prompt === 'object' && question.prompt !== null) return question.prompt as PromptPayload;
    return { text: String(question.prompt ?? '') };
  }, [question]);

  const resetAnswer = useCallback(() => {
    setAnswer(null);
    setTyped('');
    setBuilt([]);
  }, []);

  const submit = useCallback(async () => {
    if (!attemptId || answer === null || answer === '') return;
    setLoading(true);
    try {
      const res = await apiService.answerPlacement(attemptId, answer);
      resetAnswer();
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
  }, [attemptId, answer, resetAnswer]);

  const skip = useCallback(async () => {
    try {
      const res = await apiService.skipPlacement(targetLanguage, nativeLanguage);
      setResult(res);
    } catch {
      setResult({ attemptId: '', estimatedCefr: 'A1', readinessScore: 0, activeUnitId: '' });
    }
  }, [targetLanguage, nativeLanguage]);

  // -- sentence reconstruction helpers ------------------------------------
  const tapWord = useCallback((w: string) => {
    setBuilt((b) => [...b, w]);
  }, []);
  const untapWord = useCallback((i: number) => {
    setBuilt((b) => b.filter((_, idx) => idx !== i));
  }, []);
  useEffect(() => {
    setAnswer(built.length ? built.join(' ') : null);
  }, [built]);
  useEffect(() => {
    if (question?.itemType === 'gap_fill_type') setAnswer(typed.trim() || null);
  }, [typed, question?.itemType]);

  const isReconstruction = question?.itemType === 'sentence_reconstruction';
  const isGapFill = question?.itemType === 'gap_fill_type';
  const isPassage = question?.itemType === 'passage_mcq';
  const hasChoices = (question?.choices?.length ?? 0) > 0;

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {result ? (
        <View style={styles.center}>
          <Text style={styles.doneIcon}>??</Text>
          <Text style={styles.title}>You are ready to learn!</Text>
          <Text style={styles.subtitle}>
            Your starting level is <Text style={styles.accent}>{result.estimatedCefr}</Text>
            {result.readinessScore > 0 ? ` - readiness ${result.readinessScore}/1000` : ''}
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
            <Text style={styles.level}>{question.cefrLevel} · {question.module ?? ''} · {question.itemType}</Text>

            {isPassage && prompt.passage ? (
              <View>
                <Text style={styles.passage}>{prompt.passage}</Text>
                <Text style={styles.prompt}>{prompt.question}</Text>
              </View>
            ) : isGapFill && prompt.sentence_with_blank ? (
              <View>
                <Text style={styles.prompt}>{prompt.sentence_with_blank}</Text>
                <TextInput
                  style={styles.input}
                  value={typed}
                  onChangeText={setTyped}
                  autoCapitalize="none"
                  autoCorrect={false}
                  autoFocus
                  placeholder="Type the missing word"
                  placeholderTextColor={COLOR.outline}
                  onSubmitEditing={submit}
                  returnKeyType="done"
                />
                {prompt.translation ? <Text style={styles.hint}>{prompt.translation}</Text> : null}
              </View>
            ) : isReconstruction ? (
              <View>
                <Text style={styles.prompt}>{prompt.instruction ?? 'Order the words to make a correct sentence.'}</Text>
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
                  {(prompt.words ?? []).map((w, i) => {
                    const usedCount = built.filter((b) => b === w).length;
                    const totalCount = (prompt.words ?? []).filter((x) => x === w).length;
                    const exhausted = usedCount >= totalCount;
                    return (
                      <Pressable
                        key={`${w}-${i}`}
                        style={[styles.chip, exhausted && styles.chipUsed]}
                        disabled={exhausted}
                        onPress={() => tapWord(w)}>
                        <Text style={exhausted ? styles.chipTextUsed : styles.chipText}>{w}</Text>
                      </Pressable>
                    );
                  })}
                </View>
              </View>
            ) : (
              <View>
                {prompt.sentence ? (
                  <Text style={styles.contextSentence}>
                    {prompt.sentence.split(prompt.target_word ?? '\u0000').map((part, i, arr) => (
                      <React.Fragment key={i}>
                        {part}
                        {i < arr.length - 1 ? <Text style={styles.targetWord}>{prompt.target_word}</Text> : null}
                      </React.Fragment>
                    ))}
                  </Text>
                ) : null}
                <Text style={styles.prompt}>
                  {prompt.context_translation ?? prompt.text ?? (typeof question.prompt === 'string' ? question.prompt : '')}
                </Text>
              </View>
            )}

            {hasChoices && !isGapFill && !isReconstruction
              ? (question.choices ?? []).map((choice, i) => (
                  <Pressable
                    key={i}
                    style={[styles.choice, answer === choice && styles.choiceSel]}
                    onPress={() => setAnswer(choice)}>
                    <Text style={answer === choice ? styles.choiceTextSel : styles.choiceText}>{choice}</Text>
                  </Pressable>
                ))
              : null}

            <Pressable
              style={[styles.primaryButton, (answer === null || answer === '') && styles.disabled]}
              disabled={answer === null || answer === '' || loading}
              onPress={submit}>
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
  contextSentence: { ...TYPOGRAPHY.bodyLg, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  targetWord: { color: COLOR.primary, fontWeight: '800' },
  passage: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body, lineHeight: 22, marginBottom: SPACING.stackSm },
  hint: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  input: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd, ...TYPOGRAPHY.bodyLg, color: COLOR.onSurface, fontFamily: FONTS.body },
  builtRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd, minHeight: 56 },
  builtPlaceholder: { ...TYPOGRAPHY.bodyMd, color: COLOR.outline, fontFamily: FONTS.body },
  chipBuilt: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  chipBuiltText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onPrimary, fontFamily: FONTS.body },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  chipUsed: { opacity: 0.3 },
  chipText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  chipTextUsed: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  choice: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.lg, padding: SPACING.stackMd },
  choiceSel: { backgroundColor: COLOR.primary },
  choiceText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  choiceTextSel: { ...TYPOGRAPHY.bodyMd, color: COLOR.onPrimary, fontFamily: FONTS.body },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 18, paddingVertical: 12, alignItems: 'center' },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  disabled: { opacity: 0.4 },
});
