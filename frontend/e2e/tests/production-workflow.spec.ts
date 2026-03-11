import { test, expect } from '../fixtures/auth.fixture'
import { PixelsPage } from '../pages/pixels.page'
import { SalePagesPage } from '../pages/sale-pages.page'
import { SalePageEditorPage } from '../pages/sale-page-editor.page'
import { DashboardPage } from '../pages/dashboard.page'
import { ReplayPage } from '../pages/replay.page'

const WF_PIXEL_NAME = `E2E WF Pixel ${Date.now()}`
const WF_SP_NAME = `E2E WF SP ${Date.now()}`

test.describe('Production Workflow', () => {
  test.describe.configure({ mode: 'serial' })

  // Safety net cleanup even if tests fail midway
  test.afterAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: 'e2e/.auth/user.json',
    })
    const page = await context.newPage()

    try {
      // Clean up sale pages
      await page.goto('/sale-pages')
      await page.waitForLoadState('networkidle')
      for (const prefix of ['E2E WF SP']) {
        const rows = page.locator('tr', { hasText: prefix })
        let count = await rows.count()
        while (count > 0) {
          await rows.first().getByRole('button', { name: 'ลบ' }).click()
          await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
          await page.getByRole('button', { name: 'ลบ' }).last().click()
          await page.waitForTimeout(500)
          count = await rows.count()
        }
      }

      // Clean up pixels
      await page.goto('/pixels')
      await page.waitForLoadState('networkidle')
      for (const prefix of ['E2E WF Pixel']) {
        const rows = page.locator('tr', { hasText: prefix })
        let count = await rows.count()
        while (count > 0) {
          await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
          await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
          await page.waitForTimeout(500)
          count = await rows.count()
        }
      }
    } catch {
      // Best-effort cleanup
    } finally {
      await context.close()
    }
  })

  test('step 1: create pixel', async ({ page }) => {
    test.setTimeout(60_000)
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Clean up any leftover test pixels to free quota
    for (const prefix of ['E2E WF Pixel', 'Test Pixel', 'Edit Test', 'Updated', 'Delete Test']) {
      let rows = page.locator('tr', { hasText: prefix })
      let count = await rows.count()
      while (count > 0) {
        await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
        // Wait for delete dialog
        const destructiveBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
        await destructiveBtn.waitFor({ state: 'visible', timeout: 5000 })
        await destructiveBtn.click()
        await page.waitForTimeout(1000)
        // Dismiss any error toast that might intercept clicks
        const toast = page.locator('[data-sonner-toast]')
        if (await toast.count() > 0) {
          await toast.first().click()
          await page.waitForTimeout(500)
        }
        rows = page.locator('tr', { hasText: prefix })
        count = await rows.count()
      }
    }

    // Dismiss any remaining toasts before creating
    const remainingToast = page.locator('[data-sonner-toast]')
    if (await remainingToast.count() > 0) {
      await remainingToast.first().click()
      await page.waitForTimeout(500)
    }

    await pixelsPage.createPixel(WF_PIXEL_NAME, '111222333444555', 'EAAworkflow_test')
    await expect(page.getByText(WF_PIXEL_NAME)).toBeVisible()
  })

  test('step 2: create sale page linked to pixel and publish', async ({ page }) => {
    test.setTimeout(60_000)
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await salePagesPage.createButton.click()

    // Wait for block editor to load
    const editor = new SalePageEditorPage(page)
    await expect(editor.pageNameInput).toBeVisible()

    await editor.fillMinimum(WF_SP_NAME)

    // Select the pixel we created (checkbox inside collapsible that's already open)
    const pixelCheckbox = page.locator('label', { hasText: WF_PIXEL_NAME }).locator('input[type="checkbox"]')
    if (await pixelCheckbox.count() > 0) {
      await pixelCheckbox.check()
    }

    await editor.publish()

    // Wait longer for production API response
    await expect(editor.successDialogTitle).toBeVisible({ timeout: 15000 })
    await editor.goBackButton.click()

    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(WF_SP_NAME)).toBeVisible()
  })

  test('step 3: verify dashboard loads correctly', async ({ page }) => {
    const dashboard = new DashboardPage(page)
    await dashboard.goto()

    await expect(dashboard.heading).toBeVisible()
    // Dashboard should have stat cards
    await expect(dashboard.statCards.first()).toBeVisible()
    // Chart section should load
    await expect(dashboard.chartSection).toBeVisible()
  })

  test('step 4: replay center shows paywall for sandbox', async ({ page }) => {
    const replay = new ReplayPage(page)
    await replay.goto()

    await expect(replay.heading).toBeVisible()
    // Sandbox user has no replay credits — should see paywall
    await expect(replay.paywallMessage).toBeVisible()
    await expect(replay.viewReplayPacksButton).toBeVisible()
  })

  test('step 5: cleanup test data', async ({ page }) => {
    // Delete sale page
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await page.waitForLoadState('networkidle')

    const spRow = page.locator('tr', { hasText: 'E2E WF SP' })
    if (await spRow.count() > 0) {
      await salePagesPage.clickDeleteOnRow('E2E WF SP')
      await expect(salePagesPage.deleteDialogTitle).toBeVisible()
      await salePagesPage.deleteConfirmButton.click()
      await expect(spRow).not.toBeVisible()
    }

    // Delete pixel
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    const pxRow = page.locator('tr', { hasText: 'E2E WF Pixel' })
    if (await pxRow.count() > 0) {
      await pxRow.getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
      await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
      await expect(pxRow).not.toBeVisible()
    }
  })
})
