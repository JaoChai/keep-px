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
