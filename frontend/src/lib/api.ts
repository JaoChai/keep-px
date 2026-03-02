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

        // Sync customer data back to Zustand store
        if (data.data.customer) {
          useAuthStore.getState().setCustomer(data.data.customer)
        }

        onRefreshed(newAccessToken)
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
        return api(originalRequest)
      } catch (err) {
        onRefreshFailed(
          err instanceof Error ? err : new Error('Token refresh failed')
        )
        toast.error('เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่', { id: 'session-expired' })
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        useAuthStore.getState().logout()
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)

export default api
