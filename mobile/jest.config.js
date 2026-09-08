module.exports = {
  preset: 'react-native',
  setupFiles: ['./jest.setup.js'],
  moduleDirectories: ['node_modules', '<rootDir>/node_modules', '<rootDir>/../node_modules'],
  moduleNameMapper: {
    '^@chorus/shared$': '<rootDir>/../packages/shared/src/index.ts',
    '^@babel/runtime/(.*)$': '<rootDir>/node_modules/@babel/runtime/$1',
  },
  transformIgnorePatterns: [
    'node_modules/(?!(react-native|@react-native|@react-navigation|react-native-screens|react-native-safe-area-context|@react-native-async-storage|react-native-web|@testing-library)/)',
  ],
};
