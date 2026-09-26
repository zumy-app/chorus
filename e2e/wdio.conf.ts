import type { Options } from '@wdio/types'
// Minimal WebDriverIO harness for AVD (UiAutomator2) — 10.0.2.2 host
// Usage: npx wdio run e2e/wdio.conf.ts (requires Appium server on :4723)
export const config: Options.Testrunner = {
  runner: 'local',
  specs: ['./tests/99-comprehensive-two-user.spec.ts'],
  maxInstances: 1,
  capabilities: [
    {
      platformName: 'Android',
      'appium:deviceName': 'emulator-5554',
      'appium:automationName': 'UiAutomator2',
      'appium:app': '../mobile/android/app/build/outputs/apk/debug/app-debug.apk',
      'appium:ensureWebviewsHavePages': true,
    } as any,
  ],
  logLevel: 'info',
  services: ['appium'],
  framework: 'mocha',
  hostname: '127.0.0.1',
  port: 4723,
  path: '/wd/hub',
}
