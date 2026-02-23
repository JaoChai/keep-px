import { PixlinksTracker } from './tracker'
import { VisualSetup } from './visual-setup'

export { PixlinksTracker, VisualSetup }

// Auto-initialize from script tag data attributes
if (typeof window !== 'undefined' && typeof document !== 'undefined') {
  document.addEventListener('DOMContentLoaded', () => {
    const scripts = document.querySelectorAll('script[data-pixlinks-key]')
    const script = scripts[scripts.length - 1]

    if (script) {
      const apiKey = script.getAttribute('data-pixlinks-key')
      const pixelId = script.getAttribute('data-pixlinks-pixel-id')
      const endpoint = script.getAttribute('data-endpoint')
      const debug = script.getAttribute('data-debug') === 'true'

      if (apiKey && pixelId) {
        const tracker = new PixlinksTracker({
          apiKey,
          pixelId,
          endpoint: endpoint || undefined,
          debug,
        })

        ;(window as unknown as Record<string, unknown>).pixlinks = tracker
        tracker.trackPageView()
      }
    }
  })
}
