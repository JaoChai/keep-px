import { useMutation } from '@tanstack/react-query'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import type { APIResponse, AuthTokens } from '@/types'

interface LoginInput {
  email: string
  password: string
}

interface RegisterInput {
  email: string
  password: string
  name: string
}

export function useLogin() {
  const setCustomer = useAuthStore((s) => s.setCustomer)

  return useMutation({
    mutationFn: async (input: LoginInput) => {
      const { data } = await api.post<APIResponse<AuthTokens>>('/auth/login', input)
      return data.data!
    },
    onSuccess: (data) => {
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      setCustomer(data.customer)
    },
  })
}

export function useRegister() {
  const setCustomer = useAuthStore((s) => s.setCustomer)

  return useMutation({
    mutationFn: async (input: RegisterInput) => {
      const { data } = await api.post<APIResponse<AuthTokens>>('/auth/register', input)
      return data.data!
    },
    onSuccess: (data) => {
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      setCustomer(data.customer)
    },
  })
}
