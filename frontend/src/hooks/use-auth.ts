import { useMutation } from '@tanstack/react-query'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import type { APIResponse, AuthTokens, Customer } from '@/types'

function storeAuthTokens(data: AuthTokens) {
  localStorage.setItem('access_token', data.access_token)
  localStorage.setItem('refresh_token', data.refresh_token)
  useAuthStore.getState().setCustomer(data.customer)
}

export function useGoogleAuth() {
  return useMutation({
    mutationFn: async (idToken: string) => {
      const { data } = await api.post<APIResponse<AuthTokens>>('/auth/google', { id_token: idToken })
      return data.data!
    },
    onSuccess: storeAuthTokens,
  })
}

export function useRegenerateAPIKey() {
  return useMutation({
    mutationFn: async () => {
      const { data } = await api.post<APIResponse<Customer>>('/auth/regenerate-api-key')
      return data.data!
    },
    onSuccess: (customer) => {
      useAuthStore.getState().setCustomer(customer)
    },
  })
}
