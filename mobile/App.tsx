import React, { useEffect, useState } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import storage from './src/utils/storage';
import featureFlags from './src/utils/featureFlags';
import { LanguageProvider, applyImplicitLanguage, useStrings } from './src/i18n';
import { ActivityIndicator, View, StyleSheet } from 'react-native';

import LandingScreen from './src/screens/LandingScreen';
import PricingScreen from './src/screens/PricingScreen';
import AboutScreen from './src/screens/AboutScreen';
import LoginScreen from './src/screens/LoginScreen';
import RegisterScreen from './src/screens/RegisterScreen';
import ForgotPasswordScreen from './src/screens/ForgotPasswordScreen';
import ResetPasswordScreen from './src/screens/ResetPasswordScreen';
import MainTabs from './src/components/MainTabs';
import { COLOR } from './src/theme';

export type RootStackParamList = {
  Landing: undefined;
  Pricing: undefined;
  About: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: { token?: string };
  MainTabs: undefined;
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
      if (token) {
        await featureFlags.init();
        // Restored session: UI follows the profile's native language unless
        // the user explicitly picked one (persisted choice wins).
        try {
          const userStr = await storage.getItem('user');
          const native = userStr ? JSON.parse(userStr)?.nativeLanguage : null;
          await applyImplicitLanguage(native);
        } catch {}
      }
      setIsAuthenticated(!!token);
    } catch (error) {
      console.error('Auth check failed:', error);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <LanguageProvider>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color={COLOR.primary} />
        </View>
      </LanguageProvider>
    );
  }

  return (
    <LanguageProvider>
    <NavigationContainer>
      <RootStack isAuthenticated={isAuthenticated} />
    </NavigationContainer>
    </LanguageProvider>
  );
}

// Inner component so stack header titles follow the active UI language.
function RootStack({ isAuthenticated }: { isAuthenticated: boolean }) {
  const s = useStrings();
  return (
    <Stack.Navigator
      initialRouteName={isAuthenticated ? 'MainTabs' : 'Landing'}
      screenOptions={{
        headerStyle: {
          backgroundColor: COLOR.surface,
        },
        headerTintColor: COLOR.onSurface,
        headerShadowVisible: false,
        headerTitleStyle: {
          fontSize: 20,
          fontWeight: '700' as const,
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
        options={{ title: s.nav.pricing }}
      />
      <Stack.Screen
        name="About"
        component={AboutScreen}
        options={{ title: s.nav.about }}
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
        name="ForgotPassword"
        component={ForgotPasswordScreen}
        options={{ headerShown: false }}
      />
      <Stack.Screen
        name="ResetPassword"
        component={ResetPasswordScreen}
        options={{ headerShown: false }}
      />
      <Stack.Screen
        name="MainTabs"
        component={MainTabs}
        options={{ headerShown: false }}
      />
    </Stack.Navigator>
  );
}

const styles = StyleSheet.create({
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
});
