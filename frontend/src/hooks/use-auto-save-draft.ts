import { useEffect, useRef, useCallback } from 'react'

export function useAutoSaveDraft<T>(key: string, data: T, debounceMs = 2000) {
  const isFirstRender = useRef(true)
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false
      return
    }
    clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => {
      try {
        localStorage.setItem(key, JSON.stringify(data))
      } catch {
        // localStorage quota exceeded — silently ignore
      }
    }, debounceMs)
    return () => clearTimeout(timerRef.current)
  }, [key, data, debounceMs])

  const clearDraft = useCallback(() => {
    localStorage.removeItem(key)
  }, [key])

  return { clearDraft }
}

export function loadDraft<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return null
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

export function hasDraft(key: string): boolean {
  return localStorage.getItem(key) !== null
}
