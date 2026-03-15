import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, PaginatedResponse, PixelEvent } from '@/types'

export function useEvents(page = 1, perPage = 50, pixelId?: string | null, eventName?: string | null) {
  return useQuery({
    queryKey: ['events', page, perPage, pixelId ?? '', eventName ?? ''],
    queryFn: async () => {
      const params: Record<string, string | number> = { page, per_page: perPage }
      if (pixelId) params.pixel_id = pixelId
      if (eventName) params.event_name = eventName
      const { data } = await api.get<PaginatedResponse<PixelEvent>>('/events', { params })
      return data
    },
  })
}

export function useEventDetail(eventId: string | null) {
  return useQuery({
    queryKey: ['event-detail', eventId],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<PixelEvent>>(`/events/${eventId}`)
      return data.data!
    },
    enabled: !!eventId,
  })
}

export function useCustomerEventTypes() {
  return useQuery({
    queryKey: ['event-types'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<string[]>>('/events/event-types')
      return data.data ?? []
    },
  })
}
