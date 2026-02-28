import { useState, useRef, useCallback, useEffect, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, RealtimeEvent } from '@/types'

const MAX_BUFFER = 200

export function useRealtimeEvents() {
  const queryClient = useQueryClient()
  const [polledEvents, setPolledEvents] = useState<RealtimeEvent[]>([])
  const [isPaused, setIsPaused] = useState(false)
  const [pixelId, setPixelIdState] = useState<string | null>(null)
  const sinceRef = useRef<string>('')
  const processedIdsRef = useRef<Set<string>>(new Set())
  // Generation counter: bumping it creates a new query key, resetting the initial load
  const [generation, setGeneration] = useState(0)

  // Query 1: Initial load — fetch latest 100 events (no `since` param)
  const initialQuery = useQuery({
    queryKey: ['realtime-events-initial', pixelId, generation],
    queryFn: async () => {
      const params = new URLSearchParams({ limit: '100' })
      if (pixelId) params.set('pixel_id', pixelId)
      const { data } = await api.get<APIResponse<RealtimeEvent[]>>(
        `/events/recent?${params.toString()}`
      )
      return data.data ?? []
    },
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })

  // Derive isInitialLoaded from query status — no useState needed
  const isInitialLoaded = initialQuery.isSuccess && !!initialQuery.data

  // Initialize dedup refs and cursor from initial load data (refs only, no setState)
  // Backend returns events in ASC order — last element is the newest
  useEffect(() => {
    const data = initialQuery.data
    if (!data) return

    const ids = new Set<string>()
    for (const e of data) {
      ids.add(e.id)
    }
    processedIdsRef.current = ids

    const lastEvent = data[data.length - 1]
    sinceRef.current = lastEvent ? lastEvent.created_at : new Date().toISOString()
  }, [initialQuery.data])

  // Derive initial events (reversed for newest-first display) from query cache
  const initialEvents = useMemo(() => {
    if (!initialQuery.data) return []
    return [...initialQuery.data].reverse()
  }, [initialQuery.data])

  // Query 2: Polling — fetch incremental events using `since` cursor (2s interval)
  const pollingQuery = useQuery({
    queryKey: ['realtime-events-polling', pixelId],
    queryFn: async () => {
      const params = new URLSearchParams({ since: sinceRef.current })
      if (pixelId) params.set('pixel_id', pixelId)
      const { data } = await api.get<APIResponse<RealtimeEvent[]>>(
        `/events/recent?${params.toString()}`
      )
      return data.data ?? []
    },
    enabled: isInitialLoaded && !isPaused,
    refetchInterval: isPaused ? false : 2000,
  })

  // Process polling results — append new events to polled buffer
  useEffect(() => {
    const data = pollingQuery.data
    if (!data || data.length === 0) return

    const newEvents = data.filter((e) => !processedIdsRef.current.has(e.id))
    if (newEvents.length === 0) return

    for (const e of newEvents) {
      processedIdsRef.current.add(e.id)
    }

    const lastEvent = data[data.length - 1]
    if (lastEvent) {
      sinceRef.current = lastEvent.created_at
    }

    setPolledEvents((prev) => [...newEvents.reverse(), ...prev].slice(0, MAX_BUFFER))

    if (processedIdsRef.current.size > MAX_BUFFER * 2) {
      const entries = [...processedIdsRef.current]
      processedIdsRef.current = new Set(entries.slice(-MAX_BUFFER))
    }
  }, [pollingQuery.data])

  // Combine polled events (newest) + initial events, capped at MAX_BUFFER
  const events = useMemo(() => {
    return [...polledEvents, ...initialEvents].slice(0, MAX_BUFFER)
  }, [polledEvents, initialEvents])

  const setPixelId = useCallback((id: string | null) => {
    setPixelIdState(id)
    setPolledEvents([])
    sinceRef.current = ''
    processedIdsRef.current = new Set()
    setGeneration((g) => g + 1)
    // Remove ALL polling variants (old pixel key), not just the new one
    queryClient.removeQueries({ queryKey: ['realtime-events-polling'] })
  }, [queryClient])

  const togglePause = useCallback(() => setIsPaused((p) => !p), [])

  const clear = useCallback(() => {
    setPolledEvents([])
    sinceRef.current = new Date().toISOString()
    processedIdsRef.current = new Set()
    queryClient.removeQueries({ queryKey: ['realtime-events-initial'] })
    setGeneration((g) => g + 1)
  }, [queryClient])

  const refresh = useCallback(() => {
    setPolledEvents([])
    sinceRef.current = ''
    processedIdsRef.current = new Set()
    setGeneration((g) => g + 1)
  }, [])

  const isLoading = initialQuery.isLoading || initialQuery.isFetching

  return {
    events,
    isLive: !isPaused && !pollingQuery.isError,
    isPaused,
    isLoading,
    togglePause,
    clear,
    refresh,
    pixelId,
    setPixelId,
  }
}
