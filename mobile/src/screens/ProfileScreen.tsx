import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import storage from '../utils/storage';
import apiService from '../services/api';
import webSocketService from '../services/websocket';
import { SUPPORTED_LANGUAGES, User, type PrivacyVisibility } from '@chorus/shared';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

export default function ProfileScreen({ navigation }: any) {
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [displayName, setDisplayName] = useState('');
  const [nativeLanguage, setNativeLanguage] = useState('en');
  const [targetLanguages, setTargetLanguages] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [lastSeen, setLastSeen] = useState<PrivacyVisibility>('everyone');
  const [profilePhoto, setProfilePhoto] = useState<PrivacyVisibility>('everyone');
  const [contacts, setContacts] = useState<PrivacyVisibility>('everyone');
  const [privacyLoading, setPrivacyLoading] = useState(true);

  useEffect(() => {
    loadUser();
    loadPrivacy();
  }, []);

  const loadPrivacy = async () => {
    try {
      const s = await apiService.getSettings();
      setLastSeen(s.lastSeenVisibility ?? 'everyone');
      setProfilePhoto(s.profilePhotoVisibility ?? 'everyone');
      setContacts(s.contactsVisibility ?? 'everyone');
    } catch {} finally { setPrivacyLoading(false) }
  };

  const updatePrivacy = async (field: 'lastSeenVisibility' | 'profilePhotoVisibility' | 'contactsVisibility', value: PrivacyVisibility) => {
    try {
      const updated = await apiService.updateSettings({ [field]: value } as any);
      setLastSeen(updated.lastSeenVisibility);
      setProfilePhoto(updated.profilePhotoVisibility);
      setContacts(updated.contactsVisibility);
    } catch { Alert.alert('Error', 'Could not save privacy settings.'); }
  };

  const loadUser = async () => {
    const userStr = await storage.getItem('user');
    let user: User | null = null;
    if (userStr) {
      try {
        user = JSON.parse(userStr);
      } catch {
        // Corrupted storage — ignore.
      }
    }
    if (!user) {
      try {
        user = await apiService.getMe();
      } catch {
        // Backend unreachable or unauthorized.
      }
    }
    setCurrentUser(user);
    if (user) {
      setDisplayName(user.displayName || '');
      setNativeLanguage(user.nativeLanguage || 'en');
      setTargetLanguages(user.targetLanguages || []);
    }
  };

  const toggleTargetLanguage = (code: string) => {
    if (code === nativeLanguage) return;
    setTargetLanguages((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]
    );
  };

  const handleSave = async () => {
    if (!displayName.trim()) {
      Alert.alert('Error', 'Display name cannot be empty');
      return;
    }
    setSaving(true);
    try {
      const updated = await apiService.updateProfile({
        displayName: displayName.trim(),
        nativeLanguage,
        targetLanguages,
      });
      await storage.setItem('user', JSON.stringify(updated));
      setCurrentUser(updated);
      Alert.alert('Saved', 'Profile updated successfully');
    } catch {
      Alert.alert('Error', 'Could not save your profile. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = async () => {
    if (loggingOut) return;
    setLoggingOut(true);
    try {
      webSocketService.disconnect();
      await apiService.logout();
      navigation.replace('Landing');
    } catch {
      Alert.alert('Error', 'Could not log out. Please try again.');
      setLoggingOut(false);
    }
  };

  if (!currentUser) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color={COLOR.primary} />
      </View>
    );
  }

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      showsVerticalScrollIndicator={false}>
      <View style={styles.header}>
        <Text style={styles.title}>Profile</Text>
        <Text style={styles.subtitle}>Manage your account and app preferences.</Text>
      </View>

      {/* Account */}
      <View style={styles.card}>
        <Text style={styles.sectionHeader}>Account</Text>
        <View style={styles.fieldRow}>
          <Text style={styles.label}>Display Name</Text>
          <TextInput
            style={styles.input}
            value={displayName}
            onChangeText={setDisplayName}
            autoCorrect={false}
          />
          <Text style={styles.hint}>@{currentUser.username}</Text>
        </View>
        <View style={styles.settingsRow}>
          <Text style={styles.settingsRowIcon}>⭐</Text>
          <Text style={styles.settingsRowText}>Subscription</Text>
          <View style={styles.planBadge}>
            <Text style={styles.planBadgeText}>Free</Text>
          </View>
        </View>
      </View>

      {/* Language */}
      <View style={styles.card}>
        <Text style={styles.sectionHeader}>Language</Text>
        <View style={styles.fieldRow}>
          <Text style={styles.label}>Native Language</Text>
          <View style={styles.languageGrid}>
            {SUPPORTED_LANGUAGES.map((lang) => (
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
        <View style={styles.fieldRow}>
          <Text style={styles.label}>Target Languages (learning)</Text>
          <View style={styles.languageGrid}>
            {SUPPORTED_LANGUAGES.filter((l) => l.code !== nativeLanguage).map((lang) => (
              <TouchableOpacity
                key={lang.code}
                style={[
                  styles.languageButton,
                  targetLanguages.includes(lang.code) && styles.languageButtonSelected,
                ]}
                onPress={() => toggleTargetLanguage(lang.code)}>
                <Text
                  style={[
                    styles.languageButtonText,
                    targetLanguages.includes(lang.code) && styles.languageButtonTextSelected,
                  ]}>
                  {lang.nativeName}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>
      </View>

      {/* Privacy */}
      <View style={styles.card}>
        <Text style={styles.sectionHeader}>Privacy</Text>
        {privacyLoading ? <ActivityIndicator style={{margin: 16}} color={COLOR.primary} /> : <>
        {( [ ['Last seen', lastSeen, 'lastSeenVisibility', setLastSeen], ['Profile photo', profilePhoto, 'profilePhotoVisibility', setProfilePhoto], ['Contacts', contacts, 'contactsVisibility', setContacts] ] as const).map(([label, val, field, setter]) => (
          <View key={field} style={styles.settingsRow}>
            <Text style={styles.settingsRowIcon}>🔒</Text>
            <View style={styles.settingsRowTextWrap}><Text style={styles.settingsRowText}>{label}</Text><Text style={styles.settingsRowDesc}>{val === 'everyone' ? 'Everyone' : val === 'contacts' ? 'My contacts' : 'Nobody'}</Text></View>
            <View style={styles.privacyOptions}>
              {(['everyone','contacts','nobody'] as PrivacyVisibility[]).map(opt => (
                <TouchableOpacity key={opt} style={[styles.privacyChip, val === opt && styles.privacyChipSelected]} onPress={() => { (setter as any)(opt); updatePrivacy(field as any, opt) }}>
                  <Text style={[styles.privacyChipText, val === opt && styles.privacyChipTextSelected]}>{opt === 'everyone' ? 'Everyone' : opt === 'contacts' ? 'Contacts' : 'Nobody'}</Text>
                </TouchableOpacity>
              ))}
            </View>
          </View>
        ))}
        </>}
      </View>

      {/* AI Features */}
      <View style={[styles.card, styles.aiCard]}>
        <Text style={[styles.sectionHeader, styles.aiSectionHeader]}>AI Features</Text>
        <View style={styles.settingsRow}>
          <Text style={styles.settingsRowIcon}>✨</Text>
          <View style={styles.settingsRowTextWrap}>
            <Text style={styles.settingsRowText}>Auto-translation</Text>
            <Text style={styles.settingsRowDesc}>Translate incoming messages</Text>
          </View>
          <View style={styles.switchOn}>
            <View style={styles.switchThumb} />
          </View>
        </View>
        <View style={styles.settingsRow}>
          <Text style={styles.settingsRowIcon}>📊</Text>
          <View style={styles.settingsRowTextWrap}>
            <Text style={styles.settingsRowText}>Grammar Analysis</Text>
            <Text style={styles.settingsRowDesc}>Moderate</Text>
          </View>
          <Text style={styles.chevron}>›</Text>
        </View>
      </View>

      <TouchableOpacity
        style={[styles.saveButton, saving && styles.buttonDisabled]}
        onPress={handleSave}
        disabled={saving}>
        {saving ? (
          <ActivityIndicator color={COLOR.onPrimary} />
        ) : (
          <Text style={styles.saveButtonText}>Save Changes</Text>
        )}
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.logoutButton, loggingOut && styles.buttonDisabled]}
        onPress={handleLogout}
        disabled={loggingOut}>
        {loggingOut ? (
          <ActivityIndicator color={COLOR.error} />
        ) : (
          <Text style={styles.logoutButtonText}>Log Out</Text>
        )}
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  content: {
    padding: SPACING.marginMobile,
    paddingBottom: 48,
  },
  header: {
    marginBottom: SPACING.stackLg,
  },
  title: {
    ...TYPOGRAPHY.headlineSm,
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
    marginBottom: SPACING.unit,
  },
  subtitle: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
    fontFamily: FONTS.body,
  },
  card: {
    ...SHADOWS.elevation1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: RADIUS.xl,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    marginBottom: SPACING.stackMd,
    overflow: 'hidden',
  },
  sectionHeader: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.primary,
    textTransform: 'uppercase',
    letterSpacing: 1,
    backgroundColor: COLOR.surfaceContainerLow,
    paddingHorizontal: SPACING.stackMd,
    paddingVertical: SPACING.stackSm,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
    fontFamily: FONTS.label,
  },
  fieldRow: {
    padding: SPACING.stackMd,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
  },
  label: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onSurface,
    marginBottom: SPACING.stackSm,
    fontFamily: FONTS.label,
  },
  input: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: RADIUS.lg,
    padding: 14,
    fontSize: 16,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
  },
  hint: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
    marginTop: 6,
    fontFamily: FONTS.body,
  },
  settingsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: SPACING.stackMd,
    paddingVertical: SPACING.stackMd,
  },
  settingsRowIcon: {
    fontSize: 18,
    marginRight: SPACING.stackMd,
  },
  settingsRowTextWrap: {
    flex: 1,
  },
  settingsRowText: {
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
  },
  settingsRowDesc: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
    fontFamily: FONTS.body,
  },
  planBadge: {
    backgroundColor: COLOR.primaryContainer,
    borderRadius: 999,
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  planBadgeText: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.onPrimaryContainer,
    fontFamily: FONTS.label,
  },
  chevron: {
    fontSize: 24,
    color: COLOR.outlineVariant,
  },
  aiCard: {
    borderLeftWidth: 4,
    borderLeftColor: COLOR.secondary,
  },
  aiSectionHeader: {
    color: COLOR.secondary,
  },
  switchOn: {
    width: 44,
    height: 24,
    borderRadius: 12,
    backgroundColor: COLOR.secondary,
    justifyContent: 'center',
    paddingHorizontal: 2,
  },
  switchThumb: {
    width: 20,
    height: 20,
    borderRadius: 10,
    backgroundColor: '#fff',
    alignSelf: 'flex-end',
  },
  languageGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  languageButton: {
    flex: 1,
    minWidth: '45%',
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: RADIUS.lg,
    padding: 12,
    alignItems: 'center',
  },
  languageButtonSelected: {
    backgroundColor: COLOR.primary,
    borderColor: COLOR.primary,
  },
  languageButtonText: {
    fontSize: 14,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
  },
  languageButtonTextSelected: {
    color: COLOR.onPrimary,
    fontWeight: '600',
  },
  saveButton: {
    backgroundColor: COLOR.primary,
    borderRadius: RADIUS.xl,
    padding: 16,
    alignItems: 'center',
    marginTop: SPACING.stackSm,
  },
  saveButtonText: {
    color: COLOR.onPrimary,
    fontSize: 16,
    fontWeight: '600',
    fontFamily: FONTS.body,
  },
  logoutButton: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.error,
    borderRadius: RADIUS.xl,
    padding: 16,
    alignItems: 'center',
    marginTop: SPACING.stackMd,
  },
  logoutButtonText: {
    color: COLOR.error,
    fontSize: 16,
    fontWeight: '600',
    fontFamily: FONTS.body,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  privacyOptions: { flexDirection: 'row', gap: 6 },
  privacyChip: { borderWidth: 1, borderColor: COLOR.outlineVariant, borderRadius: 999, paddingHorizontal: 8, paddingVertical: 4 },
  privacyChipSelected: { backgroundColor: COLOR.primary, borderColor: COLOR.primary },
  privacyChipText: { fontSize: 11, color: COLOR.onSurface, fontFamily: FONTS.label },
  privacyChipTextSelected: { color: COLOR.onPrimary },
});