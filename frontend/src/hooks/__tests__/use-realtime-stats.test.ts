import { describe, it, expect } from 'vitest'
import { computeRealtimeStats } from '../use-realtime-stats'
import type { RealtimeEvent } from '@/types'

function makeEvent(overrides: Partial<RealtimeEvent> = {}): RealtimeEvent {
  return {
    id: crypto.randomUUID(),
    pixel_id: 'px-1',
    pixel_name: 'Test Pixel',
    event_name: 'PageView',
    source_url: 'https://example.com',
    forwarded_to_capi: true,
    event_time: new Date().toISOString(),
    created_at: new Date().toISOString(),
    ...overrides,
  }
}

describe('computeRealtimeStats', () => {
  it('returns zero/default stats for empty events', () => {
    const { stats, timeBuckets, eventTypeCounts } = computeRealtimeStats([])

    expect(stats.eventsPerMinute).toBe(0)
    expect(stats.capiSuccessRate).toBe(0)
    expect(stats.topEventType).toBe('-')
    expect(stats.totalEvents).toBe(0)
    expect(timeBuckets).toHaveLength(30)
    expect(timeBuckets.every((b) => b.count === 0)).toBe(true)
    expect(eventTypeCounts).toHaveLength(0)
  })

  it('counts events per minute from last 60 seconds', () => {
    const now = new Date()
    const recent = makeEvent({ created_at: new Date(now.getTime() - 10_000).toISOString() })
    const old = makeEvent({ created_at: new Date(now.getTime() - 120_000).toISOString() })

    const { stats } = computeRealtimeStats([recent, old])
    expect(stats.eventsPerMinute).toBe(1)
    expect(stats.totalEvents).toBe(2)
  })

  it('calculates CAPI success rate', () => {
    const events = [
      makeEvent({ forwarded_to_capi: true }),
      makeEvent({ forwarded_to_capi: true }),
      makeEvent({ forwarded_to_capi: false }),
      makeEvent({ forwarded_to_capi: false }),
    ]

    const { stats } = computeRealtimeStats(events)
    expect(stats.capiSuccessRate).toBe(50)
  })

  it('identifies top event type', () => {
    const events = [
      makeEvent({ event_name: 'PageView' }),
      makeEvent({ event_name: 'PageView' }),
      makeEvent({ event_name: 'Purchase' }),
      makeEvent({ event_name: 'Lead' }),
      makeEvent({ event_name: 'Lead' }),
      makeEvent({ event_name: 'Lead' }),
    ]

    const { stats, eventTypeCounts } = computeRealtimeStats(events)
    expect(stats.topEventType).toBe('Lead')
    expect(eventTypeCounts[0]!.name).toBe('Lead')
    expect(eventTypeCounts[0]!.count).toBe(3)
    expect(eventTypeCounts[0]!.percentage).toBe(50)
    expect(eventTypeCounts).toHaveLength(3)
  })

  it('distributes events into time buckets', () => {
    const now = Date.now()
    // Event 5 seconds ago — should land in the last bucket
    const events = [
      makeEvent({ created_at: new Date(now - 5_000).toISOString() }),
      makeEvent({ created_at: new Date(now - 5_000).toISOString() }),
    ]

    const { timeBuckets } = computeRealtimeStats(events)
    const lastBucket = timeBuckets[timeBuckets.length - 1]!
    expect(lastBucket.count).toBe(2)
  })

  it('ignores events older than 5 min window in time buckets', () => {
    const now = Date.now()
    const events = [
      makeEvent({ created_at: new Date(now - 6 * 60 * 1000).toISOString() }),
    ]

    const { timeBuckets } = computeRealtimeStats(events)
    const totalInBuckets = timeBuckets.reduce((sum, b) => sum + b.count, 0)
    expect(totalInBuckets).toBe(0)
  })

  it('sorts event type counts descending by count', () => {
    const events = [
      makeEvent({ event_name: 'A' }),
      makeEvent({ event_name: 'B' }),
      makeEvent({ event_name: 'B' }),
      makeEvent({ event_name: 'C' }),
      makeEvent({ event_name: 'C' }),
      makeEvent({ event_name: 'C' }),
    ]

    const { eventTypeCounts } = computeRealtimeStats(events)
    expect(eventTypeCounts.map((t) => t.name)).toEqual(['C', 'B', 'A'])
  })

  it('handles 100% CAPI success rate', () => {
    const events = [
      makeEvent({ forwarded_to_capi: true }),
      makeEvent({ forwarded_to_capi: true }),
    ]

    const { stats } = computeRealtimeStats(events)
    expect(stats.capiSuccessRate).toBe(100)
  })

  it('handles 0% CAPI success rate', () => {
    const events = [
      makeEvent({ forwarded_to_capi: false }),
      makeEvent({ forwarded_to_capi: false }),
    ]

    const { stats } = computeRealtimeStats(events)
    expect(stats.capiSuccessRate).toBe(0)
  })
})
