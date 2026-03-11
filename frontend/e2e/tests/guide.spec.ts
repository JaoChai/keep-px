import { test, expect } from '../fixtures/auth.fixture'
import { GuidePage } from '../pages/guide.page'

test.describe('Guide', () => {
  test('page loads with heading and search', async ({ page }) => {
    const guidePage = new GuidePage(page)
    await guidePage.goto()

    await expect(guidePage.heading).toBeVisible()
    await expect(guidePage.searchInput).toBeVisible()
  })

  test('guide sections are visible', async ({ page }) => {
    const guidePage = new GuidePage(page)
    await guidePage.goto()

    // Scope to main content area to avoid matching sidebar links
    const main = page.locator('main')

    await expect(main.getByRole('button', { name: /เริ่มต้นใช้งาน/ })).toBeVisible()
    await expect(main.getByRole('button', { name: 'แดชบอร์ด', exact: true })).toBeVisible()
    await expect(main.getByRole('button', { name: 'จัดการ Pixel', exact: true })).toBeVisible()
    await expect(main.getByRole('button', { name: 'ดู Events', exact: true })).toBeVisible()
    await expect(main.getByRole('button', { name: /Replay Center/ })).toBeVisible()
    await expect(main.getByRole('button', { name: 'เซลเพจ', exact: true })).toBeVisible()
  })

  test('search filters sections', async ({ page }) => {
    const guidePage = new GuidePage(page)
    await guidePage.goto()

    const main = page.locator('main')

    await guidePage.searchInput.fill('Pixel')
    // Wait for search to filter
    await page.waitForTimeout(300)
    // Pixel-related section button should be visible (exact match to avoid subsection)
    await expect(main.getByRole('button', { name: 'จัดการ Pixel', exact: true })).toBeVisible()
  })

  test('accordion section expands on click', async ({ page }) => {
    const guidePage = new GuidePage(page)
    await guidePage.goto()

    const main = page.locator('main')

    // "แดชบอร์ด" section is collapsed by default (not the first section)
    await main.getByRole('button', { name: 'แดชบอร์ด', exact: true }).click()

    // Subsection content should appear after expanding
    await expect(main.getByRole('button', { name: 'ตัวเลขสรุป' })).toBeVisible({ timeout: 3000 })
  })
})
