import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { NotificationListResult } from '@/types'

export function invalidateNotifications(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: ['notifications'], exact: true })
  qc.invalidateQueries({ queryKey: ['notifications', 'unread-count'], exact: true })
}

export function useNotifications(enabled: boolean) {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: async () => {
      const { data } = await api.get<{ data: NotificationListResult }>('/notifications?limit=50')
      return data.data
    },
    enabled,
    staleTime: 60_000,
  })
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: async () => {
      const { data } = await api.get<{ data: { unread_count: number } }>('/notifications/unread-count')
      return data.data.unread_count
    },
    refetchInterval: 60_000,
  })
}

export function useMarkRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/notifications/${id}/read`)
    },
    onSuccess: () => invalidateNotifications(qc),
  })
}

export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      await api.post('/notifications/read-all')
    },
    onSuccess: () => invalidateNotifications(qc),
  })
}
