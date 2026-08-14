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
import { SUPPORTED_LANGUAGES, User } from '../types';

export default function ProfileScreen({ navigation }: any) {
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [displayName, setDisplayName] = useState('');
  const [nativeLanguage, setNativeLanguage] = useState('en');
  const [targetLanguages, setTargetLanguages] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    loadUser();
  }, []);

  const loadUser = async () => {
    const userStr = await storage.getItem('user');
    let user: User | null = null;
    if (userStr) {
      try {
        user = JSON.parse(userStr);
      } catch (e) {
        console.error('Failed to parse stored user:', e);
      }
    }
    if (!user) {
      try {
        user = await apiService.getMe();
      } catch (e) {
        console.error('Failed to fetch profile:', e);
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
    } catch (error) {
      console.error('Failed to save profile:', error);
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
      navigation.replace('Login');
    } catch (error) {
      console.error('Logout error:', error);
      Alert.alert('Error', 'Could not log out. Please try again.');
      setLoggingOut(false);
    }
  };

  if (!currentUser) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#007AFF" />
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.section}>
        <Text style={styles.label}>Display Name</Text>
        <TextInput
          style={styles.input}
          value={displayName}
          onChangeText={setDisplayName}
          autoCorrect={false}
        />
        <Text style={styles.hint}>@{currentUser.username}</Text>
      </View>

      <View style={styles.section}>
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

      <View style={styles.section}>
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

      <TouchableOpacity
        style={[styles.saveButton, saving && styles.buttonDisabled]}
        onPress={handleSave}
        disabled={saving}>
        {saving ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.saveButtonText}>Save</Text>
        )}
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.logoutButton, loggingOut && styles.buttonDisabled]}
        onPress={handleLogout}
        disabled={loggingOut}>
        {loggingOut ? (
          <ActivityIndicator color="#d32f2f" />
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
    backgroundColor: '#f5f5f5',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  content: {
    padding: 20,
    paddingBottom: 40,
  },
  section: {
    marginBottom: 24,
  },
  label: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333',
    marginBottom: 8,
  },
  input: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 8,
    padding: 14,
    fontSize: 16,
    color: '#333',
  },
  hint: {
    fontSize: 13,
    color: '#888',
    marginTop: 6,
  },
  languageGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  languageButton: {
    flex: 1,
    minWidth: '45%',
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 8,
    padding: 12,
    alignItems: 'center',
  },
  languageButtonSelected: {
    backgroundColor: '#007AFF',
    borderColor: '#007AFF',
  },
  languageButtonText: {
    fontSize: 14,
    color: '#333',
  },
  languageButtonTextSelected: {
    color: '#fff',
    fontWeight: '600',
  },
  saveButton: {
    backgroundColor: '#007AFF',
    borderRadius: 8,
    padding: 16,
    alignItems: 'center',
    marginTop: 4,
  },
  saveButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  logoutButton: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#d32f2f',
    borderRadius: 8,
    padding: 16,
    alignItems: 'center',
    marginTop: 16,
  },
  logoutButtonText: {
    color: '#d32f2f',
    fontSize: 16,
    fontWeight: '600',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
});