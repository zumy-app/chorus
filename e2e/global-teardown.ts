/**
 * Global teardown: runs once after all tests complete.
 *
 * By default, we leave services running so developers can inspect state
 * and re-run tests quickly. Set E2E_STOP_SERVICES=true to tear down.
 */

export default async function globalTeardown() {
  console.log('\n🧹 Chorus E2E Global Teardown\n')

  if (process.env.E2E_STOP_SERVICES === 'true') {
    const { execSync } = await import('child_process')
    const { resolve } = await import('path')

    console.log('Stopping Chorus services...')
    try {
      execSync('docker-compose down', {
        cwd: resolve(__dirname, '..'),
        stdio: 'inherit',
        timeout: 60_000,
      })
      console.log('✅ Services stopped.')
    } catch (err) {
      console.warn('⚠️  Failed to stop services:', err)
    }
  } else {
    console.log('ℹ️  Leaving services running. Set E2E_STOP_SERVICES=true to stop them.')
  }

  console.log('\n✅ Teardown complete.\n')
}