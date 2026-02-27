import { useMutation } from '@tanstack/react-query'
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
  })
}

export function useUploadImages() {
  return useMutation({
    mutationFn: async (files: File[]) => {
      const urls: string[] = []
      for (const file of files) {
        try {
          const formData = new FormData()
          formData.append('file', file)
          const { data } = await api.post<{ data: UploadResponse }>('/uploads/image', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
          })
          urls.push(data.data.url)
        } catch (err) {
          console.error(`Failed to upload ${file.name}:`, err)
        }
      }
      if (urls.length === 0) {
        throw new Error('All uploads failed')
      }
      return urls
    },
  })
}
