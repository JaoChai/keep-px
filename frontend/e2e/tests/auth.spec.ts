import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages/login.page'
import { test as authTest, expect as authExpect } from '../fixtures/auth.fixture'

test.describe('Authentication', () => {
  test('login page shows branding and Google sign-in', async ({ page }) => {
    const loginPage = new LoginPage(page)
    await loginPage.goto()

    await expect(loginPage.heading).toBeVisible()
    await expect(loginPage.subtitle).toBeVisible()
  })

  test('/register redirects to /login', async ({ page }) => {
    await page.goto('/register')
    await expect(page).toHaveURL(/\/login/)
  })
})

authTest.describe('Authenticated', () => {
  authTest('logout button works', async ({ page }) => {
    await page.goto('/dashboard')
    await authExpect(page).toHaveURL(/\/dashboard/)

    await page.getByRole('button', { name: 'ออกจากระบบ' }).click()
    await authExpect(page).toHaveURL(/\/login/)
  })
})
