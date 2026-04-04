/**
 * Production Smoke Tests
 *
 * ออกแบบสำหรับ production โดยเฉพาะ:
 * - อ่านอย่างเดียว — ไม่สร้าง/ลบข้อมูล
 * - State-agnostic — ผ่านไม่ว่ามีข้อมูลหรือไม่
 * - เร็ว — รันจบภายใน 2-3 นาที
 * - เสถียร — ไม่ flaky เพราะ production latency
 *
 * รัน: npx playwright test production-smoke.spec.ts
 */
import { test as base, expect } from '@playwright/test'
import { test as authTest } from '../fixtures/auth.fixture'
import { SidebarPage } from '../pages/sidebar.page'
import { LoginPage } from '../pages/login.page'

// ============================================================
// Part 1: Health Check (no auth needed)
// ============================================================
base.describe('Production Smoke @smoke', () => {
  base.describe('Health Check', () => {
    base('API /health returns 200', async ({ request }) => {
      const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
      const apiURL = baseURL.replace('pixlinks.xyz', 'api.pixlinks.xyz')
        .replace('localhost:5173', 'localhost:8080')
      const res = await request.get(`${apiURL}/health`)
      expect(res.status()).toBe(200)
    })

    base('frontend loads HTML', async ({ page }) => {
      await page.goto('/')
      await expect(page.locator('html')).toBeAttached()
    })
  })

  // ============================================================
  // Part 2: Public Pages (no auth needed)
  // ============================================================
  base.describe('Public Pages', () => {
    base('login page shows branding', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.goto()
      await expect(loginPage.heading).toBeVisible()
    })

    base('/register redirects to /login', async ({ page }) => {
      await page.goto('/register')
      await page.waitForURL(/\/login/, { timeout: 15000 })
      await expect(page).toHaveURL(/\/login/)
    })

    base('nonexistent sale page shows not found', async ({ page }) => {
      const res = await page.goto('/p/e2e-nonexistent-slug-99999')
      // Accept 404 status or "not found" text on page
      const status = res?.status() ?? 0
      if (status === 404) {
        expect(status).toBe(404)
      } else {
        await expect(page.getByText(/ไม่พบ|not found|404/i)).toBeVisible({ timeout: 10000 })
      }
    })
  })
})

// ============================================================
// Part 3: Authenticated Pages (all read-only)
// ============================================================
authTest.describe('Production Smoke — Authenticated @smoke', () => {
  authTest('dashboard loads with heading and stat cards', async ({ page }) => {
    await page.goto('/dashboard', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'แดชบอร์ด' })).toBeVisible()
    // At least one stat card visible
    await expect(page.locator('[class*="card"]').first()).toBeVisible({ timeout: 10000 })
  })

  authTest('pixels page loads', async ({ page }) => {
    await page.goto('/pixels', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'พิกเซล' })).toBeVisible()
    // Either table or empty state
    const hasTable = await page.locator('table').isVisible().catch(() => false)
    const hasEmpty = await page.getByText('ยังไม่มีพิกเซล').isVisible().catch(() => false)
    expect(hasTable || hasEmpty).toBeTruthy()
  })

  authTest('sale pages page loads', async ({ page }) => {
    await page.goto('/sale-pages', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'เซลเพจ' })).toBeVisible()
  })

  authTest('events page loads', async ({ page }) => {
    await page.goto('/events', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'อีเวนต์' })).toBeVisible()
  })

  authTest('replay page loads', async ({ page }) => {
    await page.goto('/replay', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'รีเพลย์' })).toBeVisible()
  })

  authTest('billing page loads with quota', async ({ page }) => {
    await page.goto('/billing', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'การเงิน' })).toBeVisible()
    // Quota display visible
    await expect(page.getByText('อีเวนต์เดือนนี้')).toBeVisible({ timeout: 10000 })
  })

  authTest('settings page loads', async ({ page }) => {
    await page.goto('/settings', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'ตั้งค่า' })).toBeVisible()
    // API key section visible
    await expect(page.getByText('API Key', { exact: true })).toBeVisible()
  })

  authTest('guide page loads with sections', async ({ page }) => {
    await page.goto('/guide', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: 'คู่มือการใช้งาน' })).toBeVisible()
  })

  // ============================================================
  // Part 4: Navigation
  // ============================================================
  authTest('sidebar has all navigation links', async ({ page }) => {
    await page.goto('/dashboard', { waitUntil: 'networkidle' })
    const sidebar = new SidebarPage(page)

    await expect(sidebar.dashboardLink).toBeVisible()
    await expect(sidebar.pixelsLink).toBeVisible()
    await expect(sidebar.salePagesLink).toBeVisible()
    await expect(sidebar.eventsLink).toBeVisible()
    await expect(sidebar.replayCenterLink).toBeVisible()
    await expect(sidebar.billingLink).toBeVisible()
    await expect(sidebar.settingsLink).toBeVisible()
    await expect(sidebar.guideLink).toBeVisible()
  })

  // ============================================================
  // Part 5: API Response Check (GET only)
  // ============================================================
  authTest('API /auth/me returns user data', async ({ page, request }) => {
    // Get token from page localStorage
    await page.goto('/dashboard')
    const token = await page.evaluate(() => localStorage.getItem('access_token'))
    expect(token).toBeTruthy()

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
    const apiURL = baseURL.replace('pixlinks.xyz', 'api.pixlinks.xyz')
      .replace('localhost:5173', 'localhost:8080')

    const res = await request.get(`${apiURL}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status()).toBe(200)
    const data = await res.json()
    expect(data.data.email).toBeTruthy()
  })

  authTest('API /billing/quota returns quota data', async ({ page, request }) => {
    await page.goto('/dashboard')
    const token = await page.evaluate(() => localStorage.getItem('access_token'))

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
    const apiURL = baseURL.replace('pixlinks.xyz', 'api.pixlinks.xyz')
      .replace('localhost:5173', 'localhost:8080')

    const res = await request.get(`${apiURL}/api/v1/billing/quota`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status()).toBe(200)
    const data = await res.json()
    expect(data.data.plan).toBeTruthy()
  })
})
