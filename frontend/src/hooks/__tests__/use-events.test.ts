import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useEvents, useEventDetail, useCustomerEventTypes } from '../use-events'

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
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

const mockEvent = {
  id: 'ev-1',
  pixel_id: 'px-1',
  event_name: 'PageView',
  event_data: { page: '/home' },
  user_data: { em: 'hash123' },
  source_url: 'https://example.com',
  event_time: '2026-03-01T12:00:00Z',
  forwarded_to_capi: true,
  capi_response_code: 200,
  created_at: '2026-03-01T12:00:01Z',
}

describe('useEvents', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch paginated events with default params', async () => {
    const api = (await import('@/lib/api')).default
    const mockResponse = {
      data: [mockEvent],
      total: 1,
      page: 1,
      per_page: 50,
      total_pages: 1,
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: mockResponse })

    const { result } = renderHook(() => useEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockResponse)
    expect(api.get).toHaveBeenCalledWith('/events', {
      params: { page: 1, per_page: 50 },
    })
  })

  it('should pass pagination params', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 3, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(() => useEvents(3, 20), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.get).toHaveBeenCalledWith('/events', {
      params: { page: 3, per_page: 20 },
    })
  })

  it('should pass filter params when provided', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [mockEvent], total: 1, page: 1, per_page: 50, total_pages: 1 },
    })

    const { result } = renderHook(
      () => useEvents(1, 50, 'px-1', 'PageView', '2026-03-01', '2026-03-31'),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.get).toHaveBeenCalledWith('/events', {
      params: {
        page: 1,
        per_page: 50,
        pixel_id: 'px-1',
        event_name: 'PageView',
        from: '2026-03-01',
        to: '2026-03-31',
      },
    })
  })

  it('should not include null/undefined filter params', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 50, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useEvents(1, 50, null, null, null, null),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.get).toHaveBeenCalledWith('/events', {
      params: { page: 1, per_page: 50 },
    })
  })

  it('should handle fetch error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockRejectedValueOnce(new Error('Server error'))

    const { result } = renderHook(() => useEvents(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

describe('useEventDetail', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch event detail when id is provided', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockEvent } })

    const { result } = renderHook(() => useEventDetail('ev-1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockEvent)
    expect(api.get).toHaveBeenCalledWith('/events/ev-1')
  })

  it('should not fetch when id is null', async () => {
    const api = (await import('@/lib/api')).default

    const { result } = renderHook(() => useEventDetail(null), { wrapper: createWrapper() })

    // Query should remain in idle/pending state since it's disabled
    expect(result.current.fetchStatus).toBe('idle')
    expect(api.get).not.toHaveBeenCalled()
  })

  it('should handle detail fetch error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockRejectedValueOnce(new Error('Not found'))

    const { result } = renderHook(() => useEventDetail('ev-999'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe('useCustomerEventTypes', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch distinct event types', async () => {
    const api = (await import('@/lib/api')).default
    const eventTypes = ['PageView', 'Purchase', 'Lead', 'AddToCart']
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: eventTypes } })

    const { result } = renderHook(() => useCustomerEventTypes(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(eventTypes)
    expect(api.get).toHaveBeenCalledWith('/events/event-types')
  })

  it('should return empty array when data is null', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: null } })

    const { result } = renderHook(() => useCustomerEventTypes(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual([])
  })
})
