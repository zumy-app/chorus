import React from 'react';
import {
  View,
  Text,
  ScrollView,
  KeyboardAvoidingView,
  Platform,
  StyleSheet,
} from 'react-native';
import { COLOR, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

interface AuthLayoutProps {
  title?: string;
  tagline?: string;
  children: React.ReactNode;
}

export default function AuthLayout({ title = 'Chorus', tagline, children }: AuthLayoutProps) {
  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      style={styles.container}>
      <View style={styles.pattern} />
      <ScrollView contentContainerStyle={styles.scrollContent} keyboardShouldPersistTaps="handled">
        <View style={styles.content}>
          <View style={styles.brand}>
            <View style={styles.logo}>
              <Text style={styles.logoText}>💬</Text>
            </View>
            <Text style={styles.title}>{title}</Text>
            {tagline ? <Text style={styles.subtitle}>{tagline}</Text> : null}
          </View>
          {children}
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  pattern: {
    position: 'absolute',
    top: -120,
    left: '50%',
    width: 480,
    height: 480,
    marginLeft: -240,
    borderRadius: 240,
    backgroundColor: 'rgba(37, 99, 235, 0.08)',
  },
  scrollContent: {
    flexGrow: 1,
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: SPACING.marginMobile,
    paddingVertical: 48,
    maxWidth: 448,
    width: '100%',
    alignSelf: 'center',
  },
  brand: {
    alignItems: 'center',
    marginBottom: 40,
  },
  logo: {
    width: 64,
    height: 64,
    borderRadius: RADIUS.xl,
    backgroundColor: COLOR.primaryContainer,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 24,
    ...SHADOWS.elevation2,
    transform: [{ rotate: '-10deg' }],
  },
  logoText: {
    fontSize: 32,
  },
  title: {
    ...TYPOGRAPHY.headlineLg,
    color: COLOR.primary,
    textAlign: 'center',
    marginBottom: SPACING.stackSm,
  },
  subtitle: {
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurfaceVariant,
    textAlign: 'center',
    paddingHorizontal: SPACING.marginMobile,
  },
});
