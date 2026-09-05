import React, { useState, useEffect, useMemo } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
} from 'react-native';
import apiService from '../services/api';
import webSocketService from '../services/websocket';
import storage from '../utils/storage';
import { Chat } from '@chorus/shared';
import { COLOR, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';

export default function ChatListScreen({ navigation }: any) {
  const [chats, setChats] = useState<Chat[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [currentUserId, setCurrentUserId] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadCurrentUser();
    loadChats();
    webSocketService.connect();

    const unsubscribeMessage = webSocketService.onMessage((message) => {
      // New messages and finished translations change the chat list preview.
      if (message.type === 'new_message' || message.type === 'message_updated') {
        loadChats();
      }
    });

    const unsubscribeReconnect = webSocketService.onReconnect(() => {
      loadChats();
    });

    navigation.setOptions({
      // React Navigation requires a render callback here (standard pattern).
      // eslint-disable-next-line react/no-unstable-nested-components
      headerRight: () => (
        <ProfileHeaderButton onPress={() => navigation.navigate('Profile')} />
      ),
    });

    return () => {
      unsubscribeMessage();
      unsubscribeReconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadCurrentUser = async () => {
    const userStr = await storage.getItem('user');
    if (userStr) {
      try {
        setCurrentUserId(JSON.parse(userStr).id);
      } catch {
        // Corrupted storage — ignore.
      }
    }
  };

  const loadChats = async () => {
    try {
      const data = await apiService.getChats();
      setChats(data);
    } catch {
      // Backend unreachable or unauthorized — silently ignore on mount.
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = () => {
    setRefreshing(true);
    loadChats();
  };

  const filteredChats = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return chats;
    const nameOf = (chat: Chat) => {
      if (chat.name) return chat.name;
      if (chat.type === 'direct' && chat.participants) {
        const other = chat.participants.find((p) => p.user?.id !== currentUserId);
        if (other?.user?.displayName) return other.user.displayName;
      }
      return 'Group Chat';
    };
    return chats.filter((chat) => {
      const name = nameOf(chat).toLowerCase();
      const preview = (chat.lastMessage?.text || '').toLowerCase();
      return name.includes(q) || preview.includes(q);
    });
  }, [chats, searchQuery, currentUserId]);

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
    return date.toLocaleDateString();
  };

  const getChatName = (chat: Chat) => {
    if (chat.name) return chat.name;
    if (chat.type === 'direct' && chat.participants) {
      // Direct chats are titled with the OTHER participant.
      const other = chat.participants.find((p) => p.user?.id !== currentUserId);
      if (other?.user?.displayName) return other.user.displayName;
    }
    return 'Group Chat';
  };

  const getLangCode = (chat: Chat) => {
    if (chat.type === 'direct' && chat.participants) {
      const other = chat.participants.find((p) => p.user?.id !== currentUserId);
      const code =
        other?.user?.targetLanguages?.[0] ||
        other?.user?.nativeLanguage ||
        undefined;
      if (code) return code.slice(0, 2).toUpperCase();
    }
    return undefined;
  };

  const isUnread = (chat: Chat) => Boolean(chat.unreadCount && chat.unreadCount > 0);

  const renderChatItem = ({ item }: { item: Chat }) => {
    const langCode = getLangCode(item);
    const unread = isUnread(item);
    return (
    <TouchableOpacity
      style={styles.chatItem}
      onPress={() => navigation.navigate('Chat', { chatId: item.id, chatName: getChatName(item) })}>
      <View style={styles.avatarWrap}>
        <View
          style={[
            styles.chatAvatar,
            item.type === 'group' && styles.chatAvatarSquare,
          ]}>
          <Text style={styles.chatAvatarText}>👤</Text>
        </View>
        {langCode && (
          <View style={styles.langBadge}>
            <Text style={styles.langBadgeText}>{langCode}</Text>
          </View>
        )}
      </View>
      <View style={styles.chatInfo}>
        <View style={styles.chatHeader}>
          <Text
            style={[styles.chatName, unread && styles.chatNameUnread]}
            numberOfLines={1}>
            {getChatName(item)}
          </Text>
          {item.lastMessage && (
            <Text
              style={[
                styles.chatTime,
                unread && styles.chatTimeUnread,
              ]}>
              {formatTime(item.lastMessage.timestamp)}
            </Text>
          )}
        </View>
        <View style={styles.chatFooter}>
          <Text
            style={[
              styles.chatPreview,
              unread && styles.chatPreviewUnread,
            ]}
            numberOfLines={1}>
            {item.lastMessage?.text || 'No messages yet'}
          </Text>
          {unread && (
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{item.unreadCount ?? ''}</Text>
            </View>
          )}
        </View>
        {langCode && (
          <View style={styles.langRow}>
            <Text style={styles.langLabel}>
              {item.type === 'group' ? 'Group chat' : `Learning ${langCode.toLowerCase()}`}
            </Text>
          </View>
        )}
      </View>
    </TouchableOpacity>
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
    <View style={styles.container}>
      <FlatList
        data={filteredChats}
        renderItem={renderChatItem}
        keyExtractor={(item) => item.id}
        contentContainerStyle={filteredChats.length === 0 ? styles.emptyContainer : styles.listContent}
        ListHeaderComponent={
          <View>
            {/* Search Bar */}
            <View style={styles.searchWrap}>
              <Text style={styles.searchIcon}>🔍</Text>
              <TextInput
                style={styles.searchInput}
                placeholder="Search chats or languages..."
                placeholderTextColor={COLOR.outlineVariant}
                value={searchQuery}
                onChangeText={setSearchQuery}
                autoCorrect={false}
              />
            </View>
            {/* Insights bento */}
            <View style={styles.bento}>
              <TouchableOpacity style={styles.bentoPrimary}>
                <Text style={styles.bentoIcon}>🧠</Text>
                <Text style={styles.bentoTitle}>Daily Review</Text>
                <Text style={styles.bentoSubtitle}>3 new vocab words</Text>
              </TouchableOpacity>
              <TouchableOpacity style={styles.bentoSecondary}>
                <Text style={styles.bentoIconSecondary}>💬</Text>
                <Text style={styles.bentoTitle}>Practice Prompt</Text>
                <Text style={styles.bentoSubtitle}>"Order coffee in Paris"</Text>
              </TouchableOpacity>
            </View>
            <Text style={styles.sectionHeader}>ACTIVE CONVERSATIONS</Text>
          </View>
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyText}>
              {searchQuery ? 'No matching chats' : 'No chats yet'}
            </Text>
            <Text style={styles.emptySubtext}>
              {searchQuery ? 'Try a different search.' : 'Start a conversation!'}
            </Text>
          </View>
        }
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={handleRefresh}
            tintColor={COLOR.primary}
          />
        }
      />
      <TouchableOpacity
        style={styles.fab}
        onPress={() => navigation.navigate('NewChat')}>
        <Text style={styles.fabText}>✏️</Text>
      </TouchableOpacity>
    </View>
  );
}

function ProfileHeaderButton({ onPress }: { onPress: () => void }) {
  return (
    <TouchableOpacity onPress={onPress} style={styles.headerButton}>
      <Text style={styles.headerButtonText}>⚙️</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  listContent: {
    paddingBottom: 120,
  },
  headerButton: {
    marginRight: 12,
    padding: 4,
  },
  headerButtonText: {
    fontSize: 18,
    color: COLOR.onSurface,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  searchWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: COLOR.surfaceContainerLowest,
    borderRadius: RADIUS.xl,
    marginHorizontal: SPACING.marginMobile,
    marginTop: SPACING.stackMd,
    paddingHorizontal: 16,
    paddingVertical: 12,
    ...SHADOWS.elevation1,
  },
  searchIcon: {
    fontSize: 18,
    marginRight: 10,
  },
  searchInput: {
    flex: 1,
    ...TYPOGRAPHY.bodyMd,
    color: COLOR.onSurface,
    padding: 0,
  },
  bento: {
    flexDirection: 'row',
    gap: 12,
    marginHorizontal: SPACING.marginMobile,
    marginTop: SPACING.stackMd,
  },
  bentoPrimary: {
    flex: 1,
    backgroundColor: COLOR.primaryContainer,
    borderRadius: RADIUS.xl,
    padding: SPACING.stackMd,
    height: 112,
    ...SHADOWS.elevation1,
  },
  bentoSecondary: {
    flex: 1,
    backgroundColor: COLOR.surfaceContainerHigh,
    borderRadius: RADIUS.xl,
    padding: SPACING.stackMd,
    height: 112,
    overflow: 'hidden',
    ...SHADOWS.elevation1,
  },
  bentoIcon: {
    fontSize: 22,
    color: COLOR.onPrimaryContainer,
  },
  bentoIconSecondary: {
    fontSize: 22,
    color: COLOR.secondary,
  },
  bentoTitle: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onPrimaryContainer,
    marginTop: 'auto',
  },
  bentoSubtitle: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onPrimaryContainer,
    opacity: 0.8,
  },
  sectionHeader: {
    ...TYPOGRAPHY.labelMd,
    color: COLOR.onSurfaceVariant,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    marginHorizontal: 20,
    marginTop: SPACING.stackLg,
    marginBottom: SPACING.stackSm,
  },
  chatItem: {
    flexDirection: 'row',
    paddingHorizontal: SPACING.marginMobile,
    paddingVertical: 12,
    gap: 16,
    backgroundColor: COLOR.background,
  },
  avatarWrap: {
    position: 'relative',
  },
  chatAvatar: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: COLOR.surfaceVariant,
    justifyContent: 'center',
    alignItems: 'center',
    ...SHADOWS.elevation1,
  },
  chatAvatarSquare: {
    borderRadius: RADIUS.xl,
  },
  chatAvatarText: {
    fontSize: 24,
  },
  langBadge: {
    position: 'absolute',
    right: -4,
    bottom: -4,
    backgroundColor: COLOR.surface,
    borderRadius: 10,
    padding: 2,
  },
  langBadgeText: {
    backgroundColor: COLOR.primaryContainer,
    color: COLOR.onPrimaryContainer,
    fontSize: 9,
    fontWeight: '700',
    borderRadius: 8,
    overflow: 'hidden',
    paddingHorizontal: 3,
    paddingVertical: 1,
  },
  chatInfo: {
    flex: 1,
    justifyContent: 'center',
    gap: 2,
  },
  chatHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'baseline',
    gap: 8,
  },
  chatName: {
    ...TYPOGRAPHY.headlineSm,
    color: COLOR.onSurface,
    flexShrink: 1,
  },
  chatNameUnread: {
    color: COLOR.primary,
  },
  chatTime: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.onSurfaceVariant,
  },
  chatTimeUnread: {
    color: COLOR.primary,
  },
  chatFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  chatPreview: {
    flex: 1,
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
  },
  chatPreviewUnread: {
    color: COLOR.onSurface,
    fontWeight: '600',
  },
  badge: {
    backgroundColor: COLOR.primary,
    borderRadius: 10,
    minWidth: 20,
    height: 20,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 6,
    marginLeft: 8,
  },
  badgeText: {
    color: COLOR.onPrimary,
    fontSize: 11,
    fontWeight: '700',
  },
  langRow: {
    marginTop: 2,
  },
  langLabel: {
    ...TYPOGRAPHY.labelSm,
    color: COLOR.outline,
  },
  emptyContainer: {
    flexGrow: 1,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
  },
  emptyText: {
    ...TYPOGRAPHY.headlineSm,
    color: COLOR.onSurface,
    marginBottom: 8,
  },
  emptySubtext: {
    ...TYPOGRAPHY.bodySm,
    color: COLOR.onSurfaceVariant,
  },
  fab: {
    position: 'absolute',
    right: SPACING.marginMobile,
    bottom: 88,
    width: 56,
    height: 56,
    borderRadius: 16,
    backgroundColor: COLOR.primary,
    justifyContent: 'center',
    alignItems: 'center',
    ...SHADOWS.elevation2,
    shadowColor: 'rgba(0,74,198,0.25)',
  },
  fabText: {
    color: COLOR.onPrimary,
    fontSize: 24,
  },
});
