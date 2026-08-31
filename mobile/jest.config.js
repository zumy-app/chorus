module.exports = {
  preset: 'react-native',
  setupFiles: ['./jest.setup.js'],
  moduleNameMapper: {
    '^@chorus/shared$': '<rootDir>/../packages/shared/src/index.ts',
  },
  transformIgnorePatterns: [
    'node_modules/(?!(react-native|@react-native|@react-navigation|react-native-screens|react-native-safe-area-context|@react-native-async-storage|react-native-web|@testing-library)/)',
  ],
};
