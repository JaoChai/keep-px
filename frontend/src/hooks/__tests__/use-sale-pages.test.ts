import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useSalePages, useCreateSalePage, useUpdateSalePage, useDeleteSalePage } from '../use-sale-pages'

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

const mockSalePage = {
  id: 'sp-1',
  customer_id: 'cust-1',
  pixel_ids: ['px-1'],
  name: 'Test Page',
  slug: 'test-page',
  template_name: 'simple',
  content: {
    hero: { title: 'Test', subtitle: '', image_url: '' },
    body: { description: '', features: [], images: [] },
    cta: { button_text: '', button_link: '' },
    contact: { line_id: '', phone: '' },
  },
  is_published: false,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('useSalePages', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should fetch sale pages list', async () => {
    const api = (await import('@/lib/api')).default
    const mockPages = [mockSalePage, { ...mockSalePage, id: 'sp-2', name: 'Page 2', slug: 'page-2' }]
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { data: mockPages, total: 2, page: 1, per_page: 100, total_pages: 1 },
    })

    const { result } = renderHook(() => useSalePages(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(mockPages)
    expect(api.get).toHaveBeenCalledWith('/sale-pages?per_page=100')
  })
})

describe('useCreateSalePage', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should create a sale page', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockResolvedValueOnce({ data: { data: mockSalePage } })

    const { result } = renderHook(() => useCreateSalePage(), { wrapper: createWrapper() })

    result.current.mutate({
      name: 'Test Page',
      template_name: 'simple',
      content: mockSalePage.content,
      is_published: false,
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.post).toHaveBeenCalledWith('/sale-pages', {
      name: 'Test Page',
      template_name: 'simple',
      content: mockSalePage.content,
      is_published: false,
    })
  })

  it('should handle create error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Server error'))

    const { result } = renderHook(() => useCreateSalePage(), { wrapper: createWrapper() })

    result.current.mutate({
      name: 'Test Page',
      template_name: 'simple',
      content: mockSalePage.content,
      is_published: false,
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

describe('useUpdateSalePage', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should update a sale page', async () => {
    const api = (await import('@/lib/api')).default
    const updated = { ...mockSalePage, name: 'Updated' }
    vi.mocked(api.put).mockResolvedValueOnce({ data: { data: updated } })

    const { result } = renderHook(() => useUpdateSalePage(), { wrapper: createWrapper() })

    result.current.mutate({ id: 'sp-1', name: 'Updated' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.put).toHaveBeenCalledWith('/sale-pages/sp-1', { name: 'Updated' })
  })
})

describe('useDeleteSalePage', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should delete a sale page', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.delete).mockResolvedValueOnce({})

    const { result } = renderHook(() => useDeleteSalePage(), { wrapper: createWrapper() })

    result.current.mutate('sp-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(api.delete).toHaveBeenCalledWith('/sale-pages/sp-1')
  })

  it('should handle delete error', async () => {
    const api = (await import('@/lib/api')).default
    vi.mocked(api.delete).mockRejectedValueOnce(new Error('Not found'))

    const { result } = renderHook(() => useDeleteSalePage(), { wrapper: createWrapper() })

    result.current.mutate('sp-1')

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})
