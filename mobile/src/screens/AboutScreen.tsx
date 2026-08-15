import React from 'react';
import { View, Text, ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { COLORS } from '../theme';

// Mirrors the web frontend's About page (frontend/src/pages/About.tsx).
const ABOUT_FEATURES = [
  {
    icon: '🌐',
    title: 'Instant Translation',
    desc: 'Messages are translated in real-time so everyone reads in their own language.',
  },
  {
    icon: '✏️',
    title: 'Grammar Analysis',
    desc: 'AI-powered grammar feedback and CEFR difficulty ratings while you chat.',
  },
  {
    icon: '📚',
    title: 'Vocabulary Builder',
    desc: 'Spaced repetition helps you remember the words you meet in conversations.',
  },
  {
    icon: '👥',
    title: 'Group Chats',
    desc: 'Multilingual groups where each participant reads in their own language.',
  },
  {
    icon: '🔒',
    title: 'Privacy First',
    desc: 'Conversations are encrypted and secure. Messages are not stored permanently.',
  },
];

const LANGUAGES = [
  { flag: '🇬🇧', name: 'English' },
  { flag: '🇪🇸', name: 'Spanish' },
  { flag: '🇫🇷', name: 'French' },
  { flag: '🇩🇪', name: 'German' },
  { flag: '🇮🇹', name: 'Italian' },
  { flag: '🇵🇹', name: 'Portuguese' },
  { flag: '🇯🇵', name: 'Japanese' },
  { flag: '🇰🇷', name: 'Korean' },
  { flag: '🇨🇳', name: 'Chinese' },
];

export default function AboutScreen() {
  return (
    <SafeAreaView style={styles.safeArea} edges={['left', 'right']}>
      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}>
        <View style={styles.header}>
          <View style={styles.logo}>
            <Text style={styles.logoText}>💬</Text>
          </View>
          <Text style={styles.title}>About Chorus</Text>
          <Text style={styles.subtitle}>
            Break language barriers and connect with people worldwide.
          </Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardTitle}>Our Mission</Text>
          <Text style={styles.cardText}>
            Chorus makes real-time translation feel invisible, so people who speak
            different languages can chat naturally. Everyone writes in their own
            language and reads in the one they understand best.
          </Text>
          <Text style={styles.cardText}>
            Along the way, Chorus doubles as a language-learning companion — with
            grammar analysis, vocabulary tools, and on-demand translations built
            into the conversations you're already having.
          </Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardTitle}>Features</Text>
          {ABOUT_FEATURES.map((feature, i) => (
            <View key={i} style={styles.featureRow}>
              <Text style={styles.featureIcon}>{feature.icon}</Text>
              <View style={styles.featureText}>
                <Text style={styles.featureTitle}>{feature.title}</Text>
                <Text style={styles.featureDesc}>{feature.desc}</Text>
              </View>
            </View>
          ))}
        </View>

        <View style={styles.card}>
          <Text style={styles.cardTitle}>Supported Languages</Text>
          <View style={styles.languagesGrid}>
            {LANGUAGES.map((lang, i) => (
              <View key={i} style={styles.languageCell}>
                <Text style={styles.languageFlag}>{lang.flag}</Text>
                <Text style={styles.languageName}>{lang.name}</Text>
              </View>
            ))}
          </View>
        </View>

        <View style={styles.footer}>
          <Text style={styles.footerText}>Chorus Mobile · Version 0.0.1</Text>
          <Text style={styles.footerText}>© 2026 Chorus. All rights reserved.</Text>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: COLORS.bgLight,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    paddingHorizontal: 20,
    paddingVertical: 32,
    paddingBottom: 40,
  },
  header: {
    alignItems: 'center',
    marginBottom: 28,
  },
  logo: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: COLORS.primary,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 16,
  },
  logoText: {
    fontSize: 36,
  },
  title: {
    fontSize: 32,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: COLORS.textGray,
    textAlign: 'center',
  },
  card: {
    backgroundColor: COLORS.white,
    borderWidth: 1,
    borderColor: COLORS.borderGray,
    borderRadius: 16,
    padding: 20,
    marginBottom: 16,
  },
  cardTitle: {
    fontSize: 20,
    fontWeight: 'bold',
    color: COLORS.textDark,
    marginBottom: 12,
  },
  cardText: {
    fontSize: 14,
    lineHeight: 22,
    color: COLORS.textGray,
    marginBottom: 12,
  },
  featureRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginBottom: 16,
  },
  featureIcon: {
    fontSize: 22,
    marginRight: 12,
    marginTop: 2,
  },
  featureText: {
    flex: 1,
  },
  featureTitle: {
    fontSize: 15,
    fontWeight: '600',
    color: COLORS.textDark,
    marginBottom: 2,
  },
  featureDesc: {
    fontSize: 13,
    lineHeight: 20,
    color: COLORS.textGray,
  },
  languagesGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 10,
  },
  languageCell: {
    width: '30%',
    flexGrow: 1,
    backgroundColor: '#F9FAFB',
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
  },
  languageFlag: {
    fontSize: 24,
    marginBottom: 4,
  },
  languageName: {
    fontSize: 13,
    fontWeight: '500',
    color: COLORS.textDark,
  },
  footer: {
    alignItems: 'center',
    marginTop: 8,
  },
  footerText: {
    fontSize: 13,
    color: COLORS.textGray,
    marginBottom: 4,
  },
});
