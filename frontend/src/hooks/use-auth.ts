import { useMutation } from '@tanstack/react-query'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import type { APIResponse, Customer } from '@/types'

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
