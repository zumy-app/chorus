/* eslint-env jest */
// Mocks for native modules that react-native's jest preset cannot load.
jest.mock('react-native-screens', () => {
  const actual = jest.requireActual('react-native-screens');
  return {
    ...actual,
    enableScreens: jest.fn(() => false),
    screensEnabled: jest.fn(() => false),
  };
});

jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest')
);