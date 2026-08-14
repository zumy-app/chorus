import axios, { AxiosInstance } from 'axios';
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

class ApiService {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor to include auth token
    this.client.interceptors.request.use(
      async (config) => {
        const token = await storage.getItem('accessToken');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Add response interceptor for token refresh
    this.client.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;
        
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;
          
          try {
            const refreshToken = await storage.getItem('refreshToken');
            if (refreshToken) {
              const response = await this.refreshToken(refreshToken);
              await storage.setItem('accessToken', response.tokens.accessToken);
              await storage.setItem('refreshToken', response.tokens.refreshToken);
              
              originalRequest.headers.Authorization = `Bearer ${response.tokens.accessToken}`;
              return this.client(originalRequest);
            }
          } catch (refreshError) {
            await this.logout();
            throw refreshError;
          }
        }
        
        return Promise.reject(error);
      }
    );
  }

  // Auth endpoints
  async register(data: {
    username: string;
    email: string;
    password: string;
    displayName: string;
    nativeLanguage: string;
    targetLanguages: string[];
  }) {
    const response = await this.client.post('/auth/register', data);
    return response.data;
  }

  async login(username: string, password: string) {
    const response = await this.client.post('/auth/login', {
      username,
      password,
    });
    return response.data;
  }

  async refreshToken(refreshToken: string) {
    const response = await this.client.post('/auth/refresh', {
      refreshToken,
    });
    return response.data;
  }

  async logout() {
    await storage.removeItem('accessToken');
    await storage.removeItem('refreshToken');
    await storage.removeItem('user');
  }

  // User endpoints
  async getMe() {
    const response = await this.client.get('/users/me');
    return response.data;
  }

  async updateProfile(data: {
    displayName?: string;
    nativeLanguage?: string;
    targetLanguages?: string[];
  }) {
    const response = await this.client.put('/users/me', data);
    return response.data;
  }

  async searchUsers(query: string) {
    const response = await this.client.get('/users/search', {
      params: { q: query, limit: 20 },
    });
    return response.data;
  }

  // Chat endpoints
  async getChats() {
    const response = await this.client.get('/chats');
    return response.data.chats; // Extract chats array from response
  }

  async createChat(data: {
    type: 'direct' | 'group';
    participants: string[];
    name?: string;
  }) {
    const response = await this.client.post('/chats', data);
    return response.data;
  }

  async getChat(chatId: string) {
    const response = await this.client.get(`/chats/${chatId}`);
    return response.data;
  }

  // Message endpoints
  async getMessages(chatId: string, limit: number = 50, before?: string) {
    const response = await this.client.get(`/chats/${chatId}/messages`, {
      params: { limit, before },
    });
    return response.data.messages; // Extract messages array from response
  }

  async sendMessage(chatId: string, text: string, replyToId?: string) {
    const response = await this.client.post(`/chats/${chatId}/messages`, {
      text,
      replyToId,
    });
    return response.data;
  }

  async markAsRead(chatId: string, messageId: string) {
    const response = await this.client.put(`/chats/${chatId}/read`, {
      messageId,
    });
    return response.data;
  }

  // Per-message translation requested explicitly by a participant. The result
  // arrives over the WebSocket "message_updated" event once the job completes.
  async translateMessage(chatId: string, messageId: string, targetLang: string) {
    const response = await this.client.post(
      `/chats/${chatId}/messages/${messageId}/translate`,
      { targetLang }
    );
    return response.data;
  }

  // Health check
  async healthCheck() {
    try {
      const response = await axios.get(API_BASE_URL.replace('/api/v1', '/health'));
      return response.data;
    } catch (error) {
      throw error;
    }
  }
}

export default new ApiService();
