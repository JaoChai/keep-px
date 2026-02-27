import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { PixlinksTracker } from '../tracker'

// Mock fetch
const mockFetch = vi.fn()
global.fetch = mockFetch

// Mock window.location
Object.defineProperty(window, 'location', {
  value: { href: 'https://example.com/page' },
  writable: true,
})

describe('PixlinksTracker', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    mockFetch.mockResolvedValue({ ok: true })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('throws if apiKey is missing', () => {
    expect(() => new PixlinksTracker({ apiKey: '', pixelId: 'px123' } as any)).toThrow('apiKey is required')
  })

  it('throws if pixelId is missing', () => {
    expect(() => new PixlinksTracker({ apiKey: 'key123', pixelId: '' } as any)).toThrow('pixelId is required')
  })

  it('queues event with pixel_id', () => {
    const tracker = new PixlinksTracker({ apiKey: 'key123', pixelId: 'px123' })
    tracker.track('Purchase', { value: 100 })

    // Trigger flush
    tracker.flush()

    expect(mockFetch).toHaveBeenCalledOnce()
    const [, options] = mockFetch.mock.calls[0]
    const body = JSON.parse(options.body)
    expect(body.events[0].pixel_id).toBe('px123')
    expect(body.events[0].event_name).toBe('Purchase')

    tracker.destroy()
  })

  it('sends correct headers', async () => {
    const tracker = new PixlinksTracker({ apiKey: 'key123', pixelId: 'px123' })
    tracker.track('ViewContent')
    await tracker.flush()

    const [, options] = mockFetch.mock.calls[0]
    expect(options.headers['X-API-Key']).toBe('key123')
    expect(options.headers['Content-Type']).toBe('application/json')

    tracker.destroy()
  })

  it('flushes at MAX_BATCH_SIZE', () => {
    const tracker = new PixlinksTracker({ apiKey: 'key123', pixelId: 'px123' })

    for (let i = 0; i < 10; i++) {
      tracker.track(`Event${i}`)
    }

    expect(mockFetch).toHaveBeenCalledOnce()
    tracker.destroy()
  })

  it('retries failed events up to MAX_RETRIES', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'))

    const tracker = new PixlinksTracker({ apiKey: 'key123', pixelId: 'px123' })
    tracker.track('Purchase')
    await tracker.flush()

    // Event should be re-queued -- flush again
    mockFetch.mockResolvedValueOnce({ ok: true })
    await tracker.flush()

    expect(mockFetch).toHaveBeenCalledTimes(2)
    tracker.destroy()
  })

  it('includes event_id in payload', () => {
    const tracker = new PixlinksTracker({ apiKey: 'key123', pixelId: 'px123' })
    tracker.track('Purchase', { value: 100 })
    tracker.flush()

    const [, options] = mockFetch.mock.calls[0]
    const body = JSON.parse(options.body)
    expect(body.events[0].event_id).toBeDefined()
    expect(body.events[0].event_id).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/
    )

    tracker.destroy()
  })

  it('preserves explicit eventId', () => {
    const tracker = new PixlinksTracker({ apiKey: 'key123', pixelId: 'px123' })
    const customId = '11111111-2222-3333-4444-555555555555'
    tracker.track('Purchase', { value: 100 }, undefined, customId)
    tracker.flush()

    const [, options] = mockFetch.mock.calls[0]
    const body = JSON.parse(options.body)
    expect(body.events[0].event_id).toBe(customId)

    tracker.destroy()
  })
})
