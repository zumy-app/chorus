import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Linking,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { COLOR, COLORS } from '../theme';

// Mirrors the web frontend's Pricing page (frontend/src/pages/Pricing.tsx).
// Guest users get routed to Register for the premium plan (same as web, where
// guests sign up before checkout).
const FREE_FEATURES = [
  'Unlimited chats & groups',
  'Live translation up to 200 characters',
  'On-demand grammar & vocabulary tools',
  'Search across all messages',
];

const PREMIUM_FEATURES = [
  'Automatic grammar analysis',
  'Faster AI responses',
  'Messages longer than 200 characters',
  'Higher daily quotas',
];

const ENTERPRISE_FEATURES = [
  'Self-hosting & custom domains',
  'Dedicated support',
  'Volume & SLA options',
];

const COMPARISON: { label: string; free: string; premium: string }[] = [
  { label: 'Unlimited chats & groups', free: '✓', premium: '✓' },
  { label: 'Automatic grammar analysis', free: '✗', premium: '✓' },
  { label: 'Faster AI responses', free: '✗', premium: '✓' },
  { label: 'Messages longer than 200 characters', free: '✗', premium: '✓' },
  { label: 'Search across all messages', free: '✓', premium: '✓' },
  { label: 'Higher daily quotas', free: '✗', premium: '✓' },
];

export default function PricingScreen({ navigation }: any) {
  const [billing, setBilling] = useState<'monthly' | 'annual'>('annual');

  return (
    <SafeAreaView style={styles.safeArea} edges={['left', 'right']}>
      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}>
        {/* Hero */}
        <View style={styles.hero}>
          <Text style={styles.heroTitle}>Simple Pricing</Text>
          <Text style={styles.heroSubtitle}>Start free. Upgrade when you're ready for more.</Text>
          <View style={styles.heroBadge}>
            <Text style={styles.heroBadgeText}>✨ 2 months free on annual billing</Text>
          </View>
        </View>

        {/* Free */}
        <View style={styles.planCard}>
          <Text style={styles.planName}>Free</Text>
          <Text style={styles.planPrice}>
            $0<Text style={styles.planPer}>/forever</Text>
          </Text>
          {FREE_FEATURES.map((f, i) => (
            <Text key={i} style={styles.planFeature}>✓ {f}</Text>
          ))}
          <TouchableOpacity
            style={styles.planButtonOutline}
            onPress={() => navigation.navigate('Register')}>
            <Text style={styles.planButtonOutlineText}>Get Started Free</Text>
          </TouchableOpacity>
        </View>

        {/* Premium */}
        <View style={styles.planCardPremium}>
          <Text style={styles.planNamePremium}>✦ Premium</Text>
          <View style={styles.billingToggle}>
            <TouchableOpacity
              style={[styles.billingOption, billing === 'monthly' && styles.billingOptionActive]}
              onPress={() => setBilling('monthly')}>
              <Text
                style={[
                  styles.billingOptionText,
                  billing === 'monthly' && styles.billingOptionTextActive,
                ]}>
                Monthly
              </Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.billingOption, billing === 'annual' && styles.billingOptionActive]}
              onPress={() => setBilling('annual')}>
              <Text
                style={[
                  styles.billingOptionText,
                  billing === 'annual' && styles.billingOptionTextActive,
                ]}>
                Annual
              </Text>
            </TouchableOpacity>
          </View>
          {billing === 'annual' ? (
            <>
              <Text style={styles.planStrike}>
                $95.90/year
              </Text>
              <Text style={styles.planPricePremium}>
                $79.90<Text style={styles.planPerPremium}>/year</Text>
              </Text>
              <Text style={styles.planPromo}>✨ 2 months free</Text>
            </>
          ) : (
            <Text style={styles.planPricePremium}>
              $7.99<Text style={styles.planPerPremium}>/month</Text>
            </Text>
          )}
          {PREMIUM_FEATURES.map((f, i) => (
            <Text key={i} style={styles.planFeaturePremium}>✓ {f}</Text>
          ))}
          <TouchableOpacity
            style={styles.planButtonPremium}
            onPress={() => navigation.navigate('Register')}>
            <Text style={styles.planButtonPremiumText}>Get Premium</Text>
          </TouchableOpacity>
          <Text style={styles.purchaseNote}>
            You'll create a free account first, then upgrade from your profile.
          </Text>
        </View>

        {/* Enterprise */}
        <View style={styles.planCard}>
          <Text style={styles.planName}>Enterprise</Text>
          <Text style={styles.planDesc}>Self-hosted or custom deployment for teams.</Text>
          {ENTERPRISE_FEATURES.map((f, i) => (
            <Text key={i} style={styles.planFeature}>✓ {f}</Text>
          ))}
          <TouchableOpacity
            style={styles.planButtonOutline}
            onPress={() => Linking.openURL('mailto:hello@chorus.talk?subject=Enterprise%20Enquiry')}>
            <Text style={styles.planButtonOutlineText}>Contact Us</Text>
          </TouchableOpacity>
        </View>

        {/* Comparison */}
        <View style={styles.compareSection}>
          <Text style={styles.compareTitle}>Compare Features</Text>
          <View style={styles.compareTable}>
            <View style={styles.compareHeader}>
              <Text style={[styles.compareCell, styles.compareFeatureCell, styles.compareHeaderText]}>
                Feature
              </Text>
              <Text style={[styles.compareCell, styles.compareHeaderText]}>Free</Text>
              <Text style={[styles.compareCell, styles.compareHeaderText, styles.comparePremiumHeader]}>
                ✦ Premium
              </Text>
            </View>
            {COMPARISON.map((row, i) => (
              <View
                key={i}
                style={[styles.compareRow, i % 2 === 1 && styles.compareRowAlt]}>
                <Text style={[styles.compareCell, styles.compareFeatureCell, styles.compareRowText]}>
                  {row.label}
                </Text>
                <Text style={[styles.compareCell, styles.compareRowText, styles.compareCellCenter]}>
                  {row.free}
                </Text>
                <Text
                  style={[
                    styles.compareCell,
                    styles.compareCellCenter,
                    styles.comparePremiumValue,
                  ]}>
                  {row.premium}
                </Text>
              </View>
            ))}
          </View>
          <Text style={styles.compareNote}>Free forever plan — no credit card required.</Text>
        </View>

        {/* Footer */}
        <View style={styles.footer}>
          <Text style={styles.footerText}>Simple Pricing · Chorus</Text>
          <TouchableOpacity onPress={() => navigation.navigate('Landing')}>
            <Text style={styles.footerLink}>← Back to Chorus</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    paddingBottom: 24,
  },
  hero: {
    paddingHorizontal: 20,
    paddingVertical: 32,
    alignItems: 'center',
  },
  heroTitle: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    marginBottom: 8,
  },
  heroSubtitle: {
    fontSize: 16,
    color: COLOR.onSurfaceVariant,
    marginBottom: 16,
  },
  heroBadge: {
    backgroundColor: COLOR.primaryFixed,
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 6,
  },
  heroBadgeText: {
    color: COLORS.primary,
    fontSize: 13,
    fontWeight: '600',
  },
  planCard: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 16,
    padding: 22,
    marginHorizontal: 20,
    marginBottom: 16,
  },
  planName: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    marginBottom: 4,
  },
  planPrice: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    marginBottom: 16,
  },
  planPer: {
    fontSize: 14,
    fontWeight: 'normal',
    color: COLOR.onSurfaceVariant,
  },
  planDesc: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
    marginBottom: 16,
  },
  planFeature: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
    marginBottom: 8,
  },
  planButtonOutline: {
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  planButtonOutlineText: {
    color: COLOR.onSurface,
    fontSize: 15,
    fontWeight: '600',
  },
  planCardPremium: {
    backgroundColor: COLOR.secondary,
    borderRadius: 16,
    padding: 22,
    marginHorizontal: 20,
    marginBottom: 16,
    shadowColor: COLOR.secondary,
    shadowOffset: { width: 0, height: 6 },
    shadowOpacity: 0.3,
    shadowRadius: 12,
    elevation: 6,
  },
  planNamePremium: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLOR.surfaceContainerLowest,
    marginBottom: 12,
  },
  billingToggle: {
    flexDirection: 'row',
    backgroundColor: 'rgba(255,255,255,0.15)',
    borderRadius: 10,
    padding: 4,
    marginBottom: 16,
  },
  billingOption: {
    flex: 1,
    borderRadius: 8,
    paddingVertical: 8,
    alignItems: 'center',
  },
  billingOptionActive: {
    backgroundColor: COLOR.surfaceContainerLowest,
  },
  billingOptionText: {
    fontSize: 14,
    fontWeight: '600',
color: 'rgba(255,255,255,0.8)',
  },
  billingOptionTextActive: {
    color: COLOR.primary,
  },
  planStrike: {
    fontSize: 14,
    color: 'rgba(255,255,255,0.8)',
    textDecorationLine: 'line-through',
    marginBottom: 2,
  },
  planPricePremium: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLOR.surfaceContainerLowest,
    marginBottom: 4,
  },
  planPerPremium: {
    fontSize: 14,
    fontWeight: 'normal',
    color: 'rgba(255,255,255,0.8)',
  },
  planPromo: {
    fontSize: 13,
    fontWeight: '600',
    color: 'rgba(255,255,255,0.9)',
    marginBottom: 16,
  },
  planFeaturePremium: {
    fontSize: 14,
    color: COLOR.surfaceContainerLowest,
    marginBottom: 8,
  },
  planButtonPremium: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  planButtonPremiumText: {
    color: COLORS.primary,
    fontSize: 15,
    fontWeight: 'bold',
  },
  purchaseNote: {
    fontSize: 12,
    color: 'rgba(255,255,255,0.9)',
    textAlign: 'center',
    marginTop: 10,
  },
  compareSection: {
    paddingHorizontal: 20,
    paddingVertical: 24,
  },
  compareTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    textAlign: 'center',
    marginBottom: 4,
  },
  compareTable: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 16,
    overflow: 'hidden',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.08,
    shadowRadius: 6,
    elevation: 2,
  },
  compareHeader: {
    flexDirection: 'row',
    backgroundColor: COLOR.surfaceContainerLow,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
    paddingVertical: 14,
    paddingHorizontal: 16,
  },
  compareHeaderText: {
    fontSize: 14,
    fontWeight: '600',
    color: COLOR.onSurface,
  },
  comparePremiumHeader: {
    color: COLORS.primary,
  },
  compareRow: {
    flexDirection: 'row',
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
  },
  compareRowAlt: {
    backgroundColor: COLOR.surfaceContainerLow,
  },
  compareCell: {
    flex: 1,
  },
  compareFeatureCell: {
    flex: 2,
  },
  compareCellCenter: {
    textAlign: 'center',
  },
  compareRowText: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
  },
  comparePremiumValue: {
    color: COLORS.primary,
    fontWeight: '600',
  },
  compareNote: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
    textAlign: 'center',
    marginTop: 16,
  },
  footer: {
    backgroundColor: COLOR.inverseSurface,
    paddingVertical: 28,
    alignItems: 'center',
  },
  footerText: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
    marginBottom: 8,
  },
  footerLink: {
    fontSize: 14,
    color: COLOR.primaryFixed,
    fontWeight: '500',
  },
});
