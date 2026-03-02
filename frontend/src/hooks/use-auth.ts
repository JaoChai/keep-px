import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
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

function storeAuthTokens(data: AuthTokens) {
  localStorage.setItem('access_token', data.access_token)
  localStorage.setItem('refresh_token', data.refresh_token)
  useAuthStore.getState().setCustomer(data.customer)
}

export function useLogin() {
  return useMutation({
    mutationFn: async (input: LoginInput) => {
      const { data } = await api.post<APIResponse<AuthTokens>>('/auth/login', input)
      return data.data!
    },
    onSuccess: storeAuthTokens,
  })
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

export function useRegister() {
  return useMutation({
    mutationFn: async (input: RegisterInput) => {
      const { data } = await api.post<APIResponse<AuthTokens>>('/auth/register', input)
      return data.data!
    },
    onSuccess: (data) => {
      storeAuthTokens(data)
      toast.success('สมัครสมาชิกสำเร็จ')
    },
    onError: () => {
      toast.error('สมัครไม่สำเร็จ อีเมลอาจถูกใช้แล้ว')
    },
  })
}
