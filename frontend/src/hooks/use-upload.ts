import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'

interface UploadResponse {
  url: string
}

export function useUploadImage() {
  return useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      const { data } = await api.post<{ data: UploadResponse }>('/uploads/image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      return data.data.url
    },
    onError: () => {
      toast.error('อัปโหลดรูปภาพไม่สำเร็จ')
    },
  })
}

export function useUploadImages() {
  return useMutation({
    mutationFn: async (files: File[]) => {
      const results = await Promise.allSettled(
        files.map(async (file) => {
          const formData = new FormData()
          formData.append('file', file)
          const { data } = await api.post<{ data: UploadResponse }>('/uploads/image', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
          })
          return data.data.url
        })
      )
      const urls = results
        .filter((r): r is PromiseFulfilledResult<string> => r.status === 'fulfilled')
        .map((r) => r.value)
      if (urls.length === 0) {
        throw new Error('All uploads failed')
      }
      return urls
    },
    onError: () => {
      toast.error('อัปโหลดรูปภาพไม่สำเร็จ')
    },
  })
}
