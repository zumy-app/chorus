import axios from 'axios';

const API_BASE_URL = 'http://localhost:8080/api/v1';

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
    const response = await axios.get('http://localhost:8080/health');
    if (response.data.status !== 'healthy') {
      throw new Error('Health check failed');
    }
  }

  private async testRegistration(): Promise<void> {
    const timestamp = Date.now();
    const response = await axios.post(`${API_BASE_URL}/auth/register`, {
      username: `testuser_${timestamp}`,
      email: `testuser_${timestamp}@example.com`,
      password: 'TestPass123!',
      displayName: 'Test User',
      nativeLanguage: 'en',
      targetLanguages: ['es', 'fr'],
    });

    if (!response.data.user || !response.data.tokens || !response.data.tokens.accessToken) {
      throw new Error('Registration did not return user or token');
    }

    this.testUserId = response.data.user.id;
    this.testAccessToken = response.data.tokens.accessToken;
  }

  private async testLogin(): Promise<void> {
    const timestamp = Date.now();
    const username = `logintest_${timestamp}`;

    // First register a user
    await axios.post(`${API_BASE_URL}/auth/register`, {
      username,
      email: `${username}@example.com`,
      password: 'TestPass123!',
      displayName: 'Login Test User',
      nativeLanguage: 'en',
      targetLanguages: ['es'],
    });

    // Then login
    const response = await axios.post(`${API_BASE_URL}/auth/login`, {
      username,
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
    const response = await axios.post(
      `${API_BASE_URL}/chats`,
      {
        type: 'direct',
        participants: [this.testUserId],
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
    process.exit(0);
  })
  .catch((error) => {
    console.error('Test runner failed:', error);
    process.exit(1);
  });
