import axios from 'axios';

// Android emulator uses 10.0.2.2 to access host machine's localhost.
// Override with CHORUS_API_BASE_URL to run from the host (CI) or elsewhere.
const API_BASE_URL = (process.env.CHORUS_API_BASE_URL || 'http://10.0.2.2:8080') + '/api/v1';
const HEALTH_URL = (process.env.CHORUS_API_BASE_URL || 'http://10.0.2.2:8080') + '/health';

interface SmokeTestResult {
  name: string;
  passed: boolean;
  message: string;
  duration: number;
}

class MobileSmokeTest {
  private results: SmokeTestResult[] = [];
  private testUsername: string = '';
  private testPassword: string = 'TestPass123!';
  private testAccessToken: string = '';
  private testChatId: string = '';

  async runAll(): Promise<void> {
    console.log('='.repeat(70));
    console.log('CHORUS MOBILE APP - SMOKE TESTS (Android Emulator)');
    console.log('='.repeat(70));
    console.log('');

    // Basic connectivity
    await this.runTest('Backend Connectivity (10.0.2.2:8080)', this.testBackendConnectivity.bind(this));
    await this.runTest('Health Check Endpoint', this.testHealthCheck.bind(this));

    // Create test user
    await this.runTest('Create Test User', this.testCreateUser.bind(this));
    await this.runTest('Login with Test User', this.testLogin.bind(this));
    
    // Test main features
    await this.runTest('Get User Profile', this.testGetProfile.bind(this));
    await this.runTest('Create Test Chat', this.testCreateChat.bind(this));
    await this.runTest('Send Test Message', this.testSendMessage.bind(this));
    await this.runTest('Retrieve Messages', this.testGetMessages.bind(this));
    await this.runTest('Get Chat List', this.testGetChats.bind(this));

    this.printSummary();
    this.printMobileAppInstructions();
  }

  private async runTest(name: string, testFn: () => Promise<void>): Promise<void> {
    const startTime = Date.now();
    try {
      await testFn();
      const duration = Date.now() - startTime;
      this.results.push({ name, passed: true, message: 'Success', duration });
      console.log(`✅ ${name} - PASSED (${duration}ms)`);
    } catch (error: any) {
      const duration = Date.now() - startTime;
      const message = error.response?.data?.error || error.message || 'Unknown error';
      this.results.push({ name, passed: false, message, duration });
      console.log(`❌ ${name} - FAILED (${duration}ms)`);
      console.log(`   Error: ${message}`);
    }
    console.log('');
  }

  private async testBackendConnectivity(): Promise<void> {
    try {
      await axios.get(HEALTH_URL, { timeout: 5000 });
    } catch (error: any) {
      if (error.code === 'ECONNREFUSED') {
        throw new Error('Backend not accessible. Make sure Docker services are running.');
      }
      throw error;
    }
  }

  private async testHealthCheck(): Promise<void> {
    const response = await axios.get(HEALTH_URL);
    if (response.data.status !== 'healthy') {
      throw new Error('Health check failed');
    }
  }

  /**
   * Registration is invite-gated: mint an open SMS invite as the seeded
   * fixture, then register the throwaway with its single-use token.
   */
  private async testCreateUser(): Promise<void> {
    const timestamp = Date.now();
    this.testUsername = `mobiletest_${timestamp}`;

    const fixture = await axios.post(`${API_BASE_URL}/auth/login`, {
      username: 'alice.en-es@chorus.test',
      password: 'ChorusDev123!',
    });
    const invite = await axios.post(
      `${API_BASE_URL}/contacts/invites`,
      { channel: 'sms', contact: { name: 'mobiletest', phone: `+1555${String(timestamp).slice(-7)}` } },
      { headers: { Authorization: `Bearer ${fixture.data.tokens.accessToken}` } }
    );
    const inviteToken = invite.data?.data?.token;
    if (!inviteToken) throw new Error('invite mint did not return a token');

    const response = await axios.post(`${API_BASE_URL}/auth/register`, {
      username: this.testUsername,
      email: `${this.testUsername}@example.com`,
      password: this.testPassword,
      displayName: 'Mobile Test User',
      nativeLanguage: 'en',
      targetLanguages: ['es'],
      inviteToken,
    });

    if (!response.data.user || !response.data.tokens?.accessToken) {
      throw new Error('Registration did not return expected data');
    }

    this.testAccessToken = response.data.tokens.accessToken;
  }

  private async testLogin(): Promise<void> {
    const response = await axios.post(`${API_BASE_URL}/auth/login`, {
      username: this.testUsername,
      password: this.testPassword,
    });

    if (!response.data.tokens?.accessToken) {
      throw new Error('Login did not return access token');
    }
  }

  private async testGetProfile(): Promise<void> {
    const response = await axios.get(`${API_BASE_URL}/users/me`, {
      headers: { Authorization: `Bearer ${this.testAccessToken}` },
    });

    if (!response.data.username || response.data.username !== this.testUsername) {
      throw new Error('Profile data mismatch');
    }
  }

  private async testCreateChat(): Promise<void> {
    // Direct chats need a peer (participants min=1): resolve the seeded
    // ES learner via search.
    const search = await axios.get(`${API_BASE_URL}/users/search`, {
      params: { q: 'bob.es-en@chorus.test' },
      headers: { Authorization: `Bearer ${this.testAccessToken}` },
    });
    const users = search.data.users || search.data.data || [];
    const bob = users.find((u: any) => u.email === 'bob.es-en@chorus.test');
    if (!bob?.id) throw new Error('could not resolve peer user for DM (is the dev DB seeded?)');
    const response = await axios.post(
      `${API_BASE_URL}/chats`,
      {
        type: 'direct',
        participants: [bob.id],
      },
      {
        headers: { Authorization: `Bearer ${this.testAccessToken}` },
      }
    );

    if (!response.data.id) {
      throw new Error('Chat creation failed');
    }

    this.testChatId = response.data.id;
  }

  private async testSendMessage(): Promise<void> {
    const response = await axios.post(
      `${API_BASE_URL}/chats/${this.testChatId}/messages`,
      {
        text: 'Hello from mobile app smoke test! 📱',
      },
      {
        headers: { Authorization: `Bearer ${this.testAccessToken}` },
      }
    );

    if (!response.data.id || !response.data.text) {
      throw new Error('Message sending failed');
    }
  }

  private async testGetMessages(): Promise<void> {
    const response = await axios.get(
      `${API_BASE_URL}/chats/${this.testChatId}/messages`,
      {
        headers: { Authorization: `Bearer ${this.testAccessToken}` },
      }
    );

    if (!response.data.messages || !Array.isArray(response.data.messages)) {
      throw new Error('Failed to retrieve messages');
    }

    if (response.data.messages.length === 0) {
      throw new Error('No messages found');
    }
  }

  private async testGetChats(): Promise<void> {
    const response = await axios.get(`${API_BASE_URL}/chats`, {
      headers: { Authorization: `Bearer ${this.testAccessToken}` },
    });

    if (!response.data.chats || !Array.isArray(response.data.chats)) {
      throw new Error('Failed to retrieve chats');
    }
  }

  private printSummary(): void {
    console.log('='.repeat(70));
    console.log('TEST SUMMARY');
    console.log('='.repeat(70));
    
    const passed = this.results.filter(r => r.passed).length;
    const failed = this.results.filter(r => !r.passed).length;
    const total = this.results.length;
    const totalDuration = this.results.reduce((sum, r) => sum + r.duration, 0);

    console.log(`Total Tests: ${total}`);
    console.log(`Passed: ${passed}`);
    console.log(`Failed: ${failed}`);
    console.log(`Total Duration: ${totalDuration}ms`);
    console.log('');

    if (failed > 0) {
      console.log('Failed Tests:');
      this.results.filter(r => !r.passed).forEach(r => {
        console.log(`  - ${r.name}: ${r.message}`);
      });
      console.log('');
    }

    if (passed === total) {
      console.log('🎉 All smoke tests PASSED! The mobile app backend is ready.');
    } else {
      console.log('⚠️  Some tests failed. Please fix before using the mobile app.');
    }

    console.log('='.repeat(70));
  }

  failedCount(): number {
    return this.results.filter((r) => !r.passed).length;
  }

  private printMobileAppInstructions(): void {
    console.log('');
    console.log('='.repeat(70));
    console.log('MOBILE APP TESTING INSTRUCTIONS');
    console.log('='.repeat(70));
    console.log('');
    console.log('1. VERIFY APP IS RUNNING:');
    console.log('   - Check the Android emulator window');
    console.log('   - The Chorus app should be open');
    console.log('   - You should see the Login or ChatList screen');
    console.log('');
    console.log('2. TEST LOGIN:');
    console.log('   - Username: ' + this.testUsername);
    console.log('   - Password: ' + this.testPassword);
    console.log('   - Tap "Sign In"');
    console.log('   - Should navigate to ChatList screen');
    console.log('');
    console.log('3. TEST CHAT:');
    console.log('   - You should see your test chat in the list');
    console.log('   - Tap to open it');
    console.log('   - You should see the test message: "Hello from mobile app smoke test! 📱"');
    console.log('   - Try sending a new message');
    console.log('');
    console.log('4. TEST REGISTRATION:');
    console.log('   - Logout from the app');
    console.log('   - Tap "Register"');
    console.log('   - Fill in the form and create a new account');
    console.log('   - Should auto-login and show ChatList');
    console.log('');
    console.log('5. CHECK LOGS:');
    console.log('   - View Metro bundler output for any errors');
    console.log('   - Use "adb logcat" for native Android logs');
    console.log('');
    console.log('='.repeat(70));
  }
}

// Run smoke tests
const runner = new MobileSmokeTest();
runner.runAll()
  .then(() => {
    console.log('\nSmoke tests completed!');
    process.exit(runner.failedCount() > 0 ? 1 : 0);
  })
  .catch((error) => {
    console.error('Test runner failed:', error);
    process.exit(1);
  });
