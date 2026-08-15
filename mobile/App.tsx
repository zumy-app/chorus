import React, { useEffect, useState } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import storage from './src/utils/storage';
import { ActivityIndicator, View, StyleSheet } from 'react-native';

import LandingScreen from './src/screens/LandingScreen';
import PricingScreen from './src/screens/PricingScreen';
import AboutScreen from './src/screens/AboutScreen';
import LoginScreen from './src/screens/LoginScreen';
import RegisterScreen from './src/screens/RegisterScreen';
import ChatListScreen from './src/screens/ChatListScreen';
import ChatScreen from './src/screens/ChatScreen';
import NewChatScreen from './src/screens/NewChatScreen';
import ProfileScreen from './src/screens/ProfileScreen';

export type RootStackParamList = {
  Landing: undefined;
  Pricing: undefined;
  About: undefined;
  Login: undefined;
  Register: undefined;
  ChatList: undefined;
  Chat: { chatId: string; chatName: string };
  NewChat: undefined;
  Profile: undefined;
};

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function App() {
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    try {
      const token = await storage.getItem('accessToken');
      setIsAuthenticated(!!token);
    } catch (error) {
      console.error('Auth check failed:', error);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#007AFF" />
      </View>
    );
  }

  return (
    <NavigationContainer>
      <Stack.Navigator
        initialRouteName={isAuthenticated ? 'ChatList' : 'Landing'}
        screenOptions={{
          headerStyle: {
            backgroundColor: '#007AFF',
          },
          headerTintColor: '#fff',
          headerTitleStyle: {
            fontWeight: 'bold',
          },
        }}>
        <Stack.Screen
          name="Landing"
          component={LandingScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="Pricing"
          component={PricingScreen}
          options={{
            title: 'Pricing',
            headerStyle: { backgroundColor: '#4F46E5' },
          }}
        />
        <Stack.Screen
          name="About"
          component={AboutScreen}
          options={{
            title: 'About',
            headerStyle: { backgroundColor: '#4F46E5' },
          }}
        />
        <Stack.Screen
          name="Login"
          component={LoginScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="Register"
          component={RegisterScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="ChatList"
          component={ChatListScreen}
          options={{ title: 'Chorus', headerLeft: () => null }}
        />
        <Stack.Screen
          name="Chat"
          component={ChatScreen}
          options={{ title: '' }}
        />
        <Stack.Screen
          name="NewChat"
          component={NewChatScreen}
          options={{ title: 'New Chat' }}
        />
        <Stack.Screen
          name="Profile"
          component={ProfileScreen}
          options={{ title: 'Profile' }}
        />
      </Stack.Navigator>
    </NavigationContainer>
  );
}

const styles = StyleSheet.create({
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
});
