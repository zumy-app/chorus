import React from 'react';
import { Text, StyleSheet } from 'react-native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';

import ChatListScreen from '../screens/ChatListScreen';
import ChatScreen from '../screens/ChatScreen';
import NewChatScreen from '../screens/NewChatScreen';
import LearnScreen from '../screens/LearnScreen';
import PlacementScreen from '../screens/PlacementScreen';
import LessonSessionScreen from '../screens/LessonSessionScreen';
import VocabularyReviewScreen from '../screens/VocabularyReviewScreen';
import ScenariosScreen from '../screens/ScenariosScreen';
import ScenarioRoleplayScreen from '../screens/ScenarioRoleplayScreen';
import LearningRoadmapScreen from '../screens/LearningRoadmapScreen';
import RealTalkHubScreen from '../screens/RealTalkHubScreen';
import ProfileScreen from '../screens/ProfileScreen';
import BecomeTeacherScreen from '../screens/BecomeTeacherScreen';
import BrowseTutorsScreen from '../screens/BrowseTutorsScreen';
import TutorProfileScreen from '../screens/TutorProfileScreen';
import TrialCreditsScreen from '../screens/TrialCreditsScreen';
import TeacherDashboardScreen from '../screens/TeacherDashboardScreen';
import PayoutsScreen from '../screens/PayoutsScreen';
import ConfirmBookingScreen from '../screens/ConfirmBookingScreen';
import StreakRecoveryScreen from '../screens/StreakRecoveryScreen';
import AssignmentsScreen from '../screens/AssignmentsScreen';
import CallScreen from '../screens/CallScreen';
import UniversalSearchScreen from '../screens/UniversalSearchScreen';
import { COLOR, TYPOGRAPHY } from '../theme';
import { useStrings } from '../i18n';

export type MainTabsParamList = {
  ChatsTab: undefined;
  LearnTab: undefined;
  MarketplaceTab: undefined;
  ProfileTab: undefined;
};

export type ChatsStackParamList = {
  ChatList: undefined;
  Chat: { chatId: string; chatName: string };
  NewChat: undefined;
  UniversalSearch: undefined;
  Call: { callId: string; chatId: string; chatName: string };
};

export type LearnStackParamList = {
  Learn: undefined;
  Placement: undefined;
  LessonSession: { mode: string; sessionId?: string };
  VocabularyReview: undefined;
  Scenarios: undefined;
  ScenarioRoleplay: { scenarioId: string };
  LearningRoadmap: undefined;
  RealTalkHub: undefined;
  StreakRecovery: undefined;
  Assignments: undefined;
};

export type ProfileStackParamList = {
  Profile: undefined;
  BecomeTeacher: undefined;
};

export type MarketplaceStackParamList = {
  BrowseTutors: undefined;
  TutorProfile: { userId: string };
  ConfirmBooking: { userId: string };
  BecomeTeacher: undefined;
  TrialCredits: undefined;
  TeacherDashboard: undefined;
  Payouts: undefined;
};

const Tab = createBottomTabNavigator<MainTabsParamList>();
const ChatsStack = createNativeStackNavigator<ChatsStackParamList>();
const LearnStack = createNativeStackNavigator<LearnStackParamList>();
const ProfileStack = createNativeStackNavigator<ProfileStackParamList>();
const MarketplaceStack = createNativeStackNavigator<MarketplaceStackParamList>();

const stackOptions = {
  headerStyle: { backgroundColor: COLOR.surface },
  headerTintColor: COLOR.onSurface,
  headerShadowVisible: false,
  headerTitleStyle: { fontWeight: '700' as const },
};

// Each tab stack reads strings so header titles re-render on language change.
const ChatsTab = () => {
  const s = useStrings();
  return (
    <ChatsStack.Navigator screenOptions={stackOptions}>
      <ChatsStack.Screen
        name="ChatList"
        component={ChatListScreen}
        options={{ title: 'Chorus' }}
      />
      <ChatsStack.Screen
        name="Chat"
        component={ChatScreen}
        options={{ title: '' }}
      />
      <ChatsStack.Screen
        name="NewChat"
        component={NewChatScreen}
        options={{ title: s.nav.newChat }}
      />
      <ChatsStack.Screen
        name="Call"
        component={CallScreen}
        options={{ title: s.nav.call, headerShown: false, presentation: 'fullScreenModal' }}
      />
      <ChatsStack.Screen
        name="UniversalSearch"
        component={UniversalSearchScreen}
        options={{ title: s.nav.search }}
      />
    </ChatsStack.Navigator>
  );
};

const LearnTab = () => {
  const s = useStrings();
  return (
    <LearnStack.Navigator screenOptions={stackOptions}>
      <LearnStack.Screen
        name="Learn"
        component={LearnScreen}
        options={{ title: s.nav.learn }}
      />
      <LearnStack.Screen
        name="Placement"
        component={PlacementScreen}
        options={{ title: s.nav.placementTest }}
      />
      <LearnStack.Screen
        name="LessonSession"
        component={LessonSessionScreen}
        options={{ title: s.nav.practice }}
      />
      <LearnStack.Screen
        name="VocabularyReview"
        component={VocabularyReviewScreen}
        options={{ title: s.nav.vocabulary }}
      />
      <LearnStack.Screen
        name="Scenarios"
        component={ScenariosScreen}
        options={{ title: s.nav.scenarios }}
      />
      <LearnStack.Screen
        name="ScenarioRoleplay"
        component={ScenarioRoleplayScreen}
        options={{ title: s.nav.roleplay }}
      />
      <LearnStack.Screen
        name="LearningRoadmap"
        component={LearningRoadmapScreen}
        options={{ title: s.nav.roadmap }}
      />
      <LearnStack.Screen
        name="RealTalkHub"
        component={RealTalkHubScreen}
        options={{ title: s.nav.realTalk }}
      />
      <LearnStack.Screen
        name="StreakRecovery"
        component={StreakRecoveryScreen}
        options={{ title: s.nav.recoverStreak }}
      />
      <LearnStack.Screen
        name="Assignments"
        component={AssignmentsScreen}
        options={{ title: s.nav.assignments }}
      />
    </LearnStack.Navigator>
  );
};

const ProfileTab = () => {
  const s = useStrings();
  return (
    <ProfileStack.Navigator screenOptions={stackOptions}>
      <ProfileStack.Screen
        name="Profile"
        component={ProfileScreen}
        options={{ title: s.nav.profile }}
      />
      <ProfileStack.Screen
        name="BecomeTeacher"
        component={BecomeTeacherScreen}
        options={{ title: s.nav.becomeTeacher }}
      />
    </ProfileStack.Navigator>
  );
};

const MarketplaceTab = () => {
  const s = useStrings();
  return (
    <MarketplaceStack.Navigator screenOptions={stackOptions}>
      <MarketplaceStack.Screen
        name="BrowseTutors"
        component={BrowseTutorsScreen}
        options={{ title: s.nav.tutors }}
      />
      <MarketplaceStack.Screen
        name="TutorProfile"
        component={TutorProfileScreen}
        options={{ title: s.nav.tutor }}
      />
      <MarketplaceStack.Screen name="ConfirmBooking" component={ConfirmBookingScreen} options={{ title: s.nav.confirmBooking }} />
      <MarketplaceStack.Screen
        name="BecomeTeacher"
        component={BecomeTeacherScreen}
        options={{ title: s.nav.becomeTeacher }}
      />
      <MarketplaceStack.Screen name="TrialCredits" component={TrialCreditsScreen} options={{ title: s.nav.trialCredits }} />
      <MarketplaceStack.Screen name="TeacherDashboard" component={TeacherDashboardScreen} options={{ title: s.nav.dashboard }} />
      <MarketplaceStack.Screen name="Payouts" component={PayoutsScreen} options={{ title: s.nav.payouts }} />
    </MarketplaceStack.Navigator>
  );
};

function TabIcon({ focused, glyph }: { focused: boolean; glyph: string }) {
  return <Text style={[styles.tabIcon, focused && styles.tabIconFocused]}>{glyph}</Text>;
}

const TabIconChats = (props: { focused: boolean }) => <TabIcon {...props} glyph="💬" />;
const TabIconLearn = (props: { focused: boolean }) => <TabIcon {...props} glyph="🎓" />;
const TabIconMarketplace = (props: { focused: boolean }) => <TabIcon {...props} glyph="🏫" />;
const TabIconProfile = (props: { focused: boolean }) => <TabIcon {...props} glyph="👤" />;

const TAB_ICONS: Record<keyof MainTabsParamList, (props: { focused: boolean }) => React.JSX.Element> = {
  ChatsTab: TabIconChats,
  LearnTab: TabIconLearn,
  MarketplaceTab: TabIconMarketplace,
  ProfileTab: TabIconProfile,
};

export default function MainTabs() {
  const s = useStrings();
  const tabs = [
    { name: 'ChatsTab' as const, component: ChatsTab, label: s.nav.chats },
    { name: 'LearnTab' as const, component: LearnTab, label: s.nav.learn },
    { name: 'MarketplaceTab' as const, component: MarketplaceTab, label: s.nav.tutors },
    { name: 'ProfileTab' as const, component: ProfileTab, label: s.nav.profile },
  ];
  return (
    <Tab.Navigator
      initialRouteName="ChatsTab"
      screenOptions={{
        headerShown: false,
        tabBarStyle: styles.tabBar,
        tabBarActiveTintColor: COLOR.primary,
        tabBarInactiveTintColor: COLOR.onSurfaceVariant,
        tabBarLabelStyle: {
          ...TYPOGRAPHY.labelMd,
          fontSize: 12,
        },
      }}>
      {tabs.map((tab) => (
        <Tab.Screen
          key={tab.name}
          name={tab.name}
          component={tab.component}
          options={{
            title: tab.label,
            tabBarIcon: TAB_ICONS[tab.name],
          }}
        />
      ))}
    </Tab.Navigator>
  );
}

const styles = StyleSheet.create({
  tabBar: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderTopColor: COLOR.outlineVariant,
    borderTopWidth: 1,
    height: 72,
    paddingTop: 6,
    paddingBottom: 8,
  },
  tabIcon: {
    fontSize: 22,
    opacity: 0.55,
  },
  tabIconFocused: {
    opacity: 1,
  },
});
