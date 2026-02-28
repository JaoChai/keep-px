import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import type { PaginatedResponse, PixelEvent } from '@/types'

export function useEvents(page = 1, perPage = 50, pixelId?: string | null) {
  return useQuery({
    queryKey: ['events', page, perPage, pixelId ?? ''],
    queryFn: async () => {
      const params: Record<string, string | number> = { page, per_page: perPage }
      if (pixelId) params.pixel_id = pixelId
      const { data } = await api.get<PaginatedResponse<PixelEvent>>('/events', { params })
      return data
    },
  })
}
