import React, { useCallback, useEffect, useRef, useState } from 'react';
import { View, Text, StyleSheet, TextInput, Pressable, ScrollView, ActivityIndicator, KeyboardAvoidingView, Platform, Alert } from 'react-native';
import { useRoute } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { ScenarioChunk, ScenarioRun, ScenarioTurn, User } from '@chorus/shared';

export default function ScenarioRoleplayScreen({ navigation }: any) {
  const route = useRoute() as any;
  const scenarioId: string = route.params?.scenarioId;
  const [user, setUser] = useState<User | null>(null);
  const [run, setRun] = useState<ScenarioRun | null>(null);
  const [turns, setTurns] = useState<ScenarioTurn[]>([]);
  const [message, setMessage] = useState('');
  const [chunks, setChunks] = useState<ScenarioChunk[]>([]);
  const [sending, setSending] = useState(false);
  const [done, setDone] = useState(false);
  const [summary, setSummary] = useState<any>(null);
  const scrollRef = useRef<ScrollView>(null);

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
    if (!user || !scenarioId) return;
    apiService
      .startScenario(scenarioId, targetLanguage, nativeLanguage)
      .then((res) => {
        setRun(res.run);
        setChunks(res.aiResponse.suggestedChunks || []);
        setTurns([{ id: 'ai-0', runId: res.run.id, ordinal: 0, speaker: 'ai', text: res.aiResponse.aiMessage, translation: res.aiResponse.translation, phaseOrdinal: 1, createdAt: new Date().toISOString() }]);
      })
      .catch(() => {});
  }, [user, scenarioId, targetLanguage, nativeLanguage]);

  const send = useCallback(async () => {
    const text = message.trim();
    if (!text || !run || sending) return;
    setMessage('');

    // Optimistic update: show the user's bubble immediately so there is instant
    // feedback that the message was sent, before the AI reply round-trips.
    const userTurn: ScenarioTurn = {
      id: `u-${Date.now()}`,
      runId: run.id,
      ordinal: turns.length,
      speaker: 'user',
      text,
      translation: '',
      phaseOrdinal: run.currentPhaseOrdinal,
      createdAt: new Date().toISOString(),
    };
    setTurns((ts) => [...ts, userTurn]);
    setSending(true);

    try {
      const reply = await apiService.sendScenarioMessage(run.id, text);
      setTurns((ts) => [
        ...ts,
        {
          id: `ai-${Date.now()}`,
          runId: run.id,
          ordinal: ts.length,
          speaker: 'ai',
          text: reply.aiMessage,
          translation: reply.translation,
          phaseOrdinal: reply.nextPhaseOrdinal || run.currentPhaseOrdinal,
          createdAt: new Date().toISOString(),
        },
      ]);
      setChunks(reply.suggestedChunks || []);
      if (reply.runCompleted) {
        setDone(true);
        setSummary(reply.summary);
      }
    } catch {
      Alert.alert('Error', 'Could not send your message. Check the server and try again.');
      // Restore the message so nothing is lost on a failed send.
      setMessage(text);
    } finally {
      setSending(false);
    }
  }, [message, run, sending, turns.length]);

  return (
    <KeyboardAvoidingView style={styles.container} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      {done ? (
        <View style={styles.center}>
          <Text style={styles.doneIcon}>🎉</Text>
          <Text style={styles.title}>Scenario complete!</Text>
          <Text style={styles.subtitle}>{summary ? `Score ${summary.score} · +${summary.xpAwarded} XP` : 'Great conversation!'}</Text>
          <Pressable style={styles.primaryButton} onPress={() => navigation.navigate('Learn' as never)}>
            <Text style={styles.primaryButtonText}>Back to Learn</Text>
          </Pressable>
        </View>
      ) : (
        <>
          <View style={styles.topBar}>
            <Pressable onPress={() => navigation.goBack()}>
              <Text style={styles.back}>← Scenarios</Text>
            </Pressable>
            {run ? <Text style={styles.phase}>Phase {run.currentPhaseOrdinal}</Text> : null}
          </View>

          <ScrollView
            ref={scrollRef}
            style={styles.turns}
            contentContainerStyle={styles.turnsContent}
            onContentSizeChange={() => scrollRef.current?.scrollToEnd({ animated: true })}>
            {turns.map((t) => (
              <View
                key={t.id}
                style={[styles.bubble, t.speaker === 'ai' ? styles.bubbleAi : styles.bubbleUser]}>
                <Text style={t.speaker === 'ai' ? styles.bubbleAiText : styles.bubbleUserText}>{t.text}</Text>
                {t.speaker === 'ai' && t.translation ? <Text style={styles.translation}>{t.translation}</Text> : null}
              </View>
            ))}
            {sending ? (
              <View style={[styles.bubble, styles.bubbleAi, styles.typingBubble]}>
                <Text style={styles.typingText}>Sparky is typing…</Text>
              </View>
            ) : null}
          </ScrollView>

          {chunks.length > 0 ? (
            <View style={styles.hintRow}>
              {chunks.map((c, i) => (
                <Pressable key={i} style={styles.chunk} onPress={() => setMessage(c.text)}>
                  <Text style={styles.chunkText}>{c.text}</Text>
                </Pressable>
              ))}
            </View>
          ) : null}

          <View style={styles.inputRow}>
            <Pressable style={styles.hintButton} onPress={() => run && apiService.requestScenarioHint(run.id).then(setChunks)}>
              <Text style={styles.hintButtonText}>💡</Text>
            </Pressable>
            <TextInput
              value={message}
              onChangeText={setMessage}
              onSubmitEditing={send}
              placeholder="Escribe en español..."
              placeholderTextColor={COLOR.outline}
              style={styles.input}
            />
            <Pressable style={styles.sendButton} onPress={send} disabled={sending || !message.trim()}>
              <Text style={styles.sendButtonText}>➤</Text>
            </Pressable>
          </View>
        </>
      )}
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background, padding: SPACING.marginMobile },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: SPACING.stackMd },
  doneIcon: { fontSize: 52 },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onSurface, fontFamily: FONTS.headline, textAlign: 'center' },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center' },
  primaryButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 24, paddingVertical: 12, marginTop: SPACING.stackMd },
  primaryButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  topBar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: SPACING.stackSm },
  back: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  phase: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  turns: { flex: 1 },
  turnsContent: { gap: SPACING.stackSm, paddingBottom: SPACING.stackSm },
  bubble: { maxWidth: '85%', paddingHorizontal: SPACING.stackMd, paddingVertical: 10, borderRadius: 16 },
  bubbleAi: { backgroundColor: COLOR.surfaceContainerLowest, alignSelf: 'flex-start', borderLeftWidth: 2, borderLeftColor: COLOR.secondaryContainer },
  bubbleUser: { backgroundColor: COLOR.primary, alignSelf: 'flex-end' },
  bubbleAiText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body },
  bubbleUserText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onPrimary, fontFamily: FONTS.body },
  translation: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, marginTop: 4, fontFamily: FONTS.body, fontStyle: 'italic' },
  typingBubble: { alignSelf: 'flex-start' },
  typingText: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, fontFamily: FONTS.body, fontStyle: 'italic' },
  hintRow: { flexDirection: 'row', flexWrap: 'wrap', gap: SPACING.stackSm, marginTop: SPACING.stackSm },
  chunk: { backgroundColor: COLOR.surfaceContainer, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  chunkText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurface, fontFamily: FONTS.body },
  inputRow: { flexDirection: 'row', gap: SPACING.stackSm, alignItems: 'center', marginTop: SPACING.stackSm },
  hintButton: { backgroundColor: COLOR.surfaceContainer, borderRadius: 999, padding: 10 },
  hintButtonText: { fontSize: 18 },
  input: { flex: 1, backgroundColor: COLOR.surfaceContainer, borderRadius: 999, paddingHorizontal: SPACING.stackMd, paddingVertical: 12, color: COLOR.onSurface, fontFamily: FONTS.body },
  sendButton: { backgroundColor: COLOR.primary, borderRadius: 999, padding: 12 },
  sendButtonText: { color: COLOR.onPrimary, fontSize: 18 },
});
