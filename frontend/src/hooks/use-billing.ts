import { useQuery, useMutation } from '@tanstack/react-query'
import api from '@/lib/api'
import type { BillingOverview, CustomerQuota } from '@/types'

export function useBillingOverview() {
  return useQuery({
    queryKey: ['billing', 'overview'],
    queryFn: async () => {
      const { data } = await api.get<{ data: BillingOverview }>('/billing')
      return data.data
    },
  })
}

export function useQuota() {
  return useQuery({
    queryKey: ['billing', 'quota'],
    queryFn: async () => {
      const { data } = await api.get<{ data: CustomerQuota }>('/billing/quota')
      return data.data
    },
  })
}

export function useCreateCheckout() {
  return useMutation({
    mutationFn: async (params: { pack_type?: string; addon_type?: string; plan_type?: string }) => {
      const { data } = await api.post<{ data: { url: string } }>('/billing/checkout', params)
      return data.data
    },
    onSuccess: (data) => {
      window.location.href = data.url
    },
  })
}

export function useCreatePortalSession() {
  return useMutation({
    mutationFn: async () => {
      const { data } = await api.post<{ data: { url: string } }>('/billing/portal')
      return data.data
    },
    onSuccess: (data) => {
      window.location.href = data.url
    },
  })
}
