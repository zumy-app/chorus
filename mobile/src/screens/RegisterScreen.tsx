import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import storage from '../utils/storage';
import apiService from '../services/api';
import { SUPPORTED_LANGUAGES } from '@chorus/shared';
import AuthLayout from '../components/AuthLayout';
import { COLOR, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

export default function RegisterScreen({ navigation }: any) {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [displayName, setDisplayName] = useState('');
  const [nativeLanguage, setNativeLanguage] = useState('en');
  const [loading, setLoading] = useState(false);

  const handleRegister = async () => {
    if (!username.trim() || !email.trim() || !password || !displayName.trim()) {
      Alert.alert('Error', 'Please fill in all fields');
      return;
    }

    if (password.length < 8) {
      Alert.alert('Error', 'Password must be at least 8 characters');
      return;
    }

    setLoading(true);
    try {
      const response = await apiService.register({
        username: username.trim(),
        email: email.trim(),
        password,
        displayName: displayName.trim(),
        nativeLanguage,
        targetLanguages: [],
      });

      // Store auth tokens and user data
      await storage.setItem('accessToken', response.tokens.accessToken);
      await storage.setItem('refreshToken', response.tokens.refreshToken);
      await storage.setItem('user', JSON.stringify(response.user));

      Alert.alert('Success', 'Account created successfully!', [
        { text: 'OK', onPress: () => navigation.replace('MainTabs') },
      ]);
    } catch (error: any) {
      console.error('Registration error:', error);
      Alert.alert(
        'Registration Failed',
        error.response?.data?.error || 'Failed to create account. Please try again.'
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout title="Join Chorus" tagline="Set up your account and start learning.">
      <View style={styles.card}>
        <View style={styles.field}>
          <Text style={styles.label}>Username</Text>
          <TextInput
            style={styles.input}
            placeholder="Choose a username"
            placeholderTextColor={COLOR.outlineVariant}
            value={username}
            onChangeText={setUsername}
            autoCapitalize="none"
            autoCorrect={false}
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Email Address</Text>
          <TextInput
            style={styles.input}
            placeholder="you@example.com"
            placeholderTextColor={COLOR.outlineVariant}
            value={email}
            onChangeText={setEmail}
            keyboardType="email-address"
            autoCapitalize="none"
            autoCorrect={false}
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Display Name</Text>
          <TextInput
            style={styles.input}
            placeholder="How friends see you"
            placeholderTextColor={COLOR.outlineVariant}
            value={displayName}
            onChangeText={setDisplayName}
            autoCorrect={false}
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Password</Text>
          <View style={styles.inputWrap}>
            <Text style={styles.inputIcon}>🔒</Text>
            <TextInput
              style={styles.inputInner}
              placeholder="Min 8 characters"
              placeholderTextColor={COLOR.outlineVariant}
              value={password}
              onChangeText={setPassword}
              secureTextEntry={!showPassword}
              autoCapitalize="none"
            />
            <TouchableOpacity
              style={styles.visibility}
              onPress={() => setShowPassword(!showPassword)}
              accessibilityLabel={showPassword ? 'Hide password' : 'Show password'}>
              <Text style={styles.visibilityText}>{showPassword ? '🙈' : '👁️'}</Text>
            </TouchableOpacity>
          </View>
        </View>

        <View style={styles.languageSection}>
          <Text style={styles.label}>Native Language</Text>
          <View style={styles.languageGrid}>
            {SUPPORTED_LANGUAGES.slice(0, 3).map((lang) => (
              <TouchableOpacity
                key={lang.code}
                style={[
                  styles.languageButton,
                  nativeLanguage === lang.code && styles.languageButtonSelected,
                ]}
                onPress={() => setNativeLanguage(lang.code)}>
                <Text
                  style={[
                    styles.languageButtonText,
                    nativeLanguage === lang.code && styles.languageButtonTextSelected,
                  ]}>
                  {lang.nativeName}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>

        <TouchableOpacity
          style={[styles.button, loading && styles.buttonDisabled]}
          onPress={handleRegister}
          disabled={loading}>
          {loading ? (
            <ActivityIndicator color={COLOR.onPrimaryContainer} />
          ) : (
            <Text style={styles.buttonText}>Create Account</Text>
          )}
        </TouchableOpacity>

        <View style={styles.bottom}>
          <Text style={styles.bottomText}>Already have an account?</Text>
          <TouchableOpacity onPress={() => navigation.navigate('Login')}>
            <Text style={styles.bottomLink}>Sign In</Text>
          </TouchableOpacity>
        </View>
      </View>
    </AuthLayout>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: 24,
    padding: 24,
    borderWidth: 1,
    borderColor: 'rgba(195,198,215,0.3)',
    ...SHADOWS.elevation1,
  },
  field: {
    marginBottom: SPACING.stackMd,
  },
  label: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.onSurfaceVariant,
    marginLeft: 4,
    marginBottom: 6,
  },
  input: {
    backgroundColor: COLOR.surface,
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurface,
    borderRadius: RADIUS.xl,
    paddingVertical: 14,
    paddingHorizontal: 16,
    ...SHADOWS.elevation1,
  },
  inputWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: COLOR.surface,
    borderRadius: RADIUS.xl,
    ...SHADOWS.elevation1,
  },
  inputIcon: {
    fontSize: 18,
    paddingLeft: 14,
  },
  inputInner: {
    flex: 1,
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurface,
    paddingVertical: 14,
    paddingHorizontal: 12,
  },
  visibility: {
    paddingHorizontal: 14,
  },
  visibilityText: {
    fontSize: 18,
  },
  languageSection: {
    marginBottom: SPACING.stackMd,
  },
  languageGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  languageButton: {
    flex: 1,
    minWidth: '30%',
    backgroundColor: COLOR.surface,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: RADIUS.md,
    padding: 12,
    alignItems: 'center',
  },
  languageButtonSelected: {
    backgroundColor: COLOR.primaryContainer,
    borderColor: COLOR.primaryContainer,
  },
  languageButtonText: {
    fontSize: 14,
    color: COLOR.onSurface,
  },
  languageButtonTextSelected: {
    color: COLOR.onPrimaryContainer,
    fontWeight: '600',
  },
  button: {
    backgroundColor: COLOR.primaryContainer,
    borderRadius: RADIUS.xl,
    paddingVertical: 16,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 8,
    ...SHADOWS.elevation2,
  },
  buttonDisabled: {
    opacity: 0.5,
  },
  buttonText: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onPrimaryContainer,
  },
  bottom: {
    alignItems: 'center',
    marginTop: SPACING.stackMd,
    gap: 4,
  },
  bottomText: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
  },
  bottomLink: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.primary,
    padding: 8,
  },
});
