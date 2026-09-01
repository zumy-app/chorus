import React, { useEffect, useState, useCallback } from 'react';
import { View, Text, StyleSheet, ScrollView, Pressable, ActivityIndicator, Alert } from 'react-native';
import { useRoute, useNavigation } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import type { MarketplaceStackParamList } from '../components/MainTabs';
import type { TutorProfile, TutorReview } from '@chorus/shared';

type Route = RouteProp<MarketplaceStackParamList, 'TutorProfile'>;

export default function TutorProfileScreen() {
  const route = useRoute<Route>();
  const navigation = useNavigation<NativeStackNavigationProp<MarketplaceStackParamList>>();
  const { userId } = route.params;
  const [tutor, setTutor] = useState<TutorProfile | null>(null);
  const [reviews, setReviews] = useState<TutorReview[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const t = await apiService.getTutorProfile(userId);
      setTutor(t);
      try {
        const r = await apiService.getTutorReviews(userId, { limit: 3 });
        setReviews(r.reviews ?? []);
      } catch {}
    } catch {
      Alert.alert('Error', 'Could not load tutor profile');
    } finally { setLoading(false); }
  }, [userId]);

  useEffect(() => { load(); }, [load]);

  const bookTrial = () => {
    (navigation as any).navigate('ConfirmBooking', { userId });
  };

  if (loading) {
    return <View style={styles.center}><ActivityIndicator color={COLOR.primary} /></View>;
  }
  if (!tutor) {
    return <View style={styles.center}><Text style={styles.body}>Tutor not found.</Text><Pressable onPress={() => navigation.goBack()}><Text style={styles.link}>Go back</Text></Pressable></View>;
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
      <View style={styles.hero}>
        <View style={styles.avatarHero}><Text style={styles.avatarHeroText}>{(tutor.displayName || userId).slice(0, 1).toUpperCase()}</Text></View>
        <View style={styles.heroText}>
          <View style={styles.nameRow}>
            <Text style={styles.name}>{tutor.displayName || 'Tutor'}</Text>
            {tutor.verified ? <View style={styles.verifiedBadge}><Text style={styles.verifiedText}>✓ Verified</Text></View> : null}
          </View>
          <Text style={styles.sub}>{(tutor.languages || []).join(' • ') || 'Language tutor'}</Text>
          <View style={styles.metaRow}>
            <Text style={styles.meta}>★ {(tutor.ratingAvg ?? 4.9).toFixed(1)} · {tutor.ratingCount ?? 0} reviews</Text>
            <Text style={styles.meta}>· ${(Math.round((tutor.rateCents ?? 1800) / 100))}/session</Text>
          </View>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>About</Text>
        <Text style={styles.body}>{tutor.bio || 'No bio yet.'}</Text>
      </View>

      {reviews.length > 0 && (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Reviews</Text>
          {reviews.map((r) => (
            <View key={r.id} style={styles.review}>
              <Text style={styles.reviewRating}>★ {r.rating}</Text>
              <Text style={styles.body}>{r.comment || ''}</Text>
            </View>
          ))}
        </View>
      )}

      <View style={styles.ctaRow}>
        <Pressable style={styles.secondaryBtn} onPress={() => navigation.goBack()}>
          <Text style={styles.secondaryBtnText}>Back</Text>
        </Pressable>
        <Pressable style={styles.primaryBtn} onPress={bookTrial} testID="book-trial">
          <Text style={styles.primaryBtnText}>Book Trial</Text>
        </Pressable>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, gap: SPACING.stackMd, paddingBottom: 32 },
  center: { flex: 1, backgroundColor: COLOR.background, alignItems: 'center', justifyContent: 'center', padding: 24 },
  hero: { flexDirection: 'row', gap: SPACING.stackMd, alignItems: 'center', backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, ...SHADOWS.elevation1 },
  avatarHero: { width: 72, height: 72, borderRadius: 36, backgroundColor: COLOR.secondaryContainer, alignItems: 'center', justifyContent: 'center' },
  avatarHeroText: { color: COLOR.onSecondaryContainer, fontWeight: '700', fontSize: 28 },
  heroText: { flex: 1, gap: 4 },
  nameRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  name: { ...TYPOGRAPHY.headlineMd, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 22 },
  verifiedBadge: { backgroundColor: COLOR.surfaceContainerHigh, borderRadius: RADIUS.full, paddingHorizontal: 8, paddingVertical: 2 },
  verifiedText: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, fontFamily: FONTS.label },
  sub: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  metaRow: { flexDirection: 'row', gap: 6 },
  meta: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.label },
  card: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, gap: 8, ...SHADOWS.elevation1 },
  cardTitle: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 16 },
  body: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  link: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, marginTop: 12, fontFamily: FONTS.label },
  review: { borderTopWidth: 1, borderTopColor: COLOR.outlineVariant, paddingTop: 8, gap: 4 },
  reviewRating: { ...TYPOGRAPHY.labelMd, color: COLOR.onSurface, fontFamily: FONTS.label },
  ctaRow: { flexDirection: 'row', gap: 12, marginTop: 4 },
  secondaryBtn: { flex: 1, borderWidth: 1, borderColor: COLOR.primary, borderRadius: RADIUS.full, paddingVertical: 14, alignItems: 'center' },
  secondaryBtnText: { ...TYPOGRAPHY.labelMd, color: COLOR.primary, fontFamily: FONTS.label },
  primaryBtn: { flex: 1, backgroundColor: COLOR.primary, borderRadius: RADIUS.full, paddingVertical: 14, alignItems: 'center' },
  primaryBtnText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  disabled: { opacity: 0.6 },
});
