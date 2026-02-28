import { useState, useRef, useCallback, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, RealtimeEvent } from '@/types'

const MAX_BUFFER = 200

function thirtySecondsAgo() {
  return new Date(Date.now() - 30_000).toISOString()
}

export function useRealtimeEvents() {
  const [events, setEvents] = useState<RealtimeEvent[]>([])
  const [isPaused, setIsPaused] = useState(false)
  const [pixelId, setPixelIdState] = useState<string | null>(null)
  const sinceRef = useRef<string>(thirtySecondsAgo())
  const processedIdsRef = useRef<Set<string>>(new Set())

  const query = useQuery({
    queryKey: ['realtime-events', pixelId],
    queryFn: async () => {
      const params = new URLSearchParams({ since: sinceRef.current })
      if (pixelId) params.set('pixel_id', pixelId)
      const { data } = await api.get<APIResponse<RealtimeEvent[]>>(
        `/events/recent?${params.toString()}`
      )
      return data.data ?? []
    },
    refetchInterval: isPaused ? false : 2000,
  })

  useEffect(() => {
    const data = query.data
    if (!data || data.length === 0) return

    // Deduplicate using a Set of processed IDs
    const newEvents = data.filter((e) => !processedIdsRef.current.has(e.id))
    if (newEvents.length === 0) return

    // Track new event IDs
    for (const e of newEvents) {
      processedIdsRef.current.add(e.id)
    }

    // Update since cursor to the NEWEST event (last in ASC-ordered array)
    const lastEvent = data[data.length - 1]
    if (lastEvent) {
      sinceRef.current = lastEvent.created_at
    }

    // Prepend new events (reversed so newest first) and cap buffer
    setEvents((prev) => [...newEvents.reverse(), ...prev].slice(0, MAX_BUFFER))

    // Cap the processedIds set to prevent memory leak
    if (processedIdsRef.current.size > MAX_BUFFER * 2) {
      const entries = [...processedIdsRef.current]
      processedIdsRef.current = new Set(entries.slice(-MAX_BUFFER))
    }
  }, [query.data])

  const setPixelId = useCallback((id: string | null) => {
    setPixelIdState(id)
    setEvents([])
    sinceRef.current = thirtySecondsAgo()
    processedIdsRef.current = new Set()
  }, [])

  const togglePause = useCallback(() => setIsPaused((p) => !p), [])

  const clear = useCallback(() => {
    setEvents([])
    sinceRef.current = new Date().toISOString()
    processedIdsRef.current = new Set()
  }, [])

  return {
    events,
    isLive: !isPaused && !query.isError,
    isPaused,
    togglePause,
    clear,
    pixelId,
    setPixelId,
  }
}
