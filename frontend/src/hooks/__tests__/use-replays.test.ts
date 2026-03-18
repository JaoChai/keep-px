import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import {
  useReplays,
  useReplaySession,
  useCreateReplay,
  useCancelReplay,
  useRetryReplay,
  useReplayPreview,
  useEventTypes,
} from '../use-replays'

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

const mockSession = {
  id: 'rs-1',
  customer_id: 'c1',
  source_pixel_id: 'px-1',
  target_pixel_id: 'px-2',
  status: 'completed',
  total_events: 100,
  replayed_events: 98,
  failed_events: 2,
  event_types: ['PageView', 'Purchase'],
  date_from: '2026-01-01',
  date_to: '2026-01-31',
  time_mode: 'original',
  batch_delay_ms: 100,
  started_at: '2026-02-01T10:00:00Z',
  completed_at: '2026-02-01T10:05:00Z',
  created_at: '2026-02-01T09:59:00Z',
}

describe('useReplays', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch replay sessions list', async () => {
    const api = (await import('@/lib/api')).default
    const sessions = [mockSession, { ...mockSession, id: 'rs-2', status: 'running' }]
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: sessions } })

    const { result } = renderHook(() => useReplays(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(sessions)
    expect(api.get).toHaveBeenCalledWith('/replays')
  })

  it('should handle fetch error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockRejectedValueOnce(new Error('Server error'))

    const { result } = renderHook(() => useReplays(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

describe('useReplaySession', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch a single replay session by id', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockSession } })

    const { result } = renderHook(() => useReplaySession('rs-1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockSession)
    expect(api.get).toHaveBeenCalledWith('/replays/rs-1')
  })

  it('should not fetch when id is null', async () => {
    const api = (await import('@/lib/api')).default

    const { result } = renderHook(() => useReplaySession(null), { wrapper: createWrapper() })

    expect(result.current.fetchStatus).toBe('idle')
    expect(api.get).not.toHaveBeenCalled()
  })
})

describe('useCreateReplay', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should create a replay session', async () => {
    const api = (await import('@/lib/api')).default
    const newSession = { ...mockSession, id: 'rs-3', status: 'pending' }
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: newSession, message: undefined } })

    const { result } = renderHook(() => useCreateReplay(), { wrapper: createWrapper() })

    const input = {
      source_pixel_id: 'px-1',
      target_pixel_id: 'px-2',
      event_types: ['PageView'],
      date_from: '2026-01-01',
      date_to: '2026-01-31',
    }
    result.current.mutate(input)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual({ session: newSession, warning: undefined })
    expect(api.post).toHaveBeenCalledWith('/replays', input)
  })

  it('should return warning message from API', async () => {
    const api = (await import('@/lib/api')).default
    const newSession = { ...mockSession, id: 'rs-4', status: 'pending' }
    vi.mocked(api.post).mockResolvedValueOnce({
      data: { data: newSession, message: 'Credit is running low' },
    })

    const { result } = renderHook(() => useCreateReplay(), { wrapper: createWrapper() })

    result.current.mutate({ source_pixel_id: 'px-1', target_pixel_id: 'px-2' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.warning).toBe('Credit is running low')
  })

  it('should handle create error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Insufficient credits'))

    const { result } = renderHook(() => useCreateReplay(), { wrapper: createWrapper() })

    result.current.mutate({ source_pixel_id: 'px-1', target_pixel_id: 'px-2' })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

describe('useCancelReplay', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should cancel a replay session', async () => {
    const api = (await import('@/lib/api')).default
    const cancelledSession = { ...mockSession, status: 'cancelled', cancelled_at: '2026-02-01T10:03:00Z' }
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: cancelledSession } })

    const { result } = renderHook(() => useCancelReplay(), { wrapper: createWrapper() })

    result.current.mutate('rs-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(cancelledSession)
    expect(api.post).toHaveBeenCalledWith('/replays/rs-1/cancel')
  })

  it('should handle cancel error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Cannot cancel'))

    const { result } = renderHook(() => useCancelReplay(), { wrapper: createWrapper() })

    result.current.mutate('rs-1')

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe('useRetryReplay', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should retry a failed replay session', async () => {
    const api = (await import('@/lib/api')).default
    const retriedSession = { ...mockSession, id: 'rs-5', status: 'pending' }
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: retriedSession } })

    const { result } = renderHook(() => useRetryReplay(), { wrapper: createWrapper() })

    result.current.mutate('rs-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(retriedSession)
    expect(api.post).toHaveBeenCalledWith('/replays/rs-1/retry')
  })

  it('should handle retry error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Retry failed'))

    const { result } = renderHook(() => useRetryReplay(), { wrapper: createWrapper() })

    result.current.mutate('rs-1')

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe('useReplayPreview', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch replay preview', async () => {
    const api = (await import('@/lib/api')).default
    const mockPreview = {
      total_events: 150,
      sample_events: [
        { id: 'ev-1', pixel_id: 'px-1', event_name: 'PageView', event_data: {}, event_time: '2026-01-15T00:00:00Z', forwarded_to_capi: true, created_at: '2026-01-15T00:00:00Z' },
      ],
      warning: 'Large dataset',
    }
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: mockPreview } })

    const { result } = renderHook(() => useReplayPreview(), { wrapper: createWrapper() })

    const input = {
      source_pixel_id: 'px-1',
      target_pixel_id: 'px-2',
      event_types: ['PageView'],
      date_from: '2026-01-01',
      date_to: '2026-01-31',
    }
    result.current.mutate(input)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockPreview)
    expect(api.post).toHaveBeenCalledWith('/replays/preview', input)
  })
})

describe('useEventTypes', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch event types for a given pixel', async () => {
    const api = (await import('@/lib/api')).default
    const eventTypes = ['PageView', 'Purchase', 'Lead']
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: eventTypes } })

    const { result } = renderHook(() => useEventTypes('px-1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(eventTypes)
    expect(api.get).toHaveBeenCalledWith('/replays/event-types?pixel_id=px-1')
  })

  it('should not fetch when pixelId is undefined', async () => {
    const api = (await import('@/lib/api')).default

    const { result } = renderHook(() => useEventTypes(undefined), { wrapper: createWrapper() })

    expect(result.current.fetchStatus).toBe('idle')
    expect(api.get).not.toHaveBeenCalled()
  })
})
