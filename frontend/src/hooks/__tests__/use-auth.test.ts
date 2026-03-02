import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useGoogleAuth } from '../use-auth'
import api from '@/lib/api'

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

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value }),
    removeItem: vi.fn((key: string) => { delete store[key] }),
    clear: vi.fn(() => { store = {} }),
    get length() { return Object.keys(store).length },
    key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
  }
})()
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock })

// Mock auth store
const { mockSetCustomer } = vi.hoisted(() => ({ mockSetCustomer: vi.fn() }))
vi.mock('@/stores/auth-store', () => {
  const store = { setCustomer: mockSetCustomer, customer: null, isAuthenticated: false }
  return {
    useAuthStore: Object.assign(
      (selector: (state: Record<string, unknown>) => unknown) => selector(store),
      { getState: () => store },
    ),
  }
})

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useGoogleAuth', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should authenticate with Google ID token', async () => {
    const mockResponse = {
      data: {
        data: {
          access_token: 'test-access',
          refresh_token: 'test-refresh',
          customer: { id: '1', email: 'google@test.com', name: 'Google User' },
        },
      },
    }
    vi.mocked(api.post).mockResolvedValueOnce(mockResponse)

    const { result } = renderHook(() => useGoogleAuth(), { wrapper: createWrapper() })

    await act(async () => {
      result.current.mutate('google-id-token-123')
    })

    await waitFor(() => {
      expect(result.current.isSuccess || result.current.isError).toBe(true)
    })

    expect(result.current.isSuccess).toBe(true)
    expect(api.post).toHaveBeenCalledWith('/auth/google', { id_token: 'google-id-token-123' })
    expect(localStorageMock.setItem).toHaveBeenCalledWith('access_token', 'test-access')
    expect(localStorageMock.setItem).toHaveBeenCalledWith('refresh_token', 'test-refresh')
    expect(mockSetCustomer).toHaveBeenCalledWith({ id: '1', email: 'google@test.com', name: 'Google User' })
  })

  it('should handle Google auth failure', async () => {
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Google auth failed'))

    const { result } = renderHook(() => useGoogleAuth(), { wrapper: createWrapper() })

    await act(async () => {
      result.current.mutate('invalid-token')
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
