import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import {
  useBillingOverview,
  useQuota,
  useCreateCheckout,
  useUpdateSlots,
  useCreatePortalSession,
} from '../use-billing'

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

// Mock sonner to avoid toast side effects
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

// Stub window.location.href setter to prevent jsdom navigation errors
const originalLocation = window.location
beforeEach(() => {
  Object.defineProperty(window, 'location', {
    writable: true,
    value: { ...originalLocation },
  })
})
afterEach(() => {
  Object.defineProperty(window, 'location', {
    writable: true,
    value: originalLocation,
  })
})

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useBillingOverview', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch billing overview', async () => {
    const api = (await import('@/lib/api')).default
    const mockOverview = {
      plan: 'pro',
      purchases: [{ id: 'p1', customer_id: 'c1', pack_type: 'starter', amount_satang: 29900, currency: 'THB', status: 'completed', created_at: '2026-01-01T00:00:00Z' }],
      credits: [{ id: 'cr1', customer_id: 'c1', pack_type: 'starter', total_replays: 10, used_replays: 2, max_events_per_replay: 5000, expires_at: '2026-12-31T00:00:00Z', created_at: '2026-01-01T00:00:00Z' }],
      subscriptions: [],
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockOverview } })

    const { result } = renderHook(() => useBillingOverview(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockOverview)
    expect(api.get).toHaveBeenCalledWith('/billing')
  })

  it('should handle fetch error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockRejectedValueOnce(new Error('Network error'))

    const { result } = renderHook(() => useBillingOverview(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

describe('useQuota', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch customer quota', async () => {
    const api = (await import('@/lib/api')).default
    const mockQuota = {
      pixel_slots: 5,
      plan: 'pro',
      max_pixels: 10,
      max_events_per_month: 50000,
      events_used_this_month: 1200,
      retention_days: 90,
      max_sale_pages: 20,
      can_replay: true,
      remaining_replays: 8,
      max_events_per_replay: 5000,
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockQuota } })

    const { result } = renderHook(() => useQuota(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockQuota)
    expect(api.get).toHaveBeenCalledWith('/billing/quota')
  })
})

describe('useCreateCheckout', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should create a checkout session', async () => {
    const api = (await import('@/lib/api')).default
    const mockUrl = 'https://checkout.stripe.com/pay/cs_test_123'
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: { url: mockUrl } } })

    const { result } = renderHook(() => useCreateCheckout(), { wrapper: createWrapper() })

    result.current.mutate({ type: 'starter_pack' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/billing/checkout', { type: 'starter_pack' })
  })

  it('should create checkout with quantity param', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: { url: 'https://checkout.stripe.com/pay/cs_test_456' } } })

    const { result } = renderHook(() => useCreateCheckout(), { wrapper: createWrapper() })

    result.current.mutate({ type: 'pixel_slots', quantity: 3 })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/billing/checkout', { type: 'pixel_slots', quantity: 3 })
  })

  it('should handle checkout error and show toast', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Payment error'))

    const { result } = renderHook(() => useCreateCheckout(), { wrapper: createWrapper() })

    result.current.mutate({ type: 'starter_pack' })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})

describe('useUpdateSlots', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should update pixel slots', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.put).mockResolvedValueOnce({ data: { data: { url: 'https://checkout.stripe.com/pay/cs_test_789' } } })

    const { result } = renderHook(() => useUpdateSlots(), { wrapper: createWrapper() })

    result.current.mutate(5)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.put).toHaveBeenCalledWith('/billing/slots', { quantity: 5 })
  })

  it('should handle update slots error and show toast', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.put).mockRejectedValueOnce(new Error('Update error'))

    const { result } = renderHook(() => useUpdateSlots(), { wrapper: createWrapper() })

    result.current.mutate(3)

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})

describe('useCreatePortalSession', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should create a portal session', async () => {
    const api = (await import('@/lib/api')).default
    const mockUrl = 'https://billing.stripe.com/p/session_abc'
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: { url: mockUrl } } })

    const { result } = renderHook(() => useCreatePortalSession(), { wrapper: createWrapper() })

    result.current.mutate()

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/billing/portal')
  })

  it('should handle portal session error and show toast', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Portal error'))

    const { result } = renderHook(() => useCreatePortalSession(), { wrapper: createWrapper() })

    result.current.mutate()

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})
