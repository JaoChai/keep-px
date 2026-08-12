import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Customer } from '@/types'

interface AuthState {
  customer: Customer | null
  isAuthenticated: boolean
  setCustomer: (customer: Customer) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      customer: null,
      isAuthenticated: false,
      setCustomer: (customer) => set({ customer, isAuthenticated: true }),
      logout: () => set({ customer: null, isAuthenticated: false }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        // Persist only non-sensitive fields — api_key stays in memory only
        customer: state.customer
          ? {
              id: state.customer.id,
              email: state.customer.email,
              name: state.customer.name,
              plan: state.customer.plan,
              is_admin: state.customer.is_admin,
              api_key: '',
              created_at: state.customer.created_at,
              updated_at: state.customer.updated_at,
            }
          : null,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
