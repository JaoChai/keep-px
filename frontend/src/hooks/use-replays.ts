import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, ReplaySession } from '@/types'

export function useReplays() {
  return useQuery({
    queryKey: ['replays'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<ReplaySession[]>>('/replays')
      return data.data!
    },
  })
}

export function useReplaySession(id: string | null) {
  return useQuery({
    queryKey: ['replays', id],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<ReplaySession>>(`/replays/${id}`)
      return data.data!
    },
    enabled: !!id,
    refetchInterval: (query) => {
      const session = query.state.data
      if (session && (session.status === 'running' || session.status === 'pending')) {
        return 2000
      }
      return false
    },
  })
}

export function useCreateReplay() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: {
      source_pixel_id: string
      target_pixel_id: string
      event_types?: string[]
      date_from?: string
      date_to?: string
    }) => {
      const { data } = await api.post<APIResponse<ReplaySession>>('/replays', input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['replays'] })
    },
  })
}
