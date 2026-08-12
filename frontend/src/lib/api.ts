import axios from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { authClient, getAccessToken, clearAccessToken } from '@/lib/neon-auth'

const API_BASE = import.meta.env.VITE_API_URL || ''

const api = axios.create({
  baseURL: `${API_BASE}/api/v1`,
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use(async (config) => {
  const token = await getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status
    const message = error.response?.data?.error

    // ยังไม่มี customer ในระบบเรา — ลงทะเบียนแล้วลองใหม่หนึ่งครั้ง
    if (status === 401 && message === 'customer_not_provisioned' && !error.config._retried) {
      error.config._retried = true
      const { data } = await api.post('/auth/session')
      useAuthStore.getState().setCustomer(data.data)
      return api.request(error.config)
    }

    if (status === 401) {
      clearAccessToken()
      await authClient.signOut()
      useAuthStore.getState().logout()
      window.location.href = '/login'
      return Promise.reject(error)
    }

    if (status === 403 && message === 'account suspended') {
      toast.error('บัญชีนี้ถูกระงับการใช้งาน')
    }

    return Promise.reject(error)
  }
)

export default api
