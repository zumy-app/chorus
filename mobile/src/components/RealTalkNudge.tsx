import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, Pressable } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { RealTalkPrompt, User } from '@chorus/shared';

type Props = { chatId?: string; onSendToInput: (text: string) => void };

export default function RealTalkNudge({ chatId, onSendToInput }: Props) {
  const [user, setUser] = useState<User | null>(null);
  const [prompts, setPrompts] = useState<RealTalkPrompt[]>([]);
  const [idx, setIdx] = useState(0);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    storage.getItem('user').then((s) => { if (s) try { setUser(JSON.parse(s)); } catch {} });
  }, []);

  const targetLanguage = user?.targetLanguages?.[0] ?? 'es';
  const nativeLanguage = user?.nativeLanguage ?? 'en';

  useEffect(() => {
    if (!user) return;
    apiService.getRealTalkPrompts(targetLanguage, nativeLanguage, chatId).then(setPrompts).catch(() => {});
  }, [user, targetLanguage, nativeLanguage, chatId]);

  const current = prompts[idx % Math.max(1, prompts.length)];
  const send = useCallback(async () => {
    if (!current) return;
    try { await apiService.markRealTalkUsed(current.id); } catch {}
    onSendToInput(current.text);
    setDismissed(true);
  }, [current, onSendToInput]);

  if (dismissed || !current) return null;

  return (
    <View style={styles.container}>
      <Pressable onPress={() => setDismissed(true)} style={styles.close}><Text style={styles.closeText}>✕</Text></Pressable>
      <View style={styles.row}>
        <View style={styles.icon}><Text style={styles.iconText}>✨</Text></View>
        <View style={styles.body}>
          <Text style={styles.title}>Sparky’s Nudge</Text>
          <Text style={styles.subtitle}>Try this in the chat:</Text>
          <View style={styles.promptCard}><Text style={styles.prompt}>“{current.text}”</Text></View>
          <View style={styles.actions}>
            <Pressable onPress={send} style={styles.primaryButton}><Text style={styles.primaryText}>Send to Input</Text></Pressable>
            <Pressable onPress={() => setIdx((i) => (i + 1) % Math.max(1, prompts.length))} style={styles.shuffle}><Text style={styles.shuffleText}>↻</Text></Pressable>
          </View>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { backgroundColor: COLOR.primaryContainer, borderRadius: RADIUS.xl, padding: SPACING.stackMd, marginHorizontal: SPACING.marginMobile, marginBottom: 8, ...SHADOWS.elevation1, position: 'relative' },
  close: { position: 'absolute', top: 8, right: 8, padding: 6, zIndex: 1 },
  closeText: { color: COLOR.onPrimaryContainer, opacity: 0.6, fontSize: 16 },
  row: { flexDirection: 'row', gap: SPACING.stackSm },
  icon: { width: 36, height: 36, borderRadius: 18, backgroundColor: 'rgba(255,255,255,0.15)', alignItems: 'center', justifyContent: 'center' },
  iconText: { fontSize: 18 },
  body: { flex: 1, gap: 4 },
  title: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimaryContainer, fontFamily: FONTS.label },
  subtitle: { ...TYPOGRAPHY.bodySm, color: 'rgba(255,255,255,0.85)', fontFamily: FONTS.body },
  promptCard: { backgroundColor: 'rgba(255,255,255,0.12)', borderRadius: RADIUS.lg, padding: SPACING.stackSm, borderWidth: 1, borderColor: 'rgba(255,255,255,0.15)', marginVertical: 4 },
  prompt: { ...TYPOGRAPHY.headlineSm, color: COLOR.onPrimaryContainer, fontFamily: FONTS.headline, fontSize: 16 },
  actions: { flexDirection: 'row', gap: SPACING.stackSm, marginTop: 4 },
  primaryButton: { flex: 1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: 999, paddingVertical: 10, alignItems: 'center' },
  primaryText: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  shuffle: { width: 44, height: 44, borderRadius: 22, backgroundColor: 'rgba(255,255,255,0.12)', alignItems: 'center', justifyContent: 'center' },
  shuffleText: { color: COLOR.onPrimaryContainer, fontSize: 18 },
});
