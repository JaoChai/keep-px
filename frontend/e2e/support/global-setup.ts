import { chromium, type FullConfig } from '@playwright/test'
import { TEST_USER } from '../fixtures/test-data'
import path from 'path'
import fs from 'fs'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

async function globalSetup(_config: FullConfig) {
  const authDir = path.resolve(__dirname, '../.auth')
  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true })
  }

  const browser = await chromium.launch()
  const page = await browser.newPage()

  const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
  const storagePath = path.resolve(authDir, 'user.json')

  // Try to register the test user first (may fail if already exists)
  try {
    await page.goto(`${baseURL}/register`)
    await page.getByLabel('Name').fill(TEST_USER.name)
    await page.getByLabel('Email').fill(TEST_USER.email)
    await page.getByLabel('Password', { exact: true }).fill(TEST_USER.password)
    await page.getByLabel('Confirm Password').fill(TEST_USER.password)
    await page.getByRole('button', { name: 'Create account' }).click()
    await page.waitForURL('**/dashboard', { timeout: 10_000 })

    // Registration succeeded and auto-logged in
    await page.context().storageState({ path: storagePath })
    await browser.close()
    return
  } catch {
    // Registration failed (user may already exist) — fall through to login
  }

  // Login with existing user
  await page.goto(`${baseURL}/login`)
  await page.getByLabel('Email').fill(TEST_USER.email)
  await page.getByLabel('Password').fill(TEST_USER.password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.waitForURL('**/dashboard')

  await page.context().storageState({ path: storagePath })
  await browser.close()
}

export default globalSetup
