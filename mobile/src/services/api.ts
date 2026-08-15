// HTTP API client. The implementation (interceptors, token refresh, endpoint
// groups) lives in the shared package (@chorus/shared); this file wires it to
// the mobile storage adapter (src/utils/storage.ts) and platform-specific URLs.
import { createApiClient } from '@chorus/shared';
import { Platform } from 'react-native';
import storage from '../utils/storage';

// Use your backend URL - change this if deployed elsewhere.
// 10.0.2.2 is the special host-machine alias inside the Android emulator.
const API_BASE_URL = __DEV__
  ? Platform.select({
      android: 'http://10.0.2.2:8080/api/v1',
      default: 'http://localhost:8080/api/v1',
    })
  : 'https://api.chorus.talk/api/v1';

const client = createApiClient({
  baseURL: API_BASE_URL,
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