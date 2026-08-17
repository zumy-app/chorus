// HTTP API client. The implementation (interceptors, token refresh, endpoint
// groups) lives in the shared package (@chorus/shared); this file wires it to
// the mobile storage adapter (src/utils/storage.ts) and platform-specific URLs.
import { createApiClient, resolveApiConfig, type ApiPlatform } from '@chorus/shared';
import { Platform } from 'react-native';
import storage from '../utils/storage';

// Same-host /api/{version} by default; override with EXPO_PUBLIC_API_URL /
// EXPO_PUBLIC_API_VERSION to reach a remote backend (e.g. the dev PC from a
// physical device: EXPO_PUBLIC_API_URL=http://<lan-ip>:8080).
// EXPO_PUBLIC_* is only inlined by Expo tooling; under the plain react-native
// CLI, process.env.* stays undefined at runtime, so API_ORIGIN also falls back
// to the dev machine's LAN address to keep physical devices working.
export const API_ORIGIN = process.env.EXPO_PUBLIC_API_URL || 'http://10.0.0.30:8080';

const { baseURL } = resolveApiConfig({
  platform: Platform.OS as ApiPlatform,
  dev: __DEV__,
  origin: API_ORIGIN,
  version: process.env.EXPO_PUBLIC_API_VERSION,
});

const client = createApiClient({
  baseURL,
  storage,
});

// Kept default-exported as an object to preserve the previous ApiService API.
const apiService = {
  register: client.auth.register,
  login: (username: string, password: string) =>
    client.auth.login({ username, password }),
  refreshToken: client.auth.refreshToken,
  logout: client.auth.logout,
  getMe: client.auth.getMe,
  updateProfile: client.auth.updateMe,
  searchUsers: client.auth.searchUsers,
  forgotPassword: client.auth.forgotPassword,
  resetPassword: client.auth.resetPassword,
  getChats: client.chat.getChats,
  createChat: client.chat.createChat,
  getChat: client.chat.getChat,
  getMessages: client.message.getMessages,
  sendMessage: (chatId: string, text: string, replyToId?: string) =>
    client.message.sendMessage(chatId, { text, replyToId }),
  markAsRead: client.message.markAsRead,
  translateMessage: client.translation.translateMessage,
  healthCheck: client.health,
};

export default apiService;