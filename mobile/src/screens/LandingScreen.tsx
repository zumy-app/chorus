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
import { COLORS } from '../theme';

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
      <StatusBar barStyle="dark-content" backgroundColor={COLORS.white} />
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

          {/* Chat preview card */}
          <View style={styles.chatCardWrap}>
            <View style={styles.chatCard}>
              <View style={styles.bubbleIn}>
                <Text style={styles.bubbleText}>¡Hola! ¿Cómo estás?</Text>
                <Text style={styles.bubbleTrans}>Hello! How are you?</Text>
              </View>
              <View style={styles.bubbleOut}>
                <Text style={styles.bubbleTextOut}>I'm great! Learning Spanish</Text>
                <Text style={styles.bubbleTransOut}>¡Estoy genial! Aprendiendo español</Text>
              </View>
              <View style={styles.bubbleIn}>
                <Text style={styles.bubbleText}>Fantástico! 🎉</Text>
                <Text style={styles.bubbleTrans}>Fantastic! 🎉</Text>
              </View>
            </View>
          </View>
        </View>

        {/* Features Section */}
        <View style={[styles.section, styles.altSection]} onLayout={registerSection('features')}>
          <Text style={styles.sectionTitle}>Powerful Features for Global Communication</Text>
          <Text style={styles.sectionSubtitle}>Everything you need to connect with people worldwide</Text>
          <View style={styles.featuresGrid}>
            {FEATURES.map((feature, i) => (
              <View key={i} style={styles.featureCard}>
                <Text style={styles.featureIcon}>{feature.icon}</Text>
                <Text style={styles.featureTitle}>{feature.title}</Text>
                <Text style={styles.featureDesc}>{feature.desc}</Text>
              </View>
            ))}
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
    backgroundColor: COLORS.white,
  },
  nav: {
    backgroundColor: COLORS.white,
    borderBottomWidth: 1,
    borderBottomColor: COLORS.borderGray,
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
    backgroundColor: COLORS.primary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  brandLogoText: {
    fontSize: 18,
  },
  brandName: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLORS.primary,
  },
  loginButton: {
    borderWidth: 1,
    borderColor: COLORS.borderGray,
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  loginButtonText: {
    color: COLORS.textDark,
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
    color: COLORS.textGray,
    fontWeight: '500',
  },
  scroll: {
    flex: 1,
    backgroundColor: COLORS.bgLight,
  },
  scrollContent: {
    paddingBottom: 24,
  },
  section: {
    paddingHorizontal: 20,
    paddingVertical: 40,
  },
  heroSection: {
    backgroundColor: COLORS.bgLight,
  },
  altSection: {
    backgroundColor: COLORS.bgLight,
  },
  sectionPlain: {
    backgroundColor: COLORS.white,
  },
  badge: {
    alignSelf: 'flex-start',
    backgroundColor: '#E0E7FF',
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 6,
    marginBottom: 16,
  },
  badgeText: {
    color: COLORS.primary,
    fontSize: 13,
    fontWeight: '600',
  },
  heroTitle: {
    fontSize: 36,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 16,
  },
  heroTitleAccent: {
    color: COLORS.purple,
  },
  heroSubtitle: {
    fontSize: 17,
    lineHeight: 26,
    color: COLORS.textGray,
    marginBottom: 24,
  },
  heroButtons: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
    marginBottom: 28,
  },
  primaryButton: {
    backgroundColor: COLORS.primary,
    borderRadius: 10,
    paddingHorizontal: 24,
    paddingVertical: 14,
  },
  primaryButtonText: {
    color: COLORS.white,
    fontSize: 16,
    fontWeight: '600',
  },
  secondaryButton: {
    borderWidth: 2,
    borderColor: COLORS.borderGray,
    borderRadius: 10,
    paddingHorizontal: 24,
    paddingVertical: 12,
  },
  secondaryButtonText: {
    color: COLORS.textGray,
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
    color: COLORS.primary,
  },
  statLabel: {
    fontSize: 13,
    color: COLORS.textGray,
    marginTop: 2,
  },
  chatCardWrap: {
    borderRadius: 20,
    padding: 2,
    backgroundColor: COLORS.purple,
  },
  chatCard: {
    backgroundColor: COLORS.white,
    borderRadius: 18,
    padding: 16,
    gap: 12,
  },
  bubbleIn: {
    backgroundColor: '#F3F4F6',
    borderRadius: 10,
    padding: 12,
  },
  bubbleText: {
    fontSize: 14,
    color: COLORS.textGray,
    marginBottom: 2,
  },
  bubbleTrans: {
    fontSize: 12,
    color: '#9CA3AF',
  },
  bubbleOut: {
    backgroundColor: COLORS.primary,
    borderRadius: 10,
    padding: 12,
    marginLeft: 32,
  },
  bubbleTextOut: {
    fontSize: 14,
    color: COLORS.white,
    marginBottom: 2,
  },
  bubbleTransOut: {
    fontSize: 12,
    color: '#C7D2FE',
  },
  sectionTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: COLORS.textDark,
    textAlign: 'center',
    marginBottom: 8,
  },
  sectionSubtitle: {
    fontSize: 16,
    color: COLORS.textGray,
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
    backgroundColor: COLORS.white,
    borderRadius: 16,
    padding: 18,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.08,
    shadowRadius: 6,
    elevation: 2,
  },
  featureIcon: {
    fontSize: 30,
    marginBottom: 10,
  },
  featureTitle: {
    fontSize: 17,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 6,
  },
  featureDesc: {
    fontSize: 14,
    lineHeight: 21,
    color: COLORS.textGray,
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
    backgroundColor: COLORS.purple,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 12,
  },
  stepNum: {
    fontSize: 22,
    fontWeight: 'bold',
    color: COLORS.white,
  },
  stepTitle: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 6,
  },
  stepDesc: {
    fontSize: 14,
    lineHeight: 21,
    color: COLORS.textGray,
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
    backgroundColor: COLORS.white,
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
    color: COLORS.textDark,
  },
  languageNative: {
    fontSize: 12,
    color: '#9CA3AF',
    marginTop: 2,
  },
  planCard: {
    backgroundColor: COLORS.white,
    borderWidth: 1,
    borderColor: COLORS.borderGray,
    borderRadius: 16,
    padding: 22,
    marginBottom: 16,
  },
  planName: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 4,
  },
  planPrice: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 16,
  },
  planPer: {
    fontSize: 14,
    fontWeight: 'normal',
    color: COLORS.textGray,
  },
  planDesc: {
    fontSize: 14,
    color: COLORS.textGray,
    marginBottom: 16,
  },
  planFeature: {
    fontSize: 14,
    color: COLORS.textGray,
    marginBottom: 8,
  },
  planButton: {
    borderWidth: 1,
    borderColor: COLORS.borderGray,
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  planButtonDisabled: {
    backgroundColor: '#F9FAFB',
  },
  planButtonDisabledText: {
    color: COLORS.textGray,
    fontSize: 15,
    fontWeight: '600',
  },
  planCardPremium: {
    backgroundColor: COLORS.purple,
    borderRadius: 16,
    padding: 22,
    marginBottom: 16,
    shadowColor: COLORS.purple,
    shadowOffset: { width: 0, height: 6 },
    shadowOpacity: 0.3,
    shadowRadius: 12,
    elevation: 6,
  },
  planNamePremium: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLORS.white,
    marginBottom: 4,
  },
  planPricePremium: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLORS.white,
    marginBottom: 4,
  },
  planPerPremium: {
    fontSize: 14,
    fontWeight: 'normal',
    color: '#C7D2FE',
  },
  planPromo: {
    fontSize: 13,
    fontWeight: '600',
    color: '#E0E7FF',
    marginBottom: 16,
  },
  planFeaturePremium: {
    fontSize: 14,
    color: COLORS.white,
    marginBottom: 8,
  },
  planButtonPremium: {
    backgroundColor: COLORS.white,
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
  planButtonOutline: {
    borderWidth: 1,
    borderColor: COLORS.borderGray,
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  planButtonOutlineText: {
    color: COLORS.textDark,
    fontSize: 15,
    fontWeight: '600',
  },
  pricingNote: {
    fontSize: 13,
    color: COLORS.textGray,
    textAlign: 'center',
    marginTop: 8,
  },
  ctaSection: {
    backgroundColor: COLORS.purple,
    paddingHorizontal: 20,
    paddingVertical: 48,
    alignItems: 'center',
  },
  ctaTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: COLORS.white,
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
    backgroundColor: COLORS.white,
    borderRadius: 10,
    paddingHorizontal: 28,
    paddingVertical: 14,
  },
  ctaButtonText: {
    color: COLORS.primary,
    fontSize: 16,
    fontWeight: 'bold',
  },
  footer: {
    backgroundColor: COLORS.footerBg,
    paddingHorizontal: 20,
    paddingVertical: 32,
  },
  footerTagline: {
    fontSize: 14,
    lineHeight: 21,
    color: '#9CA3AF',
    marginBottom: 16,
  },
  footerLinks: {
    flexDirection: 'row',
    gap: 20,
    marginBottom: 16,
  },
  footerLink: {
    fontSize: 14,
    color: '#E0E7FF',
    fontWeight: '500',
  },
  footerRights: {
    fontSize: 13,
    color: '#6B7280',
  },
});
