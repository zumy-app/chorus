// WebSocket client. The connection/reconnect logic lives in the shared package
// (@chorus/shared); this file wires it to mobile storage and the
// platform-specific WS URL.
import { createWebSocketService, resolveApiConfig, type ApiPlatform } from '@chorus/shared';
import { Platform } from 'react-native';
import storage from '../utils/storage';

// Derived from the same resolver as the HTTP base URL so the two never drift.
const { wsUrl } = resolveApiConfig({
  platform: Platform.OS as ApiPlatform,
  dev: __DEV__,
  origin: process.env.EXPO_PUBLIC_API_URL,
  version: process.env.EXPO_PUBLIC_API_VERSION,
});

const webSocketService = createWebSocketService({
  getToken: () => storage.getItem('accessToken'),
  createUrl: (token) => `${wsUrl}?token=${encodeURIComponent(token)}`,
});

export default webSocketService;