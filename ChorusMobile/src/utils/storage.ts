// Storage adapter for web and mobile
import AsyncStorage from '@react-native-async-storage/async-storage';
import { Platform } from 'react-native';

// Minimal DOM-localStorage typing for react-native-web; the RN tsconfig does
// not include the DOM lib.
declare const localStorage: {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
  clear(): void;
};

const isWeb = Platform.OS === 'web';

const storage = {
  async getItem(key: string): Promise<string | null> {
    if (isWeb) {
      return localStorage.getItem(key);
    }
    return AsyncStorage.getItem(key);
  },

  async setItem(key: string, value: string): Promise<void> {
    if (isWeb) {
      localStorage.setItem(key, value);
      return;
    }
    await AsyncStorage.setItem(key, value);
  },

  async removeItem(key: string): Promise<void> {
    if (isWeb) {
      localStorage.removeItem(key);
      return;
    }
    await AsyncStorage.removeItem(key);
  },

  async clear(): Promise<void> {
    if (isWeb) {
      localStorage.clear();
      return;
    }
    await AsyncStorage.clear();
  },
};

export default storage;
