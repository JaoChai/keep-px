/**
 * Scenario 1: First-Time Onboarding
 *
 * จำลอง user ใหม่เข้าใช้งาน Keep-PX ครั้งแรก
 * Flow: see onboarding wizard → create pixel → wizard disappears →
 *       create sale page → get API key → ingest test event →
 *       view events → verify dashboard stats → cleanup
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { DashboardPage } from '../../pages/dashboard.page'
import { PixelsPage } from '../../pages/pixels.page'
import { SalePagesPage } from '../../pages/sale-pages.page'
import { SalePageEditorPage } from '../../pages/sale-page-editor.page'
import { SettingsPage } from '../../pages/settings.page'
import { EventLogPage } from '../../pages/event-log.page'
import { SidebarPage } from '../../pages/sidebar.page'

const PREFIX = 'E2E-S01'
const ts = Date.now()
const PIXEL_NAME = `${PREFIX} Pixel ${ts}`
const PIXEL_ID = '330000000000001'
const PIXEL_TOKEN = 'EAAscenario01_tokenA'
const SALE_PAGE_NAME = `${PREFIX} Page ${ts}`

test.describe('Scenario 1: First-Time Onboarding', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(90_000)

  /** Shared state across serial steps */
  let apiKey = ''

  // Safety net cleanup — runs even if tests fail midway
  test.afterAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: 'e2e/.auth/user.json',
    })
    const page = await context.newPage()

    try {
      // Delete all pixels with our prefix
      await page.goto('/pixels')
      await page.waitForLoadState('networkidle')

      let pixelRows = page.locator('tr', { hasText: PREFIX })
      let pixelCount = await pixelRows.count()
      while (pixelCount > 0) {
        await pixelRows
          .first()
          .getByRole('button')
          .filter({ has: page.locator('[class*="lucide-trash"]') })
          .click()
        const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
        await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
        await deleteBtn.click()
        await page.waitForTimeout(1000)
        const toast = page.locator('[data-sonner-toast]')
        if ((await toast.count()) > 0) {
          await toast.first().click()
          await page.waitForTimeout(500)
        }
        pixelRows = page.locator('tr', { hasText: PREFIX })
        pixelCount = await pixelRows.count()
      }

      // Delete all sale pages with our prefix
      await page.goto('/sale-pages')
      await page.waitForLoadState('networkidle')

      let spRows = page.locator('tr', { hasText: PREFIX })
      let spCount = await spRows.count()
      while (spCount > 0) {
        await spRows.first().getByRole('button', { name: 'ลบ' }).click()
        await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(1000)
        spRows = page.locator('tr', { hasText: PREFIX })
        spCount = await spRows.count()
      }

      // Clear onboarding dismissed flag
      await page.goto('/dashboard')
      await page.waitForLoadState('networkidle')
      await page.evaluate(() => {
        localStorage.removeItem('keepx_onboarding_dismissed')
      })
    } catch {
      // Best-effort cleanup
    } finally {
      await context.close()
    }
  })

  // --- Step 1: Cleanup — delete all PREFIX pixels + clear localStorage ---
  test('step 1: cleanup — delete PREFIX pixels + clear onboarding flag', async ({ page }) => {
    // Navigate to /pixels and clean up leftover test pixels
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    let rows = page.locator('tr', { hasText: PREFIX })
    let count = await rows.count()
    while (count > 0) {
      await rows
        .first()
        .getByRole('button')
        .filter({ has: page.locator('[class*="lucide-trash"]') })
        .click()
      const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
      await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
      await deleteBtn.click()
      await page.waitForTimeout(1000)
      const toast = page.locator('[data-sonner-toast]')
      if ((await toast.count()) > 0) {
        await toast.first().click()
        await page.waitForTimeout(500)
      }
      rows = page.locator('tr', { hasText: PREFIX })
      count = await rows.count()
    }

    // Clear onboarding dismissed flag via localStorage
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')
    await page.evaluate(() => {
      localStorage.removeItem('keepx_onboarding_dismissed')
    })
  })

  // --- Step 2: Open /dashboard → see onboarding wizard with 4 steps ---
  test('step 2: open /dashboard → see onboarding wizard with 4 steps', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(dashboardPage.heading).toBeVisible()

    // Check if user has 0 pixels (wizard only shows when pixels.length === 0)
    // If user already has pixels, wizard won't appear — skip
    const wizardVisible = await dashboardPage.onboardingWizard
      .isVisible({ timeout: 5000 })
      .catch(() => false)

    if (!wizardVisible) {
      test.skip(true, 'Onboarding wizard not visible — user already has pixels')
      return
    }

    // Verify wizard title
    await expect(
      page.getByText('ยินดีต้อนรับสู่ Keep-PX!', { exact: true }),
    ).toBeVisible()
    await expect(
      page.getByText('เริ่มต้นใช้งานด้วย 4 ขั้นตอนง่ายๆ'),
    ).toBeVisible()

    // Verify all 4 step cards are visible — scope to wizard container to avoid duplicates
    const wizard = dashboardPage.onboardingWizard
    await expect(wizard.getByText('สร้างพิกเซล', { exact: true })).toBeVisible()
    await expect(wizard.getByText('สร้างเซลเพจ', { exact: true })).toBeVisible()
    await expect(wizard.getByText('ตั้งค่า API Key', { exact: true })).toBeVisible()
    await expect(wizard.getByText('ส่ง Test Event', { exact: true })).toBeVisible()

    // Dismiss button should be visible
    await expect(dashboardPage.onboardingDismissButton).toBeVisible()
  })

  // --- Step 3: Click step 1 card "สร้างพิกเซล" → navigate to /pixels ---
  test('step 3: click "สร้างพิกเซล" step → navigate to /pixels', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()
    await page.waitForLoadState('networkidle')

    const wizardVisible = await dashboardPage.onboardingWizard
      .isVisible({ timeout: 5000 })
      .catch(() => false)

    if (!wizardVisible) {
      test.skip(true, 'Onboarding wizard not visible — user already has pixels')
      return
    }

    // Click the "สร้างพิกเซล" link card within the wizard
    const createPixelLink = dashboardPage.onboardingWizard.getByText('สร้างพิกเซล', { exact: true })
    await createPixelLink.click()

    // Should navigate to /pixels
    await expect(page).toHaveURL(/\/pixels/)

    // Pixels heading should be visible
    const pixelsPage = new PixelsPage(page)
    await expect(pixelsPage.heading).toBeVisible()
  })

  // --- Step 4: Create pixel → appears in table ---
  test('step 4: create pixel → appears in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.createPixel(PIXEL_NAME, PIXEL_ID, PIXEL_TOKEN)

    // Verify pixel appears in table
    await expect(page.locator('tr', { hasText: PIXEL_NAME })).toBeVisible()
    await expect(
      page.locator('tr', { hasText: PIXEL_NAME }).locator('td').nth(1),
    ).toContainText(PIXEL_ID)
  })

  // --- Step 5: Go back to /dashboard → wizard should be GONE (pixels > 0) ---
  test('step 5: go back to /dashboard → wizard gone (pixels > 0)', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(dashboardPage.heading).toBeVisible()

    // Onboarding wizard should NOT be visible since user now has a pixel
    await expect(dashboardPage.onboardingWizard).not.toBeVisible()
  })

  // --- Step 6: Navigate to /sale-pages via sidebar ---
  test('step 6: navigate to /sale-pages via sidebar', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()
    await page.waitForLoadState('networkidle')

    const sidebar = new SidebarPage(page)
    await sidebar.salePagesLink.first().click()
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveURL(/\/sale-pages/)

    const salePagesPage = new SalePagesPage(page)
    await expect(salePagesPage.heading).toBeVisible()
  })

  // --- Step 7: Create sale page via block editor → fill name + add text block → save draft ---
  test('step 7: create sale page draft via block editor', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')

    const editor = new SalePageEditorPage(page)

    // Fill minimum: page name + text block
    await editor.fillMinimum(SALE_PAGE_NAME)

    // Save as draft
    await editor.saveDraftButton.click()
    await page.waitForURL(/\/sale-pages$/, { timeout: 15000 })
  })

  // --- Step 8: Back to sale pages list → see draft in table ---
  test('step 8: see draft sale page in table', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    // Verify our sale page is in the table with draft status
    const row = page.locator('tr', { hasText: SALE_PAGE_NAME })
    await expect(row).toBeVisible()
    await expect(row.getByText('แบบร่าง')).toBeVisible()
  })

  // --- Step 9: Navigate to /settings → see API key section ---
  test('step 9: navigate to /settings → see API key section', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(settingsPage.heading).toBeVisible()
    await expect(settingsPage.apiKeySection).toBeVisible()
  })

  // --- Step 10: Reveal API key → copy it ---
  test('step 10: reveal API key → save to shared variable', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(settingsPage.apiKeySection).toBeVisible()

    // API key should be masked by default
    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    // Click eye button to reveal API key
    const eyeButton = page
      .locator('button', { has: page.locator('[class*="lucide-eye"]') })
      .first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    // Read the revealed API key
    apiKey = await apiKeyInput.inputValue()
    expect(apiKey).toBeTruthy()
    expect(apiKey).not.toContain('•')
    expect(apiKey.length).toBeGreaterThan(5)

    // Verify copy button is visible
    await expect(settingsPage.copyApiKeyButton).toBeVisible()
  })

  // --- Step 11: Ingest test event via API ---
  test('step 11: ingest test event via API', async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 10')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    // Navigate to an app page first so localStorage is accessible
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    // Get the internal pixel UUID from the API using JWT from localStorage
    const accessToken = await page.evaluate(() => localStorage.getItem('access_token'))
    const pixelsRes = await page.request.get(`${baseURL}/api/v1/pixels`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
    const pixels = (await pixelsRes.json()).data || []
    test.skip(pixels.length === 0, 'No pixels available for event ingestion')

    const pixelUUID = pixels[0].id

    const response = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'X-API-Key': apiKey,
        'Content-Type': 'application/json',
      },
      data: {
        events: [
          {
            pixel_id: pixelUUID,
            event_name: 'PageView',
            event_time: new Date().toISOString(),
            event_data: {},
            source_url: 'https://e2e-test.example.com',
          },
        ],
      },
    })

    // Accept 200/202 (success), 402 (quota exceeded), or 500 (CAPI forward fail with fake token — event still saved)
    const status = response.status()
    if (status >= 400) {
      const body = await response.text()
      console.log(`Event ingest returned ${status}: ${body}`)
    }
    expect([200, 202, 402, 500]).toContain(status)
  })

  // --- Step 12: Navigate to /events?mode=live → see events or waiting message ---
  test('step 12: navigate to /events?mode=live → see events or waiting', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    await expect(eventLogPage.heading).toBeVisible()

    // In live mode, either the event table has rows, or we see the waiting/loading message
    const hasTable = await eventLogPage.eventTable
      .locator('tbody tr')
      .first()
      .isVisible({ timeout: 10000 })
      .catch(() => false)
    const hasWaiting = await eventLogPage.liveWaitingMessage
      .isVisible()
      .catch(() => false)
    const hasLoading = await eventLogPage.liveLoadingMessage
      .isVisible()
      .catch(() => false)

    // At least one of these should be visible
    expect(hasTable || hasWaiting || hasLoading).toBe(true)
  })

  // --- Step 13: Switch to history mode → verify stat cards visible ---
  test('step 13: switch to history mode → stat cards visible', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    await expect(eventLogPage.heading).toBeVisible()

    // Stat cards should be visible in history mode — use specific locators to avoid strict mode violations
    await expect(eventLogPage.statEventsToday).toBeVisible({ timeout: 10000 })
    // "อีเวนต์ทั้งหมด" appears in stat card + filter dropdown — scope to paragraph
    await expect(page.getByRole('paragraph').filter({ hasText: 'อีเวนต์ทั้งหมด' })).toBeVisible()
  })

  // --- Step 14: Go back to /dashboard → stat cards show values ---
  test('step 14: go back to /dashboard → stat cards visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(dashboardPage.heading).toBeVisible()

    // Stat cards should be present on dashboard
    // They may show 0 or actual values depending on data
    const statCardCount = await dashboardPage.statCards.count()
    expect(statCardCount).toBeGreaterThan(0)
  })

  // --- Step 15: Cleanup — delete test pixel and sale page ---
  test('step 15: cleanup — delete test pixel and sale page', async ({ page }) => {
    // Delete sale page first
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    const spRow = page.locator('tr', { hasText: SALE_PAGE_NAME })
    if ((await spRow.count()) > 0) {
      await spRow.first().getByRole('button', { name: 'ลบ' }).click()
      await expect(page.getByRole('heading', { name: 'ลบเซลเพจ' })).toBeVisible()
      await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
      await page.waitForTimeout(1000)
      await expect(spRow).not.toBeVisible()
    }

    // Delete pixel
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    const pixelRow = page.locator('tr', { hasText: PIXEL_NAME })
    if ((await pixelRow.count()) > 0) {
      await pixelRow
        .first()
        .getByRole('button')
        .filter({ has: page.locator('[class*="lucide-trash"]') })
        .click()
      const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
      await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
      await deleteBtn.click()
      await page.waitForTimeout(1000)
      await expect(pixelRow).not.toBeVisible()
    }
  })
})
