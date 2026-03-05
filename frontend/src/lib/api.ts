import axios from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

const API_BASE = import.meta.env.VITE_API_URL || ''

const api = axios.create({
  baseURL: `${API_BASE}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Mutex to prevent concurrent token refresh race condition
let isRefreshing = false
let refreshSubscribers: {
  resolve: (token: string) => void
  reject: (error: Error) => void
}[] = []

function onRefreshed(token: string) {
  refreshSubscribers.forEach(({ resolve }) => resolve(token))
  refreshSubscribers = []
}

function onRefreshFailed(error: Error) {
  refreshSubscribers.forEach(({ reject }) => reject(error))
  refreshSubscribers = []
}

async function doRefresh(): Promise<string> {
  // Read from localStorage fresh — another tab may have rotated the token
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) {
    throw new Error('no refresh token')
  }
  const { data } = await axios.post(`${API_BASE}/api/v1/auth/refresh`, {
    refresh_token: refreshToken,
  })
  const newAccessToken = data.data.access_token
  localStorage.setItem('access_token', newAccessToken)
  localStorage.setItem('refresh_token', data.data.refresh_token)

  if (data.data.customer) {
    useAuthStore.getState().setCustomer(data.data.customer)
  }

  scheduleProactiveRefresh(newAccessToken)
  return newAccessToken
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    const authPaths = ['/auth/login', '/auth/register', '/auth/refresh', '/auth/google']
    const isAuthRequest = authPaths.some((path) => originalRequest.url?.includes(path))

    if (error.response?.status === 401 && !originalRequest._retry && !isAuthRequest) {
      originalRequest._retry = true

      if (isRefreshing) {
        // Another refresh is in progress — wait for it
        return new Promise((resolve, reject) => {
          refreshSubscribers.push({
            resolve: (token: string) => {
              originalRequest.headers.Authorization = `Bearer ${token}`
              resolve(api(originalRequest))
            },
            reject: (err: Error) => {
              reject(err)
            },
          })
        })
      }

      isRefreshing = true
      try {
        const newAccessToken = await doRefresh()
        onRefreshed(newAccessToken)
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
        return api(originalRequest)
      } catch (firstErr) {
        // Retry once — another tab may have rotated the token between our read and request
        const freshRefreshToken = localStorage.getItem('refresh_token')
        const usedRefreshToken = localStorage.getItem('_last_refresh_attempt')
        if (freshRefreshToken && freshRefreshToken !== usedRefreshToken) {
          try {
            localStorage.setItem('_last_refresh_attempt', freshRefreshToken)
            const newAccessToken = await doRefresh()
            onRefreshed(newAccessToken)
            originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
            return api(originalRequest)
          } catch {
            // Fall through to logout
          }
        }

        onRefreshFailed(
          firstErr instanceof Error ? firstErr : new Error('Token refresh failed')
        )
        toast.error('เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่', { id: 'session-expired' })
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        localStorage.removeItem('_last_refresh_attempt')
        useAuthStore.getState().logout()
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)

// --- Proactive refresh: refresh before access token expires ---

let proactiveTimer: ReturnType<typeof setTimeout> | null = null

function parseJwtExp(token: string): number | null {
  try {
    const parts = token.split('.')
    if (!parts[1]) return null
    const payload = JSON.parse(atob(parts[1]))
    return payload.exp ? payload.exp * 1000 : null
  } catch {
    return null
  }
}

function scheduleProactiveRefresh(accessToken: string) {
  if (proactiveTimer) clearTimeout(proactiveTimer)

  const exp = parseJwtExp(accessToken)
  if (!exp) return

  // Refresh 2 minutes before expiry (minimum 10 seconds)
  const delay = Math.max(exp - Date.now() - 2 * 60 * 1000, 10_000)

  proactiveTimer = setTimeout(async () => {
    if (!localStorage.getItem('refresh_token')) return
    try {
      await doRefresh()
    } catch {
      // Silent fail — reactive interceptor will handle it on next request
    }
  }, delay)
}

// Initialize proactive refresh on load
const initialToken = localStorage.getItem('access_token')
if (initialToken) {
  scheduleProactiveRefresh(initialToken)
}

// --- Cross-tab sync: listen for token changes from other tabs ---

window.addEventListener('storage', (e) => {
  if (e.key === 'access_token' && e.newValue) {
    // Another tab refreshed — update proactive timer
    scheduleProactiveRefresh(e.newValue)
  }

  if (e.key === 'access_token' && !e.newValue) {
    // Another tab logged out
    useAuthStore.getState().logout()
  }
})

export default api
