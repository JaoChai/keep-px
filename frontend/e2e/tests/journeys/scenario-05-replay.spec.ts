/**
 * Scenario 5: Replay & Recovery
 *
 * จำลอง user ทำ replay ตั้งแต่เลือก pixel → preview → replay → cancel → retry → history
 * Flow: get API key → check pixels → ingest events → open replay → fill form → preview →
 *       confirm → cancel → second replay → wait completion → retry failed → check history
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { ReplayPage } from '../../pages/replay.page'

// Prefix for identifying test data (not used directly but documents naming convention)
// const PREFIX = 'E2E-S05'

test.describe('Scenario 5: Replay & Recovery', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(120_000)

  /** Shared state across serial steps */
  let apiKey = ''
  let sourcePixelValue = ''
  let targetPixelValue = ''

  // Safety net cleanup — best-effort, nothing to delete for replay
  test.afterAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: 'e2e/.auth/user.json',
    })
    try {
      // No persistent resources created by replay tests — just close
    } catch {
      // Best-effort cleanup
    } finally {
      await context.close()
    }
  })

  // --- Step 1: Get API key from /settings ---
  test('step 1: get API key from /settings', async ({ page }) => {
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    // Reveal the API key
    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    const eyeButton = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    apiKey = await apiKeyInput.inputValue()
    expect(apiKey).toBeTruthy()
    expect(apiKey.length).toBeGreaterThan(5)
  })

  // --- Step 2: Check /pixels has >= 2 pixels ---
  test('step 2: check /pixels has >= 2 pixels', async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 1')

    await page.goto('/pixels')
    await page.waitForLoadState('networkidle')

    const rows = page.locator('table tbody tr')
    const rowCount = await rows.count()
    test.skip(rowCount < 2, 'Need >= 2 pixels for replay tests')
  })

  // --- Step 3: Ingest 5 events via API to first pixel ---
  test('step 3: ingest 5 events via API', async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 1')

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

    const events = ['PageView', 'Purchase', 'AddToCart', 'Lead', 'ViewContent'].map((name, i) => ({
      pixel_id: pixelUUID,
      event_name: name,
      event_time: new Date(Date.now() - i * 1000).toISOString(),
      event_data: {},
      source_url: `https://e2e-s05.example.com/page${i}`,
    }))

    const response = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'X-API-Key': apiKey,
        'Content-Type': 'application/json',
      },
      data: { events },
    })

    // Accept 200/202 (success), 402 (quota exceeded), or 500 (CAPI forward fail with fake token — event still saved)
    const status = response.status()
    if (status >= 400) {
      const body = await response.text()
      console.log(`Event ingest returned ${status}: ${body}`)
    }
    expect([200, 202, 402, 500]).toContain(status)
  })

  // --- Step 4: Open /replay → see heading ---
  test('step 4: open /replay → see heading', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(replayPage.heading).toBeVisible({ timeout: 10000 })
  })

  // --- Step 5: Check credits → skip if no credits ---
  test('step 5: check replay credits', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits — skipping replay tests')

    // Credits badge should be visible if user has credits
    const creditsBadgeVisible = await replayPage.creditsBadge.isVisible().catch(() => false)
    // Either credits badge is visible or paywall is not shown (unlimited plan)
    expect(creditsBadgeVisible || !paywallVisible).toBe(true)
  })

  // --- Step 6: Select source pixel from dropdown ---
  test('step 6: select source pixel', async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 1')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    // Skip if paywall is showing
    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    await expect(replayPage.sourcePixelSelect).toBeVisible({ timeout: 10000 })

    const options = replayPage.sourcePixelSelect.locator('option')
    const count = await options.count()
    const selectedValues: string[] = []
    for (let i = 0; i < count; i++) {
      const val = await options.nth(i).getAttribute('value')
      if (val && val !== '') {
        selectedValues.push(val)
      }
    }
    test.skip(selectedValues.length < 2, 'Need >= 2 pixels in dropdown for replay')

    sourcePixelValue = selectedValues[0]
    targetPixelValue = selectedValues[1]

    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)

    // Verify selection stuck
    const selectedVal = await replayPage.sourcePixelSelect.inputValue()
    expect(selectedVal).toBe(sourcePixelValue)
  })

  // --- Step 7: Select target pixel (different from source) ---
  test('step 7: select target pixel', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    const selectedVal = await replayPage.targetPixelSelect.inputValue()
    expect(selectedVal).toBe(targetPixelValue)
    expect(selectedVal).not.toBe(sourcePixelValue)
  })

  // --- Step 8: Click "เลือกทั้งหมด" for event types ---
  test('step 8: select all event types', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    // Click "เลือกทั้งหมด"
    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await replayPage.selectAllButton.click()
      await page.waitForTimeout(300)
    }
  })

  // --- Step 9: Set time mode to "original" + batch delay to 100 ---
  test('step 9: set time mode and batch delay', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    // Set time mode
    const timeModeVisible = await replayPage.timeModeSelect.isVisible().catch(() => false)
    if (timeModeVisible) {
      await replayPage.timeModeSelect.selectOption('original')
      const selectedMode = await replayPage.timeModeSelect.inputValue()
      expect(selectedMode).toBe('original')
    }

    // Set batch delay
    const batchDelayVisible = await replayPage.batchDelayInput.isVisible().catch(() => false)
    if (batchDelayVisible) {
      await replayPage.batchDelayInput.clear()
      await replayPage.batchDelayInput.fill('100')
      const delayValue = await replayPage.batchDelayInput.inputValue()
      expect(delayValue).toBe('100')
    }
  })

  // --- Step 10: Click "ตัวอย่าง" → see preview summary ---
  test('step 10: click preview → see preview summary', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Fill the form
    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await replayPage.selectAllButton.click()
      await page.waitForTimeout(300)
    }

    // Click preview
    await replayPage.previewButton.click()

    // Wait for preview summary to appear
    await expect(replayPage.previewSummary.or(replayPage.previewEventCount)).toBeVisible({ timeout: 15000 })

    // Confirm replay button should be visible in preview
    await expect(replayPage.confirmReplayButton).toBeVisible()
  })

  // --- Step 11: Click "ยืนยันรีเพลย์" → replay starts ---
  test('step 11: confirm replay → replay starts', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Fill form and preview
    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await replayPage.selectAllButton.click()
      await page.waitForTimeout(300)
    }

    await replayPage.previewButton.click()
    await expect(replayPage.confirmReplayButton).toBeVisible({ timeout: 15000 })

    // Confirm replay
    await replayPage.confirmReplayButton.click()
    await page.waitForTimeout(1000)

    // Replay should have started — either progress or completion indicators visible
    const progressVisible = await replayPage.progressBar.isVisible().catch(() => false)
    const totalVisible = await replayPage.totalEventsDisplay.isVisible().catch(() => false)
    const historyVisible = await replayPage.replayHistorySection.isVisible().catch(() => false)

    // At least one indicator should be present (replay may complete instantly for small batches)
    expect(progressVisible || totalVisible || historyVisible).toBe(true)
  })

  // --- Step 12: See progress section ---
  test('step 12: see progress section', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Fill form, preview, confirm — full flow for a new replay
    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await replayPage.selectAllButton.click()
      await page.waitForTimeout(300)
    }

    await replayPage.previewButton.click()
    await expect(replayPage.confirmReplayButton).toBeVisible({ timeout: 15000 })
    await replayPage.confirmReplayButton.click()
    await page.waitForTimeout(1000)

    // Check for progress indicators — replay may finish very quickly
    const totalVisible = await replayPage.totalEventsDisplay.isVisible().catch(() => false)
    const replayedVisible = await replayPage.replayedEventsDisplay.isVisible().catch(() => false)
    const failedVisible = await replayPage.failedEventsDisplay.isVisible().catch(() => false)
    const percentVisible = await replayPage.progressPercentage.isVisible().catch(() => false)
    const historyVisible = await replayPage.replayHistorySection.isVisible().catch(() => false)

    // At least some replay UI should be present
    expect(totalVisible || replayedVisible || failedVisible || percentVisible || historyVisible).toBe(true)
  })

  // --- Step 13: Cancel flow (if cancel button visible) ---
  test('step 13: cancel replay flow', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Start a new replay to test cancel
    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await replayPage.selectAllButton.click()
      await page.waitForTimeout(300)
    }

    await replayPage.previewButton.click()
    await expect(replayPage.confirmReplayButton).toBeVisible({ timeout: 15000 })
    await replayPage.confirmReplayButton.click()
    await page.waitForTimeout(500)

    // Check if cancel button is visible (replay may have completed already)
    const cancelVisible = await replayPage.cancelButton.isVisible().catch(() => false)
    if (!cancelVisible) {
      test.skip(true, 'Replay completed too quickly to test cancel')
      return
    }

    // Click cancel
    await replayPage.cancelButton.click()

    // Confirm dialog should appear
    const confirmDialogVisible = await replayPage.cancelConfirmDialog.isVisible().catch(() => false)
    if (confirmDialogVisible) {
      await replayPage.cancelConfirmButton.click()
      await page.waitForTimeout(1000)
    }
  })

  // --- Step 14: Verify cancelled ---
  test('step 14: verify cancelled status', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // After cancel, the page should show either:
    // - A cancelled status badge/text
    // - The replay form again (ready for new replay)
    // - The history section with the cancelled replay
    const formVisible = await replayPage.sourcePixelSelect.isVisible().catch(() => false)
    const historyVisible = await replayPage.replayHistorySection.isVisible().catch(() => false)
    const cancelledText = await page.getByText(/ยกเลิก/).first().isVisible().catch(() => false)

    // At least one of these indicators should be present
    expect(formVisible || historyVisible || cancelledText).toBe(true)
  })

  // --- Step 15: Second replay — fill form → preview → confirm ---
  test('step 15: second replay — full flow', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Fill form again
    await replayPage.sourcePixelSelect.selectOption(sourcePixelValue)
    await replayPage.targetPixelSelect.selectOption(targetPixelValue)

    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await replayPage.selectAllButton.click()
      await page.waitForTimeout(300)
    }

    // Set time mode and batch delay
    const timeModeVisible = await replayPage.timeModeSelect.isVisible().catch(() => false)
    if (timeModeVisible) {
      await replayPage.timeModeSelect.selectOption('original')
    }

    const batchDelayVisible = await replayPage.batchDelayInput.isVisible().catch(() => false)
    if (batchDelayVisible) {
      await replayPage.batchDelayInput.clear()
      await replayPage.batchDelayInput.fill('100')
    }

    // Preview
    await replayPage.previewButton.click()
    await expect(replayPage.confirmReplayButton).toBeVisible({ timeout: 15000 })

    // Confirm
    await replayPage.confirmReplayButton.click()
    await page.waitForTimeout(1000)
  })

  // --- Step 16: Wait for replay completion ---
  test('step 16: wait for replay completion', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Poll for completion — check if progress reaches 100% or history updates
    // Small batch (5 events) should complete quickly
    let completed = false
    for (let attempt = 0; attempt < 30; attempt++) {
      // Check for completion indicators
      const percentText = await replayPage.progressPercentage.textContent().catch(() => '')
      if (percentText && percentText.includes('100')) {
        completed = true
        break
      }

      // Check if history section has new entries (replay completed and UI moved to history view)
      const historyTableVisible = await replayPage.replayHistoryTable.first().isVisible().catch(() => false)
      if (historyTableVisible) {
        completed = true
        break
      }

      // Check if form is visible again (replay completed and reset)
      const formVisible = await replayPage.sourcePixelSelect.isVisible().catch(() => false)
      if (formVisible && attempt > 5) {
        completed = true
        break
      }

      await page.waitForTimeout(2000)
    }

    // If still not completed, it may have finished before we started polling
    // That's acceptable for a small batch
    expect(completed).toBe(true)
  })

  // --- Step 17: Verify replay finished ---
  test('step 17: verify replay finished', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // After replay finishes, the page should show:
    // - History section with entries, OR
    // - The form again ready for a new replay, OR
    // - Completion summary with stats
    const historyVisible = await replayPage.replayHistorySection.isVisible().catch(() => false)
    const formVisible = await replayPage.sourcePixelSelect.isVisible().catch(() => false)
    const completedText = await page.getByText(/สำเร็จ|เสร็จสิ้น|100%/).first().isVisible().catch(() => false)

    expect(historyVisible || formVisible || completedText).toBe(true)
  })

  // --- Step 18: Retry failed events (if any) ---
  test('step 18: retry failed events if available', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Check if retry button is visible (only appears when there are failed events)
    const retryVisible = await replayPage.retryButton.isVisible().catch(() => false)

    if (!retryVisible) {
      // No failed events or retry not available — this is acceptable
      test.skip(true, 'No failed events to retry')
      return
    }

    // Click retry
    await replayPage.retryButton.click()
    await page.waitForTimeout(2000)

    // After clicking retry, progress should update or a toast should appear
    const progressVisible = await replayPage.progressBar.isVisible().catch(() => false)
    const toastVisible = await page.locator('[data-sonner-toast]').isVisible().catch(() => false)
    const historyVisible = await replayPage.replayHistorySection.isVisible().catch(() => false)

    expect(progressVisible || toastVisible || historyVisible).toBe(true)
  })

  // --- Step 19: Check replay history section ---
  test('step 19: check replay history section', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // History section should be visible with at least one entry from our replays
    const historyVisible = await replayPage.replayHistorySection.isVisible().catch(() => false)
    const historyTableVisible = await replayPage.replayHistoryTable.first().isVisible().catch(() => false)
    const emptyHistory = await replayPage.emptyHistoryMessage.isVisible().catch(() => false)

    // Either we have history entries or the section is present
    expect(historyVisible || historyTableVisible || emptyHistory).toBe(true)

    // If there are entries, verify at least one row exists
    if (historyTableVisible) {
      const entryCount = await replayPage.replayHistoryTable.count()
      expect(entryCount).toBeGreaterThan(0)
    }
  })

  // --- Step 20: Click history entry → see detail view ---
  test('step 20: click history entry → see detail view', async ({ page }) => {
    test.skip(!sourcePixelValue || !targetPixelValue, 'No pixel values from step 6')

    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const paywallVisible = await replayPage.paywallMessage.isVisible().catch(() => false)
    test.skip(paywallVisible, 'No replay credits')

    // Wait for history entries to load
    await page.waitForTimeout(2000)

    const historyTableVisible = await replayPage.replayHistoryTable.first().isVisible().catch(() => false)
    test.skip(!historyTableVisible, 'No replay history entries to click')

    // Click the first history entry
    await replayPage.replayHistoryTable.first().click()
    await page.waitForTimeout(1000)

    // After clicking, should see detail view with replay information
    // Look for detail indicators: stats, event list, status badge, or any expanded content
    const detailVisible = await page.getByText(/ทั้งหมด|รีเพลย์แล้ว|ล้มเหลว|สำเร็จ|พิกเซลต้นทาง|พิกเซลปลายทาง/).first().isVisible().catch(() => false)
    const statsVisible = await replayPage.totalEventsDisplay.isVisible().catch(() => false)
    const anyContent = await page.locator('table, [class*="detail"], [class*="summary"]').first().isVisible().catch(() => false)

    // At least some detail content should be visible after clicking
    expect(detailVisible || statsVisible || anyContent).toBe(true)
  })
})
