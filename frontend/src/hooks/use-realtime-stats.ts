import { useMemo } from 'react'
import type { RealtimeEvent } from '@/types'

export interface TimeBucket {
  label: string
  count: number
}

export interface EventTypeCount {
  name: string
  count: number
  percentage: number
}

export interface RealtimeStats {
  eventsPerMinute: number
  capiSuccessRate: number
  topEventType: string
  totalEvents: number
}

export interface UseRealtimeStatsReturn {
  stats: RealtimeStats
  timeBuckets: TimeBucket[]
  eventTypeCounts: EventTypeCount[]
}

const BUCKET_COUNT = 30
const BUCKET_INTERVAL_S = 10
const WINDOW_MS = BUCKET_COUNT * BUCKET_INTERVAL_S * 1000 // 5 min

export function computeRealtimeStats(events: RealtimeEvent[]): UseRealtimeStatsReturn {
  const now = Date.now()
  const totalEvents = events.length

  // Events per minute — count events with created_at within last 60s
  const oneMinuteAgo = now - 60_000
  const eventsPerMinute = events.filter(
    (e) => new Date(e.created_at).getTime() >= oneMinuteAgo
  ).length

  // CAPI success rate
  const forwarded = events.filter((e) => e.forwarded_to_capi).length
  const capiSuccessRate = totalEvents > 0 ? Math.round((forwarded / totalEvents) * 100) : 0

  // Event type counts
  const typeMap = new Map<string, number>()
  for (const e of events) {
    typeMap.set(e.event_name, (typeMap.get(e.event_name) ?? 0) + 1)
  }

  const eventTypeCounts: EventTypeCount[] = [...typeMap.entries()]
    .map(([name, count]) => ({
      name,
      count,
      percentage: totalEvents > 0 ? Math.round((count / totalEvents) * 100) : 0,
    }))
    .sort((a, b) => b.count - a.count)

  const topEventType = eventTypeCounts[0]?.name ?? '-'

  // Time buckets — 30 x 10s = 5 min window
  const windowStart = now - WINDOW_MS
  const timeBuckets: TimeBucket[] = Array.from({ length: BUCKET_COUNT }, (_, i) => {
    const bucketStart = windowStart + i * BUCKET_INTERVAL_S * 1000
    const mins = Math.floor(((i + 1) * BUCKET_INTERVAL_S) / 60)
    const secs = ((i + 1) * BUCKET_INTERVAL_S) % 60
    return {
      label: `${mins}:${secs.toString().padStart(2, '0')}`,
      count: 0,
      _start: bucketStart,
    }
  })

  for (const e of events) {
    const t = new Date(e.created_at).getTime()
    if (t < windowStart) continue
    const idx = Math.min(
      Math.floor((t - windowStart) / (BUCKET_INTERVAL_S * 1000)),
      BUCKET_COUNT - 1
    )
    const bucket = timeBuckets[idx]
    if (bucket) bucket.count++
  }

  // Remove internal _start field
  const cleanBuckets = timeBuckets.map(({ label, count }) => ({ label, count }))

  return {
    stats: { eventsPerMinute, capiSuccessRate, topEventType, totalEvents },
    timeBuckets: cleanBuckets,
    eventTypeCounts,
  }
}

export function useRealtimeStats(events: RealtimeEvent[]): UseRealtimeStatsReturn {
  return useMemo(() => computeRealtimeStats(events), [events])
}
