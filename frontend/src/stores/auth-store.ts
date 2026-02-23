import { create } from 'zustand'
import type { Customer } from '@/types'

interface AuthState {
  customer: Customer | null
  isAuthenticated: boolean
  setCustomer: (customer: Customer) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  customer: null,
  isAuthenticated: false,
  setCustomer: (customer) => set({ customer, isAuthenticated: true }),
  logout: () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    set({ customer: null, isAuthenticated: false })
  },
}))
