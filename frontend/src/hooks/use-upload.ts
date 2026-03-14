import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'

interface UploadResponse {
  url: string
}

export function useUploadImage() {
  const [progress, setProgress] = useState(0)
  const mutation = useMutation({
    mutationFn: async (file: File) => {
      setProgress(0)
      const formData = new FormData()
      formData.append('file', file)
      const { data } = await api.post<{ data: UploadResponse }>('/uploads/image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (e) => {
          if (e.total) {
            setProgress(Math.round((e.loaded * 100) / e.total))
          }
        },
      })
      setProgress(100)
      return data.data.url
    },
    onError: () => {
      setProgress(0)
      toast.error('อัปโหลดรูปภาพไม่สำเร็จ')
    },
    onSettled: () => {
      setTimeout(() => setProgress(0), 500)
    },
  })
  return { ...mutation, progress }
}

export function useUploadImages() {
  const [progress, setProgress] = useState(0)
  const mutation = useMutation({
    mutationFn: async (files: File[]) => {
      setProgress(0)
      let completed = 0
      const results = await Promise.allSettled(
        files.map(async (file) => {
          const formData = new FormData()
          formData.append('file', file)
          const { data } = await api.post<{ data: UploadResponse }>('/uploads/image', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
          })
          completed++
          setProgress(Math.round((completed * 100) / files.length))
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
      setProgress(0)
      toast.error('อัปโหลดรูปภาพไม่สำเร็จ')
    },
    onSettled: () => {
      setTimeout(() => setProgress(0), 500)
    },
  })
  return { ...mutation, progress }
}
