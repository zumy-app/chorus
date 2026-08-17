import React, { useRef } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  Linking,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { TOP10_LANGUAGES } from '@chorus/shared';
import { COLOR, RADIUS } from '../theme';

// Mirrors the web frontend's Landing page (frontend/src/pages/Landing.tsx),
// so the mobile app provides the same flow: home page with content and
// navigation -> Login/Register -> Chat.
const LANGUAGES = TOP10_LANGUAGES;

const FEATURES = [
  { icon: '🌐', title: 'Instant Translation', desc: 'Messages are automatically translated to your preferred language in real-time. No delays, no manual selection.' },
  { icon: '✏️', title: 'Grammar Analysis', desc: 'AI-powered grammar checking with CEFR difficulty assessment helps you learn while you chat.' },
  { icon: '📚', title: 'Vocabulary Builder', desc: 'Smart spaced repetition system helps you remember new words and phrases from your conversations.' },
  { icon: '👥', title: 'Group Chats', desc: 'Create multilingual group conversations with up to 100 participants, each reading in their own language.' },
  { icon: '🔍', title: 'Smart Search', desc: 'Find messages across all your chats with full-text search that works in multiple languages.' },
  { icon: '🔒', title: 'Privacy First', desc: 'Your conversations are encrypted and secure. We don\'t store your messages permanently.' },
];

const FEATURE_ACCENTS = [
  { bg: 'rgba(37,99,235,0.12)', icon: '#004AC6' },
  { bg: 'rgba(132,85,239,0.12)', icon: '#6B38D4' },
  { bg: 'rgba(0,125,89,0.12)', icon: '#006242' },
  { bg: 'rgba(37,99,235,0.12)', icon: '#004AC6' },
  { bg: 'rgba(132,85,239,0.12)', icon: '#6B38D4' },
  { bg: 'rgba(0,125,89,0.12)', icon: '#006242' },
];

const STEPS = [
  { num: '1', title: 'Sign Up Free', desc: 'Create your account and select your native language and the languages you want to learn.' },
  { num: '2', title: 'Start Chatting', desc: 'Find friends or join groups. Type messages in your language—they\'ll be translated automatically.' },
  { num: '3', title: 'Learn & Grow', desc: 'Save vocabulary, review grammar suggestions, and improve your language skills naturally.' },
];

export default function LandingScreen({ navigation }: any) {
  const scrollRef = useRef<ScrollView>(null);
  const sectionY = useRef<Record<string, number>>({});

  const scrollToSection = (key: string) => {
    const y = sectionY.current[key] ?? 0;
    scrollRef.current?.scrollTo({ y, animated: true });
  };

  const registerSection = (key: string) => (e: any) => {
    sectionY.current[key] = e.nativeEvent.layout.y;
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'left', 'right']}>
      <StatusBar barStyle="dark-content" backgroundColor={COLOR.surface} />
      {/* Navigation */}
      <View style={styles.nav}>
        <View style={styles.navRow}>
          <View style={styles.brand}>
            <View style={styles.brandLogo}>
              <Text style={styles.brandLogoText}>💬</Text>
            </View>
            <Text style={styles.brandName}>Chorus</Text>
          </View>
          <TouchableOpacity
            style={styles.loginButton}
            onPress={() => navigation.navigate('Login')}>
            <Text style={styles.loginButtonText}>Log In</Text>
          </TouchableOpacity>
        </View>
        <View style={styles.navLinks}>
          <TouchableOpacity onPress={() => scrollToSection('features')}>
            <Text style={styles.navLink}>Features</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => scrollToSection('how')}>
            <Text style={styles.navLink}>How It Works</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => navigation.navigate('Pricing')}>
            <Text style={styles.navLink}>Pricing</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => scrollToSection('languages')}>
            <Text style={styles.navLink}>Languages</Text>
          </TouchableOpacity>
        </View>
      </View>

      <ScrollView
        ref={scrollRef}
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}>
        {/* Hero Section */}
        <View style={[styles.section, styles.heroSection]}>
          <View style={styles.badge}>
            <Text style={styles.badgeText}>🌍 Available in {LANGUAGES.length} languages</Text>
          </View>
          <Text style={styles.heroTitle}>
            Break Language Barriers,{'\n'}
            <Text style={styles.heroTitleAccent}>Connect Globally</Text>
          </Text>
          <Text style={styles.heroSubtitle}>
            Real-time messaging with instant translation in 10 major languages. Chat
            naturally in your language while others read in theirs.
          </Text>

          <View style={styles.heroButtons}>
            <TouchableOpacity
              style={styles.primaryButton}
              onPress={() => navigation.navigate('Register')}>
              <Text style={styles.primaryButtonText}>Get Started Free</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={styles.secondaryButton}
              onPress={() => scrollToSection('how')}>
              <Text style={styles.secondaryButtonText}>See How It Works</Text>
            </TouchableOpacity>
          </View>

          <View style={styles.stats}>
            <View style={styles.stat}>
              <Text style={styles.statValue}>{LANGUAGES.length}</Text>
              <Text style={styles.statLabel}>Languages</Text>
            </View>
            <View style={styles.stat}>
              <Text style={styles.statValue}>Real-time</Text>
              <Text style={styles.statLabel}>Translation</Text>
            </View>
            <View style={styles.stat}>
              <Text style={styles.statValue}>100%</Text>
              <Text style={styles.statLabel}>Free to Use</Text>
            </View>
          </View>

          {/* AI Tutor Sparky hero visual */}
          <View style={styles.heroVisual}>
            <View style={styles.heroVisualBg}>
              <Text style={styles.heroVisualIcon}>🌐</Text>
              <Text style={styles.heroVisualTag}>Real-time AI Translation</Text>
            </View>
            <View style={styles.sparkyCard}>
              <View style={styles.sparkyHeader}>
                <View style={styles.sparkyAvatar}>
                  <Text style={styles.sparkyAvatarText}>🤖</Text>
                </View>
                <Text style={styles.sparkyName}>AI Tutor Sparky</Text>
              </View>
              <Text style={styles.sparkyMessage}>¡Hola! ¿Cómo estuvo tu día hoy?</Text>
              <Text style={styles.sparkyTranslation}>Hello! How was your day today?</Text>
            </View>
          </View>
        </View>

        {/* Features Section */}
        <View style={[styles.section, styles.altSection]} onLayout={registerSection('features')}>
          <Text style={styles.sectionTitle}>Powerful Features for Global Communication</Text>
          <Text style={styles.sectionSubtitle}>Everything you need to connect with people worldwide</Text>
          <View style={styles.featuresGrid}>
            {FEATURES.map((feature, i) => {
              const accent = FEATURE_ACCENTS[i % FEATURE_ACCENTS.length];
              return (
                <View key={i} style={styles.featureCard}>
                  <View style={[styles.featureIconCircle, { backgroundColor: accent.bg }]}>
                    <Text style={styles.featureIcon}>{feature.icon}</Text>
                  </View>
                  <Text style={styles.featureTitle}>{feature.title}</Text>
                  <Text style={styles.featureDesc}>{feature.desc}</Text>
                </View>
              );
            })}
          </View>
        </View>

        {/* How It Works Section */}
        <View style={[styles.section, styles.sectionPlain]} onLayout={registerSection('how')}>
          <Text style={styles.sectionTitle}>How Chorus Works</Text>
          <Text style={styles.sectionSubtitle}>Start chatting in minutes, no language barriers</Text>
          <View style={styles.stepsWrap}>
            {STEPS.map((step, i) => (
              <View key={i} style={styles.step}>
                <View style={styles.stepCircle}>
                  <Text style={styles.stepNum}>{step.num}</Text>
                </View>
                <Text style={styles.stepTitle}>{step.title}</Text>
                <Text style={styles.stepDesc}>{step.desc}</Text>
              </View>
            ))}
          </View>
        </View>

        {/* Languages Section */}
        <View style={[styles.section, styles.altSection]} onLayout={registerSection('languages')}>
          <Text style={styles.sectionTitle}>Supported Languages</Text>
          <Text style={styles.sectionSubtitle}>Connect with people across 10 major languages</Text>
          <View style={styles.languagesGrid}>
            {LANGUAGES.map((lang, i) => (
              <View key={i} style={styles.languageCard}>
                <Text style={styles.languageFlag}>{lang.flag}</Text>
                <Text style={styles.languageName}>{lang.name}</Text>
                <Text style={styles.languageNative}>{lang.nativeName}</Text>
              </View>
            ))}
          </View>
        </View>

        {/* Pricing Section */}
        <View style={[styles.section, styles.sectionPlain]} onLayout={registerSection('pricing')}>
          <Text style={styles.sectionTitle}>Simple Pricing</Text>
          <Text style={styles.sectionSubtitle}>Start free. Upgrade when you're ready for more.</Text>

          <View style={styles.planCard}>
            <Text style={styles.planName}>Free</Text>
            <Text style={styles.planPrice}>
              $0<Text style={styles.planPer}>/forever</Text>
            </Text>
            {['Unlimited chats & groups', 'Live translation up to 200 characters', 'On-demand grammar & vocabulary tools', 'Search across all messages'].map((f, i) => (
              <Text key={i} style={styles.planFeature}>✓ {f}</Text>
            ))}
            <View style={[styles.planButton, styles.planButtonDisabled]}>
              <Text style={styles.planButtonDisabledText}>Current Plan</Text>
            </View>
          </View>

          <View style={styles.planCardPremium}>
            <Text style={styles.planNamePremium}>✦ Premium</Text>
            <Text style={styles.planPricePremium}>
              $79.90<Text style={styles.planPerPremium}>/year</Text>
            </Text>
            <Text style={styles.planPromo}>✦ 2 months free</Text>
            {['Automatic grammar analysis', 'Faster AI responses', 'Messages longer than 200 characters', 'Higher daily quotas'].map((f, i) => (
              <Text key={i} style={styles.planFeaturePremium}>✓ {f}</Text>
            ))}
            <TouchableOpacity
              style={styles.planButtonPremium}
              onPress={() => navigation.navigate('Pricing')}>
              <Text style={styles.planButtonPremiumText}>Get Premium</Text>
            </TouchableOpacity>
          </View>

          <View style={styles.planCard}>
            <Text style={styles.planName}>Enterprise</Text>
            <Text style={styles.planDesc}>Self-hosted or custom deployment for teams.</Text>
            {['Self-hosting & custom domains', 'Dedicated support', 'Volume & SLA options'].map((f, i) => (
              <Text key={i} style={styles.planFeature}>✓ {f}</Text>
            ))}
            <TouchableOpacity
              style={styles.planButtonOutline}
              onPress={() => Linking.openURL('mailto:hello@chorus.talk?subject=Enterprise%20Enquiry')}>
              <Text style={styles.planButtonOutlineText}>Contact Us</Text>
            </TouchableOpacity>
          </View>

          <Text style={styles.pricingNote}>Free forever plan — no credit card required.</Text>
        </View>

        {/* CTA Section */}
        <View style={styles.ctaSection}>
          <Text style={styles.ctaTitle}>Ready to Break Language Barriers?</Text>
          <Text style={styles.ctaSubtitle}>Join Chorus today and start connecting with people worldwide</Text>
          <TouchableOpacity
            style={styles.ctaButton}
            onPress={() => navigation.navigate('Register')}>
            <Text style={styles.ctaButtonText}>Get Started Now</Text>
          </TouchableOpacity>
        </View>

        {/* Footer */}
        <View style={styles.footer}>
          <Text style={styles.footerTagline}>
            Break language barriers and connect with people worldwide through real-time translation.
          </Text>
          <View style={styles.footerLinks}>
            <TouchableOpacity onPress={() => navigation.navigate('Pricing')}>
              <Text style={styles.footerLink}>Pricing</Text>
            </TouchableOpacity>
            <TouchableOpacity onPress={() => navigation.navigate('About')}>
              <Text style={styles.footerLink}>About</Text>
            </TouchableOpacity>
            <TouchableOpacity onPress={() => navigation.navigate('Login')}>
              <Text style={styles.footerLink}>Log In</Text>
            </TouchableOpacity>
          </View>
          <Text style={styles.footerRights}>© 2026 Chorus. All rights reserved.</Text>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: COLOR.surface,
  },
  nav: {
    backgroundColor: COLOR.surface,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 8,
  },
  navRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  brand: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  brandLogo: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(37,99,235,0.12)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  brandLogoText: {
    fontSize: 18,
  },
  brandName: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLOR.primary,
  },
  loginButton: {
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: RADIUS.lg,
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  loginButtonText: {
    color: COLOR.onSurface,
    fontSize: 14,
    fontWeight: '600',
  },
  navLinks: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    marginTop: 10,
  },
  navLink: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
    fontWeight: '500',
  },
  scroll: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  scrollContent: {
    paddingBottom: 24,
  },
  section: {
    paddingHorizontal: 20,
    paddingVertical: 40,
  },
  heroSection: {
    backgroundColor: COLOR.background,
  },
  altSection: {
    backgroundColor: COLOR.surfaceContainerLow,
  },
  sectionPlain: {
    backgroundColor: COLOR.surfaceContainerLowest,
  },
  badge: {
    alignSelf: 'flex-start',
    backgroundColor: COLOR.primaryFixed,
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 6,
    marginBottom: 16,
  },
  badgeText: {
    color: COLOR.primary,
    fontSize: 13,
    fontWeight: '600',
  },
  heroTitle: {
    fontSize: 36,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    marginBottom: 16,
  },
  heroTitleAccent: {
    color: COLOR.secondary,
  },
  heroSubtitle: {
    fontSize: 17,
    lineHeight: 26,
    color: COLOR.onSurfaceVariant,
    marginBottom: 24,
  },
  heroButtons: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
    marginBottom: 28,
  },
  primaryButton: {
    backgroundColor: COLOR.primary,
    borderRadius: RADIUS.xl,
    paddingHorizontal: 24,
    paddingVertical: 14,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 3,
  },
  primaryButtonText: {
    color: COLOR.onPrimary,
    fontSize: 16,
    fontWeight: '600',
  },
  secondaryButton: {
    borderWidth: 2,
    borderColor: COLOR.outlineVariant,
    borderRadius: RADIUS.xl,
    paddingHorizontal: 24,
    paddingVertical: 12,
  },
  secondaryButtonText: {
    color: COLOR.onSurfaceVariant,
    fontSize: 16,
    fontWeight: '600',
  },
  stats: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 28,
  },
  stat: {
    flex: 1,
  },
  statValue: {
    fontSize: 24,
    fontWeight: 'bold',
    color: COLOR.primary,
  },
  statLabel: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
    marginTop: 2,
  },
  heroVisual: {
    borderRadius: 32,
    overflow: 'hidden',
    backgroundColor: COLOR.surfaceContainer,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.1,
    shadowRadius: 24,
    elevation: 4,
  },
  heroVisualBg: {
    height: 240,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    backgroundColor: COLOR.surfaceContainerLow,
  },
  heroVisualIcon: {
    fontSize: 72,
  },
  heroVisualTag: {
    fontSize: 13,
    fontWeight: '600',
    color: COLOR.primary,
  },
  sparkyCard: {
    backgroundColor: 'rgba(255,255,255,0.92)',
    borderLeftWidth: 2,
    borderLeftColor: COLOR.secondaryContainer,
    borderRadius: 16,
    padding: 16,
    margin: 16,
    marginTop: 0,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.05,
    shadowRadius: 12,
    elevation: 2,
  },
  sparkyHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginBottom: 8,
  },
  sparkyAvatar: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: COLOR.surfaceContainer,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sparkyAvatarText: {
    fontSize: 16,
  },
  sparkyName: {
    fontSize: 13,
    fontWeight: '600',
    color: COLOR.onSurface,
  },
  sparkyMessage: {
    fontSize: 16,
    lineHeight: 24,
    color: COLOR.onSurface,
    marginBottom: 2,
  },
  sparkyTranslation: {
    fontSize: 14,
    lineHeight: 20,
    fontWeight: '500',
    color: COLOR.secondary,
  },
  sectionTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    textAlign: 'center',
    marginBottom: 8,
  },
  sectionSubtitle: {
    fontSize: 16,
    color: COLOR.onSurfaceVariant,
    textAlign: 'center',
    marginBottom: 28,
  },
  featuresGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 14,
  },
  featureCard: {
    width: '48%',
    flexGrow: 1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 16,
    padding: 18,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.08,
    shadowRadius: 6,
    elevation: 2,
  },
  featureIconCircle: {
    width: 48,
    height: 48,
    borderRadius: 24,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 12,
  },
  featureIcon: {
    fontSize: 22,
  },
  featureTitle: {
    fontSize: 17,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    marginBottom: 6,
  },
  featureDesc: {
    fontSize: 14,
    lineHeight: 21,
    color: COLOR.onSurfaceVariant,
  },
  stepsWrap: {
    gap: 24,
  },
  step: {
    alignItems: 'center',
  },
  stepCircle: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: COLOR.primary,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 12,
  },
  stepNum: {
    fontSize: 22,
    fontWeight: 'bold',
    color: COLOR.onPrimary,
  },
  stepTitle: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLOR.onSurface,
    marginBottom: 6,
  },
  stepDesc: {
    fontSize: 14,
    lineHeight: 21,
    color: COLOR.onSurfaceVariant,
    textAlign: 'center',
  },
  languagesGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
  },
  languageCard: {
    width: '30%',
    flexGrow: 1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 12,
    paddingVertical: 16,
    paddingHorizontal: 8,
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 4,
    elevation: 1,
  },
  languageFlag: {
    fontSize: 30,
    marginBottom: 6,
  },
  languageName: {
    fontSize: 14,
    fontWeight: '600',
    color: COLOR.onSurface,
  },
  languageNative: {
    fontSize: 12,
    color: COLOR.onSurfaceVariant,
    marginTop: 2,
  },
  planCard: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 16,
    padding: 22,
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
  planButton: {
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  planButtonDisabled: {
    backgroundColor: COLOR.surfaceContainerLow,
  },
  planButtonDisabledText: {
    color: COLOR.onSurfaceVariant,
    fontSize: 15,
    fontWeight: '600',
  },
  planCardPremium: {
    backgroundColor: COLOR.secondary,
    borderRadius: 16,
    padding: 22,
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
    color: COLOR.onSecondary,
    marginBottom: 4,
  },
  planPricePremium: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLOR.onSecondary,
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
    color: COLOR.onSecondary,
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
    color: COLOR.primary,
    fontSize: 15,
    fontWeight: 'bold',
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
  pricingNote: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
    textAlign: 'center',
    marginTop: 8,
  },
  ctaSection: {
    backgroundColor: COLOR.primary,
    paddingHorizontal: 20,
    paddingVertical: 48,
    alignItems: 'center',
  },
  ctaTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: COLOR.onPrimary,
    textAlign: 'center',
    marginBottom: 8,
  },
  ctaSubtitle: {
    fontSize: 16,
    color: 'rgba(255,255,255,0.9)',
    textAlign: 'center',
    marginBottom: 24,
  },
  ctaButton: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 10,
    paddingHorizontal: 28,
    paddingVertical: 14,
  },
  ctaButtonText: {
    color: COLOR.primary,
    fontSize: 16,
    fontWeight: 'bold',
  },
  footer: {
    backgroundColor: COLOR.inverseSurface,
    paddingHorizontal: 20,
    paddingVertical: 32,
  },
  footerTagline: {
    fontSize: 14,
    lineHeight: 21,
    color: COLOR.inverseOnSurface,
    marginBottom: 16,
  },
  footerLinks: {
    flexDirection: 'row',
    gap: 20,
    marginBottom: 16,
  },
  footerLink: {
    fontSize: 14,
    color: COLOR.primaryFixed,
    fontWeight: '500',
  },
  footerRights: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
  },
});