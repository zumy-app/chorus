import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Alert,
} from 'react-native';
import apiService from '../services/api';
import { User } from '@chorus/shared';

export default function NewChatScreen({ navigation }: any) {
  const [query, setQuery] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [searching, setSearching] = useState(false);
  const [creating, setCreating] = useState(false);

  const handleSearch = async (q: string) => {
    setQuery(q);
    if (!q.trim()) {
      setUsers([]);
      return;
    }
    setSearching(true);
    try {
      const result = await apiService.searchUsers(q.trim());
      // searchUsers already returns the users array (unwrapped).
      const found = result;
      setUsers(found);
    } catch {
      // Backend unreachable or unauthorized.
    } finally {
      setSearching(false);
    }
  };

  const startChat = async (user: User) => {
    if (creating) return;
    setCreating(true);
    try {
      const chat = await apiService.createChat({
        type: 'direct',
        participants: [user.id],
      });
      navigation.replace('Chat', { chatId: chat.id, chatName: user.displayName });
    } catch {
      Alert.alert('Error', 'Could not start a chat with this user.');
    } finally {
      setCreating(false);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.searchBox}>
        <TextInput
          style={styles.input}
          placeholder="Search by name or username"
          value={query}
          onChangeText={handleSearch}
          autoCapitalize="none"
          autoCorrect={false}
        />
      </View>

      {searching && (
        <ActivityIndicator style={styles.loader} color="#007AFF" />
      )}

      <FlatList
        data={users}
        keyExtractor={(item) => item.id}
        keyboardShouldPersistTaps="handled"
        ListEmptyComponent={
          !searching ? (
            <View style={styles.emptyState}>
              <Text style={styles.emptyText}>
                {query.trim() ? 'No users found' : 'Search for someone to chat with'}
              </Text>
            </View>
          ) : null
        }
        renderItem={({ item }) => (
          <TouchableOpacity
            style={styles.userItem}
            onPress={() => startChat(item)}
            disabled={creating}>
            <View style={styles.avatar}>
              <Text style={styles.avatarText}>
                {item.displayName?.charAt(0).toUpperCase() || '?'}
              </Text>
            </View>
            <View style={styles.userInfo}>
              <Text style={styles.userName}>{item.displayName}</Text>
              <Text style={styles.userUsername}>@{item.username}</Text>
            </View>
          </TouchableOpacity>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  searchBox: {
    padding: 12,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  input: {
    backgroundColor: '#f5f5f5',
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontSize: 16,
  },
  loader: {
    marginTop: 24,
  },
  userItem: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  avatar: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#007AFF',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  avatarText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: 'bold',
  },
  userInfo: {
    flex: 1,
  },
  userName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333',
  },
  userUsername: {
    fontSize: 13,
    color: '#888',
  },
  emptyState: {
    paddingTop: 60,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 14,
    color: '#888',
  },
});