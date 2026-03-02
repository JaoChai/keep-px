import { useState, useMemo, useEffect } from 'react'
import { useSearchParams } from 'react-router'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Pause,
  Play,
  Trash2,
  Check,
  X,
  Zap,
  CheckCircle,
  Hash,
  Clock,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Activity,
  ScrollText,
} from 'lucide-react'
import { useRealtimeEvents } from '@/hooks/use-realtime-events'
import { useRealtimeStats } from '@/hooks/use-realtime-stats'
import { useEvents } from '@/hooks/use-events'
import { useOverviewStats } from '@/hooks/use-analytics'
import { usePixels } from '@/hooks/use-pixels'
import { formatDistanceToNow } from 'date-fns'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

const eventBadgeVariant = (name: string) => {
  switch (name) {
    case 'PageView':
      return 'secondary'
    case 'Purchase':
      return 'success'
    case 'Lead':
    case 'CompleteRegistration':
      return 'default'
    case 'AddToCart':
    case 'InitiateCheckout':
      return 'warning'
    default:
      return 'outline'
  }
}

interface StatCardProps {
  title: string
  value: number | string
  subtitle?: string
  icon: React.ReactNode
}

function StatCard({ title, value, subtitle, icon }: StatCardProps) {
  return (
    <Card>
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-muted-foreground">{title}</p>
            <p className="text-2xl font-bold text-foreground mt-1">{value}</p>
            {subtitle && <p className="text-xs text-muted-foreground mt-1">{subtitle}</p>}
          </div>
          <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center text-foreground">
            {icon}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

const EVENT_TYPE_COLORS: Record<string, string> = {
  PageView: 'bg-muted-foreground',
  Purchase: 'bg-emerald-500',
  Lead: 'bg-primary',
  CompleteRegistration: 'bg-primary/80',
  AddToCart: 'bg-amber-500',
  InitiateCheckout: 'bg-amber-400',
}

function getEventColor(name: string): string {
  return EVENT_TYPE_COLORS[name] ?? 'bg-muted-foreground'
}

export function EventsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const rawMode = searchParams.get('mode')
  const mode: 'live' | 'history' = rawMode === 'history' ? 'history' : 'live'
  const pixelId = searchParams.get('pixel_id') || null

  const setMode = (m: 'live' | 'history') => {
    setSearchParams({ mode: m, ...(pixelId ? { pixel_id: pixelId } : {}) })
  }

  const setPixelFilter = (id: string | null) => {
    const params: Record<string, string> = { mode }
    if (id) params.pixel_id = id
    setSearchParams(params)
    setHistoryPage(1)
  }

  // All hooks called unconditionally (React rules of hooks)
  const {
    events: realtimeEvents,
    isLive,
    isPaused,
    isLoading: realtimeLoading,
    togglePause,
    clear,
    refresh,
    pixelId: realtimePixelId,
    setPixelId: setRealtimePixelId,
  } = useRealtimeEvents()
  const { stats, timeBuckets, eventTypeCounts } = useRealtimeStats(realtimeEvents)
  const [historyPage, setHistoryPage] = useState(1)
  const historyQuery = useEvents(historyPage, 50, pixelId)
  const { data: overviewStats } = useOverviewStats()
  const { data: pixels } = usePixels()

  // Sync pixel filter to realtime hook
  useEffect(() => {
    if (realtimePixelId !== pixelId) {
      setRealtimePixelId(pixelId)
    }
  }, [pixelId, realtimePixelId, setRealtimePixelId])

  // Auto-pause realtime when in history mode
  useEffect(() => {
    if (mode === 'history' && !isPaused) {
      togglePause()
    } else if (mode === 'live' && isPaused) {
      togglePause()
    }
  }, [mode, isPaused, togglePause])

  // Pixel name resolver for History mode (PixelEvent doesn't have pixel_name)
  const pixelNameMap = useMemo(() => {
    const map = new Map<string, string>()
    if (pixels) {
      for (const p of pixels) {
        map.set(p.id, p.name)
      }
    }
    return map
  }, [pixels])

  // History data
  const historyEvents = historyQuery.data?.data ?? []
  const totalPages = historyQuery.data?.total_pages ?? 1

  // CAPI rate computed from overview stats
  const capiRate =
    overviewStats && overviewStats.total_events > 0
      ? `${Math.round((overviewStats.forwarded_events / overviewStats.total_events) * 100)}%`
      : '-'

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <div>
            <h1 className="text-2xl font-bold text-foreground">Events</h1>
            <p className="text-sm text-muted-foreground mt-1">
              {mode === 'live'
                ? `${realtimeEvents.length} events${stats.eventsPerMinute > 0 ? ` (${stats.eventsPerMinute}/min)` : ''}`
                : historyQuery.data
                  ? `${historyQuery.data.total} events total`
                  : 'Loading events...'}
            </p>
          </div>
          {mode === 'live' && (
            <Badge variant={isLive ? 'success' : 'warning'}>
              {isLive ? 'Live' : 'Paused'}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-3">
          {/* Mode toggle */}
          <div className="flex rounded-lg border border-border p-0.5 bg-secondary">
            <Button
              variant={mode === 'live' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setMode('live')}
              className={mode === 'live' ? '' : 'text-muted-foreground'}
            >
              <Activity className="h-4 w-4 mr-1" />
              Live
            </Button>
            <Button
              variant={mode === 'history' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setMode('history')}
              className={mode === 'history' ? '' : 'text-muted-foreground'}
            >
              <ScrollText className="h-4 w-4 mr-1" />
              History
            </Button>
          </div>

          {/* Live controls */}
          {mode === 'live' && (
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={togglePause}>
                {isPaused ? (
                  <Play className="h-4 w-4 mr-1" />
                ) : (
                  <Pause className="h-4 w-4 mr-1" />
                )}
                {isPaused ? 'Resume' : 'Pause'}
              </Button>
              <Button variant="outline" size="sm" onClick={clear}>
                <Trash2 className="h-4 w-4 mr-1" />
                Clear
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={refresh}
                disabled={realtimeLoading}
              >
                <RefreshCw
                  className={`h-4 w-4 mr-1 ${realtimeLoading ? 'animate-spin' : ''}`}
                />
                Refresh
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Pixel filter */}
      <div className="mb-4">
        <select
          className="border border-border rounded-lg px-3 py-2 text-sm bg-background text-foreground"
          value={pixelId ?? ''}
          onChange={(e) => setPixelFilter(e.target.value || null)}
        >
          <option value="">All Pixels</option>
          {pixels?.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          title="Events Today"
          value={overviewStats?.events_today ?? '-'}
          subtitle="Today's total"
          icon={<Clock className="h-5 w-5" />}
        />
        <StatCard
          title="Total Events"
          value={
            overviewStats?.total_events != null
              ? overviewStats.total_events.toLocaleString()
              : '-'
          }
          subtitle="All time"
          icon={<Hash className="h-5 w-5" />}
        />
        <StatCard
          title="CAPI Rate"
          value={capiRate}
          subtitle="Forwarded to Facebook"
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <StatCard
          title="Events/Min"
          value={mode === 'live' ? stats.eventsPerMinute : '-'}
          subtitle={mode === 'live' ? 'Last 60 seconds' : 'Live mode only'}
          icon={<Zap className="h-5 w-5" />}
        />
      </div>

      {/* Charts (Live mode only) */}
      {mode === 'live' && realtimeEvents.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
          {/* Event Rate Chart */}
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-base">Event Rate (5 min window)</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={timeBuckets}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#E4E4E7" />
                  <XAxis
                    dataKey="label"
                    tick={{ fontSize: 10, fill: '#737373' }}
                    interval={4}
                  />
                  <YAxis tick={{ fontSize: 12, fill: '#737373' }} allowDecimals={false} />
                  <Tooltip
                    contentStyle={{
                      borderRadius: '8px',
                      border: '1px solid #e5e5e5',
                      fontSize: '12px',
                    }}
                  />
                  <Bar dataKey="count" fill="#18181B" radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {/* Event Type Breakdown */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Event Types</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {eventTypeCounts.slice(0, 6).map((type) => (
                  <div key={type.name}>
                    <div className="flex items-center justify-between mb-1">
                      <Badge variant={eventBadgeVariant(type.name)} className="text-xs">
                        {type.name}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {type.count} ({type.percentage}%)
                      </span>
                    </div>
                    <div className="h-2 bg-secondary rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full ${getEventColor(type.name)}`}
                        style={{ width: `${type.percentage}%` }}
                      />
                    </div>
                  </div>
                ))}
                {eventTypeCounts.length === 0 && (
                  <p className="text-sm text-muted-foreground text-center py-4">No data</p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Event Table */}
      {mode === 'live' ? (
        /* Live mode table */
        realtimeEvents.length === 0 ? (
          <div className="text-center py-16 border border-dashed border-border rounded-lg">
            {realtimeLoading ? (
              <>
                <RefreshCw className="h-6 w-6 animate-spin text-foreground mx-auto mb-3" />
                <p className="text-muted-foreground">Loading recent events...</p>
              </>
            ) : (
              <>
                <div className="animate-pulse mb-3">
                  <div className="inline-block h-3 w-3 rounded-full bg-emerald-400" />
                </div>
                <p className="text-muted-foreground">Waiting for events...</p>
                <p className="text-sm text-muted-foreground mt-1">
                  New events will appear here in realtime
                </p>
              </>
            )}
          </div>
        ) : (
          <div className="border border-border rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted">
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Event
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Pixel
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Source URL
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    CAPI
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody>
                {realtimeEvents.map((event) => (
                  <tr
                    key={event.id}
                    className="border-b border-border last:border-0 animate-[fadeIn_0.3s_ease-in]"
                  >
                    <td className="px-4 py-3">
                      <Badge variant={eventBadgeVariant(event.event_name)}>
                        {event.event_name}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-sm text-foreground">{event.pixel_name}</td>
                    <td
                      className="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate"
                      title={event.source_url}
                    >
                      {event.source_url || '-'}
                    </td>
                    <td className="px-4 py-3">
                      {event.forwarded_to_capi ? (
                        <Check className="h-4 w-4 text-emerald-600" />
                      ) : (
                        <X className="h-4 w-4 text-red-400" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {formatDistanceToNow(new Date(event.event_time), { addSuffix: true })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      ) : /* History mode table */
      historyQuery.isLoading ? (
        <div className="text-center py-12 text-muted-foreground">Loading events...</div>
      ) : historyEvents.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-border rounded-lg">
          <p className="text-muted-foreground">No events recorded yet</p>
          <p className="text-sm text-muted-foreground mt-1">
            Events will appear here once your sale pages start receiving traffic
          </p>
        </div>
      ) : (
        <>
          <div className="border border-border rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted">
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Event
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Pixel
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Source URL
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    CAPI
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody>
                {historyEvents.map((event) => (
                  <tr key={event.id} className="border-b border-border last:border-0">
                    <td className="px-4 py-3">
                      <Badge variant={eventBadgeVariant(event.event_name)}>
                        {event.event_name}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-sm text-foreground">
                      {pixelNameMap.get(event.pixel_id) ?? event.pixel_id}
                    </td>
                    <td
                      className="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate"
                      title={event.source_url}
                    >
                      {event.source_url || '-'}
                    </td>
                    <td className="px-4 py-3">
                      {event.forwarded_to_capi ? (
                        <Check className="h-4 w-4 text-emerald-600" />
                      ) : (
                        <X className="h-4 w-4 text-red-400" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {formatDistanceToNow(new Date(event.event_time), { addSuffix: true })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <p className="text-sm text-muted-foreground">
                Page {historyPage} of {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={historyPage <= 1}
                  onClick={() => setHistoryPage((p) => p - 1)}
                >
                  <ChevronLeft className="h-4 w-4" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={historyPage >= totalPages}
                  onClick={() => setHistoryPage((p) => p + 1)}
                >
                  Next
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
