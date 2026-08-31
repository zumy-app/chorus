module.exports = {
  presets: ['module:@react-native/babel-preset'],
  plugins: [
    // Inline process.env.* from the project's .env at bundle time. Without
    // this, Metro (react-native start) leaves process.env.EXPO_PUBLIC_*
    // undefined at runtime, so local-dev env settings (e.g. the prefilled
    // test login in LoginScreen) wouldn't reach the app.
    ['module:react-native-dotenv'],
  ],
};
