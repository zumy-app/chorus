import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, Pressable, ActivityIndicator, Alert } from 'react-native';
import { useRoute, useNavigation } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import type { MarketplaceStackParamList } from '../components/MainTabs';
import type { TutorProfile } from '@chorus/shared';

type Route = RouteProp<MarketplaceStackParamList, 'ConfirmBooking'>;

export default function ConfirmBookingScreen() {
  const route = useRoute<Route>();
  const navigation = useNavigation<any>();
  const { userId } = (route.params as any) || {};
  const [tutor, setTutor] = useState<TutorProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [booking, setBooking] = useState(false);

  useEffect(() => {
    if (!userId) return;
    apiService.getTutorProfile(userId).then(setTutor).catch(() => {}).finally(() => setLoading(false));
  }, [userId]);

  const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
  const dateLabel = tomorrow.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' } as any);

  const confirm = async () => {
    setBooking(true);
    try {
      const start = new Date(tomorrow);
      start.setHours(10, 0, 0, 0);
      const end = new Date(start.getTime() + 30 * 60 * 1000);
      await apiService.bookTutor(userId, { startTime: start.toISOString(), endTime: end.toISOString(), isTrial: true });
      Alert.alert('Booked', 'Trial booked! Check your credits.');
      navigation.navigate('TrialCredits' as never);
    } catch (e: any) {
      Alert.alert('Failed', e?.response?.data?.error || e.message);
    } finally { setBooking(false); }
  };

  if (loading) return <View style={styles.center}><ActivityIndicator color={COLOR.primary} /></View>;
  if (!tutor) return <View style={styles.center}><Text style={styles.body}>Tutor not found</Text></View>;

  return (
    <View style={styles.container}>
      <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
        <View style={styles.heroCenter}>
          <View style={styles.check}><Text style={styles.checkText}>✓</Text></View>
          <Text style={styles.title}>Great choice!</Text>
          <Text style={styles.sub}>Review your trial session details below.</Text>
        </View>
        <View style={styles.cardRow}>
          <View style={styles.avatar}><Text style={styles.avatarText}>{(tutor.displayName || userId || 'T').slice(0, 1).toUpperCase()}</Text></View>
          <View><Text style={styles.label}>Your Tutor</Text><Text style={styles.name}>{tutor.displayName}</Text><Text style={styles.subSm}>{(tutor.languages || []).slice(0, 1).join('') || 'Language'} · Native</Text></View>
        </View>
        <View style={styles.card}>
          <View style={styles.row}><View style={styles.icon}><Text>📅</Text></View><View><Text style={styles.label}>Date</Text><Text style={styles.bodyBold}>{dateLabel}</Text></View></View>
          <View style={styles.divider} />
          <View style={styles.row}><View style={styles.icon}><Text>⏰</Text></View><View><Text style={styles.label}>Time</Text><Text style={styles.bodyBold}>10:00 AM - 10:30 AM</Text></View></View>
        </View>
        <View style={styles.summary}>
          <Text style={styles.label}>Payment Summary</Text>
          <View style={styles.summaryRow}><Text style={styles.sub}>Trial Session</Text><Text style={styles.body}>1 Credit</Text></View>
          <View style={styles.summaryRow}><Text style={styles.sub}>Credits Applied</Text><Text style={styles.body}>-1 Credit</Text></View>
          <View style={styles.divider} />
          <View style={styles.summaryRow}><Text style={styles.bodyBold}>Total</Text><Text style={styles.bodyBold}>$0.00</Text></View>
        </View>
        <View style={styles.policy}><Text style={styles.sub}>ℹ Cancellation Policy: You can reschedule or cancel for free up to 24 hours before your trial.</Text></View>
      </ScrollView>
      <View style={styles.footer}>
        <Pressable style={[styles.primaryBtn, booking && styles.disabled]} onPress={confirm} disabled={booking} testID="confirm-booking">
          {booking ? <ActivityIndicator color={COLOR.onPrimary} /> : <Text style={styles.primaryBtnText}>Confirm Booking</Text>}
        </Pressable>
      </View>
    </View>
  );
}
const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLOR.background },
  content: { padding: SPACING.marginMobile, gap: SPACING.stackMd, paddingBottom: 88 },
  center: { flex: 1, backgroundColor: COLOR.background, alignItems: 'center', justifyContent: 'center', padding: 24 },
  heroCenter: { alignItems: 'center', gap: 4 },
  check: { width: 56, height: 56, borderRadius: 28, backgroundColor: 'rgba(0,125,85,0.12)', alignItems: 'center', justifyContent: 'center' },
  checkText: { color: COLOR.tertiary, fontSize: 28 },
  title: { ...TYPOGRAPHY.headlineMd, color: COLOR.onSurface, fontFamily: FONTS.headline },
  sub: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  subSm: { ...TYPOGRAPHY.labelSm, color: COLOR.primary, fontFamily: FONTS.label },
  card: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, gap: 8, ...SHADOWS.elevation1 },
  cardRow: { backgroundColor: COLOR.surfaceContainerLowest, borderRadius: RADIUS.xl, padding: SPACING.stackMd, flexDirection: 'row', gap: 12, alignItems: 'center', ...SHADOWS.elevation1 },
  avatar: { width: 48, height: 48, borderRadius: 24, backgroundColor: COLOR.secondaryContainer, alignItems: 'center', justifyContent: 'center' },
  avatarText: { color: COLOR.onSecondaryContainer, fontWeight: '700' },
  label: { ...TYPOGRAPHY.labelSm, color: COLOR.onSurfaceVariant, textTransform: 'uppercase', fontFamily: FONTS.label, fontSize: 11 },
  name: { ...TYPOGRAPHY.headlineSm, color: COLOR.onSurface, fontFamily: FONTS.headline, fontSize: 18 },
  body: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, fontFamily: FONTS.body },
  bodyBold: { ...TYPOGRAPHY.bodySm, color: COLOR.onSurface, fontWeight: '600', fontFamily: FONTS.body },
  row: { flexDirection: 'row', gap: 12, alignItems: 'center' },
  icon: { width: 36, height: 36, borderRadius: 18, backgroundColor: 'rgba(37,99,235,0.10)', alignItems: 'center', justifyContent: 'center' },
  divider: { height: 1, backgroundColor: COLOR.outlineVariant, opacity: 0.3 },
  summary: { backgroundColor: COLOR.surfaceContainerLow, borderRadius: RADIUS.xl, padding: SPACING.stackMd, gap: 8 },
  summaryRow: { flexDirection: 'row', justifyContent: 'space-between' },
  policy: { backgroundColor: 'rgba(33,49,69,0.06)', borderRadius: 8, padding: 12 },
  footer: { position: 'absolute', bottom: 0, left: 0, right: 0, backgroundColor: COLOR.surfaceContainerLowest, padding: SPACING.marginMobile, borderTopWidth: 1, borderTopColor: COLOR.outlineVariant },
  primaryBtn: { backgroundColor: COLOR.primary, borderRadius: RADIUS.full, paddingVertical: 14, alignItems: 'center' },
  primaryBtnText: { ...TYPOGRAPHY.labelMd, color: COLOR.onPrimary, fontFamily: FONTS.label },
  disabled: { opacity: 0.6 },
});
