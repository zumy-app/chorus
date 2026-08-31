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
  getLearningDashboard: client.learning.getDashboard,
  getLearningProfile: client.learning.getProfile,
  getLearningCapabilities: client.learning.getCapabilities,
  getLearningPath: client.learning.getPath,
  updateLearningProfile: client.learning.updateProfile,
  startPlacement: client.learning.startPlacement,
  answerPlacement: client.learning.answerPlacement,
  skipPlacement: client.learning.skipPlacement,
  getPlacement: client.learning.getPlacement,
  getUnit: client.learning.getUnit,
  startLesson: client.learning.startLesson,
  answerLessonStep: client.learning.answerLessonStep,
  completeLesson: client.learning.completeLesson,
  startSession: client.learning.startSession,
  getSession: client.learning.getSession,
  answerSessionItem: client.learning.answerSessionItem,
  completeSession: client.learning.completeSession,
  getMinedItems: client.learning.getMinedItems,
  acceptMinedItem: client.learning.acceptMinedItem,
  ignoreMinedItem: client.learning.ignoreMinedItem,
  getScenarios: client.learning.getScenarios,
  getScenario: client.learning.getScenario,
  startScenario: client.learning.startScenario,
  getScenarioRun: client.learning.getScenarioRun,
  sendScenarioMessage: client.learning.sendScenarioMessage,
  requestScenarioHint: client.learning.requestScenarioHint,
  completeScenario: client.learning.completeScenario,
  getRealTalkPrompts: client.learning.getRealTalkPrompts,
  markRealTalkUsed: client.learning.markRealTalkUsed,
  recoverStreak: client.learning.recoverStreak,
};

export default apiService;