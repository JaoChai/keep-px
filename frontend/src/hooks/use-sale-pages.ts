import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, SalePage, SalePageContent, SalePageContentV2 } from '@/types'

export function useSalePages() {
  return useQuery({
    queryKey: ['sale-pages'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<SalePage[]>>('/sale-pages')
      return data.data!
    },
  })
}

export function useCreateSalePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: {
      name: string
      slug?: string
      pixel_id?: string
      template_name: string
      content: SalePageContent | SalePageContentV2
      is_published: boolean
    }) => {
      const { data } = await api.post<APIResponse<SalePage>>('/sale-pages', input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sale-pages'] })
    },
  })
}

export function useUpdateSalePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...input
    }: {
      id: string
      name?: string
      slug?: string
      pixel_id?: string
      template_name?: string
      content?: SalePageContent | SalePageContentV2
      is_published?: boolean
    }) => {
      const { data } = await api.put<APIResponse<SalePage>>(`/sale-pages/${id}`, input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sale-pages'] })
    },
  })
}

export function useDeleteSalePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/sale-pages/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sale-pages'] })
    },
  })
}
