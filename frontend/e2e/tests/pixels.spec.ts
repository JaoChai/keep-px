import { test, expect } from '../fixtures/auth.fixture'
import { PixelsPage } from '../pages/pixels.page'

test.describe('Pixels', () => {
  test('create pixel and see in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()

    const pixelName = `Test Pixel ${Date.now()}`
    await pixelsPage.createPixel(pixelName, '123456789012345', 'EAAtest123token')

    // Snippet dialog appears after creation - close it
    await page.getByRole('button', { name: 'Done' }).click()

    await expect(page.getByText(pixelName)).toBeVisible()
  })

  test('edit pixel name and see updated in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()

    // Create a pixel first
    const originalName = `Edit Test ${Date.now()}`
    await pixelsPage.createPixel(originalName, '123456789012345', 'EAAtest123token')
    await page.getByRole('button', { name: 'Done' }).click()

    // Click edit button on the pixel row
    const pixelRow = page.locator('tr', { hasText: originalName })
    await pixelRow.getByRole('button').filter({ has: page.locator('[class*="lucide-pencil"]') }).click()

    // Update name
    const updatedName = `Updated ${Date.now()}`
    await pixelsPage.nameInput.clear()
    await pixelsPage.nameInput.fill(updatedName)
    await page.getByRole('button', { name: 'Save Changes' }).click()

    await expect(page.getByText(updatedName)).toBeVisible()
  })

  test('delete pixel with confirmation dialog', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()

    // Create a pixel first
    const pixelName = `Delete Test ${Date.now()}`
    await pixelsPage.createPixel(pixelName, '123456789012345', 'EAAtest123token')
    await page.getByRole('button', { name: 'Done' }).click()

    // Click delete button
    const pixelRow = page.locator('tr', { hasText: pixelName })
    await pixelRow.getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()

    // Confirm deletion
    await expect(page.getByText('Are you sure?')).toBeVisible()
    await pixelsPage.deleteConfirmButton.click()

    await expect(page.getByText(pixelName)).not.toBeVisible()
  })

  test('show create pixel dialog with correct fields', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()

    await pixelsPage.addPixelButton.click()

    await expect(page.getByRole('heading', { name: 'Add Pixel' })).toBeVisible()
    await expect(pixelsPage.nameInput).toBeVisible()
    await expect(pixelsPage.pixelIdInput).toBeVisible()
    await expect(pixelsPage.accessTokenInput).toBeVisible()
    await expect(pixelsPage.cancelButton).toBeVisible()
  })
})
