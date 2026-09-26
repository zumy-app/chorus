import axios from 'axios';

// Same-host backend by default; override with CHORUS_API_BASE_URL for CI or
// remote hosts (e.g. the dev PC from a physical device).
const API_BASE_URL = process.env.CHORUS_API_BASE_URL || 'http://localhost:8080/api/v1';
const HEALTH_URL = (process.env.CHORUS_API_BASE_URL || 'http://localhost:8080') + '/health';

interface TestResult {
  name: string;
  passed: boolean;
  message: string;
  duration: number;
}

class TestRunner {
  private results: TestResult[] = [];
  private testUserId: string = '';
  private testAccessToken: string = '';
  private testChatId: string = '';

  async runAllTests(): Promise<void> {
    console.log('='.repeat(60));
    console.log('CHORUS MOBILE APP - PHASE 1 FEATURE TESTS');
    console.log('='.repeat(60));
    console.log('');

    await this.runTest('Health Check', this.testHealthCheck.bind(this));
    await this.runTest('User Registration', this.testRegistration.bind(this));
    await this.runTest('User Login', this.testLogin.bind(this));
    await this.runTest('Get User Profile', this.testGetProfile.bind(this));
    await this.runTest('Create Direct Chat', this.testCreateChat.bind(this));
    await this.runTest('Send Message', this.testSendMessage.bind(this));
    await this.runTest('Get Messages', this.testGetMessages.bind(this));
    await this.runTest('Get Chats List', this.testGetChats.bind(this));

    this.printSummary();
  }

  failedCount(): number {
    return this.results.filter((r) => !r.passed).length;
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

  private async testHealthCheck(): Promise<void> {
    const response = await axios.get(HEALTH_URL);
    if (response.data.status !== 'healthy') {
      throw new Error('Health check failed');
    }
  }

  /**
   * Registers a throwaway user through the real invite gate (registration is
   * invite-gated by default): mint an open SMS invite as the seeded fixture,
   * then register with its single-use token.
   */
  private async registerThrowaway(prefix: string): Promise<{ token: string; userId: string; email: string }> {
    const fixture = await axios.post(`${API_BASE_URL}/auth/login`, {
      username: 'alice.en-es@chorus.test',
      password: 'ChorusDev123!',
    });
    const fixtureToken = fixture.data.tokens.accessToken;
    const email = `${prefix}_${Date.now()}@chorus.test`;
    const invite = await axios.post(
      `${API_BASE_URL}/contacts/invites`,
      { channel: 'sms', contact: { name: prefix, phone: `+1555${String(Date.now()).slice(-7)}` } },
      { headers: { Authorization: `Bearer ${fixtureToken}` } }
    );
    const inviteToken = invite.data?.data?.token;
    if (!inviteToken) throw new Error('invite mint did not return a token');
    const response = await axios.post(`${API_BASE_URL}/auth/register`, {
      username: email,
      email,
      password: 'TestPass123!',
      displayName: 'Test User',
      nativeLanguage: 'en',
      targetLanguages: ['es', 'fr'],
      inviteToken,
    });

    if (!response.data.user || !response.data.tokens || !response.data.tokens.accessToken) {
      throw new Error('Registration did not return user or token');
    }

    return { token: response.data.tokens.accessToken, userId: response.data.user.id, email };
  }

  private async testRegistration(): Promise<void> {
    const { token, userId } = await this.registerThrowaway('testuser');
    this.testUserId = userId;
    this.testAccessToken = token;
  }

  private async testLogin(): Promise<void> {
    // Register a throwaway, then prove the same credentials log in.
    const { email } = await this.registerThrowaway('logintest');

    const response = await axios.post(`${API_BASE_URL}/auth/login`, {
      username: email,
      password: 'TestPass123!',
    });

    if (!response.data.tokens || !response.data.tokens.accessToken) {
      throw new Error('Login did not return access token');
    }
  }

  private async testGetProfile(): Promise<void> {
    const response = await axios.get(`${API_BASE_URL}/users/me`, {
      headers: { Authorization: `Bearer ${this.testAccessToken}` },
    });

    if (!response.data.id || response.data.id !== this.testUserId) {
      throw new Error('Profile data mismatch');
    }
  }

  private async testCreateChat(): Promise<void> {
    // Direct chats need a peer: resolve the seeded ES learner via search.
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
      throw new Error('Chat creation did not return chat ID');
    }

    this.testChatId = response.data.id;
  }

  private async testSendMessage(): Promise<void> {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/chats/${this.testChatId}/messages`,
        {
          text: 'Hello, this is a test message!',
        },
        {
          headers: { Authorization: `Bearer ${this.testAccessToken}` },
        }
      );

      if (!response.data.id || !response.data.text) {
        throw new Error('Message sending failed - incomplete response');
      }
    } catch (error: any) {
      console.error('Send message error:', error.response?.data);
      throw new Error(error.response?.data?.error || 'Failed to send message');
    }
  }

  private async testGetMessages(): Promise<void> {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/chats/${this.testChatId}/messages`,
        {
          headers: { Authorization: `Bearer ${this.testAccessToken}` },
        }
      );

      if (!response.data.messages || !Array.isArray(response.data.messages) || response.data.messages.length === 0) {
        throw new Error('No messages found');
      }
    } catch (error: any) {
      console.error('Get messages error:', error.response?.data);
      throw new Error(error.response?.data?.error || 'Failed to retrieve messages');
    }
  }

  private async testGetChats(): Promise<void> {
    try {
      const response = await axios.get(`${API_BASE_URL}/chats`, {
        headers: { Authorization: `Bearer ${this.testAccessToken}` },
      });

      if (!response.data.chats || !Array.isArray(response.data.chats)) {
        throw new Error('Invalid response format');
      }
    } catch (error: any) {
      console.error('Get chats error:', error.response?.data);
      throw new Error(error.response?.data?.error || 'Failed to retrieve chats list');
    }
  }

  private printSummary(): void {
    console.log('='.repeat(60));
    console.log('TEST SUMMARY');
    console.log('='.repeat(60));
    
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

    console.log('='.repeat(60));
  }
}

// Run tests
const runner = new TestRunner();
runner.runAllTests()
  .then(() => {
    console.log('All tests completed!');
    process.exit(runner.failedCount() > 0 ? 1 : 0);
  })
  .catch((error) => {
    console.error('Test runner failed:', error);
    process.exit(1);
  });
