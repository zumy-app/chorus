import { test, expect } from '@playwright/test'

test.describe('smoke', () => {
  test('landing page renders and links to login', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('link', { name: 'Log In' }).last()).toBeVisible()

    await page.getByRole('link', { name: 'Log In' }).last().click()
    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByRole('heading', { name: 'Chorus' })).toBeVisible()
  })

  test('login page rejects empty submission with validation', async ({ page }) => {
    await page.goto('/login')
    await page.getByRole('button', { name: /log in|sign in/i }).first().click()
    await expect(page.getByText(/username|email/i).first()).toBeVisible()
  })
})
