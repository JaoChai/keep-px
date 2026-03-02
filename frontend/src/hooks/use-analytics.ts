import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, RealtimeEvent } from '@/types'

interface OverviewStats {
  total_pixels: number
  active_pixels: number
  total_events: number
  events_today: number
  events_yesterday: number
  events_this_week: number
  total_replays: number
  active_replays: number
  forwarded_events: number
}

interface EventChartData {
  date: string
  count: number
}

export function useOverviewStats() {
  return useQuery({
    queryKey: ['analytics', 'overview'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<OverviewStats>>('/analytics/overview')
      return data.data!
    },
  })
}

export function useDashboardRecentEvents(limit = 8) {
  return useQuery({
    queryKey: ['dashboard', 'recent-events', limit],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<RealtimeEvent[]>>('/events/recent', {
        params: { limit },
      })
      return data.data ?? []
    },
  })
}

export function useEventChart(days = 30) {
  return useQuery({
    queryKey: ['analytics', 'events', days],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<EventChartData[]>>('/analytics/events', {
        params: { days },
      })
      return data.data!
    },
  })
}
