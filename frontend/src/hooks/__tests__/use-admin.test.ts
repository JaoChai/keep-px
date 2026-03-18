import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import {
  useAdminCustomers,
  useAdminCustomerDetail,
  useAdminAnalytics,
  useAdminRevenueChart,
  useAdminGrowthChart,
  useAdminPurchases,
  useAdminSubscriptions,
  useAdminCreditGrants,
  useAdminUpdateCustomerPlan,
  useAdminSuspendCustomer,
  useAdminActivateCustomer,
  useAdminGrantCredits,
  useAdminSalePages,
  useAdminSalePageDetail,
  useAdminToggleSalePage,
  useAdminDeleteSalePage,
  useAdminPixels,
  useAdminPixelDetail,
  useAdminTogglePixel,
  useAdminReplays,
  useAdminReplayDetail,
  useAdminCancelReplay,
  useAdminEvents,
  useAdminEventStats,
  useAdminAuditLog,
} from '../use-admin'

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

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

// --- Query tests ---

describe('useAdminCustomers', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch customers with filters', async () => {
    const api = (await import('@/lib/api')).default
    const mockResponse = {
      data: [{ id: 'c1', email: 'user@test.com', name: 'Test User', plan: 'pro', is_admin: false, created_at: '', updated_at: '' }],
      total: 1,
      page: 1,
      per_page: 20,
      total_pages: 1,
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: mockResponse })

    const { result } = renderHook(
      () => useAdminCustomers('test', 'pro', 'active', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockResponse)
    expect(api.get).toHaveBeenCalledWith(
      expect.stringContaining('/admin/customers?'),
    )
    // Verify query params
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('search=test')
    expect(calledUrl).toContain('plan=pro')
    expect(calledUrl).toContain('status=active')
    expect(calledUrl).toContain('page=1')
    expect(calledUrl).toContain('per_page=20')
  })

  it('should omit empty filter params', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminCustomers('', '', '', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).not.toContain('search=')
    expect(calledUrl).not.toContain('plan=')
    expect(calledUrl).not.toContain('status=')
  })
})

describe('useAdminCustomerDetail', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch customer detail', async () => {
    const api = (await import('@/lib/api')).default
    const mockDetail = {
      customer: { id: 'c1', email: 'user@test.com', name: 'Test', plan: 'pro', is_admin: false, api_key: 'key123', created_at: '', updated_at: '' },
      pixel_count: 3,
      event_count: 1000,
      sale_page_count: 2,
      replay_count: 5,
      purchases: [],
      credits: [],
      subscriptions: [],
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockDetail } })

    const { result } = renderHook(() => useAdminCustomerDetail('c1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockDetail)
    expect(api.get).toHaveBeenCalledWith('/admin/customers/c1')
  })

  it('should not fetch when id is null', async () => {
    const api = (await import('@/lib/api')).default

    const { result } = renderHook(() => useAdminCustomerDetail(null), { wrapper: createWrapper() })

    expect(result.current.fetchStatus).toBe('idle')
    expect(api.get).not.toHaveBeenCalled()
  })
})

describe('useAdminAnalytics', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch admin analytics overview', async () => {
    const api = (await import('@/lib/api')).default
    const mockAnalytics = {
      total_customers: 50,
      active_customers: 45,
      suspended_customers: 5,
      total_pixels: 120,
      events_today: 3500,
      events_this_month: 85000,
      total_replays: 200,
      successful_replays: 180,
      failed_replays: 20,
      total_revenue_thb: 150000,
      revenue_this_month_thb: 25000,
      customers_by_plan: { free: 20, pro: 25, enterprise: 5 },
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockAnalytics } })

    const { result } = renderHook(() => useAdminAnalytics(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockAnalytics)
    expect(api.get).toHaveBeenCalledWith('/admin/analytics/overview')
  })
})

describe('useAdminRevenueChart', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch revenue chart data', async () => {
    const api = (await import('@/lib/api')).default
    const mockData = [
      { date: '2026-03-01', amount_satang: 29900, purchase_count: 1 },
      { date: '2026-03-02', amount_satang: 59800, purchase_count: 2 },
    ]
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockData } })

    const { result } = renderHook(() => useAdminRevenueChart(30), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockData)
    expect(api.get).toHaveBeenCalledWith('/admin/analytics/revenue', { params: { days: 30 } })
  })
})

describe('useAdminGrowthChart', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch growth chart data', async () => {
    const api = (await import('@/lib/api')).default
    const mockData = [
      { date: '2026-03-01', new_customers: 3, total_customers: 48 },
      { date: '2026-03-02', new_customers: 2, total_customers: 50 },
    ]
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockData } })

    const { result } = renderHook(() => useAdminGrowthChart(7), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockData)
    expect(api.get).toHaveBeenCalledWith('/admin/analytics/growth', { params: { days: 7 } })
  })
})

describe('useAdminPurchases', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch purchases with status filter', async () => {
    const api = (await import('@/lib/api')).default
    const mockResponse = {
      data: [{ id: 'p1', customer_id: 'c1', customer_email: 'a@b.com', customer_name: 'A', pack_type: 'starter', amount_satang: 29900, currency: 'THB', status: 'completed', created_at: '' }],
      total: 1,
      page: 1,
      per_page: 20,
      total_pages: 1,
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: mockResponse })

    const { result } = renderHook(
      () => useAdminPurchases('completed', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockResponse)
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('status=completed')
  })
})

describe('useAdminSubscriptions', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch subscriptions', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminSubscriptions('active', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('/admin/subscriptions')
    expect(calledUrl).toContain('status=active')
  })
})

describe('useAdminCreditGrants', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch credit grants', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminCreditGrants(1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('/admin/credit-grants')
    expect(calledUrl).toContain('page=1')
    expect(calledUrl).toContain('per_page=20')
  })
})

// --- Mutation tests ---

describe('useAdminUpdateCustomerPlan', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should update customer plan', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.put).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminUpdateCustomerPlan(), { wrapper: createWrapper() })

    result.current.mutate({ customerId: 'c1', plan: 'enterprise' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.put).toHaveBeenCalledWith('/admin/customers/c1/plan', { plan: 'enterprise' })
    expect(toast.success).toHaveBeenCalled()
  })

  it('should show error toast on failure', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.put).mockRejectedValueOnce(new Error('Forbidden'))

    const { result } = renderHook(() => useAdminUpdateCustomerPlan(), { wrapper: createWrapper() })

    result.current.mutate({ customerId: 'c1', plan: 'enterprise' })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})

describe('useAdminSuspendCustomer', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should suspend a customer', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminSuspendCustomer(), { wrapper: createWrapper() })

    result.current.mutate('c1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/customers/c1/suspend')
    expect(toast.success).toHaveBeenCalled()
  })

  it('should show error toast on failure', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Error'))

    const { result } = renderHook(() => useAdminSuspendCustomer(), { wrapper: createWrapper() })

    result.current.mutate('c1')

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})

describe('useAdminActivateCustomer', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should activate a customer', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminActivateCustomer(), { wrapper: createWrapper() })

    result.current.mutate('c1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/customers/c1/activate')
    expect(toast.success).toHaveBeenCalled()
  })
})

describe('useAdminGrantCredits', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should grant credits to a customer', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminGrantCredits(), { wrapper: createWrapper() })

    result.current.mutate({
      customerId: 'c1',
      pack_type: 'starter',
      total_replays: 10,
      max_events_per_replay: 5000,
      expires_in_days: 30,
      reason: 'Promotional grant',
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/customers/c1/credits', {
      pack_type: 'starter',
      total_replays: 10,
      max_events_per_replay: 5000,
      expires_in_days: 30,
      reason: 'Promotional grant',
    })
    expect(toast.success).toHaveBeenCalled()
  })

  it('should show error toast on failure', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Error'))

    const { result } = renderHook(() => useAdminGrantCredits(), { wrapper: createWrapper() })

    result.current.mutate({
      customerId: 'c1',
      pack_type: 'starter',
      total_replays: 10,
      max_events_per_replay: 5000,
      expires_in_days: 30,
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})

// --- F1: Sale Pages ---

describe('useAdminSalePages', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch admin sale pages with filters', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminSalePages('test', 'c1', 'true', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('search=test')
    expect(calledUrl).toContain('customer_id=c1')
    expect(calledUrl).toContain('published=true')
  })
})

describe('useAdminSalePageDetail', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch sale page detail', async () => {
    const api = (await import('@/lib/api')).default
    const mockDetail = {
      sale_page: { id: 'sp-1', customer_id: 'c1', pixel_ids: [], name: 'Test', slug: 'test', template_name: 'simple', content: {}, is_published: true, created_at: '', updated_at: '' },
      customer_email: 'a@b.com',
      customer_name: 'A',
      linked_pixels: [],
      event_count: 100,
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockDetail } })

    const { result } = renderHook(() => useAdminSalePageDetail('sp-1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockDetail)
  })

  it('should not fetch when id is null', async () => {
    const api = (await import('@/lib/api')).default
    renderHook(() => useAdminSalePageDetail(null), { wrapper: createWrapper() })
    expect(api.get).not.toHaveBeenCalled()
  })
})

describe('useAdminToggleSalePage', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should enable a sale page', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminToggleSalePage(), { wrapper: createWrapper() })

    result.current.mutate({ id: 'sp-1', enable: true })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/sale-pages/sp-1/enable')
    expect(toast.success).toHaveBeenCalled()
  })

  it('should disable a sale page', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminToggleSalePage(), { wrapper: createWrapper() })

    result.current.mutate({ id: 'sp-1', enable: false })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/sale-pages/sp-1/disable')
  })
})

describe('useAdminDeleteSalePage', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should delete a sale page', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.delete).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminDeleteSalePage(), { wrapper: createWrapper() })

    result.current.mutate('sp-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.delete).toHaveBeenCalledWith('/admin/sale-pages/sp-1')
    expect(toast.success).toHaveBeenCalled()
  })
})

// --- F2: Pixels ---

describe('useAdminPixels', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch admin pixels with filters', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminPixels('pixel1', 'c1', 'true', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('search=pixel1')
    expect(calledUrl).toContain('customer_id=c1')
    expect(calledUrl).toContain('active=true')
  })
})

describe('useAdminPixelDetail', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch pixel detail', async () => {
    const api = (await import('@/lib/api')).default
    const mockDetail = {
      pixel: { id: 'px-1', customer_id: 'c1', fb_pixel_id: '111', name: 'Pixel 1', is_active: true, created_at: '', updated_at: '' },
      customer_email: 'a@b.com',
      customer_name: 'A',
      event_count: 500,
      linked_sale_pages: [],
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockDetail } })

    const { result } = renderHook(() => useAdminPixelDetail('px-1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockDetail)
  })

  it('should not fetch when id is null', async () => {
    const api = (await import('@/lib/api')).default
    renderHook(() => useAdminPixelDetail(null), { wrapper: createWrapper() })
    expect(api.get).not.toHaveBeenCalled()
  })
})

describe('useAdminTogglePixel', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should enable a pixel', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminTogglePixel(), { wrapper: createWrapper() })

    result.current.mutate({ id: 'px-1', enable: true })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/pixels/px-1/enable')
    expect(toast.success).toHaveBeenCalled()
  })

  it('should disable a pixel', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminTogglePixel(), { wrapper: createWrapper() })

    result.current.mutate({ id: 'px-1', enable: false })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/pixels/px-1/disable')
  })
})

// --- F3: Replays ---

describe('useAdminReplays', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch admin replays with filters', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminReplays('running', 'c1', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('status=running')
    expect(calledUrl).toContain('customer_id=c1')
  })
})

describe('useAdminReplayDetail', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch replay detail', async () => {
    const api = (await import('@/lib/api')).default
    const mockDetail = {
      session: { id: 'rs-1', customer_id: 'c1', source_pixel_id: 'px-1', target_pixel_id: 'px-2', status: 'completed', total_events: 100, replayed_events: 98, failed_events: 2, created_at: '', customer_email: 'a@b.com', customer_name: 'A', source_pixel_name: 'Src', target_pixel_name: 'Tgt' },
      customer_email: 'a@b.com',
      customer_name: 'A',
      source_pixel: { id: 'px-1', name: 'Src', fb_pixel_id: '111', is_active: true },
      target_pixel: { id: 'px-2', name: 'Tgt', fb_pixel_id: '222', is_active: true },
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockDetail } })

    const { result } = renderHook(() => useAdminReplayDetail('rs-1'), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockDetail)
  })

  it('should not fetch when id is null', async () => {
    const api = (await import('@/lib/api')).default
    renderHook(() => useAdminReplayDetail(null), { wrapper: createWrapper() })
    expect(api.get).not.toHaveBeenCalled()
  })
})

describe('useAdminCancelReplay', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should cancel a replay', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockResolvedValueOnce({ data: {} })

    const { result } = renderHook(() => useAdminCancelReplay(), { wrapper: createWrapper() })

    result.current.mutate('rs-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/admin/replays/rs-1/cancel')
    expect(toast.success).toHaveBeenCalled()
  })

  it('should show error toast on failure', async () => {
    const api = (await import('@/lib/api')).default
    const { toast } = await import('sonner')
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Cannot cancel'))

    const { result } = renderHook(() => useAdminCancelReplay(), { wrapper: createWrapper() })

    result.current.mutate('rs-1')

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalled()
  })
})

// --- F4: Events ---

describe('useAdminEvents', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch admin events with filters', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 50, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminEvents('c1', 'px-1', 'PageView', 1, 50),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('customer_id=c1')
    expect(calledUrl).toContain('pixel_id=px-1')
    expect(calledUrl).toContain('event_name=PageView')
  })
})

describe('useAdminEventStats', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch event stats', async () => {
    const api = (await import('@/lib/api')).default
    const mockStats = {
      total_today: 500,
      total_this_hour: 30,
      capi_success_rate: 95.5,
      capi_failure_count: 12,
      top_event_types: [{ event_name: 'PageView', count: 300 }],
      timeseries: [],
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockStats } })

    const { result } = renderHook(() => useAdminEventStats(24), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockStats)
    expect(api.get).toHaveBeenCalledWith('/admin/events/stats', { params: { hours: 24 } })
  })
})

// --- F5: Audit Log ---

describe('useAdminAuditLog', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch audit log with filters', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminAuditLog('admin-1', 'suspend', 'c1', '2026-03-01', '2026-03-31', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).toContain('admin_id=admin-1')
    expect(calledUrl).toContain('action=suspend')
    expect(calledUrl).toContain('target_customer_id=c1')
    expect(calledUrl).toContain('from=2026-03-01')
    expect(calledUrl).toContain('to=2026-03-31')
  })

  it('should omit empty filter params', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: [], total: 0, page: 1, per_page: 20, total_pages: 0 },
    })

    const { result } = renderHook(
      () => useAdminAuditLog('', '', '', '', '', 1, 20),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const calledUrl = vi.mocked(api.get).mock.calls[0]![0] as string
    expect(calledUrl).not.toContain('admin_id=')
    expect(calledUrl).not.toContain('action=')
    expect(calledUrl).not.toContain('target_customer_id=')
  })
})
