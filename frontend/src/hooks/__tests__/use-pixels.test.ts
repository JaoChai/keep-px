import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { usePixels, useCreatePixel } from '../use-pixels'

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

describe('usePixels', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch pixel list', async () => {
    const api = (await import('@/lib/api')).default
    const mockPixels = [
      { id: '1', name: 'Pixel 1', fb_pixel_id: '111', is_active: true, status: 'active', customer_id: 'c1', created_at: '', updated_at: '' },
      { id: '2', name: 'Pixel 2', fb_pixel_id: '222', is_active: false, status: 'inactive', customer_id: 'c1', created_at: '', updated_at: '' },
    ]
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: mockPixels } })

    const { result } = renderHook(() => usePixels(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockPixels)
    expect(api.get).toHaveBeenCalledWith('/pixels')
  })
})

describe('useCreatePixel', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should create a new pixel', async () => {
    const api = (await import('@/lib/api')).default
    const newPixel = { id: '3', name: 'New Pixel', fb_pixel_id: '333', is_active: true, status: 'active', customer_id: 'c1', created_at: '', updated_at: '' }
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: newPixel } })

    const { result } = renderHook(() => useCreatePixel(), { wrapper: createWrapper() })

    result.current.mutate({ fb_pixel_id: '333', fb_access_token: 'token', name: 'New Pixel' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/pixels', { fb_pixel_id: '333', fb_access_token: 'token', name: 'New Pixel' })
  })
})
