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
  Modal,
} from 'react-native';
import storage from '../utils/storage';
import apiService from '../services/api';
import webSocketService from '../services/websocket';
import { Message, WebSocketMessage, User } from '@chorus/shared';
import { COLOR, FONTS } from '../theme';

export default function ChatScreen({ route, navigation }: any) {
  const { chatId, chatName } = route.params;
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputText, setInputText] = useState('');
  const [loading, setLoading] = useState(true);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [typing, setTyping] = useState(false);
  const [translatingId, setTranslatingId] = useState<string | null>(null);
  const [translateAsType, setTranslateAsType] = useState(false);
  const [deepDiveVisible, setDeepDiveVisible] = useState(false);
  const [sparkyInput, setSparkyInput] = useState('');
  const flatListRef = useRef<FlatList>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const latestMessageId = useRef<string | null>(null);

  const nativeLanguage = currentUser?.nativeLanguage || 'en';
  const targetLang = currentUser?.targetLanguages?.[0]?.toUpperCase();

  const handleWebSocket = useCallback((message: WebSocketMessage) => {
    const payload = message.data || {};
    if (message.type === 'new_message' && payload.chatId === chatId) {
      setMessages((prev) => [
        payload,
        ...prev.filter((m) => m.id !== payload.id),
      ]);
    } else if (message.type === 'message_updated' && payload.chatId === chatId) {
      setMessages((prev) =>
        prev.map((m) => (m.id === payload.id ? payload : m))
      );
    } else if (message.type === 'user_typing' && payload.chatId === chatId) {
      setTyping(payload.isTyping === true);
    } else if (message.type === 'call_incoming' && payload.chatId === chatId) {
      const callId = (payload as Record<string, string>).callId || (payload as Record<string, string>).call_id;
      if (callId) navigation.navigate('Call', { callId, chatId, chatName });
    }
  }, [chatId, chatName, navigation]);

  const loadCurrentUser = useCallback(async () => {
    const userStr = await storage.getItem('user');
    if (userStr) {
      try {
        setCurrentUser(JSON.parse(userStr));
      } catch {
        // Corrupted storage — ignore.
      }
    }
  }, []);

  const loadMessages = useCallback(async () => {
    try {
      const data = await apiService.getMessages(chatId);
      setMessages(data);
      if (data.length > 0) {
        latestMessageId.current = data[0].id;
        apiService.markAsRead(chatId, data[0].id);
      }
    } catch {
      // Backend unreachable or unauthorized — silently ignore.
    } finally {
      setLoading(false);
    }
  }, [chatId]);

  const startCall = async () => {
    try {
      const res = await apiService.initiateCall(chatId, 'audio');
      navigation.navigate('Call', { callId: res.session.id, chatId, chatName });
    } catch {}
  };

  useEffect(() => {
    navigation.setOptions({
      title: chatName,
      headerTintColor: COLOR.primary,
      headerTitleStyle: { fontSize: 18, fontWeight: '600', color: COLOR.primary },
      headerRight: () => (
        <TouchableOpacity onPress={startCall} style={{ marginRight: 4, width: 36, height: 36, borderRadius: 18, backgroundColor: COLOR.primaryContainer, alignItems: 'center', justifyContent: 'center' }}>
          <Text style={{ color: '#fff', fontSize: 16 }}>📞</Text>
        </TouchableOpacity>
      ),
    });
  }, [navigation, chatName, chatId]);

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
    } catch {
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
      await apiService.translateMessage(chatId, messageId, nativeLanguage);
    } catch {
      // Translation request failed — non-fatal.
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
            <View style={[styles.translationBlock, isOwn ? styles.ownTranslationBlock : styles.otherTranslationBlock]}>
              <Text style={[styles.translationLabel, isOwn ? styles.ownLabel : styles.otherLabel]}>
                🌐 In your language
              </Text>
              <Text style={[styles.translatedText, isOwn ? styles.ownTranslated : styles.otherTranslated]}>
                {nativeTranslation}
              </Text>
            </View>
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
          {isOwn && (
            <View style={styles.checkRow}>
              <Text style={styles.checkmark}>✓✓</Text>
            </View>
          )}
        </View>
        <Text style={[styles.messageTime, isOwn ? styles.ownTime : styles.otherTime]}>
          {new Date(item.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </Text>
      </View>
    );
  };

  if (loading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color={COLOR.primary} />
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
      <View style={styles.inputArea}>
        <View style={styles.toggleRow}>
          <TouchableOpacity
            style={styles.toggleLabel}
            onPress={() => setTranslateAsType((prev) => !prev)}
            accessibilityRole="switch"
            accessibilityState={{ checked: translateAsType }}>
            <Text style={styles.toggleIcon}>🌐</Text>
            <Text style={[styles.toggleText, translateAsType && styles.toggleTextActive]}>
              Translate as I type
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.switch, translateAsType && styles.switchOn]}
            onPress={() => setTranslateAsType((prev) => !prev)}>
            <View style={[styles.switchThumb, translateAsType && styles.switchThumbOn]} />
          </TouchableOpacity>
        </View>

        {translateAsType && inputText.trim().length > 0 && (
          <View style={styles.livePreview}>
            <Text style={styles.livePreviewLabel}>
              ✨ Live translation to {targetLang}
            </Text>
            <Text style={styles.livePreviewText}>{inputText}</Text>
          </View>
        )}

        <View style={styles.inputRow}>
          <TouchableOpacity style={styles.iconButton}>
            <Text style={styles.iconButtonText}>＋</Text>
          </TouchableOpacity>
          <TextInput
            style={styles.input}
            value={inputText}
            onChangeText={(text) => {
              setInputText(text);
              handleTyping();
            }}
            placeholder="Type a message..."
            placeholderTextColor={COLOR.onSurfaceVariant}
            multiline
            maxLength={1000}
          />
          <TouchableOpacity style={styles.iconButton}>
            <Text style={styles.iconButtonText}>🎙</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.sendButton, !inputText.trim() && styles.sendButtonDisabled]}
            onPress={handleSend}
            disabled={!inputText.trim()}>
            <Text style={styles.sendButtonText}>➤</Text>
          </TouchableOpacity>
        </View>
      </View>

      {/* Sparky FAB */}
      <TouchableOpacity
        style={styles.sparkyFab}
        onPress={() => setDeepDiveVisible(true)}
        accessibilityLabel="Ask Sparky">
        <Text style={styles.sparkyFabIcon}>🤖</Text>
        <View style={styles.sparkyDot} />
      </TouchableOpacity>

      {/* Deep Dive bottom sheet */}
      <Modal
        visible={deepDiveVisible}
        transparent
        animationType="slide"
        onRequestClose={() => setDeepDiveVisible(false)}>
        <View style={styles.sheetOverlay}>
          <TouchableOpacity
            style={styles.sheetBackdrop}
            activeOpacity={1}
            onPress={() => setDeepDiveVisible(false)}
          />
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <View style={styles.sheetHeader}>
              <View style={styles.sheetTitleRow}>
                <Text style={styles.sheetTitleIcon}>✨</Text>
                <Text style={styles.sheetTitle}>AI Deep Dive</Text>
              </View>
              <TouchableOpacity onPress={() => setDeepDiveVisible(false)}>
                <Text style={styles.sheetClose}>✕</Text>
              </TouchableOpacity>
            </View>
            <View style={styles.sheetBody}>
              <View style={styles.sheetIntro}>
                <Text style={styles.sheetIntroTitle}>Sparky's Insight</Text>
                <Text style={styles.sheetIntroText}>Let's look at that last sentence.</Text>
              </View>
              <View style={styles.sheetTutorBubble}>
                <Text style={styles.sheetTutorText}>
                  Ask Sparky about any message for grammar help and practice ideas.
                </Text>
              </View>
            </View>
            <View style={styles.sheetInputRow}>
              <TextInput
                style={styles.sheetInput}
                value={sparkyInput}
                onChangeText={setSparkyInput}
                placeholder="Ask Sparky..."
                placeholderTextColor={COLOR.onSurfaceVariant}
                multiline
              />
              <TouchableOpacity
                style={[styles.sheetSend, !sparkyInput.trim() && styles.sendButtonDisabled]}
                disabled={!sparkyInput.trim()}>
                <Text style={styles.sheetSendText}>➤</Text>
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  messageList: {
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  messageContainer: {
    marginVertical: 6,
    maxWidth: '80%',
  },
  ownMessage: {
    alignSelf: 'flex-end',
    alignItems: 'flex-end',
  },
  otherMessage: {
    alignSelf: 'flex-start',
    alignItems: 'flex-start',
  },
  senderName: {
    fontSize: 12,
    fontWeight: '600',
    color: COLOR.onSurfaceVariant,
    marginBottom: 4,
    marginLeft: 16,
  },
  messageBubble: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    shadowColor: '#000',
    shadowOpacity: 0.05,
    shadowRadius: 4,
    shadowOffset: { width: 0, height: 2 },
    elevation: 1,
  },
  ownBubble: {
    backgroundColor: COLOR.primary,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    borderBottomLeftRadius: 24,
    borderBottomRightRadius: 4,
  },
  otherBubble: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderColor: COLOR.outlineVariant,
    borderWidth: 1,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    borderBottomLeftRadius: 4,
    borderBottomRightRadius: 24,
  },
  messageText: {
    fontSize: 16,
    lineHeight: 24,
    fontFamily: FONTS.body,
  },
  ownText: {
    color: COLOR.onPrimary,
  },
  otherText: {
    color: COLOR.onSurface,
  },
  translationBlock: {
    marginTop: 8,
    paddingTop: 8,
    borderTopWidth: 1,
  },
  ownTranslationBlock: {
    borderTopColor: 'rgba(255, 255, 255, 0.3)',
  },
  otherTranslationBlock: {
    borderTopColor: 'rgba(67, 70, 85, 0.3)',
  },
  translationLabel: {
    fontSize: 11,
    marginBottom: 2,
  },
  ownLabel: {
    color: 'rgba(255, 255, 255, 0.75)',
  },
  otherLabel: {
    color: COLOR.onSurfaceVariant,
  },
  translatedText: {
    fontSize: 14,
    lineHeight: 20,
    fontStyle: 'italic',
  },
  ownTranslated: {
    color: COLOR.onPrimary,
  },
  otherTranslated: {
    color: COLOR.onSurfaceVariant,
  },
  translateButton: {
    marginTop: 8,
    alignSelf: 'flex-start',
  },
  translateButtonText: {
    fontSize: 12,
    color: COLOR.primary,
    fontWeight: '600',
  },
  checkRow: {
    alignItems: 'flex-end',
    marginTop: 2,
  },
  checkmark: {
    fontSize: 12,
    color: COLOR.onPrimary,
    opacity: 0.7,
  },
  messageTime: {
    fontSize: 11,
    marginTop: 4,
  },
  ownTime: {
    color: COLOR.onSurfaceVariant,
    marginRight: 4,
  },
  otherTime: {
    color: COLOR.onSurfaceVariant,
    marginLeft: 4,
  },
  typingIndicator: {
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  typingText: {
    fontSize: 12,
    color: COLOR.onSurfaceVariant,
    fontStyle: 'italic',
  },
  inputArea: {
    backgroundColor: COLOR.surface,
    borderTopWidth: 1,
    borderTopColor: COLOR.outlineVariant,
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
    paddingHorizontal: 4,
  },
  toggleLabel: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  toggleIcon: {
    fontSize: 18,
    marginRight: 6,
  },
  toggleText: {
    fontSize: 13,
    fontWeight: '600',
    color: COLOR.onSurfaceVariant,
  },
  toggleTextActive: {
    color: COLOR.primary,
  },
  switch: {
    width: 40,
    height: 20,
    borderRadius: 10,
    backgroundColor: COLOR.surfaceContainerLow,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    justifyContent: 'center',
  },
  switchOn: {
    backgroundColor: COLOR.primary,
    borderColor: COLOR.primary,
  },
  switchThumb: {
    width: 16,
    height: 16,
    borderRadius: 8,
    backgroundColor: '#fff',
    shadowColor: '#000',
    shadowOpacity: 0.2,
    shadowRadius: 2,
    shadowOffset: { width: 0, height: 1 },
    marginLeft: 2,
  },
  switchThumbOn: {
    marginLeft: 22,
  },
  livePreview: {
    marginBottom: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(107, 56, 212, 0.3)',
    backgroundColor: 'rgba(233, 221, 255, 0.25)',
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  livePreviewLabel: {
    fontSize: 11,
    fontWeight: '600',
    color: COLOR.secondary,
    marginBottom: 2,
  },
  livePreviewText: {
    fontSize: 14,
    fontStyle: 'italic',
    color: COLOR.onSurfaceVariant,
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
  },
  iconButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  iconButtonText: {
    fontSize: 20,
    color: COLOR.primary,
  },
  input: {
    flex: 1,
    backgroundColor: COLOR.surfaceContainerLow,
    borderRadius: 24,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    paddingHorizontal: 16,
    paddingVertical: 10,
    marginHorizontal: 6,
    maxHeight: 100,
    fontSize: 16,
    color: COLOR.onSurface,
  },
  sendButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: COLOR.primary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendButtonDisabled: {
    opacity: 0.4,
  },
  sendButtonText: {
    color: COLOR.onPrimary,
    fontSize: 18,
    lineHeight: 20,
  },
  sparkyFab: {
    position: 'absolute',
    bottom: 96,
    right: 16,
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: COLOR.secondary,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: COLOR.secondary,
    shadowOpacity: 0.35,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 4 },
    elevation: 6,
  },
  sparkyFabIcon: {
    fontSize: 28,
  },
  sparkyDot: {
    position: 'absolute',
    top: 4,
    right: 4,
    width: 12,
    height: 12,
    borderRadius: 6,
    backgroundColor: COLOR.tertiary,
    borderWidth: 2,
    borderColor: COLOR.surface,
  },
  sheetOverlay: {
    flex: 1,
    justifyContent: 'flex-end',
  },
  sheetBackdrop: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(11, 28, 48, 0.4)',
  },
  sheet: {
    backgroundColor: COLOR.surface,
    borderTopLeftRadius: 32,
    borderTopRightRadius: 32,
    paddingHorizontal: 16,
    paddingBottom: 24,
    maxHeight: '85%',
  },
  sheetHandle: {
    alignSelf: 'center',
    width: 48,
    height: 5,
    borderRadius: 3,
    backgroundColor: COLOR.outlineVariant,
    marginTop: 10,
    marginBottom: 8,
  },
  sheetHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
  },
  sheetTitleRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  sheetTitleIcon: {
    fontSize: 18,
    marginRight: 8,
  },
  sheetTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
  },
  sheetClose: {
    fontSize: 18,
    color: COLOR.onSurfaceVariant,
    padding: 4,
  },
  sheetBody: {
    paddingVertical: 16,
  },
  sheetIntro: {
    alignItems: 'center',
    marginBottom: 16,
  },
  sheetIntroTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
    marginBottom: 4,
  },
  sheetIntroText: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
    fontFamily: FONTS.body,
  },
  sheetTutorBubble: {
    alignSelf: 'flex-start',
    maxWidth: '85%',
    backgroundColor: COLOR.surfaceContainerLow,
    borderRadius: 16,
    borderBottomLeftRadius: 4,
    padding: 12,
  },
  sheetTutorText: {
    fontSize: 14,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
    lineHeight: 20,
  },
  sheetInputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    borderTopWidth: 1,
    borderTopColor: COLOR.outlineVariant,
    paddingTop: 12,
  },
  sheetInput: {
    flex: 1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 10,
    marginRight: 8,
    maxHeight: 100,
    fontSize: 14,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
  },
  sheetSend: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: COLOR.secondary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sheetSendText: {
    color: '#fff',
    fontSize: 18,
    lineHeight: 20,
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
    color: COLOR.onSurface,
    marginBottom: 8,
  },
  emptySubtext: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
  },
});