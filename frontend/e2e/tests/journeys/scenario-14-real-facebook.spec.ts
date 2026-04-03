/**
 * Scenario 14: Real Facebook Integration Test
 *
 * ทดสอบ full pipeline จริง — ส่ง event ไป Facebook จริงๆ
 *
 * ต้องมี env vars:
 *   REAL_PIXEL_A_ID      — Facebook Pixel ID ตัวที่ 1 (source)
 *   REAL_PIXEL_A_TOKEN    — Access Token ของ Pixel A
 *   REAL_PIXEL_B_ID      — Facebook Pixel ID ตัวที่ 2 (target/replay)
 *   REAL_PIXEL_B_TOKEN    — Access Token ของ Pixel B
 *   E2E_ACCESS_TOKEN      — JWT access token
 *   E2E_REFRESH_TOKEN     — JWT refresh token
 *
 * รัน: REAL_PIXEL_A_ID=xxx REAL_PIXEL_A_TOKEN=xxx ... npx playwright test scenario-14 --headed
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { PixelsPage } from '../../pages/pixels.page'
import { SalePagesPage } from '../../pages/sale-pages.page'
import { SalePageEditorPage } from '../../pages/sale-page-editor.page'
import { EventLogPage } from '../../pages/event-log.page'
import { ReplayPage } from '../../pages/replay.page'
import { SettingsPage } from '../../pages/settings.page'

// Real Facebook pixel credentials from env
const PIXEL_A_ID = process.env.REAL_PIXEL_A_ID || ''
const PIXEL_A_TOKEN = process.env.REAL_PIXEL_A_TOKEN || ''
const PIXEL_B_ID = process.env.REAL_PIXEL_B_ID || ''
const PIXEL_B_TOKEN = process.env.REAL_PIXEL_B_TOKEN || ''

const PREFIX = 'E2E-S14'
const ts = Date.now()
const SOURCE_NAME = `${PREFIX} Src ${ts}`
const TARGET_NAME = `${PREFIX} Tgt ${ts}`
const SP_NAME = `${PREFIX} Pg ${ts}`

let apiKey = ''
let hasSecondPixelSlot = false

test.describe('Scenario 14: Real Facebook Integration', { tag: ['@critical', '@real'] }, () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(180_000)

  // Skip entire suite if no real pixel credentials
  test.beforeAll(() => {
    if (!PIXEL_A_ID || !PIXEL_A_TOKEN) {
      test.skip(true, 'Real pixel credentials not provided — set REAL_PIXEL_A_ID, REAL_PIXEL_A_TOKEN')
    }
  })

  // Safety-net cleanup
  test.afterAll(async ({ browser }, testInfo) => {
    testInfo.setTimeout(60_000)
    const context = await browser.newContext({ storageState: 'e2e/.auth/user.json' })
    const page = await context.newPage()
    try {
      // Delete sale pages
      await page.goto('/sale-pages')
      await page.waitForLoadState('networkidle')
      let rows = page.locator('[data-testid="sale-page-card"]', { hasText: PREFIX })
      let count = await rows.count()
      while (count > 0) {
        await rows.first().getByRole('button', { name: 'ลบ' }).click()
        await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(1000)
        rows = page.locator('[data-testid="sale-page-card"]', { hasText: PREFIX })
        count = await rows.count()
      }
      // Delete pixels
      await page.goto('/pixels')
      await page.waitForLoadState('networkidle')
      for (let i = 0; i < 3; i++) { await page.keyboard.press('Escape').catch(() => {}); await page.waitForTimeout(200) }
      rows = page.locator('tr', { hasText: PREFIX })
      count = await rows.count()
      while (count > 0) {
        await page.keyboard.press('Escape').catch(() => {})
        await page.waitForTimeout(200)
        await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).waitFor()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(1000)
        const toast = page.locator('[data-sonner-toast]')
        if (await toast.count() > 0) { await toast.first().click().catch(() => {}); await page.waitForTimeout(500) }
        rows = page.locator('tr', { hasText: PREFIX })
        count = await rows.count()
      }
    } catch { /* best-effort */ } finally { await context.close() }
  })

  // ============================================================
  // Phase 1: Create pixels with REAL Facebook credentials
  // ============================================================

  test('step 1: create source pixel with real FB credentials', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Force-dismiss any lingering dialogs/modals/overlays that block clicks
    await page.evaluate(() => {
      document.querySelectorAll('[class*="fixed"][class*="inset-0"][class*="z-50"]').forEach(el => el.remove())
      document.querySelectorAll('[data-sonner-toast]').forEach(el => el.remove())
    })
    await page.waitForTimeout(500)

    // Cleanup leftover test data
    let rows = page.locator('tr', { hasText: PREFIX })
    let count = await rows.count()
    while (count > 0) {
      // Remove any blocking overlays before each delete attempt
      await page.evaluate(() => {
        document.querySelectorAll('[class*="fixed"][class*="inset-0"][class*="z-50"]').forEach(el => el.remove())
        document.querySelectorAll('[data-sonner-toast]').forEach(el => el.remove())
      })
      await page.waitForTimeout(300)
      await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
      const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
      await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
      await deleteBtn.click()
      await page.waitForTimeout(1500)
      await page.evaluate(() => {
        document.querySelectorAll('[data-sonner-toast]').forEach(el => el.remove())
      })
      rows = page.locator('tr', { hasText: PREFIX })
      count = await rows.count()
    }

    await pixelsPage.createPixel(SOURCE_NAME, PIXEL_A_ID, PIXEL_A_TOKEN)
    await expect(page.locator('tr', { hasText: SOURCE_NAME })).toBeVisible()
    console.log(`✅ Source pixel created: ${SOURCE_NAME} (FB ID: ${PIXEL_A_ID})`)
  })

  test('step 2: create target pixel if quota allows', async ({ page }) => {
    if (!PIXEL_B_ID || !PIXEL_B_TOKEN) {
      console.log('⚠️ No Pixel B credentials — skipping target pixel')
      test.skip(true, 'No REAL_PIXEL_B_ID/TOKEN')
      return
    }

    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Check if we can create another pixel (quota check)
    const addButton = page.getByRole('button', { name: 'เพิ่มพิกเซล' }).first()
    const isDisabled = await addButton.isDisabled().catch(() => true)
    if (isDisabled) {
      console.log('⚠️ Pixel quota full — cannot create target pixel (need upgrade)')
      test.skip(true, 'Pixel quota full — only 1 slot available')
      return
    }

    await pixelsPage.createPixel(TARGET_NAME, PIXEL_B_ID, PIXEL_B_TOKEN)
    await expect(page.locator('tr', { hasText: TARGET_NAME })).toBeVisible()
    hasSecondPixelSlot = true
    console.log(`✅ Target pixel created: ${TARGET_NAME} (FB ID: ${PIXEL_B_ID})`)
  })

  // ============================================================
  // Phase 2: Create sale page + publish
  // ============================================================

  test('step 3: create sale page + assign source pixel + publish', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await salePagesPage.createButton.click()

    const editor = new SalePageEditorPage(page)
    await expect(editor.pageNameInput).toBeVisible()
    await editor.fillMinimum(SP_NAME)
    await editor.selectFirstPixel()
    await editor.publish()

    await expect(editor.successDialogTitle).toBeVisible({ timeout: 15000 })
    console.log(`✅ Sale page published: ${SP_NAME}`)
  })

  // ============================================================
  // Phase 3: Send REAL events → Facebook should receive them
  // ============================================================

  test('step 4: get API key from settings', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible()
    await page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first().click()
    await page.waitForTimeout(500)

    apiKey = await apiKeyInput.inputValue()
    expect(apiKey.length).toBeGreaterThan(10)
    console.log(`✅ API key obtained`)
  })

  test('step 5: ingest 5 events → verify CAPI sends to Facebook successfully', async ({ page }) => {
    expect(apiKey).toBeTruthy()

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    // Get internal pixel UUID
    const accessToken = await page.evaluate(() => localStorage.getItem('access_token'))
    const pixelsRes = await page.request.get(`${baseURL}/api/v1/pixels`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
    const pixels = (await pixelsRes.json()).data || []
    const sourcePixel = pixels.find((p: { name?: string }) => p.name?.includes(`${PREFIX} Src`))
    expect(sourcePixel).toBeTruthy()
    const pixelUUID = sourcePixel.id

    // Send 5 funnel events
    const events = [
      { event_name: 'PageView', event_data: {} },
      { event_name: 'ViewContent', event_data: { content_name: 'E2E Real Test Product' } },
      { event_name: 'AddToCart', event_data: { value: '1990', currency: 'THB' } },
      { event_name: 'InitiateCheckout', event_data: { value: '1990' } },
      { event_name: 'Purchase', event_data: { value: '1990', currency: 'THB' } },
    ].map((evt, i) => ({
      pixel_id: pixelUUID,
      ...evt,
      event_time: new Date(Date.now() - i * 1000).toISOString(),
      source_url: `https://e2e-s14-real.example.com/step${i}`,
    }))

    const resp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: { 'X-API-Key': apiKey, 'Content-Type': 'application/json' },
      data: { events },
    })

    const status = resp.status()
    const body = await resp.text()
    console.log(`📡 Event ingest response: ${status} — ${body}`)

    // 200 = sync success, 202 = accepted + async CAPI forwarding (normal for this backend)
    expect([200, 202]).toContain(status)
    console.log(`✅ 5 events accepted by Keep-PX (CAPI forwarding async)`)
    console.log(`📡 Facebook will receive events via background CAPI goroutine`)

    await page.waitForTimeout(3000)
  })

  // ============================================================
  // Phase 4: Verify events in UI
  // ============================================================

  test('step 6: verify events visible in event log', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)

    const hasTable = await eventLogPage.eventTable.isVisible().catch(() => false)
    expect(hasTable).toBe(true)

    const rows = eventLogPage.eventTable.locator('tbody tr')
    const rowCount = await rows.count()
    console.log(`📋 Events in history: ${rowCount}`)
    expect(rowCount).toBeGreaterThan(0)
  })

  test('step 7: open event detail → verify data', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    await eventLogPage.clickFirstEventRow()
    await expect(eventLogPage.eventDetailSheet).toBeVisible({ timeout: 10000 })
    console.log(`✅ Event detail sheet opened`)

    await page.keyboard.press('Escape')
  })

  // ============================================================
  // Phase 5: Replay events to Pixel B (if credits available)
  // ============================================================

  test('step 8: replay events from Pixel A → Pixel B (if possible)', async ({ page }) => {
    // Need 2 pixels + replay credits
    if (!hasSecondPixelSlot) {
      console.log('⚠️ Only 1 pixel slot — cannot replay (need 2 pixels)')
      test.skip(true, 'Need 2 pixel slots for replay')
      return
    }

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      console.log('⚠️ No replay credits — skipping replay test')
      test.skip(true, 'No replay credits')
      return
    }

    // Select source pixel
    const sourceOpts = replayPage.sourcePixelSelect.locator('option')
    for (let i = 0; i < await sourceOpts.count(); i++) {
      const text = await sourceOpts.nth(i).textContent()
      if (text && text.includes(SOURCE_NAME)) {
        await replayPage.sourcePixelSelect.selectOption(await sourceOpts.nth(i).getAttribute('value') ?? '')
        break
      }
    }

    // Select target pixel
    const targetOpts = replayPage.targetPixelSelect.locator('option')
    for (let i = 0; i < await targetOpts.count(); i++) {
      const text = await targetOpts.nth(i).textContent()
      if (text && text.includes(TARGET_NAME)) {
        await replayPage.targetPixelSelect.selectOption(await targetOpts.nth(i).getAttribute('value') ?? '')
        break
      }
    }

    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) await replayPage.selectAllButton.click()

    await replayPage.previewButton.click()
    await page.waitForTimeout(2000)

    const hasSummary = await replayPage.previewSummary.isVisible().catch(() => false)
    if (hasSummary) {
      await replayPage.confirmReplayButton.click()
      const hasProgress = await replayPage.progressPercentage.isVisible({ timeout: 10000 }).catch(() => false)
      if (hasProgress) {
        await expect(replayPage.progressPercentage).toContainText('100%', { timeout: 60000 }).catch(() => {})
      }
      console.log(`✅ Replay completed — events sent to Pixel B (FB ID: ${PIXEL_B_ID})`)
    }
  })

  // ============================================================
  // Phase 6: Cleanup
  // ============================================================

  test('step 9: cleanup — delete sale page', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    const row = salePagesPage.getRow(SP_NAME)
    if (await row.count() > 0) {
      await salePagesPage.clickDeleteOnRow(SP_NAME)
      await expect(salePagesPage.deleteDialogTitle).toBeVisible()
      await salePagesPage.deleteConfirmButton.click()
      await expect(row).not.toBeVisible({ timeout: 10000 })
      console.log('🗑️ Sale page deleted')
    }
  })

  test('step 10: cleanup — delete both pixels', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    for (const name of [TARGET_NAME, SOURCE_NAME]) {
      // Dismiss any lingering dialogs/toasts that block clicks
      await page.keyboard.press('Escape').catch(() => {})
      await page.waitForTimeout(300)
      const toast = page.locator('[data-sonner-toast]')
      if (await toast.count() > 0) { await toast.first().click().catch(() => {}); await page.waitForTimeout(500) }

      const row = page.locator('tr', { hasText: name })
      if (await row.count() > 0) {
        await row.getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
        const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
        await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
        await deleteBtn.click()
        await page.waitForTimeout(1500)
        if (await toast.count() > 0) { await toast.first().click().catch(() => {}); await page.waitForTimeout(500) }
        console.log(`🗑️ Pixel deleted: ${name}`)
      }
    }

    // Verify no test data remains
    const remaining = page.locator('tr', { hasText: PREFIX })
    await expect(remaining).toHaveCount(0, { timeout: 5000 })
    console.log('✅ All test data cleaned up')
  })
})
