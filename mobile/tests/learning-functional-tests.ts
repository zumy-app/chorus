/**
 * Comprehensive learning-engine functional tests (Phase 1-8 of the Chorus
 * learning implementation plan). These hit the real backend over HTTP, the
 * same way the mobile/web clients do, and validate every learning feature:
 * capabilities, profile, dashboard, path, placement, lessons, sessions, mined
 * vocabulary, scenarios, real-talk prompts, and streak recovery.
 *
 * Run with:
 *   cd c:\dev\chorus\mobile
 *   npx ts-node --project tsconfig.tests.json tests/learning-functional-tests.ts
 */
import axios from 'axios';

const API = 'http://localhost:8080/api/v1';
const TARGET = 'es';
const NATIVE = 'en';

interface TestResult {
  name: string;
  passed: boolean;
  message: string;
  duration: number;
}

class LearningTestRunner {
  private results: TestResult[] = [];
  private token = '';
  private userId = '';

  private headers() {
    return { Authorization: `Bearer ${this.token}` };
  }

  async run() {
    return new Promise<void>((resolve) => {
      const boot = async () => {
        console.log('='.repeat(72));
        console.log('CHORUS MOBILE — LEARNING ENGINE FEATURE TESTS');
        console.log('='.repeat(72));

        await this.runTest('1. Health', () => this.tHealth());
        await this.runTest('2. Register learning user', () => this.tRegister());
        await this.runTest('3. Get capabilities (en->es = full_course)', () => this.tCapabilities());
        await this.runTest('4. Get/create learning profile', () => this.tProfile());
        await this.runTest('5. Update learning profile', () => this.tUpdateProfile());
        await this.runTest('6. Get learning dashboard', () => this.tDashboard());
        await this.runTest('7. Get learning path', () => this.tPath());
        await this.runTest('8. Placement start + answer', () => this.tPlacement());
        await this.runTest('9. Placement skip', () => this.tSkipPlacement());
        await this.runTest('10. Get unit', () => this.tUnit());
        await this.runTest('11. Start + answer + complete lesson', () => this.tLesson());
        await this.runTest('12. Start session + answer + complete', () => this.tSession());
        await this.runTest('13. Mined items list/accept/ignore', () => this.tMinedItems());
        await this.runTest('14. Scenarios list + start roleplay', () => this.tScenarios());
        await this.runTest('15. Scenario send message + hint', () => this.tScenarioMessage());
        await this.runTest('16. Real-talk prompts + mark used', () => this.tRealTalk());
        await this.runTest('17. Streak recovery', () => this.tStreak());

        this.printSummary();
        resolve();
      };
      boot().catch((e) => {
        console.error('Fatal runner error:', e);
        resolve();
      });
    });
  }

  private async runTest(name: string, fn: () => Promise<void>) {
    const start = Date.now();
    try {
      await fn();
      this.results.push({ name, passed: true, message: 'ok', duration: Date.now() - start });
      console.log(`✅ ${name}  (${Date.now() - start}ms)`);
    } catch (e: any) {
      const msg = e.response?.data?.error || e.message || 'unknown';
      this.results.push({ name, passed: false, message: msg, duration: Date.now() - start });
      console.log(`❌ ${name}  (${Date.now() - start}ms)`);
      console.log(`   ${msg}`);
    }
  }

  // ---- tests ---------------------------------------------------------------

  private async tHealth() {
    const r = await axios.get('http://localhost:8080/health');
    if (r.data.status !== 'healthy') throw new Error('not healthy');
  }

  private async tRegister() {
    // The backend gates registration behind an invite, so authenticate with the
    // local pre-seeded test account instead of creating a user per run.
    const r = await axios.post(`${API}/auth/login`, {
      username: 'uhsarp@gmail.com',
      password: 'Demor@cer1',
    });
    this.userId = r.data.user?.id;
    this.token = r.data.tokens?.accessToken;
    if (!this.token) throw new Error('no token');
  }

  private async tCapabilities() {
    const r = await axios.get(`${API}/learning/capabilities`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const cap = r.data.data;
    if (!cap || cap.supportTier !== 'full_course') throw new Error('expected full_course, got ' + cap?.supportTier);
  }

  private async tProfile() {
    const r = await axios.get(`${API}/learning/profile`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const p = r.data.data;
    if (!p || p.targetLanguage !== TARGET) throw new Error('missing profile');
    if (!['not_started', 'in_progress', 'completed', 'skipped'].includes(p.placementStatus)) {
      throw new Error('unexpected placementStatus ' + p.placementStatus);
    }
  }

  private async tUpdateProfile() {
    const r = await axios.put(
      `${API}/learning/profile`,
      { targetLanguage: TARGET, nativeLanguage: NATIVE, dailyGoalItems: 15 },
      { headers: this.headers() }
    );
    if (!r.data.data || r.data.data.dailyGoalItems !== 15) throw new Error('update not applied');
  }

  private async tDashboard() {
    const r = await axios.get(`${API}/learning/dashboard`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const d = r.data.data;
    if (!d || !d.capability || !d.profile || !d.dailyGoal || !d.fluency || !d.vocabulary || !d.grammar || !d.scenario) {
      throw new Error('dashboard missing sections');
    }
  }

  private async tPath() {
    const r = await axios.get(`${API}/learning/path`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const path = r.data.data;
    if (!path || !Array.isArray(path.units) || path.units.length < 20) {
      throw new Error('expected a full A1-B2 unit list, got ' + path?.units?.length);
    }
  }

  private async tPlacement() {
    const start = await axios.post(`${API}/learning/placement/start`, null, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const attemptId = start.data.data.attemptId;
    if (!attemptId) throw new Error('no attemptId');
    let attempt = attemptId as string;
    // Answer with the first choice until the test finalizes (returns estimatedCefr).
    for (let i = 0; i < 20; i++) {
      const res = await axios.post(`${API}/learning/placement/${attempt}/answer`, { answer: 'a' }, { headers: this.headers() });
      const data = res.data.data;
      if (data?.estimatedCefr) {
        if (!['A1', 'A2', 'B1', 'B2'].includes(data.estimatedCefr)) throw new Error('bad level ' + data.estimatedCefr);
        return;
      }
      attempt = data.attemptId || attempt;
      if (!data?.question) throw new Error('placement stalled');
    }
    throw new Error('placement never finalized');
  }

  private async tSkipPlacement() {
    const r = await axios.post(`${API}/learning/placement/skip`, null, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    if (!r.data.data || r.data.data.estimatedCefr !== 'A1') throw new Error('skip did not assign A1');
  }

  private async tUnit() {
    const path = await axios.get(`${API}/learning/path`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const unit = path.data.data.units[0];
    if (!unit?.id) throw new Error('no unit');
    const r = await axios.get(`${API}/learning/units/${unit.id}`, { headers: this.headers() });
    if (!r.data.data || !r.data.data.id) throw new Error('unit not returned');
  }

  private async tLesson() {
    const path = await axios.get(`${API}/learning/path`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const lessons = (path.data.data.units[0]?.lessons || []);
    if (lessons.length === 0) throw new Error('unit has no lessons');
    const lessonId = lessons[0].id;
    const start = await axios.post(`${API}/learning/lessons/${lessonId}/start`, null, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const attempt = start.data.data.attempt;
    if (!attempt?.id) throw new Error('no attempt');
    const steps = start.data.data.steps || [];
    for (const step of steps) {
      await axios.post(
        `${API}/learning/lesson-attempts/${attempt.id}/steps/${step.id}/answer`,
        { answer: 'a' },
        { headers: this.headers() }
      );
    }
    const complete = await axios.post(`${API}/learning/lesson-attempts/${attempt.id}/complete`, null, { headers: this.headers() });
    if (!complete.data.data) throw new Error('no completion result');
  }

  private async tSession() {
    const start = await axios.post(`${API}/learning/sessions/start`, {
      targetLanguage: TARGET,
      nativeLanguage: NATIVE,
      mode: 'daily',
      source: 'test',
    }, { headers: this.headers() });
    const session = start.data.data.session;
    const items = start.data.data.items || [];
    if (!session?.id) throw new Error('no session');
    for (const item of items) {
      await axios.post(`${API}/learning/sessions/${session.id}/items/${item.id}/answer`, {
        answer: { text: 'a', choice: 'a' },
        latencyMs: 500,
      }, { headers: this.headers() });
    }
    const done = await axios.post(`${API}/learning/sessions/${session.id}/complete`, null, { headers: this.headers() });
    if (!done.data.data) throw new Error('no completion result');
  }

  private async tMinedItems() {
    const list = await axios.get(`${API}/learning/vocabulary/mined`, {
      params: { targetLanguage: TARGET, status: 'candidate' },
      headers: this.headers(),
    });
    if (!Array.isArray(list.data.data)) throw new Error('mined list not an array');
    // Validate accept/ignore routes against a real item when one exists;
    // otherwise just confirm the list endpoint shape is correct.
    const item = list.data.data[0];
    if (item) {
      const accept = await axios.post(`${API}/learning/vocabulary/mined/${item.id}/accept`, null, { headers: this.headers() });
      if (!accept.data.data?.id) throw new Error('accept did not return a card');
      const ignore = await axios.post(`${API}/learning/vocabulary/mined/${item.id}/ignore`, null, { headers: this.headers() });
      if (ignore.status !== 200) throw new Error('ignore route failed');
    }
  }

  private async tScenarios() {
    const list = await axios.get(`${API}/learning/scenarios`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const scenarios = list.data.data || [];
    if (!Array.isArray(scenarios)) throw new Error('scenarios not an array');
    if (scenarios.length === 0) throw new Error('no scenarios seeded');
    const sc = scenarios.find((s: any) => s.slug === 'ordering-coffee') || scenarios[0];
    const one = await axios.get(`${API}/learning/scenarios/${sc.id}`, { headers: this.headers() });
    if (!one.data.data || !one.data.data.id) throw new Error('scenario detail missing');
  }

  private async tScenarioMessage() {
    const sc = (await axios.get(`${API}/learning/scenarios`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    })).data.data[0];
    const start = await axios.post(`${API}/learning/scenarios/${sc.id}/start`, null, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    const runId = start.data.data.run?.id;
    if (!runId) throw new Error('no run');
    const send = await axios.post(`${API}/learning/scenario-runs/${runId}/message`, { message: 'Hola, buenos días.' }, { headers: this.headers() });
    if (send.data.data.aiMessage === undefined) throw new Error('no aiMessage');
    const hint = await axios.post(`${API}/learning/scenario-runs/${runId}/hint`, null, { headers: this.headers() });
    if (!Array.isArray(hint.data.data)) throw new Error('hint not an array');
  }

  private async tRealTalk() {
    const prompts = await axios.get(`${API}/learning/real-talk/prompts`, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    if (!Array.isArray(prompts.data.data)) throw new Error('prompts not an array');
    if (prompts.data.data.length > 0) {
      await axios.post(`${API}/learning/real-talk/prompts/${prompts.data.data[0].id}/used`, null, { headers: this.headers() });
    }
  }

  private async tStreak() {
    const r = await axios.post(`${API}/learning/streak/recover`, null, {
      params: { nativeLanguage: NATIVE, targetLanguage: TARGET },
      headers: this.headers(),
    });
    if (!r.data.data || typeof r.data.data.recovered !== 'boolean') throw new Error('bad streak result');
  }

  private printSummary() {
    const passed = this.results.filter((r) => r.passed).length;
    const failed = this.results.filter((r) => !r.passed).length;
    console.log('');
    console.log('='.repeat(72));
    console.log(`TOTAL: ${this.results.length}  |  PASSED: ${passed}  |  FAILED: ${failed}`);
    console.log('='.repeat(72));
    if (failed > 0) {
      console.log('FAILED:');
      this.results.filter((r) => !r.passed).forEach((r) => console.log(`  - ${r.name}: ${r.message}`));
      console.log('');
    }
  }
}

new LearningTestRunner()
  .run()
  .then(() => process.exit(0))
  .catch(() => process.exit(1));
