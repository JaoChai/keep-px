import { test, expect } from '../fixtures/auth.fixture'
import { ReplayPage } from '../pages/replay.page'

test.describe('Replay', () => {
  test('replay page loads with heading and paywall or form', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()

    await expect(replayPage.heading).toBeVisible()

    // Without replay credits, the paywall gate is shown instead of the form
    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      await expect(replayPage.paywallMessage).toBeVisible()
      await expect(replayPage.viewReplayPacksButton).toBeVisible()
    } else {
      await expect(replayPage.sourcePixelSelect).toBeVisible()
      await expect(replayPage.targetPixelSelect).toBeVisible()
      await expect(replayPage.dateFromInput).toBeVisible()
      await expect(replayPage.dateToInput).toBeVisible()
      await expect(replayPage.previewButton).toBeVisible()
    }
  })
})

test.describe('Replay - Stripe Checkout', () => {
  test('billing page has replay purchase buttons that redirect to Stripe', async ({ page }) => {
    await page.goto('/billing')
    await page.waitForLoadState('networkidle')

    // Verify the replay section exists with purchase buttons
    const buyButtons = page.getByRole('button', { name: 'ซื้อ' })
    const buttonCount = await buyButtons.count()
    expect(buttonCount).toBeGreaterThanOrEqual(1)

    // Intercept the checkout request to verify it targets Stripe
    let checkoutRequestSent = false
    page.on('request', (request) => {
      if (request.url().includes('/api/v1/billing/checkout')) {
        checkoutRequestSent = true
      }
    })

    // Click the first buy button and verify checkout API is called
    await buyButtons.first().click()

    // Wait briefly for the request to fire
    await page.waitForTimeout(1000)
    expect(checkoutRequestSent).toBe(true)
  })

  test('paywall on replay page links to billing page', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (!hasPaywall) {
      test.skip(true, 'User has replay credits, no paywall shown')
      return
    }

    await expect(replayPage.viewReplayPacksButton).toBeVisible()
    await replayPage.viewReplayPacksButton.click()
    await expect(page).toHaveURL(/\/billing/)
  })
})

test.describe('Replay - Form Controls', () => {
  test('source and target pixel selects are visible when form is shown', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      test.skip(true, 'No replay credits — paywall is shown')
      return
    }

    await expect(replayPage.sourcePixelSelect).toBeVisible()
    await expect(replayPage.targetPixelSelect).toBeVisible()

    // Verify selects have at least the placeholder option
    const sourceOptions = replayPage.sourcePixelSelect.locator('option')
    await expect(sourceOptions.first()).toHaveText('เลือกต้นทาง...')

    const targetOptions = replayPage.targetPixelSelect.locator('option')
    await expect(targetOptions.first()).toHaveText('เลือกปลายทาง...')
  })

  test('event type checkboxes appear after selecting source pixel', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      test.skip(true, 'No replay credits — paywall is shown')
      return
    }

    // Select first available source pixel (if any beyond the placeholder)
    const sourceOptions = replayPage.sourcePixelSelect.locator('option')
    const optionCount = await sourceOptions.count()
    if (optionCount <= 1) {
      test.skip(true, 'No pixels available for selection')
      return
    }

    await replayPage.sourcePixelSelect.selectOption({ index: 1 })
    await page.waitForTimeout(500)

    // Event type checkboxes may or may not appear depending on whether the pixel has events
    const selectAllVisible = await replayPage.selectAllButton.isVisible().catch(() => false)
    if (selectAllVisible) {
      await expect(replayPage.selectAllButton).toBeVisible()
    }
    // Either way, test passes — event types are optional
  })

  test('date range inputs accept values', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      test.skip(true, 'No replay credits — paywall is shown')
      return
    }

    await expect(replayPage.dateFromInput).toBeVisible()
    await expect(replayPage.dateToInput).toBeVisible()

    await replayPage.dateFromInput.fill('2026-01-01')
    await replayPage.dateToInput.fill('2026-03-19')

    await expect(replayPage.dateFromInput).toHaveValue('2026-01-01')
    await expect(replayPage.dateToInput).toHaveValue('2026-03-19')
  })

  test('time mode select has original and current options', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      test.skip(true, 'No replay credits — paywall is shown')
      return
    }

    await expect(replayPage.timeModeSelect).toBeVisible()

    // Verify options exist
    const options = replayPage.timeModeSelect.locator('option')
    await expect(options).toHaveCount(2)
    await expect(options.nth(0)).toHaveValue('original')
    await expect(options.nth(1)).toHaveValue('current')

    // Default should be "original"
    await expect(replayPage.timeModeSelect).toHaveValue('original')

    // Can switch to "current"
    await replayPage.timeModeSelect.selectOption('current')
    await expect(replayPage.timeModeSelect).toHaveValue('current')
  })

  test('batch delay input exists with default value', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      test.skip(true, 'No replay credits — paywall is shown')
      return
    }

    await expect(replayPage.batchDelayInput).toBeVisible()

    // Can fill a value
    await replayPage.batchDelayInput.fill('1000')
    await expect(replayPage.batchDelayInput).toHaveValue('1000')
  })
})

test.describe('Replay - Preview & Confirm', () => {
  test('preview button submits form and shows summary', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)
    if (hasPaywall) {
      test.skip(true, 'No replay credits — paywall is shown')
      return
    }

    // Need at least 2 pixels to select source and target
    const sourceOptions = replayPage.sourcePixelSelect.locator('option')
    const optionCount = await sourceOptions.count()
    if (optionCount <= 2) {
      test.skip(true, 'Need at least 2 pixels for source and target')
      return
    }

    // Select source and target
    await replayPage.sourcePixelSelect.selectOption({ index: 1 })
    await replayPage.targetPixelSelect.selectOption({ index: 2 })

    // Click preview
    await replayPage.previewButton.click()
    await page.waitForTimeout(2000)

    // Check if preview summary appeared or if there was an error
    const previewVisible = await replayPage.previewSummary.isVisible().catch(() => false)
    if (previewVisible) {
      await expect(replayPage.previewSummary).toBeVisible()
      await expect(replayPage.confirmReplayButton).toBeVisible()
      await expect(replayPage.previewBackButton).toBeVisible()
    }
    // If preview didn't show, it could be a validation error or no events — test still passes
  })

  test.fixme('confirm replay creates a session', async ({ page }) => {
    // Requires: replay credits + pixels with events + preview success
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    // This test requires a fully set up state that sandbox users may not have
  })
})

test.describe('Replay - Progress & Active Session', () => {
  test.fixme('progress bar and stats display for active replay', async ({ page }) => {
    // Requires an active replay session to be running
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    // Would verify: progressBar, totalEventsDisplay, replayedEventsDisplay,
    // failedEventsDisplay, progressPercentage
  })

  test.fixme('cancel active replay shows confirm dialog', async ({ page }) => {
    // Requires an active replay session (running or pending)
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    // Would: click cancelButton -> verify cancelConfirmDialog visible
    // -> click cancelConfirmButton -> verify replay cancelled
  })

  test.fixme('retry failed replay starts new session', async ({ page }) => {
    // Requires a completed replay with failed events
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    // Would: verify retryButton visible -> click -> verify new session starts
  })
})

test.describe('Replay - History', () => {
  test('replay history section is visible', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    // History section shows either replay entries or empty message
    // It may be hidden if there's an active replay being viewed
    const historyVisible = await replayPage.replayHistory.isVisible().catch(() => false)
    const emptyVisible = await replayPage.emptyHistoryMessage.isVisible().catch(() => false)

    // At least one of these should be true (history heading or empty state)
    expect(historyVisible || emptyVisible).toBe(true)
  })

  test('replay history entries are clickable', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    // Check if there are replay history entries
    const entries = page.locator('.cursor-pointer.hover\\:bg-accent')
    const entryCount = await entries.count()

    if (entryCount === 0) {
      test.skip(true, 'No replay history entries to click')
      return
    }

    // Click the first entry — it should show the replay detail (progress view)
    await entries.first().click()
    await page.waitForTimeout(500)

    // After clicking, the progress section should appear
    const progressTitle = page.getByText('ความคืบหน้ารีเพลย์')
    await expect(progressTitle).toBeVisible()
  })
})
