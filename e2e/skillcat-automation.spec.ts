import { test, chromium, Page } from '@playwright/test';

const EMAIL = 'trolldown@gmail.com';
const PASSWORD = 'en&L&@4KHb7S4Kc&!Fb!N7B';
const BASE_URL = 'https://skillcat.app';

async function waitForPageSettle(page: Page) {
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(1000);
}

async function clickByText(page: Page, text: string, timeout = 10000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const btn = page.locator(`text="${text}"`).first();
      if (await btn.isVisible({ timeout: 1000 })) {
        await btn.click();
        return true;
      }
      const partial = page.locator(`text=${text}`).first();
      if (await partial.isVisible({ timeout: 1000 })) {
        await partial.click();
        return true;
      }
    } catch {
      // continue
    }
    await page.waitForTimeout(500);
  }
  return false;
}

async function findAllClickableTexts(page: Page): Promise<string[]> {
  const allLinks = await page.locator('a').all();
  const allButtons = await page.locator('button').all();
  const elements = [...allLinks, ...allButtons];
  const candidates: string[] = [];
  for (const el of elements) {
    const text = await el.textContent();
    if (text && text.trim().length > 0 && text.trim().length < 100) {
      candidates.push(text.trim());
    }
  }
  return [...new Set(candidates)];
}

async function processTests(page: Page) {
  const testSelectors = [
    'button:has-text("Take Test")',
    'button:has-text("Start Quiz")',
    'button:has-text("Start Test")',
    'button:has-text("Take Quiz")',
    'button:has-text("Begin")',
    'button:has-text("Start")',
    'button:has-text("Continue")',
    '[class*="quiz"]',
    '[class*="test"]',
    '[class*="assessment"]',
  ];

  let testsFound = 0;
  for (const sel of testSelectors) {
    for (let attempts = 0; attempts < 10; attempts++) {
      try {
        const btn = page.locator(sel).first();
        if (await btn.isVisible({ timeout: 2000 })) {
          await btn.click();
          console.log(`  ✅ Clicked: ${sel}`);
          await page.waitForTimeout(2000);
          await waitForPageSettle(page);
          testsFound++;

          // Answer questions if needed
          const answerOptions = page.locator('input[type="radio"], input[type="checkbox"], [class*="option"], [class*="answer"], [class*="choice"]');
          const answerCount = await answerOptions.count();
          if (answerCount > 0) {
            await answerOptions.first().click();
            await page.waitForTimeout(500);
          }

          await clickByText(page, 'Submit');
          await clickByText(page, 'Next');
          await clickByText(page, 'Finish');
          await page.waitForTimeout(1000);
        } else {
          break;
        }
      } catch {
        break;
      }
    }
  }
  return testsFound;
}

test('Complete all Skillcat courses', async () => {
  test.setTimeout(1800000); // 30 minutes

  const browser = await chromium.launch({
    headless: false,
    args: ['--start-maximized'],
  });

  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
  });
  const page = await context.newPage();

  const POLL_INTERVAL = 5000;
  const MAX_POLL_TIME = 25 * 60 * 1000;
  const startTime = Date.now();
  let consecutiveStuck = 0;
  const maxStuck = 12;

  try {
    // Navigate
    console.log('📍 Navigating to Skillcat...');
    await page.goto(`${BASE_URL}/my/`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await waitForPageSettle(page);

    let currentUrl = page.url();
    console.log(`📍 URL: ${currentUrl}`);

    // Login if needed
    if (currentUrl.includes('/login') || currentUrl.includes('/auth') || currentUrl.includes('signin')) {
      console.log('📍 Logging in...');
      const emailInput = page.locator('input[type="email"], input[name="email"], input[placeholder*="email" i], input[placeholder*="Email" i]').first();
      const passwordInput = page.locator('input[type="password"], input[name="password"], input[placeholder*="password" i]').first();

      if (await emailInput.isVisible({ timeout: 5000 })) {
        await emailInput.fill(EMAIL);
        await passwordInput.fill(PASSWORD);
        const loginBtn = page.locator('button[type="submit"], button:has-text("Log in"), button:has-text("Login"), button:has-text("Sign in")').first();
        if (await loginBtn.isVisible()) {
          await loginBtn.click();
        }
        await page.waitForTimeout(5000);
        await waitForPageSettle(page);
        console.log(`📍 After login: ${page.url()}`);
      }
    }

    // Main polling loop
    while (Date.now() - startTime < MAX_POLL_TIME) {
      currentUrl = page.url();
      console.log(`\n📍 ${currentUrl}`);
      console.log(`⏱️  ${((Date.now() - startTime) / 1000 / 60).toFixed(1)} min elapsed`);

      await page.screenshot({ path: `skillcat-progress.png`, fullPage: true }).catch(() => {});

      // Try Dashboard
      if (await clickByText(page, 'Dashboard')) {
        console.log('✅ Clicked Dashboard');
        await page.waitForTimeout(3000);
        await waitForPageSettle(page);
        consecutiveStuck = 0;
        continue;
      }

      // Find page texts
      const pageTexts = await findAllClickableTexts(page);
      console.log('📋 Elements:', pageTexts.slice(0, 15));

      const testKeywords = ['test', 'quiz', 'exam', 'assessment'];
      const courseKeywords = ['course', 'module', 'lesson', 'chapter', 'unit'];

      let clickedSomething = false;

      // Try test elements
      for (const text of pageTexts) {
        const lower = text.toLowerCase();
        if (testKeywords.some(k => lower.includes(k))) {
          if (await clickByText(page, text)) {
            console.log(`✅ Clicked test: ${text}`);
            await page.waitForTimeout(3000);
            await waitForPageSettle(page);
            await processTests(page);
            await clickByText(page, 'Submit');
            await clickByText(page, 'Finish');
            await clickByText(page, 'Complete');
            await clickByText(page, 'Done');
            await page.waitForTimeout(2000);
            clickedSomething = true;
            consecutiveStuck = 0;
            break;
          }
        }
      }

      if (!clickedSomething) {
        // Try course elements
        for (const text of pageTexts) {
          const lower = text.toLowerCase();
          if (courseKeywords.some(k => lower.includes(k))) {
            if (await clickByText(page, text)) {
              console.log(`✅ Clicked course: ${text}`);
              await page.waitForTimeout(3000);
              await waitForPageSettle(page);
              const found = await processTests(page);
              console.log(`  Found ${found} tests inside`);
              clickedSomething = true;
              consecutiveStuck = 0;
              break;
            }
          }
        }
      }

      if (!clickedSomething) {
        const found = await processTests(page);
        if (found > 0) {
          clickedSomething = true;
          consecutiveStuck = 0;
        }
      }

      if (!clickedSomething) {
        consecutiveStuck++;
        console.log(`⏳ Stuck count: ${consecutiveStuck}`);

        if (consecutiveStuck >= maxStuck) {
          console.log('🔄 Refreshing...');
          await page.goto(`${BASE_URL}/my/`, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {});
          await page.waitForTimeout(3000);
          await waitForPageSettle(page);
          consecutiveStuck = 0;
        }

        // Check for completion
        const doneKeywords = ['complete', 'finished', 'congratulations', 'certificate', 'you did it'];
        const bodyText = await page.textContent('body').catch(() => '');
        if (bodyText && doneKeywords.some(k => bodyText.toLowerCase().includes(k))) {
          console.log('🎉 Completion detected!');
          await page.screenshot({ path: 'skillcat-done.png', fullPage: true }).catch(() => {});
        }

        await page.waitForTimeout(POLL_INTERVAL);
      }
    }

    console.log('\n✨ Automation finished!');
    await page.screenshot({ path: 'skillcat-final.png', fullPage: true }).catch(() => {});
  } catch (error) {
    console.error('❌ Error:', error);
    await page.screenshot({ path: 'skillcat-error.png', fullPage: true }).catch(() => {});
  }

  // Keep browser open for user
  await page.waitForTimeout(600000);
  await browser.close();
});