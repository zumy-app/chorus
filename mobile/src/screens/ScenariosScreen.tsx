import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, Pressable, ScrollView, ActivityIndicator } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { ScenarioScript, User } from '@chorus/shared';

export default function ScenariosScreen({ navigation }: any) {
  const [user, setUser] = useState<User | null>(null);
  const [scenarios, setScenarios] = useState<ScenarioScript[]>([]);
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
      .getScenarios(targetLanguage, nativeLanguage)
      .then(setScenarios)
      .catch(() => setScenarios([]))
      .finally(() => setLoading(false));
  }, [user, targetLanguage, nativeLanguage]);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Real-World Scenarios</Text>
      <Text style={styles.subtitle}>Practice real conversations with AI</Text>
      {loading && <ActivityIndicator color={COLOR.primary} style={{ marginTop: 32 }} />}
      {!loading && scenarios.length === 0 && <Text style={styles.empty}>No scenarios available yet.</Text>}
      {scenarios.map((sc) => {
        const done = sc.metadata?.completed === true;
        return (
          <Pressable key={sc.id} style={styles.card} onPress={() => navigation.navigate('ScenarioRoleplay' as never, { scenarioId: sc.id } as never)}>
            <View style={styles.cardTop}>
              <Text style={styles.cardTitle}>{sc.title}</Text>
              {done ? <Text style={styles.done}>✓</Text> : null}
            </View>
            <Text style={styles.canDo}>{sc.canDoStatement}</Text>
            <View style={styles.metaRow}>
              <Text style={styles.level}>{sc.cefrLevel}</Text>
              <Text style={styles.meta}>{sc.estimatedMinutes} min · {sc.domain}</Text>
            </View>
          </Pressable>
        );
      })}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, marginBottom: SPACING.stackMd, fontFamily: FONTS.body },
  empty: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, marginTop: 24, fontFamily: FONTS.body },
  card: { ...SHADOWS.elevation1, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackLg, marginBottom: SPACING.stackSm, gap: SPACING.stackSm },
  cardTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  cardTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline },
  done: { color: COLOR.tertiary, fontSize: 20, fontWeight: '700' },
  canDo: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  metaRow: { flexDirection: 'row', gap: SPACING.stackSm, alignItems: 'center', marginTop: 2 },
  level: { ...TYPOGRAPHY.labelSm, color: COLOR.secondary, backgroundColor: 'rgba(208,188,255,0.3)', borderRadius: 999, paddingHorizontal: 8, paddingVertical: 2, overflow: 'hidden', fontFamily: FONTS.label },
  meta: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, fontFamily: FONTS.label },
});
