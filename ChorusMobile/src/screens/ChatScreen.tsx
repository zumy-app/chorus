import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  ActivityIndicator,
} from 'react-native';
import storage from '../utils/storage';
import apiService from '../services/api';
import webSocketService from '../services/websocket';
import { Message, WebSocketMessage, User } from '../types';

export default function ChatScreen({ route, navigation }: any) {
  const { chatId, chatName } = route.params;
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputText, setInputText] = useState('');
  const [loading, setLoading] = useState(true);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [typing, setTyping] = useState(false);
  const [translatingId, setTranslatingId] = useState<string | null>(null);
  const flatListRef = useRef<FlatList>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const latestMessageId = useRef<string | null>(null);

  const nativeLanguage = currentUser?.nativeLanguage || 'en';

  const handleWebSocket = useCallback((message: WebSocketMessage) => {
    const payload = message.data || {};
    if (message.type === 'new_message' && payload.chatId === chatId) {
      setMessages((prev) => [
        payload,
        ...prev.filter((m) => m.id !== payload.id),
      ]);
    } else if (message.type === 'message_updated' && payload.chatId === chatId) {
      // A translation completed for this message — swap in the fresh copy.
      setMessages((prev) =>
        prev.map((m) => (m.id === payload.id ? payload : m))
      );
    } else if (message.type === 'user_typing' && payload.chatId === chatId) {
      setTyping(payload.isTyping === true);
    }
  }, [chatId]);

  const loadCurrentUser = useCallback(async () => {
    const userStr = await storage.getItem('user');
    if (userStr) {
      try {
        setCurrentUser(JSON.parse(userStr));
      } catch (e) {
        console.error('Failed to parse stored user:', e);
      }
    }
  }, []);

  const loadMessages = useCallback(async () => {
    try {
      // The backend returns messages newest-first; the inverted FlatList
      // renders index 0 at the bottom, so pass the list through as-is.
      const data = await apiService.getMessages(chatId);
      setMessages(data);
      if (data.length > 0) {
        latestMessageId.current = data[0].id;
        apiService.markAsRead(chatId, data[0].id);
      }
    } catch (error) {
      console.error('Failed to load messages:', error);
    } finally {
      setLoading(false);
    }
  }, [chatId]);

  useEffect(() => {
    navigation.setOptions({ title: chatName });
  }, [navigation, chatName]);

  useEffect(() => {
    loadCurrentUser();
    loadMessages();
    webSocketService.connect();

    const unsubscribe = webSocketService.onMessage(handleWebSocket);

    return () => {
      unsubscribe();
    };
  }, [chatId, handleWebSocket, loadMessages, loadCurrentUser]);

  const handleSend = async () => {
    if (!inputText.trim()) return;

    const messageText = inputText.trim();
    setInputText('');

    try {
      const newMessage = await apiService.sendMessage(chatId, messageText);
      setMessages((prev) => [newMessage, ...prev.filter((m) => m.id !== newMessage.id)]);
      latestMessageId.current = newMessage.id;
      apiService.markAsRead(chatId, newMessage.id);
    } catch (error) {
      console.error('Failed to send message:', error);
      setInputText(messageText);
    }
  };

  const handleTyping = () => {
    webSocketService.sendTyping(chatId, true);

    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    typingTimeoutRef.current = setTimeout(() => {
      webSocketService.sendTyping(chatId, false);
    }, 1500);
  };

  const handleTranslate = async (messageId: string) => {
    if (translatingId) return;
    setTranslatingId(messageId);
    try {
      // The finished translation arrives via the "message_updated" event.
      await apiService.translateMessage(chatId, messageId, nativeLanguage);
    } catch (error) {
      console.error('Failed to request translation:', error);
    } finally {
      setTranslatingId(null);
    }
  };

  const renderMessage = ({ item }: { item: Message }) => {
    const isOwn = item.senderId === currentUser?.id;
    const nativeTranslation =
      item.translations?.[nativeLanguage] &&
      item.translations[nativeLanguage] !== item.text
        ? item.translations[nativeLanguage]
        : null;

    return (
      <View style={[styles.messageContainer, isOwn ? styles.ownMessage : styles.otherMessage]}>
        {!isOwn && (
          <Text style={styles.senderName}>{item.sender?.displayName || 'Unknown'}</Text>
        )}
        <View style={[styles.messageBubble, isOwn ? styles.ownBubble : styles.otherBubble]}>
          <Text style={[styles.messageText, isOwn ? styles.ownText : styles.otherText]}>
            {item.text}
          </Text>
          {nativeTranslation && (
            <Text style={[styles.translatedText, isOwn ? styles.ownTranslated : styles.otherTranslated]}>
              {nativeTranslation}
            </Text>
          )}
          {!isOwn && !nativeTranslation && (
            <TouchableOpacity
              onPress={() => handleTranslate(item.id)}
              disabled={translatingId === item.id}
              style={styles.translateButton}
            >
              <Text style={styles.translateButtonText}>
                {translatingId === item.id ? 'Translating...' : '🌐 Translate'}
              </Text>
            </TouchableOpacity>
          )}
          <Text style={[styles.messageTime, isOwn ? styles.ownTime : styles.otherTime]}>
            {new Date(item.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
          </Text>
        </View>
      </View>
    );
  };

  if (loading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#007AFF" />
      </View>
    );
  }

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={90}>
      <FlatList
        ref={flatListRef}
        data={messages}
        renderItem={renderMessage}
        keyExtractor={(item) => item.id}
        inverted
        contentContainerStyle={styles.messageList}
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyText}>No messages yet</Text>
            <Text style={styles.emptySubtext}>Start the conversation!</Text>
          </View>
        }
      />
      {typing && (
        <View style={styles.typingIndicator}>
          <Text style={styles.typingText}>Someone is typing...</Text>
        </View>
      )}
      <View style={styles.inputContainer}>
        <TextInput
          style={styles.input}
          value={inputText}
          onChangeText={(text) => {
            setInputText(text);
            handleTyping();
          }}
          placeholder="Type a message..."
          multiline
          maxLength={1000}
        />
        <TouchableOpacity
          style={[styles.sendButton, !inputText.trim() && styles.sendButtonDisabled]}
          onPress={handleSend}
          disabled={!inputText.trim()}>
          <Text style={styles.sendButtonText}>Send</Text>
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
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
  messageList: {
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  messageContainer: {
    marginVertical: 4,
    maxWidth: '75%',
  },
  ownMessage: {
    alignSelf: 'flex-end',
  },
  otherMessage: {
    alignSelf: 'flex-start',
  },
  senderName: {
    fontSize: 12,
    color: '#666',
    marginBottom: 2,
    marginLeft: 12,
  },
  messageBubble: {
    padding: 12,
    borderRadius: 18,
  },
  ownBubble: {
    backgroundColor: '#007AFF',
  },
  otherBubble: {
    backgroundColor: '#fff',
  },
  messageText: {
    fontSize: 16,
    lineHeight: 20,
  },
  ownText: {
    color: '#fff',
  },
  otherText: {
    color: '#333',
  },
  translatedText: {
    fontSize: 14,
    marginTop: 6,
    fontStyle: 'italic',
    opacity: 0.8,
  },
  ownTranslated: {
    color: '#fff',
  },
  otherTranslated: {
    color: '#666',
  },
  translateButton: {
    marginTop: 6,
    alignSelf: 'flex-start',
  },
  translateButtonText: {
    fontSize: 12,
    color: '#007AFF',
    fontWeight: '600',
  },
  messageTime: {
    fontSize: 11,
    marginTop: 4,
  },
  ownTime: {
    color: 'rgba(255, 255, 255, 0.7)',
    textAlign: 'right',
  },
  otherTime: {
    color: '#999',
  },
  typingIndicator: {
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  typingText: {
    fontSize: 12,
    color: '#666',
    fontStyle: 'italic',
  },
  inputContainer: {
    flexDirection: 'row',
    padding: 12,
    backgroundColor: '#fff',
    borderTopWidth: 1,
    borderTopColor: '#ddd',
  },
  input: {
    flex: 1,
    backgroundColor: '#f5f5f5',
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 8,
    marginRight: 8,
    maxHeight: 100,
    fontSize: 16,
  },
  sendButton: {
    backgroundColor: '#007AFF',
    borderRadius: 20,
    paddingHorizontal: 20,
    justifyContent: 'center',
    alignItems: 'center',
  },
  sendButtonDisabled: {
    backgroundColor: '#ccc',
  },
  sendButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingTop: 100,
  },
  emptyText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#333',
    marginBottom: 8,
  },
  emptySubtext: {
    fontSize: 14,
    color: '#666',
  },
});
