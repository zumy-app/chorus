import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, Pressable, ScrollView, ActivityIndicator } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import storage from '../utils/storage';
import type { ScenarioScript, User } from '@chorus/shared';

const ES_SCENARIOS_FALLBACK: ScenarioScript[] = [
  {
    id: 'es-cafe',
    courseId: 'es-course',
    slug: 'pedir-cafe',
    title: 'Pedir café en una cafetería',
    domain: 'food_drink',
    cefrLevel: 'A1',
    canDoStatement: 'Pedir una bebida',
    aiRoleName: 'Sparky',
    aiRoleDescription: 'Friendly cafe barista. Speaks clear A1 Spanish.',
    openingLine: 'Hola. ¿Qué te gustaría pedir hoy?',
    maxTurns: 10,
    estimatedMinutes: 5,
    phases: [
      { id: 'p1', scenarioId: 'es-cafe', ordinal: 1, title: 'Greeting', learnerGoal: 'Greet the barista.', requiredIntents: ['greet'], chunkBank: [{ text: 'Hola, buenos días.', translation: 'Hello, good morning.' }, { text: 'Buenas tardes.', translation: 'Good afternoon.' }] },
      { id: 'p2', scenarioId: 'es-cafe', ordinal: 2, title: 'Order', learnerGoal: 'Order one drink.', requiredIntents: ['order_drink'], chunkBank: [{ text: 'Quisiera un café con leche, por favor.', translation: 'I would like a coffee with milk, please.' }, { text: '¿Me puede dar un café, por favor?', translation: 'Can you give me a coffee, please?' }] },
      { id: 'p3', scenarioId: 'es-cafe', ordinal: 3, title: 'Customization', learnerGoal: 'Answer or request a simple option.', requiredIntents: ['customize'], chunkBank: [{ text: 'Para llevar, por favor.', translation: 'To go, please.' }, { text: 'Sin azúcar, por favor.', translation: 'Without sugar, please.' }] },
      { id: 'p4', scenarioId: 'es-cafe', ordinal: 4, title: 'Payment', learnerGoal: 'Ask or understand the price.', requiredIntents: ['pay'], chunkBank: [{ text: '¿Cuánto cuesta?', translation: 'How much does it cost?' }, { text: '¿Aceptan tarjeta?', translation: 'Do you accept card?' }] },
      { id: 'p5', scenarioId: 'es-cafe', ordinal: 5, title: 'Closing', learnerGoal: 'Close politely.', requiredIntents: ['close'], chunkBank: [{ text: 'Gracias.', translation: 'Thank you.' }, { text: 'Que tenga un buen día.', translation: 'Have a good day.' }] },
    ],
    metadata: { openingTranslation: 'Hello. What would you like to order today?' },
  } as any,
  {
    id: 'es-mercado',
    courseId: 'es-course',
    slug: 'mercado',
    title: 'Comprar en el mercado',
    domain: 'shopping',
    cefrLevel: 'A2',
    canDoStatement: 'Comprar frutas',
    aiRoleName: 'Sparky',
    aiRoleDescription: 'Friendly market vendor.',
    openingLine: 'Hola, ¿qué buscas?',
    maxTurns: 10,
    estimatedMinutes: 6,
    phases: [
      { id: 'm1', scenarioId: 'es-mercado', ordinal: 1, title: 'Greeting', learnerGoal: 'Greet the vendor.', requiredIntents: ['greet'], chunkBank: [{ text: 'Hola, buenos días.', translation: 'Hello, good morning.' }, { text: 'Buenas tardes.', translation: 'Good afternoon.' }] },
      { id: 'm2', scenarioId: 'es-mercado', ordinal: 2, title: 'Request', learnerGoal: 'Ask for fruit.', requiredIntents: ['order_drink'], chunkBank: [{ text: 'Quisiera un kilo de manzanas, por favor.', translation: 'I would like a kilo of apples, please.' }, { text: '¿Cuánto cuesta el kilo?', translation: 'How much is it per kilo?' }] },
      { id: 'm3', scenarioId: 'es-mercado', ordinal: 3, title: 'Payment', learnerGoal: 'Ask price and pay.', requiredIntents: ['pay'], chunkBank: [{ text: '¿Cuánto cuesta?', translation: 'How much does it cost?' }, { text: '¿Aceptan tarjeta?', translation: 'Do you accept card?' }] },
    ],
    metadata: { openingTranslation: 'Hi, what are you looking for?' },
  } as any,
];

const OPENING_TRANSLATIONS: Record<string, string> = {
  'Hola. ¿Qué te gustaría pedir hoy?': 'Hello. What would you like to order today?',
  'Hola, ¿qué buscas?': 'Hi, what are you looking for?',
};

function getOpeningTranslation(sc: any): string {
  if (sc.metadata?.openingTranslation) return sc.metadata.openingTranslation;
  if (sc.openingTranslation) return sc.openingTranslation;
  if (OPENING_TRANSLATIONS[sc.openingLine]) return OPENING_TRANSLATIONS[sc.openingLine];
  return sc.translation || '';
}

function getChunks(sc: any): { text: string; translation: string }[] {
  if (sc.phases?.[0]?.chunkBank?.length) return sc.phases[0].chunkBank;
  if (sc.phases?.length) {
    for (const p of sc.phases) if (p.chunkBank?.length) return p.chunkBank;
  }
  if (sc.chunkBank?.length) return sc.chunkBank;
  if (sc.suggestedChunks?.length) return sc.suggestedChunks;
  if (sc.metadata?.chunks?.length) return sc.metadata.chunks;
  for (const f of ES_SCENARIOS_FALLBACK) if (f.id === sc.id || f.slug === sc.slug) return (f as any).phases[0].chunkBank;
  return [];
}

function mergeSpanishScenarios(fetched: ScenarioScript[], targetLanguage: string): ScenarioScript[] {
  if (targetLanguage !== 'es') return fetched;
  const byId = new Map(fetched.map((s) => [s.id, s]));
  const bySlug = new Map(fetched.map((s) => [s.slug, s]));
  const out = [...fetched];
  for (const fb of ES_SCENARIOS_FALLBACK) {
    if (!byId.has(fb.id) && !bySlug.has(fb.slug)) out.push(fb);
  }
  return out.map((sc: any) => {
    const fb = ES_SCENARIOS_FALLBACK.find((f) => f.id === sc.id || f.slug === sc.slug);
    if (!fb) return sc;
    if (!sc.openingLine) sc.openingLine = fb.openingLine;
    if (!sc.phases || sc.phases.length === 0) sc.phases = (fb as any).phases;
    if (!sc.metadata) sc.metadata = {};
    if (!sc.metadata.openingTranslation) sc.metadata.openingTranslation = (fb as any).metadata.openingTranslation;
    return sc;
  });
}

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
        } catch {}
      }
    });
  }, []);

  useEffect(() => {
    if (!user) return;
    setLoading(true);
    apiService
      .getScenarios(targetLanguage, nativeLanguage)
      .then((data) => setScenarios(mergeSpanishScenarios(data as any, targetLanguage)))
      .catch(() => setScenarios(mergeSpanishScenarios([], targetLanguage)))
      .finally(() => setLoading(false));
  }, [user, targetLanguage, nativeLanguage]);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Real-World Scenarios</Text>
      <Text style={styles.subtitle}>Practice real conversations with AI</Text>
      {loading && <ActivityIndicator color={COLOR.primary} style={{ marginTop: 32 }} />}
      {!loading && scenarios.length === 0 && <Text style={styles.empty}>No scenarios available yet.</Text>}
      {scenarios.map((sc) => {
        const done = (sc as any).metadata?.completed === true;
        const openingLine = (sc as any).openingLine || '';
        const openingTranslation = getOpeningTranslation(sc as any);
        const chunks = getChunks(sc as any);
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
            {openingLine ? <Text style={styles.openingLine}>{openingLine}</Text> : null}
            {openingTranslation ? <Text style={styles.openingTranslation}>{openingTranslation}</Text> : null}
            {chunks.length > 0 ? (
              <View style={styles.chunkRow}>
                {chunks.map((c: any, i: number) => (
                  <View key={i} style={styles.chunk}>
                    <Text style={styles.chunkText}>{c.text}</Text>
                    {c.translation ? <Text style={styles.chunkTranslation}>{c.translation}</Text> : null}
                  </View>
                ))}
              </View>
            ) : null}
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
  openingLine: { ...TYPOGRAPHY.bodyMd, color: COLOR.onSurface, fontFamily: FONTS.body, marginTop: 4 },
  openingTranslation: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, fontStyle: 'italic' },
  chunkRow: { flexDirection: 'row', flexWrap: 'wrap', gap: SPACING.stackSm, marginTop: 6 },
  chunk: { backgroundColor: COLOR.surfaceContainer, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  chunkText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurface, fontFamily: FONTS.body },
  chunkTranslation: { ...TYPOGRAPHY.labelSm, color: COLOR.outline, fontFamily: FONTS.body, fontSize: 10 },
});
