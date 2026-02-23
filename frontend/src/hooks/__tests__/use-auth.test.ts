import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useLogin, useRegister } from '../use-auth'
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
const mockSetCustomer = vi.fn()
vi.mock('@/stores/auth-store', () => ({
  useAuthStore: (selector: (state: Record<string, unknown>) => unknown) => {
    const store = { setCustomer: mockSetCustomer, customer: null, isAuthenticated: false }
    return selector(store)
  },
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useLogin', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should login successfully', async () => {
    const mockResponse = {
      data: {
        data: {
          access_token: 'test-access',
          refresh_token: 'test-refresh',
          customer: { id: '1', email: 'test@test.com', name: 'Test' },
        },
      },
    }
    vi.mocked(api.post).mockResolvedValueOnce(mockResponse)

    const { result } = renderHook(() => useLogin(), { wrapper: createWrapper() })

    await act(async () => {
      result.current.mutate({ email: 'test@test.com', password: 'password123' })
    })

    await waitFor(() => {
      expect(result.current.isSuccess || result.current.isError).toBe(true)
    })

    if (result.current.isError) {
      console.error('Mutation error:', result.current.error)
    }

    expect(result.current.isSuccess).toBe(true)
    expect(api.post).toHaveBeenCalledWith('/auth/login', { email: 'test@test.com', password: 'password123' })
  })

  it('should handle login failure', async () => {
    vi.mocked(api.post).mockRejectedValueOnce(new Error('Invalid credentials'))

    const { result } = renderHook(() => useLogin(), { wrapper: createWrapper() })

    await act(async () => {
      result.current.mutate({ email: 'test@test.com', password: 'wrong' })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe('useRegister', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should register successfully', async () => {
    const mockResponse = {
      data: {
        data: {
          access_token: 'test-access',
          refresh_token: 'test-refresh',
          customer: { id: '1', email: 'new@test.com', name: 'New User' },
        },
      },
    }
    vi.mocked(api.post).mockResolvedValueOnce(mockResponse)

    const { result } = renderHook(() => useRegister(), { wrapper: createWrapper() })

    await act(async () => {
      result.current.mutate({ email: 'new@test.com', password: 'password123', name: 'New User' })
    })

    await waitFor(() => {
      expect(result.current.isSuccess || result.current.isError).toBe(true)
    })

    if (result.current.isError) {
      console.error('Mutation error:', result.current.error)
    }

    expect(result.current.isSuccess).toBe(true)
    expect(api.post).toHaveBeenCalledWith('/auth/register', { email: 'new@test.com', password: 'password123', name: 'New User' })
  })
})
