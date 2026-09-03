import { test, expect } from '@playwright/test'
import { loginAsUser } from '../fixtures/test-helpers'

/**
 * C-02 — Learning Journey (placement / scenarios / real-talk / streak / lesson / monthly)
 * Authority: docs/QA_CRITIQUE_AND_IMPROVEMENTS.md:§2.2, docs/TEST_PLAN.md:§12.2 C-02
 * Backend: POST /learning/placement/start:663, POST /learning/placement/:attemptId/answer:664, GET :attemptId:667,
 *          GET /learning/dashboard:659, GET /learning/scenarios:696, POST /scenarios/:id/start:698,
 *          GET /learning/real-talk/prompts:705, POST /prompts/:id/used:706, POST /learning/streak/recover:708,
 *          POST /learning/sessions/start:681, POST /sessions/:sessionId/items/:itemId/answer:683, POST /sessions/:sessionId/complete:684,
 *          GET /learning/srs/queue:687, GET /learning/vocabulary/mined:691, GET /learning/path:660
 * Frontend: /learn/placement:134, /learn/scenarios:143, /learn/real-talk:170, /learn/session:138, /learn/vocabulary:148, /learn/roadmap:168
 * Mobile: MainTabs.tsx:89-124 LearnTab stack + Placement/LessonSession/VocabularyReview/Scenarios/ScenarioRoleplay/RealTalkHub/LearningRoadmap/StreakRecovery
 * Gherkin in QA doc §2.2 — this skeleton is FAILING-FIRST (RED until real placement/scenarios/real-talk/streak/lesson wired against dev seed, not mocked qa-scenarios-drills.spec.ts:194).
 */
test.describe('@C-02 @learning @wireframe-placement @wireframe-scenarios @wireframe-real-talk @wireframe-streak', () => {
  test.describe.configure({ mode: 'serial' })

  test('C-02-01 — placement start → vocab + reading answers → results summary (POST /learning/placement/*)', async ({ page }) => {
    await loginAsUser(page, { email: 'alice.dev@chorus.test', password: 'ChorusDev123!', nativeLanguage: 'en', displayName: 'Alice Dev' } as any)
    await page.goto('/learn/placement')
    // Failing-first: real placement not yet wired to dev seed — will be RED until POST /learning/placement/start:663 works
    await expect(page.getByRole('heading', { name: /Placement Test/i })).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('C-02-01-placement-results-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-02-02 — dashboard weeklyActivity + fluency after placement (GET /learning/dashboard:659)', async ({ page }) => {
    await page.goto('/learn')
    await expect(page.getByText(/Your Learning Path|Fluency|weeklyActivity/i).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('C-02-02-dashboard-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-02-03 — scenarios list Pedir café → start → ScenarioRoleplay openingLine + chunks + translation', async ({ page }) => {
    await page.goto('/learn/scenarios')
    await expect(page.getByText(/Pedir café en una cafetería|Real-World Scenarios/)).toBeVisible({ timeout: 10_000 })
    // Start scenario — requires GET /learning/scenarios:696 + POST /learning/scenarios/:id/start:698 via curriculum.go ordering-coffee
    await expect(page.getByTestId('C-02-03-scenario-roleplay-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-02-04 — real-talk hub prompts → POST /prompts/:id/used:706 + RealTalkNudge in Chat', async ({ page }) => {
    await page.goto('/learn/real-talk')
    await expect(page.getByText(/Real Talk/i).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('C-02-04-real-talk-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-02-05 — streak + StreakRecoveryScreen → POST /learning/streak/recover:708', async ({ page }) => {
    await page.goto('/learn/roadmap')
    await expect(page.getByText(/Streak|Roadmap/i).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('C-02-05-streak-recovery-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-02-06 — lesson session daily practice: start → cloze Yo ____ cansado. → answer → ¡Excelente! → complete recap', async ({ page }) => {
    await page.goto('/learn/session')
    // Placeholder: real StartSession via session_composer.go StartSession + GetDueCards + NextLessonStep
    await expect(page.getByText(/Yo ____ cansado|Start Session|Daily Practice/i).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('C-02-06-lesson-session-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-02-07 — SRS queue interleaved (srs_queue.go interleaveQueue) + mined + monthlyActivity after session', async ({ page }) => {
    await page.goto('/learn/vocabulary')
    await expect(page.getByText(/Vocabulary|Due|SRS/i).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('C-02-07-srs-queue-proven')).toBeVisible({ timeout: 1_000 })
  })
})
