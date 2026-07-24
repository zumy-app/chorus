import { chromium, Page } from '@playwright/test';

const EMAIL = 'trolldown@gmail.com';
const PASSWORD = 'en&L&@4KHb7S4Kc&!Fb!N7B';
const BASE_URL = 'https://skillcat.app';

/**
 * Wait for page to settle after navigation
 */
async function waitForPageSettle(page: Page) {
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(1000);
}

/**
 * Try clicking a text match with retry
 */
async function clickByText(page: Page, text: string, timeout = 10000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const btn = page.locator(`text="${text}"`).first();
      if (await btn.isVisible({ timeout: 1000 })) {
        await btn.click();
        return true;
      }
      // Also try partial match
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

/**
 * Find course cards/sections on the dashboard
 */
async function findCourses(page: Page): Promise<string[]> {
  // Try multiple strategies to find course links
  const courseSelectors = [
    'a[href*="/course/"]',
    'a[href*="/my/"]',
    '[class*="course"]',
    '[class*="lesson"]',
    '[class*="card"]',
    'a[href*="learn"]',
    'button:has-text("Resume")',
    'button:has-text("Start")',
    'button:has-text("Continue")',
    '[class*="module"]',
  ];

  for (const sel of courseSelectors) {
    const items = page.locator(sel);
    const count = await items.count();
    if (count > 0) {
      console.log(`Found ${count} items matching selector: ${sel}`);
    }
  }

  // Get all links and buttons as potential courses
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

/**
 * Process tests on the current page
 */
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
    let attempts = 0;
    while (attempts < 10) {
      try {
        const btn = page.locator(sel).first();
        if (await btn.isVisible({ timeout: 2000 })) {
          await btn.click();
          console.log(`✅ Clicked: ${sel}`);
          await page.waitForTimeout(2000);
          await waitForPageSettle(page);
          testsFound++;

          // Answer questions if needed (try all options)
          const answerOptions = page.locator('input[type="radio"], input[type="checkbox"], [class*="option"], [class*="answer"], [class*="choice"]');
          const answerCount = await answerOptions.count();
          if (answerCount > 0) {
            await answerOptions.first().click();
            await page.waitForTimeout(500);
          }

          // Submit
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
      attempts++;
    }
  }
  return testsFound;
}

async function main() {
  console.log('🚀 Starting Skillcat automation...');

  const browser = await chromium.launch({
    headless: false, // Let user watch the progress
    args: ['--start-maximized'],
  });

  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
  });
  const page = await context.newPage();

  // Polling-based approach: keep trying until user intervenes or all courses are complete
  const POLL_INTERVAL = 5000; // 5 seconds
  const MAX_POLL_TIME = 30 * 60 * 1000; // 30 minutes max
  const startTime = Date.now();

  let allComplete = false;
  let consecutiveStuck = 0;
  const maxStuck = 12; // ~1 minute of being stuck before logging extra info

  try {
    // Step 1: Navigate to the site
    console.log('📌 Step 1: Navigating to Skillcat...');
    await page.goto(`${BASE_URL}/my/`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await waitForPageSettle(page);

    // Check if we're already logged in
    let currentUrl = page.url();
    console.log(`📍 Current URL: ${currentUrl}`);

    // Step 2: Login if needed
    if (currentUrl.includes('/login') || currentUrl.includes('/auth') || currentUrl.includes('signin') || currentUrl.includes('log-in')) {
      console.log('📌 Step 2: Logging in...');

      const emailInput = page.locator('input[type="email"], input[name="email"], input[placeholder*="email" i], input[placeholder*="Email" i]').first();
      const passwordInput = page.locator('input[type="password"], input[name="password"], input[placeholder*="password" i]').first();

      if (await emailInput.isVisible({ timeout: 5000 })) {
        await emailInput.fill(EMAIL);
        await passwordInput.fill(PASSWORD);

        const loginBtn = page.locator('button[type="submit"], button:has-text("Log in"), button:has-text("Login"), button:has-text("Sign in"), button:has-text("Sign In")').first();
        if (await loginBtn.isVisible()) {
          await loginBtn.click();
        }

        await page.waitForTimeout(5000);
        await waitForPageSettle(page);
        console.log(`📍 After login URL: ${page.url()}`);
      }
    }

    // Main polling loop
    while (Date.now() - startTime < MAX_POLL_TIME && !allComplete) {
      currentUrl = page.url();
      console.log(`\n📍 Current URL: ${currentUrl}`);
      console.log(`⏱️  Elapsed: ${((Date.now() - startTime) / 1000 / 60).toFixed(1)} min`);

      // Take a screenshot to track progress
      await page.screenshot({ path: `skillcat-progress-${Math.floor((Date.now() - startTime) / 10000)}.png`, fullPage: true }).catch(() => {});

      // Step 3: Navigate to Dashboard
      const dashboardClicked = await clickByText(page, 'Dashboard');
      if (dashboardClicked) {
        console.log('✅ Clicked Dashboard');
        await page.waitForTimeout(3000);
        await waitForPageSettle(page);
        consecutiveStuck = 0;
        continue; // Re-evaluate on Dashboard
      }

      // Step 4: Find courses and tests
      const pageTexts = await findCourses(page);
      console.log('📋 Found page elements:', pageTexts.slice(0, 20));

      const testKeywords = ['test', 'quiz', 'exam', 'assessment', 'take test', 'start test'];
      const courseKeywords = ['course', 'module', 'lesson', 'chapter', 'unit'];

      let clickedSomething = false;

      // Look for test buttons first
      for (const text of pageTexts) {
        const lower = text.toLowerCase();

        // If it looks like a test/quiz, click it
        if (testKeywords.some(k => lower.includes(k))) {
          console.log(`🎯 Found test element: "${text}"`);
          const clicked = await clickByText(page, text);
          if (clicked) {
            console.log(`✅ Clicked test: ${text}`);
            await page.waitForTimeout(3000);
            await waitForPageSettle(page);

            // Process questions on this page
            await processTests(page);

            // Try to submit/complete
            await clickByText(page, 'Submit');
            await clickByText(page, 'Finish');
            await clickByText(page, 'Complete');
            await clickByText(page, 'Done');

            await page.waitForTimeout(2000);
            clickedSomething = true;
            consecutiveStuck = 0;
            break; // Re-scan the page
          }
        }
      }

      if (!clickedSomething) {
        // Try course elements
        for (const text of pageTexts) {
          const lower = text.toLowerCase();
          if (courseKeywords.some(k => lower.includes(k))) {
            console.log(`📚 Found course element: "${text}"`);
            const clicked = await clickByText(page, text);
            if (clicked) {
              console.log(`✅ Clicked course: ${text}`);
              await page.waitForTimeout(3000);
              await waitForPageSettle(page);

              // Process tests inside
              const found = await processTests(page);
              console.log(`  Found ${found} tests inside this course`);

              clickedSomething = true;
              consecutiveStuck = 0;
              break;
            }
          }
        }
      }

      if (!clickedSomething) {
        // Try generic test button selectors
        const found = await processTests(page);
        if (found > 0) {
          clickedSomething = true;
          consecutiveStuck = 0;
        }
      }

      if (!clickedSomething) {
        consecutiveStuck++;
        console.log(`⏳ Waiting... (stuck count: ${consecutiveStuck})`);

        if (consecutiveStuck >= maxStuck) {
          console.log('🔄 Appears stuck. Trying to refresh and navigate...');
          await page.goto(`${BASE_URL}/my/`, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {});
          await page.waitForTimeout(3000);
          await waitForPageSettle(page);
          consecutiveStuck = 0;
        }

        // Check if we might be done
        const doneKeywords = ['complete', 'finished', 'congratulations', 'certificate', 'you did it'];
        const pageContent = await page.textContent('body').catch(() => '');
        if (pageContent && doneKeywords.some(k => pageContent.toLowerCase().includes(k))) {
          console.log('🎉 Possible completion state detected!');
          // Take a screenshot for the user
          await page.screenshot({ path: 'skillcat-completion-state.png', fullPage: true }).catch(() => {});
        }

        await page.waitForTimeout(POLL_INTERVAL);
      }
    }

    // Final screenshot
    await page.screenshot({ path: 'skillcat-final.png', fullPage: true }).catch(() => {});
    console.log('\n✨ Polling complete!');

    if (consecutiveStuck < maxStuck) {
      console.log('✅ Looks like we made progress!');
    } else {
      console.log('⚠️  Automation might be stuck. You can take over manually now.');
    }

  } catch (error) {
    console.error('❌ Error:', error);
    await page.screenshot({ path: 'skillcat-error.png', fullPage: true }).catch(() => {});
  }

  console.log('\n📸 Screenshots saved to e2e/skillcat-*.png');
  console.log('💡 Browser will stay open for you to review and manually complete if needed.');
  console.log('   Close the browser window when done.');

  // Keep browser open for user to see/intervene
  await page.waitForTimeout(600000); // 10 minutes
  await browser.close();
}

main();