/* eslint-env jest */
// React 19 concurrent-mode tests need this flag for act() to flush effects.
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

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
