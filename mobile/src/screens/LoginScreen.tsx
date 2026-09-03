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
import AuthLayout from '../components/AuthLayout';
import DevAccountSwitcher from '../components/DevAccountSwitcher';
import { COLOR, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

export default function LoginScreen({ navigation }: any) {
  // Local-dev convenience: prefill the test account when
  // EXPO_PUBLIC_TEST_USER_* are present (mobile/.env or the shell env). Falls
  // back to empty fields otherwise.
  const [username, setUsername] = useState(process.env.EXPO_PUBLIC_TEST_USER_EMAIL || '');
  const [password, setPassword] = useState(process.env.EXPO_PUBLIC_TEST_USER_PASSWORD || '');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);

  const [requires2FA, setRequires2FA] = useState(false);
  const [tempToken, setTempToken] = useState('');
  const [phoneMasked, setPhoneMasked] = useState('');
  const [code, setCode] = useState('');

  const handleLogin = async () => {
    if (!username.trim() || !password) {
      Alert.alert('Error', 'Please enter username and password');
      return;
    }
    setLoading(true);
    try {
      const raw: any = await (apiService as any).api?.post?.('/auth/login', { username: username.trim(), password }) ?? await (await import('../services/api')).api.post('/auth/login', { username: username.trim(), password });
      if (raw.data?.requires2FA) {
        setTempToken(raw.data.tempToken);
        setPhoneMasked(raw.data.phoneMasked || '');
        setRequires2FA(true);
        return;
      }
      const response = raw.data.tokens ? raw.data : await apiService.login(username.trim(), password);
      if (response.tokens) {
        await storage.setItem('accessToken', response.tokens.accessToken);
        await storage.setItem('refreshToken', response.tokens.refreshToken);
        await storage.setItem('user', JSON.stringify(response.user));
        navigation.replace('MainTabs');
      }
    } catch (error: any) {
      Alert.alert('Login Failed', error.response?.data?.error || 'Invalid credentials. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleVerify2FA = async () => {
    if (code.length !== 6) { Alert.alert('Error','Enter 6-digit code'); return; }
    setLoading(true);
    try {
      const r: any = await (apiService as any).verify2FA(tempToken, code);
      await storage.setItem('accessToken', r.tokens.accessToken);
      await storage.setItem('refreshToken', r.tokens.refreshToken);
      await storage.setItem('user', JSON.stringify(r.user));
      navigation.replace('MainTabs');
    } catch (e: any) { Alert.alert('Failed', e.response?.data?.error || 'Invalid code') }
    finally { setLoading(false) }
  };

  return (
    <AuthLayout tagline="Master new languages through seamless, conversational learning.">
      <View style={styles.card}>
        <DevAccountSwitcher onSelect={({ email, password }) => { setUsername(email); setPassword(password) }} />
        <View style={styles.field}>
          <Text style={styles.label}>Email Address</Text>
          <View style={styles.inputWrap}>
            <Text style={styles.inputIcon}>✉️</Text>
            <TextInput
              style={styles.input}
              placeholder="you@example.com"
              placeholderTextColor={COLOR.outlineVariant}
              value={username}
              onChangeText={setUsername}
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="email-address"
              autoFocus
            />
          </View>
        </View>

        <View style={styles.field}>
          <View style={styles.labelRow}>
            <Text style={styles.label}>Password</Text>
            <TouchableOpacity onPress={() => navigation.navigate('ForgotPassword')}>
              <Text style={styles.link}>Forgot password?</Text>
            </TouchableOpacity>
          </View>
          <View style={styles.inputWrap}>
            <Text style={styles.inputIcon}>🔒</Text>
            <TextInput
              style={styles.input}
              placeholder="Enter your password"
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

        {requires2FA ? (
          <>
            <Text style={{...TYPOGRAPHY.bodySm, color: COLOR.onSurfaceVariant, marginBottom: 8}}>Code sent to {phoneMasked}</Text>
            <View style={styles.inputWrap}>
              <TextInput style={styles.input} placeholder="123456" keyboardType="number-pad" maxLength={6} value={code} onChangeText={setCode} />
            </View>
            <TouchableOpacity style={[styles.button, loading && styles.buttonDisabled]} onPress={handleVerify2FA} disabled={loading}>
              <Text style={styles.buttonText}>Verify</Text>
            </TouchableOpacity>
            <TouchableOpacity onPress={()=>setRequires2FA(false)} style={{alignItems:'center', marginTop:8}}><Text style={styles.link}>Back</Text></TouchableOpacity>
          </>
        ) : (
          <TouchableOpacity
            style={[styles.button, loading && styles.buttonDisabled]}
            onPress={handleLogin}
            disabled={loading}>
            {loading ? (
              <ActivityIndicator color={COLOR.onPrimaryContainer} />
            ) : (
              <Text style={styles.buttonText}>Log In</Text>
            )}
          </TouchableOpacity>
        )}

        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>OR CONTINUE WITH</Text>
          <View style={styles.dividerLine} />
        </View>

        <TouchableOpacity style={styles.googleButton}>
          <Text style={styles.googleButtonText}>G</Text>
          <Text style={styles.googleButtonLabel}>Google</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.appleButton}>
          <Text style={styles.appleButtonText}>Log in with Apple</Text>
        </TouchableOpacity>

        <View style={styles.bottom}>
          <Text style={styles.bottomText}>New here?</Text>
          <TouchableOpacity onPress={() => navigation.navigate('Register')}>
            <Text style={styles.bottomLink}>Create an account</Text>
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
  labelRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-end',
    marginLeft: 4,
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
  visibility: {
    paddingHorizontal: 14,
  },
  visibilityText: {
    fontSize: 18,
  },
  link: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.primary,
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
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginVertical: 24,
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: 'rgba(195,198,215,0.5)',
  },
  dividerText: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.outline,
    letterSpacing: 0.5,
  },
  googleButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 12,
    backgroundColor: COLOR.surface,
    borderWidth: 1,
    borderColor: 'rgba(195,198,215,0.5)',
    borderRadius: RADIUS.xl,
    paddingVertical: 14,
    marginBottom: 12,
  },
  googleButtonText: {
    fontSize: 20,
    fontWeight: '700',
    color: '#4285F4',
  },
  googleButtonLabel: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onSurface,
  },
  appleButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: COLOR.inverseSurface,
    borderRadius: RADIUS.xl,
    paddingVertical: 14,
  },
  appleButtonText: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.inverseOnSurface,
  },
  bottom: {
    alignItems: 'center',
    marginTop: SPACING.stackLg,
    gap: SPACING.stackSm,
  },
  bottomText: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
  },
  bottomLink: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.primary,
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
});
