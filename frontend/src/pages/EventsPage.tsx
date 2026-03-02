import { useState, useEffect } from 'react'
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
import { usePixelNameMap } from '@/hooks/use-pixel-name-map'
import { QueryErrorAlert } from '@/components/shared/QueryErrorAlert'
import { eventBadgeVariant, getEventColor } from '@/lib/event-utils'
import { timeAgo } from '@/lib/utils'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

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
  const { data: overviewStats, isError: overviewError, error: overviewErr, refetch: refetchOverview } = useOverviewStats()
  const { data: pixels } = usePixels()
  const pixelNameMap = usePixelNameMap()

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
            <h1 className="text-2xl font-bold text-foreground">อีเวนต์</h1>
            <p className="text-sm text-muted-foreground mt-1">
              {mode === 'live'
                ? `${realtimeEvents.length} อีเวนต์${stats.eventsPerMinute > 0 ? ` (${stats.eventsPerMinute}/นาที)` : ''}`
                : historyQuery.data
                  ? `ทั้งหมด ${historyQuery.data.total} อีเวนต์`
                  : 'กำลังโหลดอีเวนต์...'}
            </p>
          </div>
          {mode === 'live' && (
            <Badge variant={isLive ? 'success' : 'warning'}>
              {isLive ? 'สด' : 'หยุดชั่วคราว'}
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
              สด
            </Button>
            <Button
              variant={mode === 'history' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setMode('history')}
              className={mode === 'history' ? '' : 'text-muted-foreground'}
            >
              <ScrollText className="h-4 w-4 mr-1" />
              ประวัติ
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
                {isPaused ? 'ดำเนินต่อ' : 'หยุด'}
              </Button>
              <Button variant="outline" size="sm" onClick={clear}>
                <Trash2 className="h-4 w-4 mr-1" />
                ล้าง
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
                รีเฟรช
              </Button>
            </div>
          )}
        </div>
      </div>

      {overviewError && (
        <QueryErrorAlert error={overviewErr} onRetry={refetchOverview} className="mb-4" />
      )}

      {mode === 'history' && historyQuery.isError && (
        <QueryErrorAlert error={historyQuery.error} onRetry={historyQuery.refetch} className="mb-4" />
      )}

      {/* Pixel filter */}
      <div className="mb-4">
        <select
          className="border border-border rounded-lg px-3 py-2 text-sm bg-background text-foreground"
          value={pixelId ?? ''}
          onChange={(e) => setPixelFilter(e.target.value || null)}
        >
          <option value="">พิกเซลทั้งหมด</option>
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
          title="อีเวนต์วันนี้"
          value={overviewStats?.events_today ?? '-'}
          subtitle="รวมวันนี้"
          icon={<Clock className="h-5 w-5" />}
        />
        <StatCard
          title="อีเวนต์ทั้งหมด"
          value={
            overviewStats?.total_events != null
              ? overviewStats.total_events.toLocaleString()
              : '-'
          }
          subtitle="ตลอดทั้งหมด"
          icon={<Hash className="h-5 w-5" />}
        />
        <StatCard
          title="อัตรา CAPI"
          value={capiRate}
          subtitle="ส่งต่อไป Facebook แล้ว"
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <StatCard
          title="อีเวนต์/นาที"
          value={mode === 'live' ? stats.eventsPerMinute : '-'}
          subtitle={mode === 'live' ? '60 วินาทีล่าสุด' : 'เฉพาะโหมดสด'}
          icon={<Zap className="h-5 w-5" />}
        />
      </div>

      {/* Charts (Live mode only) */}
      {mode === 'live' && realtimeEvents.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
          {/* Event Rate Chart */}
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-base">อัตราอีเวนต์ (5 นาที)</CardTitle>
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
              <CardTitle className="text-base">ประเภทอีเวนต์</CardTitle>
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
                  <p className="text-sm text-muted-foreground text-center py-4">ไม่มีข้อมูล</p>
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
                <p className="text-muted-foreground">กำลังโหลดอีเวนต์ล่าสุด...</p>
              </>
            ) : (
              <>
                <div className="animate-pulse mb-3">
                  <div className="inline-block h-3 w-3 rounded-full bg-emerald-400" />
                </div>
                <p className="text-muted-foreground">รอรับอีเวนต์...</p>
                <p className="text-sm text-muted-foreground mt-1">
                  อีเวนต์ใหม่จะปรากฏที่นี่แบบเรียลไทม์
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
                    อีเวนต์
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    พิกเซล
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    URL ต้นทาง
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    CAPI
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    เวลา
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
                      {timeAgo(event.event_time)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      ) : /* History mode table */
      historyQuery.isLoading ? (
        <div className="text-center py-12 text-muted-foreground">กำลังโหลดอีเวนต์...</div>
      ) : historyEvents.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-border rounded-lg">
          <p className="text-muted-foreground">ยังไม่มีอีเวนต์ที่บันทึก</p>
          <p className="text-sm text-muted-foreground mt-1">
            อีเวนต์จะปรากฏที่นี่เมื่อหน้าขายของคุณเริ่มได้รับการเข้าชม
          </p>
        </div>
      ) : (
        <>
          <div className="border border-border rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted">
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    อีเวนต์
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    พิกเซล
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    URL ต้นทาง
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    CAPI
                  </th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">
                    เวลา
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
                      {timeAgo(event.event_time)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <p className="text-sm text-muted-foreground">
                หน้า {historyPage} จาก {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={historyPage <= 1}
                  onClick={() => setHistoryPage((p) => p - 1)}
                >
                  <ChevronLeft className="h-4 w-4" />
                  ก่อนหน้า
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={historyPage >= totalPages}
                  onClick={() => setHistoryPage((p) => p + 1)}
                >
                  ถัดไป
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
