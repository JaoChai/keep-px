import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages/login.page'
import { RegisterPage } from '../pages/register.page'
import { TEST_USER } from '../fixtures/test-data'

test.describe('Authentication', () => {
  test('register with valid data redirects to dashboard', async ({ page }) => {
    const registerPage = new RegisterPage(page)
    await registerPage.goto()

    const uniqueEmail = `e2e-${Date.now()}@example.com`
    await registerPage.register('Test User', uniqueEmail, 'Test1234!')

    await expect(page).toHaveURL(/\/dashboard/)
  })

  test('login with valid credentials redirects to /dashboard @smoke', async ({ page }) => {
    const loginPage = new LoginPage(page)
    await loginPage.goto()

    await loginPage.login(TEST_USER.email, TEST_USER.password)

    await expect(page).toHaveURL(/\/dashboard/)
  })

  test('show validation errors on empty form submit', async ({ page }) => {
    const loginPage = new LoginPage(page)
    await loginPage.goto()

    await loginPage.signInButton.click()

    await expect(page.getByText('Invalid email address')).toBeVisible()
    await expect(page.getByText('Password is required')).toBeVisible()
  })

  test('logout button works', async ({ page }) => {
    const loginPage = new LoginPage(page)
    await loginPage.goto()

    await loginPage.login(TEST_USER.email, TEST_USER.password)
    await expect(page).toHaveURL(/\/dashboard/)

    await page.getByRole('button', { name: 'Logout' }).click()

    await expect(page).toHaveURL(/\/login/)
  })
})
