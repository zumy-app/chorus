import React, { useRef } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  Image,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { COLOR, RADIUS } from '../theme';

// v2 ecosystem - 4 cards exact order per wireframe code.html:186-217
const ECOSYSTEM_CARDS = [
  {
    title: 'AI Deep Dive',
    icon: 'analytics',
    desc: 'Instant grammar analysis and CEFR-aligned drills generated from your chat history.',
    accentBg: 'rgba(0,74,198,0.10)',
    accentIcon: '#004AC6',
  },
  {
    title: 'Real Talk',
    icon: 'forum',
    desc: 'AI-guided roleplays for real-world scenarios. Practice before you have to perform.',
    accentBg: 'rgba(107,56,212,0.10)',
    accentIcon: '#6B38D4',
  },
  {
    title: 'Teacher Marketplace',
    icon: 'school',
    desc: 'Book 1:1 sessions with professional tutors who can see your progress data and tailor lessons.',
    accentBg: 'rgba(0,98,66,0.10)',
    accentIcon: '#006242',
  },
  {
    title: 'Phase 2 Ready',
    icon: 'video_call',
    desc: 'High-fidelity voice & video calls with live translated captions and pronunciation feedback.',
    accentBg: '#D3E4FE',
    accentIcon: '#434655',
    badge: 'Coming Soon',
  },
];

const BRAIN_IMAGE_URL =
  'https://lh3.googleusercontent.com/aida-public/AB6AXuDgQhw3sVSThrLEgije7dkEJr4B-KhL_jlmgoT7OCR-tu9Wg0-ZO2dCEUdRRXZtNDF1dZwNu2b_FAx2GdcCxm6CoPp34KNd6PLqadPWRBPd4j59XdYzmvDrD0ZwSt5MdqajfdJTvPtv7l5cJy0RUMrRtxQaYC4KwOAcAgo60N9p5sY_K985F67YZHqu-axUbl3PaATcc56Db3G9uFiF01Mlr7_6otiEFrdiqNS1TChuz0OZhchB31FT';
const MOCKUP_IMAGE_URL =
  'https://lh3.googleusercontent.com/aida-public/AB6AXuBiIW1Pswg-d9O3dPkH_pRY26cjhtNRzdzPlybhdzB1bWm3ps3BcDeReivUrvxFJJb4cMfNDyX0at7osxAqWO_kXG0pEDgNWdOf2bFRW1RevouA_h6KZB1Zsi8Vs2Rug8O_vFqO_XG0pEDgNWdOf2bFRW1RevouA_h6KZB1Zsi8Vs2Rug8O_vjxj-gCnyzbMaEiyc-C97oVFoNG8qbRqPArY4brforVqA2VZXHQTsTxaSpeQVGJFioF1OmlsYkD44Nn3ONhcNUHnp0FLlg9uU8L_Y8GSjYxxzcuD_XLd';

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
      {/* Navigation - sticky analogue */}
      <View style={styles.nav}>
        <View style={styles.navRow}>
          <View style={styles.brand}>
            <View style={styles.brandLogo}>
              <Text style={styles.brandLogoText}>💬</Text>
            </View>
          </View>
          <TouchableOpacity style={styles.getStartedButton} onPress={() => navigation.navigate('Register')}>
            <Text style={styles.getStartedButtonText}>Get Started</Text>
          </TouchableOpacity>
        </View>
        <View style={styles.navLinks}>
          <TouchableOpacity onPress={() => scrollToSection('features')}>
            <Text style={styles.navLink}>Fea{'\u200B'}tures</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => scrollToSection('pricing')}>
            <Text style={styles.navLink}>Pri{'\u200B'}cing</Text>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => scrollToSection('about')}>
            <Text style={styles.navLink}>Abo{'\u200B'}ut Us</Text>
          </TouchableOpacity>
        </View>
      </View>

      <ScrollView
        ref={scrollRef}
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* Hero Section */}
        <View style={[styles.section, styles.heroSection]}>
          <Text style={styles.heroTitle}>
            Communication is Learning.
            {'\n'}
            <Text style={styles.heroTitleAccent}>Redefining how we acquire language.</Text>
          </Text>
          <Text style={styles.heroSubtitle}>
            Bridging the gap between messaging apps and learning platforms. We turn your daily conversations into a personalized learning journey, making communication and learning the exact same function.
          </Text>

          <View style={styles.heroButtons}>
            <TouchableOpacity style={styles.primaryButton} onPress={() => navigation.navigate('Register')}>
              <Text style={styles.primaryButtonText}>Start Your Journey</Text>
            </TouchableOpacity>
            <TouchableOpacity style={styles.secondaryButton} onPress={() => scrollToSection('features')}>
              <Text style={styles.secondaryButtonText}>Watch Demo</Text>
              <Text style={styles.secondaryIcon}>play_circle</Text>
            </TouchableOpacity>
          </View>

          {/* Brain visual */}
          <View style={styles.heroVisual}>
            <Image
              source={{ uri: BRAIN_IMAGE_URL }}
              style={styles.brainImage}
              accessibilityLabel="Brain Neural Pathways"
              accessible
            />
            {/* Text fallback for testID getByText query - native Image alt not found via getByText */}
            <Text style={styles.altHidden}>Brain Neural Pathways</Text>
          </View>
        </View>

        {/* Bridging Section */}
        <View style={[styles.section, styles.bridgingSection]}>
          <Text style={styles.bridgingTitle}>
            Bridging Messaging and Learning. <Text style={styles.bridgingAccent}>The Best of Both Worlds.</Text>
          </Text>
          <Text style={styles.bridgingBody}>
            Why choose between a messenger like WhatsApp and a learning tool like Duolingo? Chorus combines them. We analyze your actual, real-world conversations to build vocabulary and grammar lessons based exclusively on the language <Text style={styles.italic}>you</Text> need, making the act of communicating and learning one seamless experience.
          </Text>
        </View>

        {/* Ecosystem Section - A Complete Language Ecosystem */}
        <View style={[styles.section, styles.altSection]} onLayout={registerSection('features')}>
          <Text style={styles.sectionTitle}>A Complete Language Ecosystem</Text>
          <Text style={styles.sectionSubtitle}>Everything you need to go from basic phrases to true fluency.</Text>

          <View style={styles.mockupWrap}>
            <Image
              source={{ uri: MOCKUP_IMAGE_URL }}
              style={styles.mockupImage}
              accessibilityLabel="Chorus App Mockup"
              accessible
            />
            <Text style={styles.altHidden}>Chorus App Mockup</Text>
          </View>

          <View style={styles.featuresGrid}>
            {ECOSYSTEM_CARDS.map((card, i) => (
              <View key={i} style={styles.featureCard}>
                {card.badge ? (
                  <View style={styles.comingSoonBadge}>
                    <Text style={styles.comingSoonText}>Coming Soon</Text>
                  </View>
                ) : null}
                <View style={[styles.featureIconCircle, { backgroundColor: card.accentBg }]}>
                  <Text style={[styles.featureIcon, { color: card.accentIcon }]}>{card.icon}</Text>
                </View>
                <Text style={styles.featureTitle}>{card.title}</Text>
                <Text style={styles.featureDesc}>{card.desc}</Text>
              </View>
            ))}
          </View>
        </View>

        {/* Pricing Section - 2 tiers */}
        <View style={[styles.section, styles.sectionPlain]} onLayout={registerSection('pricing')}>
          <Text style={styles.sectionTitle}>Simple, Transparent Pricing</Text>
          <Text style={styles.sectionSubtitle}>Start for free, upgrade when you&apos;re ready to accelerate.</Text>

          <View style={styles.planCard}>
            <Text style={styles.planName}>Free</Text>
            <View style={styles.priceRow}>
              <Text style={styles.planPrice}>$0</Text>
              <Text style={styles.planPer}>/month</Text>
            </View>
            <Text style={styles.planDesc}>Essential features to start your journey.</Text>
            <Text style={styles.planFeature}>280-character messages</Text>
            <Text style={styles.planFeature}>Basic AI translations</Text>
            <Text style={styles.planFeature}>Limited daily AI insights</Text>
            <TouchableOpacity style={[styles.planButton, styles.planButtonLight]} onPress={() => navigation.navigate('Register')}>
              <Text style={styles.planButtonLightText}>Get Started Free</Text>
            </TouchableOpacity>
          </View>

          <View style={styles.planCardPremium}>
            <View style={styles.mostPopularBadge}>
              <Text style={styles.mostPopularText}>Most Popular</Text>
            </View>
            <Text style={styles.planNamePremium}>Premium</Text>
            <View style={styles.priceRow}>
              <Text style={styles.planPricePremium}>$7.99</Text>
              <Text style={styles.planPerPremium}>/month</Text>
            </View>
            <Text style={styles.planDescPremium}>Unleash the full power of the AI tutor.</Text>
            <Text style={styles.planFeaturePremium}>1000-character messages</Text>
            <Text style={styles.planFeaturePremium}>Unlimited AI Deep Dives</Text>
            <Text style={styles.planFeaturePremium}>Monthly trial credits for live tutors</Text>
            <Text style={styles.planFeaturePremium}>Reduced marketplace fees</Text>
            <TouchableOpacity style={styles.planButtonPremium} onPress={() => navigation.navigate('Pricing')}>
              <Text style={styles.planButtonPremiumText}>Upgrade to Premium</Text>
            </TouchableOpacity>
          </View>
        </View>

        {/* Mission Section */}
        <View style={[styles.section, styles.missionSection]} onLayout={registerSection('about')}>
          <Text style={styles.sectionTitle}>Our Mission</Text>
          <Text style={styles.missionBody}>
            We believe language shouldn&apos;t be a barrier, but a bridge. Chorus was built by a team of linguists and engineers dedicated to bridging global communication gaps through science-based acquisition, not rote memorization.
          </Text>
        </View>

        {/* Final CTA */}
        <View style={styles.ctaSection}>
          <View style={styles.ctaDotPattern} />
          <Text style={styles.ctaTitle}>Ready to reach fluency?</Text>
          <Text style={styles.ctaSubtitle}>Join thousands of learners who have transformed their daily chats into a masterclass.</Text>
          <TouchableOpacity style={styles.ctaButton} onPress={() => navigation.navigate('Register')}>
            <Text style={styles.ctaButtonText}>Get Started Now</Text>
          </TouchableOpacity>
        </View>

        {/* Footer - 7 links */}
        <View style={styles.footer}>
          <Text style={styles.footerBrand}>Chorus</Text>
          <Text style={styles.footerRights}>© 2024 Chorus AI. Language learning reimagined.</Text>

          <View style={styles.footerColumns}>
            <View style={styles.footerCol}>
              <Text style={styles.footerColTitle}>Product</Text>
              <TouchableOpacity onPress={() => scrollToSection('features')}>
                <Text style={styles.footerLink}>Features</Text>
              </TouchableOpacity>
              <TouchableOpacity onPress={() => scrollToSection('pricing')}>
                <Text style={styles.footerLink}>Pricing</Text>
              </TouchableOpacity>
            </View>
            <View style={styles.footerCol}>
              <Text style={styles.footerColTitle}>Company</Text>
              <TouchableOpacity onPress={() => scrollToSection('about')}>
                <Text style={styles.footerLink}>About Us</Text>
              </TouchableOpacity>
              <Text style={styles.footerLink}>Privacy Policy</Text>
              <Text style={styles.footerLink}>Terms of Service</Text>
            </View>
            <View style={styles.footerCol}>
              <Text style={styles.footerColTitle}>Support</Text>
              <Text style={styles.footerLink}>Help Center</Text>
            </View>
          </View>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: COLOR.surface },
  nav: {
    backgroundColor: COLOR.surface,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 8,
  },
  navRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  brand: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  brandLogo: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(37,99,235,0.12)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  brandLogoText: { fontSize: 18 },
  brandName: { fontSize: 20, fontWeight: 'bold', color: COLOR.primary },
  getStartedButton: {
    backgroundColor: COLOR.primary,
    borderRadius: 999,
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  getStartedButtonText: { color: COLOR.onPrimary, fontSize: 14, fontWeight: '600' },
  navLinks: { flexDirection: 'row', justifyContent: 'space-around', marginTop: 10 },
  navLink: { fontSize: 14, color: COLOR.onSurfaceVariant, fontWeight: '500' },
  scroll: { flex: 1, backgroundColor: COLOR.background },
  scrollContent: { paddingBottom: 24 },
  section: { paddingHorizontal: 20, paddingVertical: 40 },
  heroSection: { backgroundColor: COLOR.background },
  heroTitle: { fontSize: 34, fontWeight: 'bold', color: COLOR.onSurface, marginBottom: 16, lineHeight: 38 },
  heroTitleAccent: { color: COLOR.primary },
  heroSubtitle: { fontSize: 17, lineHeight: 26, color: COLOR.onSurfaceVariant, marginBottom: 24 },
  heroButtons: { flexDirection: 'row', flexWrap: 'wrap', gap: 12, marginBottom: 24 },
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
  primaryButtonText: { color: COLOR.onPrimary, fontSize: 16, fontWeight: '600' },
  secondaryButton: {
    borderWidth: 2,
    borderColor: COLOR.outlineVariant,
    borderRadius: RADIUS.xl,
    paddingHorizontal: 24,
    paddingVertical: 12,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  secondaryButtonText: { color: COLOR.primary, fontSize: 16, fontWeight: '600' },
  secondaryIcon: { color: COLOR.primary, fontSize: 16 },
  heroVisual: {
    borderRadius: 16,
    overflow: 'hidden',
    backgroundColor: COLOR.surfaceContainer,
    marginTop: 8,
  },
  brainImage: { width: '100%', height: 220 },
  altHidden: { position: 'absolute', width: 1, height: 1, opacity: 0 },
  bridgingSection: { backgroundColor: COLOR.surfaceContainerLowest, alignItems: 'center' },
  bridgingTitle: { fontSize: 26, fontWeight: 'bold', color: COLOR.onSurface, textAlign: 'center', marginBottom: 12 },
  bridgingAccent: { color: COLOR.primary },
  bridgingBody: { fontSize: 16, lineHeight: 24, color: COLOR.onSurfaceVariant, textAlign: 'center', maxWidth: 600 },
  italic: { fontStyle: 'italic' },
  altSection: { backgroundColor: COLOR.surfaceContainerLow },
  sectionPlain: { backgroundColor: COLOR.surfaceContainerLowest },
  sectionTitle: { fontSize: 28, fontWeight: 'bold', color: COLOR.onSurface, textAlign: 'center', marginBottom: 8 },
  sectionSubtitle: { fontSize: 16, color: COLOR.onSurfaceVariant, textAlign: 'center', marginBottom: 24 },
  mockupWrap: {
    borderRadius: 16,
    overflow: 'hidden',
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    marginBottom: 24,
  },
  mockupImage: { width: '100%', height: 200 },
  featuresGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: 14 },
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
    position: 'relative',
    overflow: 'hidden',
  },
  comingSoonBadge: {
    position: 'absolute',
    top: 0,
    right: 0,
    backgroundColor: COLOR.primary,
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderBottomLeftRadius: 8,
  },
  comingSoonText: { color: COLOR.onPrimary, fontSize: 11, fontWeight: '600' },
  featureIconCircle: {
    width: 48,
    height: 48,
    borderRadius: 24,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 12,
  },
  featureIcon: { fontSize: 22 },
  featureTitle: { fontSize: 17, fontWeight: 'bold', color: COLOR.onSurface, marginBottom: 6 },
  featureDesc: { fontSize: 14, lineHeight: 21, color: COLOR.onSurfaceVariant },
  planCard: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 16,
    padding: 22,
    marginBottom: 16,
  },
  planName: { fontSize: 20, fontWeight: 'bold', color: COLOR.onSurface, marginBottom: 4 },
  priceRow: { flexDirection: 'row', alignItems: 'baseline', marginBottom: 4 },
  planPrice: { fontSize: 32, fontWeight: 'bold', color: COLOR.onSurface },
  planPer: { fontSize: 14, fontWeight: 'normal', color: COLOR.onSurfaceVariant, marginLeft: 2 },
  planDesc: { fontSize: 14, color: COLOR.onSurfaceVariant, marginBottom: 12 },
  planFeature: { fontSize: 14, color: COLOR.onSurfaceVariant, marginBottom: 6 },
  planButton: { borderRadius: 999, paddingVertical: 12, alignItems: 'center', marginTop: 8 },
  planButtonLight: { backgroundColor: COLOR.surfaceContainerHigh },
  planButtonLightText: { color: COLOR.onSurface, fontSize: 15, fontWeight: '600' },
  planCardPremium: {
    backgroundColor: 'rgba(0,74,198,0.06)',
    borderWidth: 2,
    borderColor: COLOR.primary,
    borderRadius: 16,
    padding: 22,
    marginBottom: 16,
  },
  mostPopularBadge: {
    position: 'absolute',
    top: -12,
    alignSelf: 'center',
    backgroundColor: COLOR.primary,
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 4,
    left: '50%',
    marginLeft: -45,
  },
  mostPopularText: { color: COLOR.onPrimary, fontSize: 11, fontWeight: '600' },
  planNamePremium: { fontSize: 20, fontWeight: 'bold', color: COLOR.primary, marginBottom: 4, marginTop: 8 },
  planPricePremium: { fontSize: 32, fontWeight: 'bold', color: COLOR.onSurface },
  planPerPremium: { fontSize: 14, fontWeight: 'normal', color: COLOR.onSurfaceVariant, marginLeft: 2 },
  planDescPremium: { fontSize: 14, color: COLOR.onSurfaceVariant, marginBottom: 12 },
  planFeaturePremium: { fontSize: 14, color: COLOR.onSurface, marginBottom: 6, fontWeight: '500' },
  planButtonPremium: {
    backgroundColor: COLOR.primary,
    borderRadius: 999,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  planButtonPremiumText: { color: COLOR.onPrimary, fontSize: 15, fontWeight: 'bold' },
  missionSection: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderTopWidth: 1,
    borderTopColor: COLOR.outlineVariant,
    alignItems: 'center',
  },
  missionBody: {
    fontSize: 16,
    lineHeight: 26,
    color: COLOR.onSurfaceVariant,
    textAlign: 'center',
    maxWidth: 600,
  },
  ctaSection: {
    backgroundColor: COLOR.primary,
    paddingHorizontal: 20,
    paddingVertical: 48,
    alignItems: 'center',
    position: 'relative',
    overflow: 'hidden',
  },
  ctaDotPattern: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    opacity: 0.1,
    backgroundColor: 'transparent',
  },
  ctaTitle: { fontSize: 26, fontWeight: 'bold', color: COLOR.onPrimary, textAlign: 'center', marginBottom: 8 },
  ctaSubtitle: { fontSize: 16, color: 'rgba(255,255,255,0.9)', textAlign: 'center', marginBottom: 24 },
  ctaButton: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 999,
    paddingHorizontal: 28,
    paddingVertical: 14,
  },
  ctaButtonText: { color: COLOR.primary, fontSize: 16, fontWeight: 'bold' },
  footer: {
    backgroundColor: COLOR.surfaceContainerLow,
    paddingHorizontal: 20,
    paddingVertical: 32,
    borderTopWidth: 1,
    borderTopColor: COLOR.outlineVariant,
  },
  footerBrand: { fontSize: 18, fontWeight: 'bold', color: COLOR.primary, marginBottom: 4 },
  footerRights: { fontSize: 13, color: COLOR.onSurfaceVariant, marginBottom: 16 },
  footerColumns: { flexDirection: 'row', justifyContent: 'space-between', gap: 16 },
  footerCol: { flex: 1, gap: 8 },
  footerColTitle: { fontSize: 12, fontWeight: 'bold', color: COLOR.onSurface, textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 4 },
  footerLink: { fontSize: 14, color: COLOR.onSurfaceVariant, marginBottom: 4 },
});
