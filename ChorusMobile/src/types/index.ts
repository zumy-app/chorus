export interface User {
  id: string;
  username: string;
  email: string;
  displayName: string;
  nativeLanguage: string;
  targetLanguages: string[];
  createdAt: string;
  lastActiveAt: string;
}

export interface Chat {
  id: string;
  type: 'direct' | 'group';
  name?: string;
  createdBy: string;
  settings?: Record<string, any>;
  createdAt: string;
  participants?: ChatParticipant[];
  lastMessage?: Message;
  unreadCount?: number;
}

export interface ChatParticipant {
  chatId: string;
  userId: string;
  role: 'admin' | 'member';
  joinedAt: string;
  user?: User;
}

export interface Message {
  id: string;
  chatId: string;
  senderId: string;
  text: string;
  originalLanguage?: string;
  translations?: Record<string, string>;
  translationEnhanced?: boolean;
  deliveryStatus: 'sent' | 'delivered' | 'failed';
  replyToId?: string;
  timestamp: string;
  sender?: User;
}

export interface WebSocketMessage {
  type: string;
  data: any;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface Language {
  code: string;
  name: string;
  nativeName: string;
}

export const SUPPORTED_LANGUAGES: Language[] = [
  { code: 'en', name: 'English', nativeName: 'English' },
  { code: 'es', name: 'Spanish', nativeName: 'Español' },
  { code: 'fr', name: 'French', nativeName: 'Français' },
  { code: 'de', name: 'German', nativeName: 'Deutsch' },
  { code: 'it', name: 'Italian', nativeName: 'Italiano' },
  { code: 'pt', name: 'Portuguese', nativeName: 'Português' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語' },
  { code: 'ko', name: 'Korean', nativeName: '한국어' },
  { code: 'zh', name: 'Chinese', nativeName: '中文' },
];
