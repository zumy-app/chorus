import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, Pressable, ScrollView, ActivityIndicator } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { LearningPath, UnitProgressSummary, User } from '@chorus/shared';

export default function LearningRoadmapScreen({ navigation }: any) {
  const [user, setUser] = useState<User | null>(null);
  const [path, setPath] = useState<LearningPath | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
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
      .getLearningPath(targetLanguage, nativeLanguage)
      .then(setPath)
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, [user, targetLanguage, nativeLanguage]);

  const byLevel = (level: string) => path?.units?.filter((u) => u.cefrLevel === level) || [];

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Your Roadmap</Text>
      <Text style={styles.subtitle}>Progress through A1 to B2</Text>
      {loading && <ActivityIndicator color={COLOR.primary} style={{ marginTop: 32 }} />}
      {!loading && error && <Text style={styles.empty}>Could not load your roadmap.</Text>}
      {!loading && !error && path ? (
        ['A1', 'A2', 'B1', 'B2'].map((level) => {
          const units = byLevel(level);
          if (units.length === 0) return null;
          return (
            <View key={level} style={styles.section}>
              <View style={styles.sectionHeader}>
                <Text style={styles.levelBadge}>{level}</Text>
                <View style={styles.line} />
              </View>
              {units.map((u) => (
                <UnitRow key={u.id} unit={u} onOpen={() => navigation.navigate('Learn' as never)} />
              ))}
            </View>
          );
        })
      ) : null}
    </ScrollView>
  );
}

function UnitRow({ unit, onOpen }: { unit: UnitProgressSummary; onOpen: () => void }) {
  const completed = unit.status === 'completed';
  const available = unit.status === 'available' || unit.status === 'in_progress';
  const locked = unit.status === 'locked';
  return (
    <Pressable
      style={[styles.unit, locked && styles.unitLocked]}
      onPress={onOpen}
      disabled={locked}>
      <View style={[styles.unitIcon, completed ? styles.unitIconDone : available ? styles.unitIconActive : styles.unitIconLocked]}>
        <Text style={styles.unitIconText}>{completed ? '✓' : locked ? '🔒' : unit.ordinal}</Text>
      </View>
      <View style={{ flex: 1 }}>
        <View style={styles.unitTop}>
          <Text style={styles.unitTitle}>{unit.title}</Text>
          {unit.progressPct > 0 ? <Text style={styles.unitPct}>{unit.progressPct}%</Text> : null}
        </View>
        <Text style={styles.unitCanDo} numberOfLines={1}>{unit.canDoStatement}</Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 48 },
  title: { ...TYPOGRAPHY.headlineLg, color: COLOR.onBackground, fontFamily: FONTS.headline },
  subtitle: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, marginBottom: SPACING.stackMd, fontFamily: FONTS.body },
  empty: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurfaceVariant, marginTop: 24, fontFamily: FONTS.body },
  section: { marginBottom: SPACING.stackLg },
  sectionHeader: { flexDirection: 'row', alignItems: 'center', gap: SPACING.stackSm, marginBottom: SPACING.stackSm },
  levelBadge: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, backgroundColor: 'rgba(0,74,198,0.08)', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 3, overflow: 'hidden', fontFamily: FONTS.label },
  line: { flex: 1, height: 1, backgroundColor: COLOR.surfaceContainerHigh },
  unit: { ...SHADOWS.elevation1, flexDirection: 'row', alignItems: 'center', gap: SPACING.stackMd, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, marginBottom: SPACING.stackSm },
  unitLocked: { opacity: 0.5 },
  unitIcon: { width: 36, height: 36, borderRadius: 18, alignItems: 'center', justifyContent: 'center' },
  unitIconDone: { backgroundColor: 'rgba(0,124,85,0.15)' },
  unitIconActive: { backgroundColor: 'rgba(0,74,198,0.12)' },
  unitIconLocked: { backgroundColor: COLOR.surfaceContainerHigh },
  unitIconText: { color: COLOR.onSurface, fontWeight: '700', fontFamily: FONTS.label },
  unitTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  unitTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline },
  unitPct: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, fontFamily: FONTS.label },
  unitCanDo: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, marginTop: 2, fontFamily: FONTS.body },
});
