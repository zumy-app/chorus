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
import apiService from '../services/api';
import { useStrings, t } from '../i18n';
import AuthLayout from '../components/AuthLayout';
import { COLOR, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

export default function ForgotPasswordScreen({ navigation }: any) {
  const s = useStrings();
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  const handleSubmit = async () => {
    if (!email.trim()) {
      Alert.alert(t('common.error'), t('auth.enterEmailB'));
      return;
    }
    setLoading(true);
    try {
      const response = await apiService.forgotPassword(email.trim().toLowerCase());
      setSent(true);
      Alert.alert(t('auth.inboxT'), response.message);
    } catch (error: any) {
      Alert.alert(t('common.error'), error.response?.data?.error || t('auth.genericErrB'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout tagline={s.auth.forgotTagline}>
      <View style={styles.card}>
        <View style={styles.field}>
          <Text style={styles.label}>{s.auth.email}</Text>
          <View style={styles.inputWrap}>
            <Text style={styles.inputIcon}>✉️</Text>
            <TextInput
              style={styles.input}
              placeholder={s.auth.emailPh}
              placeholderTextColor={COLOR.outlineVariant}
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="email-address"
              autoFocus
            />
          </View>
        </View>

        <TouchableOpacity
          style={[styles.button, loading && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={loading}>
          {loading ? (
            <ActivityIndicator color={COLOR.onPrimaryContainer} />
          ) : (
            <Text style={styles.buttonText}>{sent ? s.auth.resendLink : s.auth.sendLink}</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity style={styles.backLink} onPress={() => navigation.goBack()}>
          <Text style={styles.backLinkText}>{s.auth.backToLogin}</Text>
        </TouchableOpacity>
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
  input: {
    flex: 1,
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurface,
    paddingVertical: 14,
    paddingHorizontal: 12,
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
  backLink: {
    alignItems: 'center',
    marginTop: SPACING.stackMd,
    padding: 8,
  },
  backLinkText: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.primary,
  },
});
