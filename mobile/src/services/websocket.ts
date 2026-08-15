// WebSocket client. The connection/reconnect logic lives in the shared package
// (@chorus/shared); this file wires it to mobile storage and the
// platform-specific WS URL.
import { createWebSocketService } from '@chorus/shared';
import { Platform } from 'react-native';
import storage from '../utils/storage';

// 10.0.2.2 is the host-machine alias inside the Android emulator.
const WS_URL = __DEV__
  ? Platform.select({
      android: 'ws://10.0.2.2:8080/ws',
      default: 'ws://localhost:8080/ws',
    })
  : 'wss://api.chorus.talk/ws';

const webSocketService = createWebSocketService({
  getToken: () => storage.getItem('accessToken'),
  createUrl: (token) => `${WS_URL}?token=${encodeURIComponent(token)}`,
});

export default webSocketService;