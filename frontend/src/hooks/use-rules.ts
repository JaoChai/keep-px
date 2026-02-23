import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, EventRule } from '@/types'

export function useRulesByPixel(pixelId: string | null) {
  return useQuery({
    queryKey: ['rules', pixelId],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<EventRule[]>>(`/pixels/${pixelId}/rules`)
      return data.data!
    },
    enabled: !!pixelId,
  })
}

export function useCreateRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ pixelId, ...input }: { pixelId: string; page_url: string; event_name: string; trigger_type: string; css_selector?: string; element_text?: string }) => {
      const { data } = await api.post<APIResponse<EventRule>>(`/pixels/${pixelId}/rules`, input)
      return data.data!
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['rules', variables.pixelId] })
    },
  })
}

export function useUpdateRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, ...input }: { id: string; page_url?: string; event_name?: string; trigger_type?: string; css_selector?: string; element_text?: string; is_active?: boolean }) => {
      const { data } = await api.put<APIResponse<EventRule>>(`/rules/${id}`, input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
    },
  })
}

export function useDeleteRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/rules/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
    },
  })
}
