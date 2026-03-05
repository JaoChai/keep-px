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
  AdminSalePage,
  AdminSalePageDetail,
  AdminPixel,
  AdminPixelDetail,
  AdminReplaySession,
  AdminReplaySessionDetail,
  AdminEvent,
  AdminEventStats,
  AuditLogEntry,
} from '@/types/admin'

// --- Queries ---

export function useAdminCustomers(search: string, plan: string, status: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'customers', { search, plan, status, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (search) params.set('search', search)
      if (plan) params.set('plan', plan)
      if (status) params.set('status', status)
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

// F1: Sale Pages
export function useAdminSalePages(search: string, customerID: string, published: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'sale-pages', { search, customerID, published, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (search) params.set('search', search)
      if (customerID) params.set('customer_id', customerID)
      if (published) params.set('published', published)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminSalePage>>(`/admin/sale-pages?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminSalePageDetail(id: string | null) {
  return useQuery({
    queryKey: ['admin', 'sale-pages', id],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<AdminSalePageDetail>>(`/admin/sale-pages/${id}`)
      return data.data!
    },
    enabled: !!id,
  })
}

export function useAdminToggleSalePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, enable }: { id: string; enable: boolean }) => {
      const { data } = await api.post(`/admin/sale-pages/${id}/${enable ? 'enable' : 'disable'}`)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'sale-pages'] })
      toast.success('อัปเดตสถานะเซลเพจสำเร็จ')
    },
    onError: () => toast.error('อัปเดตสถานะไม่สำเร็จ'),
  })
}

export function useAdminDeleteSalePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.delete(`/admin/sale-pages/${id}`)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'sale-pages'] })
      toast.success('ลบเซลเพจสำเร็จ')
    },
    onError: () => toast.error('ลบเซลเพจไม่สำเร็จ'),
  })
}

// F2: Pixels
export function useAdminPixels(search: string, customerID: string, active: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'pixels', { search, customerID, active, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (search) params.set('search', search)
      if (customerID) params.set('customer_id', customerID)
      if (active) params.set('active', active)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminPixel>>(`/admin/pixels?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminPixelDetail(id: string | null) {
  return useQuery({
    queryKey: ['admin', 'pixels', id],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<AdminPixelDetail>>(`/admin/pixels/${id}`)
      return data.data!
    },
    enabled: !!id,
  })
}

export function useAdminTogglePixel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, enable }: { id: string; enable: boolean }) => {
      const { data } = await api.post(`/admin/pixels/${id}/${enable ? 'enable' : 'disable'}`)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'pixels'] })
      toast.success('อัปเดตสถานะพิกเซลสำเร็จ')
    },
    onError: () => toast.error('อัปเดตสถานะไม่สำเร็จ'),
  })
}

// F3: Replays
export function useAdminReplays(status: string, customerID: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'replays', { status, customerID, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (status) params.set('status', status)
      if (customerID) params.set('customer_id', customerID)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminReplaySession>>(`/admin/replays?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminReplayDetail(id: string | null) {
  return useQuery({
    queryKey: ['admin', 'replays', id],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<AdminReplaySessionDetail>>(`/admin/replays/${id}`)
      return data.data!
    },
    enabled: !!id,
  })
}

export function useAdminCancelReplay() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.post(`/admin/replays/${id}/cancel`)
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'replays'] })
      toast.success('ยกเลิกรีเพลย์สำเร็จ')
    },
    onError: () => toast.error('ยกเลิกรีเพลย์ไม่สำเร็จ'),
  })
}

// F4: Events
export function useAdminEvents(customerID: string, pixelID: string, eventName: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'events', { customerID, pixelID, eventName, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (customerID) params.set('customer_id', customerID)
      if (pixelID) params.set('pixel_id', pixelID)
      if (eventName) params.set('event_name', eventName)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AdminEvent>>(`/admin/events?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
  })
}

export function useAdminEventStats(hours: number = 24) {
  return useQuery({
    queryKey: ['admin', 'events', 'stats', hours],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<AdminEventStats>>('/admin/events/stats', { params: { hours } })
      return data.data!
    },
    refetchInterval: 60000,
  })
}

// F5: Audit Log
export function useAdminAuditLog(adminID: string, action: string, targetCustomerID: string, from: string, to: string, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'audit-log', { adminID, action, targetCustomerID, from, to, page, perPage }],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (adminID) params.set('admin_id', adminID)
      if (action) params.set('action', action)
      if (targetCustomerID) params.set('target_customer_id', targetCustomerID)
      if (from) params.set('from', from)
      if (to) params.set('to', to)
      params.set('page', String(page))
      params.set('per_page', String(perPage))
      const { data } = await api.get<PaginatedResponse<AuditLogEntry>>(`/admin/audit-log?${params}`)
      return data
    },
    placeholderData: (prev) => prev,
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
