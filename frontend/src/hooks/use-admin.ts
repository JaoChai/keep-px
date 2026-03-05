import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'
import type { PaginatedResponse, APIResponse } from '@/types'
import type {
  AdminCustomer,
  AdminCustomerDetail,
  AdminAnalytics,
  AdminPurchase,
  AdminSubscription,
  AdminCreditGrant,
  RevenueChartPoint,
  GrowthChartPoint,
} from '@/types/admin'

// --- Queries ---

export function useAdminCustomers(search: string, plan: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'customers', { search, plan, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (search) params.set('search', search)
      if (plan) params.set('plan', plan)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminCustomer>>(`/admin/customers?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminCustomerDetail(customerId: string | null) {
  return useQuery({
    queryKey: ['admin', 'customers', customerId],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<AdminCustomerDetail>>(`/admin/customers/${customerId}`)
      return data.data!
    },
    enabled: !!customerId,
  })
}

export function useAdminAnalytics() {
  return useQuery({
    queryKey: ['admin', 'analytics'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<AdminAnalytics>>('/admin/analytics/overview')
      return data.data!
    },
  })
}

export function useAdminRevenueChart(days: number) {
  return useQuery({
    queryKey: ['admin', 'analytics', 'revenue', days],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<RevenueChartPoint[]>>('/admin/analytics/revenue', {
        params: { days },
      })
      return data.data!
    },
  })
}

export function useAdminGrowthChart(days: number) {
  return useQuery({
    queryKey: ['admin', 'analytics', 'growth', days],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<GrowthChartPoint[]>>('/admin/analytics/growth', {
        params: { days },
      })
      return data.data!
    },
  })
}

export function useAdminPurchases(status: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'purchases', { status, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (status) params.set('status', status)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminPurchase>>(`/admin/purchases?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminSubscriptions(status: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'subscriptions', { status, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (status) params.set('status', status)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminSubscription>>(`/admin/subscriptions?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminCreditGrants(page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'credit-grants', { page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminCreditGrant>>(`/admin/credit-grants?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

// --- Mutations ---

export function useAdminUpdateCustomerPlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ customerId, plan }: { customerId: string; plan: string }) => {
      const { data } = await api.put(`/admin/customers/${customerId}/plan`, { plan })
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'customers'] })
      toast.success('เปลี่ยนแผนสำเร็จ')
    },
    onError: () => {
      toast.error('เปลี่ยนแผนไม่สำเร็จ')
    },
  })
}

export function useAdminSuspendCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (customerId: string) => {
      const { data } = await api.post(`/admin/customers/${customerId}/suspend`)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'customers'] })
      toast.success('ระงับบัญชีสำเร็จ')
    },
    onError: () => {
      toast.error('ระงับบัญชีไม่สำเร็จ')
    },
  })
}

export function useAdminActivateCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (customerId: string) => {
      const { data } = await api.post(`/admin/customers/${customerId}/activate`)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'customers'] })
      toast.success('เปิดใช้งานบัญชีสำเร็จ')
    },
    onError: () => {
      toast.error('เปิดใช้งานบัญชีไม่สำเร็จ')
    },
  })
}

export function useAdminGrantCredits() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      customerId,
      ...body
    }: {
      customerId: string
      pack_type: string
      total_replays: number
      max_events_per_replay: number
      expires_in_days: number
      reason?: string
    }) => {
      const { data } = await api.post(`/admin/customers/${customerId}/credits`, body)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'customers'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'credit-grants'] })
      toast.success('เพิ่มเครดิตสำเร็จ')
    },
    onError: () => {
      toast.error('เพิ่มเครดิตไม่สำเร็จ')
    },
  })
}
