import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, CustomDomain } from '@/types'

export function useCustomDomains() {
  return useQuery({
    queryKey: ['custom-domains'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<CustomDomain[]>>('/domains')
      return data.data!
    },
  })
}

interface CreateDomainData {
  domain: CustomDomain
  cname_target: string
}

export function useCreateCustomDomain() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: { domain: string; sale_page_id: string }) => {
      const { data } = await api.post<APIResponse<CreateDomainData>>('/domains', input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-domains'] })
    },
  })
}

export function useVerifyCustomDomain() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.post<APIResponse<CustomDomain>>(`/domains/${id}/verify`)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-domains'] })
    },
  })
}

export function useDeleteCustomDomain() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/domains/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-domains'] })
    },
  })
}
