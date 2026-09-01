import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, Pressable, TextInput, ActivityIndicator, RefreshControl, Alert } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import type { MarketplaceStackParamList } from '../components/MainTabs';
import type { TutorProfile } from '@chorus/shared';

type Nav = NativeStackNavigationProp<MarketplaceStackParamList, 'BrowseTutors'>;

export default function BrowseTutorsScreen() {
  const navigation = useNavigation<Nav>();
  const [q, setQ] = useState('');
  const [tutors, setTutors] = useState<TutorProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiService.browseTutors({ search: q || undefined, limit: 20 });
      setTutors(res.tutors ?? []);
    } catch {
      // keep previous
    } finally {
      setLoading(false);
    }
  }, [q]);

  useEffect(() => { load(); }, [load]);

  const onRefresh = async () => {
    setRefreshing(true);
    try {
      const res = await apiService.browseTutors({ search: q || undefined, limit: 20 });
      setTutors(res.tutors ?? []);
    } catch {} finally { setRefreshing(false); }
  };

  const featured = tutors.slice(0, 2);
  const rest = tutors.slice(2);

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content} refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} />} showsVerticalScrollIndicator={false}>
      <View style={styles.searchRow}>
        <Text style={styles.searchIcon}>⌕</Text>
        <TextInput value={q} onChangeText={setQ} placeholder="Find a tutor or language..." placeholderTextColor={COLOR.outline} style={styles.searchInput} returnKeyType="search" onSubmitEditing={load} testID="tutor-search" />
        {q.length > 0 && (
          <Pressable onPress={() => setQ('')} style={styles.clearBtn}>
            <Text style={styles.clearText}>✕</Text>
          </Pressable>
        )}
      </View>

      <View style={styles.filterRow}>
        <View style={styles.filterChip}><Text style={styles.filterChipText}>Language ▾</Text></View>
        <View style={styles.filterChip}><Text style={styles.filterChipText}>Price ▾</Text></View>
        <View style={styles.filterChip}><Text style={styles.filterChipText}>Rating ▾</Text></View>
      </View>
      <View style={styles.linksRow}>
        <Pressable onPress={() => (navigation as any).navigate('TrialCredits')}><Text style={styles.link}>Trial credits</Text></Pressable>
        <Text style={styles.dot}>·</Text>
        <Pressable onPress={() => (navigation as any).navigate('TeacherDashboard')}><Text style={styles.link}>Dashboard</Text></Pressable>
        <Text style={styles.dot}>·</Text>
        <Pressable onPress={() => (navigation as any).navigate('Payouts')}><Text style={styles.link}>Payouts</Text></Pressable>
      </View>

      {loading ? (
        <View style={styles.center}><ActivityIndicator color={COLOR.primary} /></View>
      ) : tutors.length === 0 ? (
        <View style={styles.emptyCard}>
          <Text style={styles.emptyTitle}>No tutors yet</Text>
          <Text style={styles.emptySub}>Try a different search or check back later.</Text>
          <Pressable style={styles.becomeBtn} onPress={() => (navigation as any).navigate('BecomeTeacher')}>
            <Text style={styles.becomeText}>Become a teacher</Text>
          </Pressable>
        </View>
      ) : (
        <>
          {featured.length > 0 && (
            <View style={styles.featuredSection}>
              <View style={styles.sectionHead}>
                <Text style={styles.sectionTitle}>Featured Tutors</Text>
                <Pressable onPress={load}><Text style={styles.seeAll}>See all</Text></Pressable>
              </View>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.featuredRow}>
                {featured.map((t) => (
                  <Pressable key={t.userId} style={styles.featuredCard} onPress={() => navigation.navigate('TutorProfile', { userId: t.userId })} testID={`tutor-featured-${t.userId}`}>
                    <View style={styles.featuredTop} />
                    <View style={styles.featuredBody}>
                      <View style={styles.avatarLg}><Text style={styles.avatarLgText}>{(t.displayName || t.userId).slice(0, 1).toUpperCase()}</Text></View>
                      <Text style={styles.featuredName}>{t.displayName || t.userId}</Text>
                      <Text style={styles.featuredLang}>{(t.languages || []).slice(0, 2).join(' • ') || 'Tutor'}</Text>
                      <View style={styles.ratingPill}><Text style={styles.ratingPillText}>★ {(t.ratingAvg ?? 5).toFixed(1)}</Text></View>
                    </View>
                  </Pressable>
                ))}
              </ScrollView>
            </View>
          )}

          <View style={styles.listSection}>
            <Text style={styles.sectionTitle}>Available Now</Text>
            {rest.length === 0 && featured.length > 0 ? (
              <Text style={styles.emptySub}>More tutors coming soon.</Text>
            ) : null}
            {rest.map((t) => (
              <View key={t.userId} style={styles.listCard} testID={`tutor-card-${t.userId}`}>
                <View style={styles.listRow}>
                  <View style={styles.avatarSm}><Text style={styles.avatarSmText}>{(t.displayName || t.userId).slice(0, 1).toUpperCase()}</Text></View>
                  <View style={styles.listMain}>
                    <View style={styles.listHead}>
                      <Text style={styles.listName}>{t.displayName || t.userId}</Text>
                      <Text style={styles.listRating}>★ {(t.ratingAvg ?? 4.8).toFixed(1)}</Text>
                    </View>
                    <Text style={styles.listLang}>{(t.languages || []).join(' • ') || 'Language tutor'}</Text>
                    <Text style={styles.listPrice}>${Math.round((t.rateCents ?? 1800) / 100)} / session</Text>
                  </View>
                </View>
                <View style={styles.listActions}>
                  <Pressable style={styles.secondaryBtn} onPress={() => navigation.navigate('TutorProfile', { userId: t.userId })}>
                    <Text style={styles.secondaryBtnText}>View Profile</Text>
                  </Pressable>
                  <Pressable style={styles.primaryBtn} onPress={() => navigation.navigate('TutorProfile', { userId: t.userId })}>
                    <Text style={styles.primaryBtnText}>Book Trial</Text>
                  </Pressable>
                </View>
              </View>
            ))}
          </View>
        </>
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, paddingBottom: 32, gap: SPACING.stackMd },
  center: { padding: 24, alignItems: 'center' },
  searchRow: { flexDirection: 'row', alignItems: 'center', backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.full, borderWidth: 1, borderColor: COLOR.outlineVariant, paddingHorizontal: 12, height: 44, ...SHADOWS.elevation1 },
  searchIcon: { fontSize: 16, color: COLOR.outline, marginRight: 8 },
  searchInput: { flex: 1, color: COLOR.onSurface, fontFamily: FONTS.body, fontSize: 14 },
  clearBtn: { padding: 4 },
  clearText: { color: COLOR.outline },
  filterRow: { flexDirection: 'row', gap: 8 },
  filterChip: { borderWidth: 1, borderColor: COLOR.outlineVariant, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.full, paddingHorizontal: 12, paddingVertical: 6 },
  filterChipText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  linksRow: { flexDirection: 'row', gap: 8, alignItems: 'center', justifyContent: 'center' },
  link: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label, fontSize: 13 },
  dot: { color: COLOR.outlineVariant },
  emptyCard: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: 20, alignItems: 'center', gap: 8, ...SHADOWS.elevation1 },
  emptyTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline },
  emptySub: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body, textAlign: 'center' },
  becomeBtn: { backgroundColor: COLOR.primary, borderRadius: RADIUS.full, paddingHorizontal: 16, paddingVertical: 10, marginTop: 8 },
  becomeText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  featuredSection: { gap: 8 },
  sectionHead: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'baseline' },
  sectionTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 16 },
  seeAll: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  featuredRow: { gap: 12, paddingRight: 16, paddingBottom: 4 },
  featuredCard: { width: 180, backgroundColor: COLOR.surfaceContainerLowest, borderRadius: 20, borderWidth: 1, borderColor: COLOR.outlineVariant, overflow: 'hidden', ...SHADOWS.elevation1 },
  featuredTop: { height: 56, backgroundColor: COLOR.surfaceContainerHigh },
  featuredBody: { alignItems: 'center', padding: 12, marginTop: -28, gap: 4 },
  avatarLg: { width: 56, height: 56, borderRadius: 28, backgroundColor: COLOR.secondaryContainer, borderWidth: 3, borderColor: COLOR.surfaceContainerLowest, alignItems: 'center', justifyContent: 'center' },
  avatarLgText: { color: COLOR.onSecondaryContainer, fontWeight: '700', fontSize: 18 },
  featuredName: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontSize: 16, fontFamily: FONTS.headline },
  featuredLang: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  ratingPill: { backgroundColor: COLOR.surfaceContainer, borderRadius: RADIUS.full, paddingHorizontal: 8, paddingVertical: 2, marginTop: 4 },
  ratingPillText: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurface, fontFamily: FONTS.label },
  listSection: { gap: 12 },
  listCard: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: 14, borderWidth: 1, borderColor: COLOR.outlineVariant, gap: 12, ...SHADOWS.elevation1 },
  listRow: { flexDirection: 'row', gap: 12 },
  avatarSm: { width: 48, height: 48, borderRadius: 24, backgroundColor: COLOR.surfaceVariant, alignItems: 'center', justifyContent: 'center' },
  avatarSmText: { color: COLOR.onSurfaceVariant, fontWeight: '700' },
  listMain: { flex: 1, gap: 2 },
  listHead: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  listName: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontSize: 16, fontFamily: FONTS.headline },
  listRating: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurface, fontFamily: FONTS.label },
  listLang: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  listPrice: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  listActions: { flexDirection: 'row', gap: 8, justifyContent: 'flex-end' },
  secondaryBtn: { borderWidth: 1, borderColor: COLOR.primary, borderRadius: RADIUS.full, paddingHorizontal: 14, paddingVertical: 8 },
  secondaryBtnText: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  primaryBtn: { backgroundColor: COLOR.primary, borderRadius: RADIUS.full, paddingHorizontal: 14, paddingVertical: 8 },
  primaryBtnText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
});
