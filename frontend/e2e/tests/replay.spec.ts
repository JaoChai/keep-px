import { test, expect } from '../fixtures/auth.fixture'
import { ReplayPage } from '../pages/replay.page'

test.describe('Replay', () => {
  test('replay form elements visible', async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()

    await expect(replayPage.heading).toBeVisible()
    await expect(replayPage.sourcePixelSelect).toBeVisible()
    await expect(replayPage.targetPixelSelect).toBeVisible()
    await expect(replayPage.dateFromInput).toBeVisible()
    await expect(replayPage.dateToInput).toBeVisible()
    await expect(replayPage.startReplayButton).toBeVisible()
  })
})
