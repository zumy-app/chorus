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
import { SUPPORTED_LANGUAGES, User, type PrivacyVisibility, DEV_ACCOUNTS, apiErrorMessage } from '@chorus/shared';
import { useStrings, t, useAppLocale, setExplicitLanguage, applyImplicitLanguage, BUNDLED_LOCALES, type Locale } from '../i18n';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

export default function ProfileScreen({ navigation }: any) {
  const s = useStrings();
  const appLang = useAppLocale();
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
  const [blocked, setBlocked] = useState<any[]>([]);
  const [phoneStatus, setPhoneStatus] = useState<any>(null);
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');

  useEffect(() => {
    loadUser();
    loadPrivacy();
    (apiService as any).getBlocked?.().then(setBlocked).catch(()=>{});
    (apiService as any).getPhoneStatus?.().then(setPhoneStatus).catch(()=>{});
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
    } catch { Alert.alert(t('common.error'), t('profile.privacyFailB')); }
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
      Alert.alert(t('common.error'), t('profile.emptyNameB'));
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
      // A changed learning native refines the UI language (no-op with an
      // explicit app-language pick). "App language" vs "I speak" stay separate.
      await applyImplicitLanguage(updated.nativeLanguage);
      Alert.alert(t('common.success'), t('profile.savedB'));
    } catch {
      Alert.alert(t('common.error'), t('profile.saveFailB'));
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
      Alert.alert(t('common.error'), t('profile.logoutFailB'));
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

  const isDev = typeof (globalThis as any).__DEV__ !== 'undefined' ? (globalThis as any).__DEV__ : (typeof __DEV__ !== 'undefined' ? (__DEV__ as unknown as boolean) : true)
  const handleDevSwitch = async (a: typeof DEV_ACCOUNTS[number]) => {
    try {
      webSocketService.disconnect()
      // Typed re-login: apiService.switchUser resolves {tokens, user} or
      // throws a human-readable Error. Never reach into response envelopes
      // here — `(apiService as any).api?.post(...)` was undefined (the
      // default export has no `.api`), which crashed on `raw.data`.
      const { tokens, user } = await apiService.switchUser(a.email, a.password)
      await storage.setItem('accessToken', tokens.accessToken)
      await storage.setItem('refreshToken', tokens.refreshToken)
      await storage.setItem('user', JSON.stringify(user))
      await applyImplicitLanguage(user.nativeLanguage)
      navigation.replace('MainTabs')
    } catch (e: any) {
      Alert.alert(t('profile.switchFailT'), apiErrorMessage(e, t('profile.switchFailB')))
    }
  }

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      showsVerticalScrollIndicator={false}>
      <View style={styles.header}>
        <Text style={styles.title}>{s.profile.title}</Text>
        <Text style={styles.subtitle}>{s.profile.subtitle}</Text>
      </View>
      {isDev && (
        <View style={{ borderWidth: 1, borderColor: '#FDE68A', backgroundColor: '#FFFBEB', borderRadius: RADIUS.lg, padding: SPACING.stackMd, gap: 8, marginBottom: SPACING.stackMd }}>
          <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6 }}>
            <Text style={{ fontSize: 10, fontWeight: '700', letterSpacing: 0.8, color: '#92400E' }}>{s.profile.devOnly}</Text>
            <Text style={{ fontSize: 11, color: '#B45309', flex: 1 }}>{s.profile.devHint}</Text>
          </View>
          {DEV_ACCOUNTS.map((a) => (
            <TouchableOpacity key={a.email} onPress={() => handleDevSwitch(a)} style={{ flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: 'white', borderWidth: 1, borderColor: '#FDE68A', borderRadius: RADIUS.lg, paddingHorizontal: 12, paddingVertical: 10 }}>
              <View style={{ flex: 1, gap: 2 }}><Text style={{ fontSize: 13, fontWeight: '600', color: COLOR.onSurface }}>{a.label}</Text><Text style={{ fontSize: 11, color: COLOR.onSurfaceVariant }}>{a.email}</Text></View>
              <Text style={{ fontSize: 12, color: COLOR.primary, marginLeft: 8 }}>{s.profile.switchAction}</Text>
            </TouchableOpacity>
          ))}
        </View>
      )}

      {/* Account */}
      <View style={styles.card}>
        <Text style={styles.sectionHeader}>{s.profile.account}</Text>
        <View style={styles.fieldRow}>
          <Text style={styles.label}>{s.profile.displayName}</Text>
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
          <Text style={styles.settingsRowText}>{s.profile.subscription}</Text>
          <View style={styles.planBadge}>
            <Text style={styles.planBadgeText}>{s.profile.free}</Text>
          </View>
        </View>
      </View>

      {/* App language (interface) — separate from the learning languages below. */}
      <View style={styles.card}>
        <Text style={styles.sectionHeader}>{s.profile.appLanguage}</Text>
        <Text style={[styles.settingsRowDesc, { paddingHorizontal: 16, paddingBottom: 8 }]}>{s.profile.appLanguageDesc}</Text>
        <View style={[styles.languageGrid, { paddingHorizontal: 16, paddingBottom: 16 }]}>
          {(BUNDLED_LOCALES as readonly string[]).map((code) => {
            const info = SUPPORTED_LANGUAGES.find((l) => l.code === code);
            return (
              <TouchableOpacity
                key={code}
                style={[
                  styles.languageButton,
                  appLang === code && styles.languageButtonSelected,
                ]}
                onPress={() => setExplicitLanguage(code as Locale)}>
                <Text
                  style={[
                    styles.languageButtonText,
                    appLang === code && styles.languageButtonTextSelected,
                  ]}>
                  {info?.nativeName ?? code}
                </Text>
              </TouchableOpacity>
            );
          })}
        </View>
      </View>

      {/* Language */}
      <View style={styles.card}>
        <Text style={styles.sectionHeader}>{s.profile.language}</Text>
        <View style={styles.fieldRow}>
          <Text style={styles.label}>{s.profile.nativeLang}</Text>
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
          <Text style={styles.label}>{s.profile.targetLangs}</Text>
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
        <Text style={styles.sectionHeader}>{s.profile.privacy}</Text>
        {privacyLoading ? <ActivityIndicator style={{margin: 16}} color={COLOR.primary} /> : <>
        {( [ [s.profile.lastSeen, lastSeen, 'lastSeenVisibility', setLastSeen], [s.profile.profilePhoto, profilePhoto, 'profilePhotoVisibility', setProfilePhoto], [s.profile.contacts, contacts, 'contactsVisibility', setContacts] ] as const).map(([label, val, field, setter]) => (
          <View key={field} style={styles.settingsRow}>
            <Text style={styles.settingsRowIcon}>🔒</Text>
            <View style={styles.settingsRowTextWrap}><Text style={styles.settingsRowText}>{label}</Text><Text style={styles.settingsRowDesc}>{val === 'everyone' ? s.profile.privacyEveryone : val === 'contacts' ? s.profile.privacyContactsDesc : s.profile.privacyNobody}</Text></View>
            <View style={styles.privacyOptions}>
              {(['everyone','contacts','nobody'] as PrivacyVisibility[]).map(opt => (
                <TouchableOpacity key={opt} style={[styles.privacyChip, val === opt && styles.privacyChipSelected]} onPress={() => { (setter as any)(opt); updatePrivacy(field as any, opt) }}>
                  <Text style={[styles.privacyChipText, val === opt && styles.privacyChipTextSelected]}>{opt === 'everyone' ? s.profile.privacyEveryone : opt === 'contacts' ? s.profile.privacyContacts : s.profile.privacyNobody}</Text>
                </TouchableOpacity>
              ))}
            </View>
          </View>
        ))}
        </>}
      </View>

      <View style={styles.card}>
        <Text style={styles.sectionHeader}>{s.profile.twoFA}</Text>
        <View style={{padding: 16, gap: 8}}>
          <Text style={styles.settingsRowDesc}>{s.profile.phoneLabel} {phoneStatus?.phoneMasked || s.profile.notSet} {phoneStatus?.phoneVerified ? '✓' : ''}  {s.profile.twofaLabel} {phoneStatus?.twoFactorEnabled ? s.profile.on : s.profile.off}</Text>
          <TextInput style={styles.input} value={phone} onChangeText={setPhone} placeholder="+14155551234" keyboardType="phone-pad" />
          <TouchableOpacity style={styles.saveButton} onPress={async()=>{ try{ const r=await (apiService as any).requestOTP(phone); Alert.alert(t('profile.codeSentT'), t('profile.codeSentB', { phone: r.phoneMasked }))} catch(e:any){ Alert.alert(t('common.error'), e.response?.data?.error||t('auth.genericErrB'))}}}><Text style={styles.saveButtonText}>{s.profile.sendCode}</Text></TouchableOpacity>
          <TextInput style={styles.input} value={code} onChangeText={setCode} placeholder="123456" keyboardType="number-pad" maxLength={6} />
          <TouchableOpacity style={styles.saveButton} onPress={async()=>{ try{ await (apiService as any).verifyPhone(phone, code); Alert.alert(t('profile.verifiedT'),t('profile.verifiedB')); const s2=await (apiService as any).getPhoneStatus(); setPhoneStatus(s2)} catch(e:any){ Alert.alert(t('common.error'), e.response?.data?.error||t('auth.verify2faFailedB'))}}}><Text style={styles.saveButtonText}>{s.profile.verifyPhoneBtn}</Text></TouchableOpacity>
          <TouchableOpacity style={[styles.saveButton, !phoneStatus?.phoneVerified && styles.buttonDisabled]} disabled={!phoneStatus?.phoneVerified} onPress={async()=>{ try{ const s2=await (apiService as any).setTwoFactor(!phoneStatus?.twoFactorEnabled); setPhoneStatus(s2); Alert.alert(s2.twoFactorEnabled?t('profile.enabledState'):t('profile.disabledState'))} catch(e:any){ Alert.alert(t('common.error'), e.response?.data?.error||t('auth.genericErrB'))}}}><Text style={styles.saveButtonText}>{phoneStatus?.twoFactorEnabled?s.profile.disable2fa:s.profile.enable2fa}</Text></TouchableOpacity>
        </View>
      </View>

      {/* AI Features */}
      <View style={[styles.card, styles.aiCard]}>
        <Text style={[styles.sectionHeader, styles.aiSectionHeader]}>{s.profile.aiFeatures}</Text>
        <View style={styles.settingsRow}>
          <Text style={styles.settingsRowIcon}>✨</Text>
          <View style={styles.settingsRowTextWrap}>
            <Text style={styles.settingsRowText}>{s.profile.autoTranslate}</Text>
            <Text style={styles.settingsRowDesc}>{s.profile.autoTranslateDesc}</Text>
          </View>
          <View style={styles.switchOn}>
            <View style={styles.switchThumb} />
          </View>
        </View>
        <View style={styles.settingsRow}>
          <Text style={styles.settingsRowIcon}>📊</Text>
          <View style={styles.settingsRowTextWrap}>
            <Text style={styles.settingsRowText}>{s.profile.grammarAnalysis}</Text>
            <Text style={styles.settingsRowDesc}>{s.profile.moderate}</Text>
          </View>
          <Text style={styles.chevron}>›</Text>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.sectionHeader}>{s.profile.blockedUsers}</Text>
        {blocked.length === 0 ? <Text style={{padding:16, color: COLOR.onSurfaceVariant}}>{s.profile.noBlocked}</Text> : blocked.map((b:any)=>(
          <View key={b.id} style={{flexDirection:'row', justifyContent:'space-between', alignItems:'center', padding:12, borderTopWidth:1, borderTopColor: COLOR.outlineVariant}}>
            <Text style={{color: COLOR.onSurface, flex:1}}>{b.blocked?.displayName || b.blocked?.username || s.profile.userFallback}</Text>
            <TouchableOpacity style={{borderWidth:1, borderColor: COLOR.outlineVariant, borderRadius: 999, paddingHorizontal:12, paddingVertical:6}} onPress={async()=>{ try{ await (apiService as any).unblockUser(b.blockedId); setBlocked(prev=>prev.filter(x=>x.blockedId!==b.blockedId)) } catch{ Alert.alert(t('common.error'),t('profile.unblockFailB'))}}}>
              <Text style={{color: COLOR.onSurface}}>{s.profile.unblock}</Text>
            </TouchableOpacity>
          </View>
        ))}
      </View>
      <TouchableOpacity style={{backgroundColor: COLOR.primary, borderRadius: RADIUS.xl, padding: 16, alignItems:'center', marginTop: SPACING.stackSm}} onPress={()=>navigation.navigate('BecomeTeacher')}>
        <Text style={styles.saveButtonText}>{s.profile.becomeTeacher}</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.saveButton, saving && styles.buttonDisabled]}
        onPress={handleSave}
        disabled={saving}>
        {saving ? (
          <ActivityIndicator color={COLOR.onPrimary} />
        ) : (
          <Text style={styles.saveButtonText}>{s.profile.saveChanges}</Text>
        )}
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.logoutButton, loggingOut && styles.buttonDisabled]}
        onPress={handleLogout}
        disabled={loggingOut}>
        {loggingOut ? (
          <ActivityIndicator color={COLOR.error} />
        ) : (
          <Text style={styles.logoutButtonText}>{s.profile.logout}</Text>
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