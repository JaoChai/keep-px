import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'
import type { APIResponse, PaginatedResponse, SalePage, SalePageContent, SalePageContentV2 } from '@/types'

export function useSalePages() {
  return useQuery({
    queryKey: ['sale-pages'],
    queryFn: async () => {
      const { data } = await api.get<PaginatedResponse<SalePage>>('/sale-pages?per_page=100')
      return data.data
    },
  })
}

export function useCreateSalePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: {
      name: string
      slug?: string
      pixel_ids?: string[]
      template_name: string
      content: SalePageContent | SalePageContentV2
      is_published: boolean
    }) => {
      const { data } = await api.post<APIResponse<SalePage>>('/sale-pages', input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sale-pages'] })
      toast.success('สร้าง Sale Page สำเร็จ')
    },
    onError: () => {
      toast.error('สร้าง Sale Page ไม่สำเร็จ')
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
      pixel_ids?: string[]
      template_name?: string
      content?: SalePageContent | SalePageContentV2
      is_published?: boolean
    }) => {
      const { data } = await api.put<APIResponse<SalePage>>(`/sale-pages/${id}`, input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sale-pages'] })
      toast.success('อัปเดต Sale Page สำเร็จ')
    },
    onError: () => {
      toast.error('อัปเดต Sale Page ไม่สำเร็จ')
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
      toast.success('ลบ Sale Page สำเร็จ')
    },
    onError: () => {
      toast.error('ลบ Sale Page ไม่สำเร็จ')
    },
  })
}
