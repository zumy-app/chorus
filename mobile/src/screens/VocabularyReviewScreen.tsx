import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, Pressable, ScrollView, ActivityIndicator } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { MinedItem, User } from '@chorus/shared';

export default function VocabularyReviewScreen({ navigation }: any) {
  const [user, setUser] = useState<User | null>(null);
  const [items, setItems] = useState<MinedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const targetLanguage = user?.targetLanguages?.[0] ?? 'es';

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
      .getMinedItems(targetLanguage, 'candidate')
      .then((items) => setItems(items ?? []))
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, [user, targetLanguage]);

  const accept = useCallback(async (id: string) => {
    await apiService.acceptMinedItem(id);
    setItems((cs) => cs.filter((c) => c.id !== id));
  }, []);
  const ignore = useCallback(async (id: string) => {
    await apiService.ignoreMinedItem(id);
    setItems((cs) => cs.filter((c) => c.id !== id));
  }, []);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <View>
          <Text style={styles.title}>Vocabulary</Text>
          <Text style={styles.subtitle}>Words found in your chats</Text>
        </View>
        <Pressable style={styles.practiceButton} onPress={() => navigation.navigate('LessonSession' as never, { mode: 'vocabulary' } as never)}>
          <Text style={styles.practiceButtonText}>Practice</Text>
        </Pressable>
      </View>

      {loading && <ActivityIndicator color={COLOR.primary} style={{ marginTop: 32 }} />}
      {!loading && items.length === 0 && (
        <View style={styles.empty}>
          <Text style={styles.emptyText}>No vocabulary yet. Chat in Spanish to discover words.</Text>
        </View>
      )}

      {items.map((item) => (
        <View key={item.id} style={styles.card}>
          <View style={styles.cardTop}>
            <Text style={styles.term}>{item.surfaceText}</Text>
            <Text style={styles.route}>{item.routeStatus}</Text>
          </View>
          {item.translation ? <Text style={styles.translation}>{item.translation}</Text> : null}
          {item.contextSentence ? <Text style={styles.context}>«{item.contextSentence}»</Text> : null}
          <View style={styles.actions}>
            <Pressable style={styles.saveButton} onPress={() => accept(item.id)}>
              <Text style={styles.saveButtonText}>Save</Text>
            </Pressable>
            <Pressable style={styles.dismissButton} onPress={() => ignore(item.id)}>
              <Text style={styles.dismissButtonText}>Dismiss</Text>
            </Pressable>
          </View>
        </View>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: SPACING.stackMd },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  practiceButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 10 },
  practiceButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  empty: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackLg },
  emptyText: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center' },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, marginBottom: SPACING.stackSm, gap: SPACING.stackSm },
  cardTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  term: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline },
  route: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, backgroundColor: 'rgba(0,74,198,0.08)', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 3, overflow: 'hidden', fontFamily: FONTS.label },
  translation: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  context: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, fontFamily: FONTS.body },
  actions: { flexDirection: 'row', gap: SPACING.stackSm, marginTop: 4 },
  saveButton: { backgroundColor: COLOR.primary, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 8 },
  saveButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  dismissButton: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: 999, paddingHorizontal: 16, paddingVertical: 8 },
  dismissButtonText: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
});
