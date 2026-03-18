import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useRealtimeEvents } from '../use-realtime-events'

// Mock api module
vi.mock('@/lib/api', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
  },
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

function makeRealtimeEvent(overrides: Record<string, unknown> = {}) {
  return {
    id: crypto.randomUUID(),
    pixel_id: 'px-1',
    pixel_name: 'Test Pixel',
    event_name: 'PageView',
    source_url: 'https://example.com',
    forwarded_to_capi: true,
    event_time: new Date().toISOString(),
    created_at: new Date().toISOString(),
    ...overrides,
  }
}

describe('useRealtimeEvents', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should load initial events', async () => {
    const api = (await import('@/lib/api')).default
    const events = [
      makeRealtimeEvent({ id: 'e1', created_at: '2026-03-01T10:00:00Z' }),
      makeRealtimeEvent({ id: 'e2', created_at: '2026-03-01T10:00:01Z' }),
    ]
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: events } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    // Initial events are reversed for newest-first display
    expect(result.current.events).toHaveLength(2)
    expect(result.current.events[0]!.id).toBe('e2')
    expect(result.current.events[1]!.id).toBe('e1')
  })

  it('should return empty events when API returns null', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: null } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.events).toEqual([])
  })

  it('should start in live mode (not paused)', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: [] } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.isPaused).toBe(false)
  })

  it('should toggle pause state', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValue({ data: { data: [] } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => { result.current.togglePause() })
    expect(result.current.isPaused).toBe(true)

    act(() => { result.current.togglePause() })
    expect(result.current.isPaused).toBe(false)
  })

  it('should clear events on clear()', async () => {
    const api = (await import('@/lib/api')).default
    const events = [
      makeRealtimeEvent({ id: 'e1' }),
      makeRealtimeEvent({ id: 'e2' }),
    ]
    // First call: initial load. After clear: another initial load (empty).
    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: { data: events } })
      .mockResolvedValueOnce({ data: { data: [] } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.events).toHaveLength(2))

    act(() => { result.current.clear() })

    await waitFor(() => expect(result.current.events).toHaveLength(0))
  })

  it('should reset events on setPixelId()', async () => {
    const api = (await import('@/lib/api')).default
    const events = [makeRealtimeEvent({ id: 'e1', pixel_id: 'px-1' })]
    const filteredEvents = [makeRealtimeEvent({ id: 'e3', pixel_id: 'px-2' })]

    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: { data: events } })
      .mockResolvedValueOnce({ data: { data: filteredEvents } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.pixelId).toBe(null)

    act(() => { result.current.setPixelId('px-2') })

    expect(result.current.pixelId).toBe('px-2')
    // After setPixelId, it resets polled events and refetches with new pixel_id
    await waitFor(() => {
      const getCall = vi.mocked(api.get).mock.calls.find(
        (call) => (call[0] as string).includes('pixel_id=px-2'),
      )
      return expect(getCall).toBeTruthy()
    })
  })

  it('should pass pixel_id filter to initial load query', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: { data: [] } })
      .mockResolvedValueOnce({ data: { data: [] } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => { result.current.setPixelId('px-5') })

    await waitFor(() => {
      const calls = vi.mocked(api.get).mock.calls
      const hasPixelFilter = calls.some(
        (call) => (call[0] as string).includes('pixel_id=px-5'),
      )
      return expect(hasPixelFilter).toBe(true)
    })
  })

  it('should have pixelId=null initially', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: [] } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    expect(result.current.pixelId).toBe(null)
  })

  it('should call refresh and reload events', async () => {
    const api = (await import('@/lib/api')).default
    const events = [makeRealtimeEvent({ id: 'e1' })]
    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: { data: events } })
      .mockResolvedValueOnce({ data: { data: events } })

    const { result } = renderHook(() => useRealtimeEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.events).toHaveLength(1))

    act(() => { result.current.refresh() })

    // refresh bumps generation, which triggers a new initial load
    await waitFor(() => expect(vi.mocked(api.get).mock.calls.length).toBeGreaterThanOrEqual(2))
  })
})
