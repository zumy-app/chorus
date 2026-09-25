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

interface ResetPasswordScreenProps {
  navigation: any;
  route: { params?: { token?: string } };
}

export default function ResetPasswordScreen({ navigation, route }: ResetPasswordScreenProps) {
  const s = useStrings();
  const token = route?.params?.token || '';
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    if (!token) {
      Alert.alert(t('auth.invalidLinkT'), t('auth.invalidLinkB'));
      return;
    }
    if (password.length < 8) {
      Alert.alert(t('common.error'), t('auth.shortPwB'));
      return;
    }
    if (password !== confirmPassword) {
      Alert.alert(t('common.error'), t('auth.pwMismatchB'));
      return;
    }
    setLoading(true);
    try {
      const response = await apiService.resetPassword(token, password);
      Alert.alert(t('auth.resetDoneT'), response.message, [
        { text: t('common.ok'), onPress: () => navigation.replace('Login') },
      ]);
    } catch (error: any) {
      Alert.alert(t('common.error'), error.response?.data?.error || t('auth.resetExpiredB'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout tagline={s.auth.resetTagline}>
      <View style={styles.card}>
        <View style={styles.field}>
          <Text style={styles.label}>{s.auth.newPw}</Text>
          <View style={styles.inputWrap}>
            <Text style={styles.inputIcon}>🔒</Text>
            <TextInput
              style={styles.input}
              placeholder={s.auth.passwordPhMin}
              placeholderTextColor={COLOR.outlineVariant}
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoCapitalize="none"
              autoFocus
            />
          </View>
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>{s.auth.confirmPw}</Text>
          <View style={styles.inputWrap}>
            <Text style={styles.inputIcon}>🔒</Text>
            <TextInput
              style={styles.input}
              placeholder={s.auth.confirmPwPh}
              placeholderTextColor={COLOR.outlineVariant}
              value={confirmPassword}
              onChangeText={setConfirmPassword}
              secureTextEntry
              autoCapitalize="none"
            />
          </View>
        </View>

        <TouchableOpacity
          style={[styles.button, (loading || !token) && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={loading}>
          {loading ? (
            <ActivityIndicator color={COLOR.onPrimaryContainer} />
          ) : (
            <Text style={styles.buttonText}>{s.auth.resetBtn}</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity style={styles.backLink} onPress={() => navigation.replace('Login')}>
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
