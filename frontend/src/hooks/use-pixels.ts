import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, Pixel } from '@/types'

export function usePixels() {
  return useQuery({
    queryKey: ['pixels'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<Pixel[]>>('/pixels')
      return data.data!
    },
  })
}

export function useCreatePixel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: { fb_pixel_id: string; fb_access_token: string; name: string }) => {
      const { data } = await api.post<APIResponse<Pixel>>('/pixels', input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pixels'] })
    },
  })
}

export function useUpdatePixel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, ...input }: { id: string; fb_pixel_id?: string; fb_access_token?: string; name?: string; is_active?: boolean }) => {
      const { data } = await api.put<APIResponse<Pixel>>(`/pixels/${id}`, input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pixels'] })
    },
  })
}

export function useDeletePixel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/pixels/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pixels'] })
    },
  })
}

export function useTestPixel() {
  return useMutation({
    mutationFn: async (pixelId: string) => {
      const { data } = await api.post<APIResponse<{ events_received: number; fbtrace_id?: string }>>(`/pixels/${pixelId}/test`)
      return data.data!
    },
  })
}
