module.exports = {
  root: true,
  extends: '@react-native',
  overrides: [
    {
      // The account-switch outage was an `as any` cast hiding an undefined
      // `.api` member from tsc (ProfileScreen crashed on `raw.data`).
      // Keep the API-client boundary any-free so the type checker can see it.
      files: ['src/services/*.ts'],
      rules: { '@typescript-eslint/no-explicit-any': 'error' },
    },
  ],
};
