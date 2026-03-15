import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip as ShadTooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatCard } from '@/components/shared/StatCard'
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
  ChevronDown,
  ChevronUp,
  RefreshCw,
  Activity,
  ScrollText,
  AlertCircle,
  ExternalLink,
  Globe,
  Monitor,
  CalendarDays,
} from 'lucide-react'
import { useRealtimeEvents } from '@/hooks/use-realtime-events'
import { useRealtimeStats } from '@/hooks/use-realtime-stats'
import { useEvents, useEventDetail, useCustomerEventTypes } from '@/hooks/use-events'
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
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
} from 'recharts'

function formatAbsoluteTime(date: string | Date): string {
  return new Date(date).toLocaleString('th-TH', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function SkeletonRows({ count = 5 }: { count?: number }) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <tr key={i} className="border-b border-border last:border-0">
          <td className="px-4 py-3"><Skeleton className="h-5 w-20" /></td>
          <td className="px-4 py-3"><Skeleton className="h-4 w-24" /></td>
          <td className="px-4 py-3"><Skeleton className="h-4 w-40" /></td>
          <td className="px-4 py-3"><Skeleton className="h-4 w-4" /></td>
          <td className="px-4 py-3"><Skeleton className="h-4 w-16" /></td>
        </tr>
      ))}
    </>
  )
}

export function EventsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const rawMode = searchParams.get('mode')
  const mode: 'live' | 'history' = rawMode === 'history' ? 'history' : 'live'
  const pixelId = searchParams.get('pixel_id') || null
  const eventNameFilter = searchParams.get('event_name') || null
  const fromFilter = searchParams.get('from') || null
  const toFilter = searchParams.get('to') || null

  const setMode = (m: 'live' | 'history') => {
    const params: Record<string, string> = { mode: m }
    if (pixelId) params.pixel_id = pixelId
    if (eventNameFilter) params.event_name = eventNameFilter
    if (fromFilter) params.from = fromFilter
    if (toFilter) params.to = toFilter
    setSearchParams(params)
  }

  const setPixelFilter = (id: string | null) => {
    const params: Record<string, string> = { mode }
    if (id) params.pixel_id = id
    if (eventNameFilter) params.event_name = eventNameFilter
    if (fromFilter) params.from = fromFilter
    if (toFilter) params.to = toFilter
    setSearchParams(params)
    setHistoryPage(1)
  }

  const setEventNameFilter = (name: string | null) => {
    const params: Record<string, string> = { mode }
    if (pixelId) params.pixel_id = pixelId
    if (name) params.event_name = name
    if (fromFilter) params.from = fromFilter
    if (toFilter) params.to = toFilter
    setSearchParams(params)
    setHistoryPage(1)
  }

  const setDateRange = (from: string | null, to: string | null) => {
    const params: Record<string, string> = { mode }
    if (pixelId) params.pixel_id = pixelId
    if (eventNameFilter) params.event_name = eventNameFilter
    if (from) params.from = from
    if (to) params.to = to
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
  const historyQuery = useEvents(historyPage, 50, pixelId, eventNameFilter, fromFilter, toFilter)
  const { data: overviewStats, isLoading: overviewLoading, isError: overviewError, error: overviewErr, refetch: refetchOverview } = useOverviewStats()
  const { data: eventTypes } = useCustomerEventTypes()
  const { data: pixels } = usePixels()
  const pixelNameMap = usePixelNameMap()
  const [showAllTypes, setShowAllTypes] = useState(false)
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null)
  const { data: eventDetail, isLoading: detailLoading } = useEventDetail(selectedEventId)

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

  const statsLoading = overviewLoading && !overviewStats
  const displayedTypes = showAllTypes ? eventTypeCounts : eventTypeCounts.slice(0, 6)
  const bufferFull = realtimeEvents.length >= 200

  return (
    <TooltipProvider>
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

        {/* Buffer warning */}
        {mode === 'live' && bufferFull && (
          <div className="flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 mb-4">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span>
              บัฟเฟอร์เต็ม — อีเวนต์เก่าจะถูกลบอัตโนมัติ กด{' '}
              <button onClick={clear} className="font-medium underline underline-offset-2">
                ล้าง
              </button>{' '}
              เพื่อรีเซ็ต
            </span>
          </div>
        )}

        {/* Filters */}
        <div className="flex gap-3 mb-4">
          <Select
            value={pixelId ?? 'all'}
            onValueChange={(v) => setPixelFilter(v === 'all' ? null : v)}
          >
            <SelectTrigger className="w-[220px]">
              <SelectValue placeholder="พิกเซลทั้งหมด" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">พิกเซลทั้งหมด</SelectItem>
              {pixels?.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {mode === 'history' && (
            <>
              <Select
                value={eventNameFilter ?? 'all'}
                onValueChange={(v) => setEventNameFilter(v === 'all' ? null : v)}
              >
                <SelectTrigger className="w-[200px]">
                  <SelectValue placeholder="อีเวนต์ทั้งหมด" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">อีเวนต์ทั้งหมด</SelectItem>
                  {eventTypes?.map((name) => (
                    <SelectItem key={name} value={name}>
                      {name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Popover>
                <PopoverTrigger className="inline-flex items-center justify-center gap-1.5 rounded-md border border-input bg-background px-3 h-9 text-sm font-medium shadow-xs hover:bg-accent hover:text-accent-foreground">
                  <CalendarDays className="h-4 w-4" />
                  {fromFilter || toFilter ? (
                    <span className="text-xs">
                      {fromFilter ? new Date(fromFilter).toLocaleDateString('th-TH') : '...'} — {toFilter ? new Date(toFilter).toLocaleDateString('th-TH') : '...'}
                    </span>
                  ) : (
                    <span className="text-sm">ช่วงวันที่</span>
                  )}
                </PopoverTrigger>
                <PopoverContent className="w-auto p-4" align="start">
                  <div className="space-y-3">
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">จาก</label>
                      <Input
                        type="datetime-local"
                        className="mt-1"
                        value={fromFilter ? fromFilter.slice(0, 16) : ''}
                        onChange={(e) => {
                          const val = e.target.value
                          setDateRange(val ? new Date(val).toISOString() : null, toFilter)
                        }}
                      />
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">ถึง</label>
                      <Input
                        type="datetime-local"
                        className="mt-1"
                        value={toFilter ? toFilter.slice(0, 16) : ''}
                        onChange={(e) => {
                          const val = e.target.value
                          setDateRange(fromFilter, val ? new Date(val).toISOString() : null)
                        }}
                      />
                    </div>
                    {(fromFilter || toFilter) && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="w-full text-xs"
                        onClick={() => setDateRange(null, null)}
                      >
                        ล้างช่วงวันที่
                      </Button>
                    )}
                  </div>
                </PopoverContent>
              </Popover>
            </>
          )}
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <StatCard
            title="อีเวนต์วันนี้"
            value={overviewStats?.events_today ?? '-'}
            subtitle="รวมวันนี้"
            icon={<Clock className="h-5 w-5" />}
            loading={statsLoading}
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
            loading={statsLoading}
          />
          <StatCard
            title="อัตรา CAPI"
            value={capiRate}
            subtitle="ส่งต่อไป Facebook แล้ว"
            icon={<CheckCircle className="h-5 w-5" />}
            loading={statsLoading}
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
                    <RechartsTooltip
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
                  {displayedTypes.map((type) => (
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
                  {eventTypeCounts.length > 6 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="w-full text-xs"
                      onClick={() => setShowAllTypes(!showAllTypes)}
                    >
                      {showAllTypes ? (
                        <>
                          <ChevronUp className="h-3 w-3 mr-1" />
                          แสดงน้อยลง
                        </>
                      ) : (
                        <>
                          <ChevronDown className="h-3 w-3 mr-1" />
                          แสดงเพิ่มเติม ({eventTypeCounts.length - 6})
                        </>
                      )}
                    </Button>
                  )}
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
                      onClick={() => setSelectedEventId(event.id)}
                      className="border-b border-border last:border-0 animate-[fadeIn_0.3s_ease-in] cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <td className="px-4 py-3">
                        <Badge variant={eventBadgeVariant(event.event_name)}>
                          {event.event_name}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-sm text-foreground">{event.pixel_name}</td>
                      <td className="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate">
                        {event.source_url ? (
                          <ShadTooltip>
                            <TooltipTrigger asChild>
                              <span className="truncate block">{event.source_url}</span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-md break-all">
                              {event.source_url}
                            </TooltipContent>
                          </ShadTooltip>
                        ) : (
                          '-'
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {event.forwarded_to_capi ? (
                          <Check className="h-4 w-4 text-emerald-600" />
                        ) : (
                          <X className="h-4 w-4 text-red-400" />
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        <ShadTooltip>
                          <TooltipTrigger asChild>
                            <span>{timeAgo(event.event_time)}</span>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            {formatAbsoluteTime(event.event_time)}
                          </TooltipContent>
                        </ShadTooltip>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        ) : /* History mode table */
        historyQuery.isLoading ? (
          <div className="border border-border rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted">
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">อีเวนต์</th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">พิกเซล</th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">URL ต้นทาง</th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">CAPI</th>
                  <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">เวลา</th>
                </tr>
              </thead>
              <tbody>
                <SkeletonRows />
              </tbody>
            </table>
          </div>
        ) : historyEvents.length === 0 ? (
          <div className="text-center py-12 border border-dashed border-border rounded-lg">
            <p className="text-muted-foreground">ยังไม่มีอีเวนต์ที่บันทึก</p>
            <p className="text-sm text-muted-foreground mt-1">
              อีเวนต์จะปรากฏที่นี่เมื่อเซลเพจของคุณเริ่มได้รับการเข้าชม
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
                    <tr
                      key={event.id}
                      onClick={() => setSelectedEventId(event.id)}
                      className="border-b border-border last:border-0 cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <td className="px-4 py-3">
                        <Badge variant={eventBadgeVariant(event.event_name)}>
                          {event.event_name}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-sm text-foreground">
                        {pixelNameMap.get(event.pixel_id) ?? event.pixel_id}
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate">
                        {event.source_url ? (
                          <ShadTooltip>
                            <TooltipTrigger asChild>
                              <span className="truncate block">{event.source_url}</span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-md break-all">
                              {event.source_url}
                            </TooltipContent>
                          </ShadTooltip>
                        ) : (
                          '-'
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {event.forwarded_to_capi ? (
                          <Check className="h-4 w-4 text-emerald-600" />
                        ) : (
                          <X className="h-4 w-4 text-red-400" />
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        <ShadTooltip>
                          <TooltipTrigger asChild>
                            <span>{timeAgo(event.event_time)}</span>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            {formatAbsoluteTime(event.event_time)}
                          </TooltipContent>
                        </ShadTooltip>
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

        {/* Event Detail Sheet */}
        <Sheet open={!!selectedEventId} onOpenChange={(open) => { if (!open) setSelectedEventId(null) }}>
          <SheetContent className="sm:max-w-lg overflow-y-auto">
            <SheetHeader>
              <SheetTitle className="flex items-center gap-2">
                {eventDetail && (
                  <>
                    <Badge variant={eventBadgeVariant(eventDetail.event_name)}>
                      {eventDetail.event_name}
                    </Badge>
                    <span className="text-sm font-normal text-muted-foreground">
                      {pixelNameMap.get(eventDetail.pixel_id) ?? eventDetail.pixel_id}
                    </span>
                  </>
                )}
              </SheetTitle>
              <SheetDescription>รายละเอียดอีเวนต์</SheetDescription>
            </SheetHeader>

            {detailLoading ? (
              <div className="space-y-4 mt-6">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-32 w-full" />
                <Skeleton className="h-32 w-full" />
              </div>
            ) : eventDetail ? (
              <div className="space-y-5 mt-6">
                {/* Event Time */}
                <div className="flex items-start gap-3">
                  <Clock className="h-4 w-4 mt-0.5 text-muted-foreground shrink-0" />
                  <div>
                    <p className="text-sm font-medium">เวลา</p>
                    <p className="text-sm text-muted-foreground">
                      {formatAbsoluteTime(eventDetail.event_time)} ({timeAgo(eventDetail.event_time)})
                    </p>
                  </div>
                </div>

                {/* Source URL */}
                {eventDetail.source_url && (
                  <div className="flex items-start gap-3">
                    <Globe className="h-4 w-4 mt-0.5 text-muted-foreground shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium">URL ต้นทาง</p>
                      <a
                        href={eventDetail.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-blue-600 hover:underline break-all inline-flex items-center gap-1"
                      >
                        {eventDetail.source_url}
                        <ExternalLink className="h-3 w-3 shrink-0" />
                      </a>
                    </div>
                  </div>
                )}

                {/* CAPI Status */}
                <div className="flex items-start gap-3">
                  <Monitor className="h-4 w-4 mt-0.5 text-muted-foreground shrink-0" />
                  <div>
                    <p className="text-sm font-medium">CAPI</p>
                    <div className="flex items-center gap-2 mt-0.5">
                      {eventDetail.forwarded_to_capi ? (
                        <>
                          <Badge variant="success" className="text-xs">ส่งแล้ว</Badge>
                          {eventDetail.capi_response_code && (
                            <span className="text-xs text-muted-foreground">
                              Response: {eventDetail.capi_response_code}
                            </span>
                          )}
                        </>
                      ) : (
                        <Badge variant="secondary" className="text-xs">ยังไม่ได้ส่ง</Badge>
                      )}
                    </div>
                  </div>
                </div>

                {/* Event Data */}
                <div>
                  <p className="text-sm font-medium mb-2">Event Data</p>
                  <ScrollArea className="h-[200px] rounded-md border border-border">
                    <pre className="text-xs p-3 text-muted-foreground whitespace-pre-wrap break-all">
                      {JSON.stringify(eventDetail.event_data, null, 2)}
                    </pre>
                  </ScrollArea>
                </div>

                {/* User Data */}
                {eventDetail.user_data && Object.keys(eventDetail.user_data).length > 0 && (
                  <div>
                    <p className="text-sm font-medium mb-2">User Data</p>
                    <ScrollArea className="h-[200px] rounded-md border border-border">
                      <pre className="text-xs p-3 text-muted-foreground whitespace-pre-wrap break-all">
                        {JSON.stringify(eventDetail.user_data, null, 2)}
                      </pre>
                    </ScrollArea>
                  </div>
                )}
              </div>
            ) : null}
          </SheetContent>
        </Sheet>
      </div>
    </TooltipProvider>
  )
}
