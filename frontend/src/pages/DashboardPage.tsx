import { useState, useMemo } from 'react'
import { Link } from 'react-router'
import {
  Radio,
  Zap,
  Send,
  RotateCcw,
  TrendingUp,
  TrendingDown,
  Check,
  X,
  ArrowRight,
  Activity,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  useOverviewStats,
  useEventChart,
  useDashboardRecentEvents,
} from '@/hooks/use-analytics'
import { useQuota } from '@/hooks/use-billing'
import { usePixels } from '@/hooks/use-pixels'
import { useReplays } from '@/hooks/use-replays'
import { usePixelNameMap } from '@/hooks/use-pixel-name-map'
import { QueryErrorAlert } from '@/components/shared/QueryErrorAlert'
import { ReplayStatusBadge } from '@/components/shared/ReplayStatusBadge'
import { eventBadgeVariant, getEventColor } from '@/lib/event-utils'
import { timeAgo } from '@/lib/utils'
import type { RealtimeEvent } from '@/types'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

// --- Stat Card ---

interface StatCardProps {
  title: string
  value: number | string
  subtitle?: string
  icon: React.ReactNode
  trend?: { value: number; label: string } | null
  indicator?: 'green' | 'yellow' | 'red'
}

function StatCard({ title, value, subtitle, icon, trend, indicator }: StatCardProps) {
  return (
    <Card>
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-muted-foreground">{title}</p>
            <div className="flex items-center gap-2 mt-1">
              <p className="text-2xl font-bold text-foreground">{value}</p>
              {indicator && (
                <div
                  className={`h-2.5 w-2.5 rounded-full ${
                    indicator === 'green'
                      ? 'bg-emerald-500'
                      : indicator === 'yellow'
                        ? 'bg-amber-500'
                        : 'bg-red-500'
                  }`}
                />
              )}
            </div>
            {trend && (
              <p
                className={`text-xs mt-1 flex items-center gap-1 ${
                  trend.value > 0
                    ? 'text-emerald-600'
                    : trend.value < 0
                      ? 'text-red-500'
                      : 'text-muted-foreground'
                }`}
              >
                {trend.value > 0 ? (
                  <TrendingUp className="h-3 w-3" />
                ) : trend.value < 0 ? (
                  <TrendingDown className="h-3 w-3" />
                ) : null}
                {trend.value > 0 ? '+' : ''}
                {trend.value}% {trend.label}
              </p>
            )}
            {!trend && subtitle && (
              <p className="text-xs text-muted-foreground mt-1">{subtitle}</p>
            )}
          </div>
          <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center text-foreground">
            {icon}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// --- Event Volume Chart ---

const TIME_RANGES = [
  { label: '7d', days: 7 },
  { label: '14d', days: 14 },
  { label: '30d', days: 30 },
  { label: '90d', days: 90 },
] as const

function EventVolumeChart() {
  const [days, setDays] = useState(30)
  const { data: chartData } = useEventChart(days)

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">ปริมาณอีเวนต์</CardTitle>
          <div className="flex rounded-lg border border-border p-0.5 bg-secondary">
            {TIME_RANGES.map((range) => (
              <button
                key={range.days}
                onClick={() => setDays(range.days)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                  days === range.days
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {range.label}
              </button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {chartData && chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorEvents" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#18181B" stopOpacity={0.1} />
                  <stop offset="95%" stopColor="#18181B" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#E4E4E7" />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 12, fill: '#737373' }}
                tickFormatter={(val: string) => {
                  const d = new Date(val)
                  return `${d.getMonth() + 1}/${d.getDate()}`
                }}
              />
              <YAxis tick={{ fontSize: 12, fill: '#737373' }} />
              <Tooltip
                contentStyle={{
                  borderRadius: '8px',
                  border: '1px solid #e5e5e5',
                  fontSize: '12px',
                }}
                labelFormatter={(val) => {
                  const d = new Date(String(val))
                  return d.toLocaleDateString('th-TH', {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })
                }}
              />
              <Area
                type="monotone"
                dataKey="count"
                stroke="#18181B"
                fillOpacity={1}
                fill="url(#colorEvents)"
              />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="h-[300px] flex items-center justify-center text-muted-foreground">
            ยังไม่มีข้อมูลอีเวนต์
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// --- Recent Activity Feed ---

function RecentActivityFeed({ events }: { events: RealtimeEvent[] }) {
  const pixelNameMap = usePixelNameMap()

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">กิจกรรมล่าสุด</CardTitle>
          <Link
            to="/events"
            className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            ดูทั้งหมด <ArrowRight className="h-4 w-4 ml-1" />
          </Link>
        </div>
      </CardHeader>
      <CardContent>
        {events.length > 0 ? (
          <div className="space-y-3">
            {events.slice(0, 8).map((event) => (
              <div
                key={event.id}
                className="flex items-center justify-between py-2 border-b border-border last:border-0"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <Badge variant={eventBadgeVariant(event.event_name)} className="text-xs shrink-0">
                    {event.event_name}
                  </Badge>
                  <span className="text-sm text-foreground truncate">
                    {event.pixel_name || pixelNameMap.get(event.pixel_id) || 'ไม่ทราบ'}
                  </span>
                </div>
                <div className="flex items-center gap-2 shrink-0 ml-2">
                  {event.forwarded_to_capi ? (
                    <Check className="h-3.5 w-3.5 text-emerald-600" />
                  ) : (
                    <X className="h-3.5 w-3.5 text-red-400" />
                  )}
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {timeAgo(event.event_time)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            ยังไม่มีกิจกรรมล่าสุด
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// --- Pixel Status List ---

function PixelStatusList() {
  const { data: pixels } = usePixels()

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">สถานะพิกเซล</CardTitle>
          <Link
            to="/pixels"
            className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            จัดการ <ArrowRight className="h-4 w-4 ml-1" />
          </Link>
        </div>
      </CardHeader>
      <CardContent>
        {pixels && pixels.length > 0 ? (
          <div className="space-y-3">
            {pixels.map((pixel) => (
              <div
                key={pixel.id}
                className="flex items-center justify-between py-2 border-b border-border last:border-0"
              >
                <div className="flex items-center gap-2">
                  <div
                    className={`h-2 w-2 rounded-full ${
                      pixel.is_active ? 'bg-emerald-500' : 'bg-muted-foreground'
                    }`}
                  />
                  <span className="text-sm font-medium text-foreground">{pixel.name}</span>
                </div>
                <Badge variant={pixel.is_active ? 'success' : 'secondary'} className="text-xs">
                  {pixel.is_active ? 'ใช้งาน' : 'หยุดชั่วคราว'}
                </Badge>
              </div>
            ))}
          </div>
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            ยังไม่ได้ตั้งค่าพิกเซล
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// --- Top Event Types ---

function TopEventTypes({ events }: { events: RealtimeEvent[] }) {
  const eventTypeCounts = useMemo(() => {
    if (events.length === 0) return []
    const counts: Record<string, number> = {}
    for (const e of events) {
      counts[e.event_name] = (counts[e.event_name] || 0) + 1
    }
    const total = events.length
    return Object.entries(counts)
      .map(([name, count]) => ({
        name,
        count,
        percentage: Math.round((count / total) * 100),
      }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 5)
  }, [events])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">ประเภทอีเวนต์ยอดนิยม</CardTitle>
      </CardHeader>
      <CardContent>
        {eventTypeCounts.length > 0 ? (
          <div className="space-y-3">
            {eventTypeCounts.map((type) => (
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
          </div>
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            ยังไม่มีข้อมูลอีเวนต์
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// --- Recent Replays ---

function RecentReplays() {
  const { data: replays } = useReplays()
  const pixelNameMap = usePixelNameMap()

  const recentReplays = useMemo(() => {
    if (!replays) return []
    return [...replays]
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, 3)
  }, [replays])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">รีเพลย์ล่าสุด</CardTitle>
          <Link
            to="/replay"
            className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            ดูทั้งหมด <ArrowRight className="h-4 w-4 ml-1" />
          </Link>
        </div>
      </CardHeader>
      <CardContent>
        {recentReplays.length > 0 ? (
          <div className="space-y-3">
            {recentReplays.map((replay) => {
              const progress =
                replay.total_events > 0
                  ? Math.round((replay.replayed_events / replay.total_events) * 100)
                  : 0
              return (
                <div
                  key={replay.id}
                  className="py-2 border-b border-border last:border-0"
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm text-foreground">
                      {pixelNameMap.get(replay.source_pixel_id) || 'ต้นทาง'}{' '}
                      <span className="text-muted-foreground">→</span>{' '}
                      {pixelNameMap.get(replay.target_pixel_id) || 'ปลายทาง'}
                    </span>
                    <ReplayStatusBadge status={replay.status} className="text-xs" />
                  </div>
                  {(replay.status === 'running' || replay.status === 'completed') && (
                    <div className="flex items-center gap-2">
                      <div className="flex-1 h-1.5 bg-secondary rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full ${
                            replay.status === 'completed' ? 'bg-emerald-500' : 'bg-primary'
                          }`}
                          style={{ width: `${progress}%` }}
                        />
                      </div>
                      <span className="text-xs text-muted-foreground">{progress}%</span>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            ยังไม่มีรีเพลย์
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// --- Dashboard Page ---

export function DashboardPage() {
  const { data: stats, isError: statsError, error: statsErr, refetch: refetchStats } = useOverviewStats()
  const { data: recentEvents = [], isError: eventsError, error: eventsErr, refetch: refetchEvents } = useDashboardRecentEvents(100)
  const { data: quota } = useQuota()

  const capiRate = stats && stats.total_events > 0
    ? Math.round((stats.forwarded_events / stats.total_events) * 100)
    : 0

  const capiIndicator: 'green' | 'yellow' | 'red' =
    capiRate >= 90 ? 'green' : capiRate >= 70 ? 'yellow' : 'red'

  const eventsTrend = stats && stats.events_yesterday > 0
    ? {
        value: Math.round(
          ((stats.events_today - stats.events_yesterday) / stats.events_yesterday) * 100
        ),
        label: 'เทียบกับเมื่อวาน',
      }
    : null

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">แดชบอร์ด</h1>
          <p className="text-sm text-muted-foreground mt-1">
            ภาพรวมระบบติดตามพิกเซลของคุณ
          </p>
        </div>
      </div>

      {statsError && (
        <QueryErrorAlert error={statsErr} onRetry={refetchStats} className="mb-6" />
      )}

      {eventsError && (
        <QueryErrorAlert error={eventsErr} onRetry={refetchEvents} className="mb-6" />
      )}

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
        <StatCard
          title="พิกเซลที่ใช้งาน"
          value={`${stats?.active_pixels ?? 0}/${stats?.total_pixels ?? 0}`}
          subtitle={`ทั้งหมด ${stats?.total_pixels ?? 0}`}
          icon={<Radio className="h-5 w-5" />}
        />
        <StatCard
          title="อีเวนต์วันนี้"
          value={(stats?.events_today ?? 0).toLocaleString()}
          trend={eventsTrend}
          icon={<Zap className="h-5 w-5" />}
        />
        <StatCard
          title="อัตรา CAPI"
          value={stats ? `${capiRate}%` : '-'}
          subtitle="ส่งต่อไป Facebook แล้ว"
          indicator={stats ? capiIndicator : undefined}
          icon={<Send className="h-5 w-5" />}
        />
        <StatCard
          title="อีเวนต์สัปดาห์นี้"
          value={(stats?.events_this_week ?? 0).toLocaleString()}
          icon={<Activity className="h-5 w-5" />}
        />
        <StatCard
          title="รีเพลย์ที่ทำงาน"
          value={stats?.active_replays ?? 0}
          subtitle={`ทั้งหมด ${stats?.total_replays ?? 0}`}
          icon={<RotateCcw className="h-5 w-5" />}
        />
      </div>

      {/* Event Usage */}
      {quota && (
        <Card className="mb-6">
          <CardContent className="p-6">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm font-medium text-foreground">ปริมาณอีเวนต์รายเดือน</p>
              <p className="text-sm text-muted-foreground">
                {quota.events_used_this_month.toLocaleString()} / {quota.max_events_per_month.toLocaleString()}
              </p>
            </div>
            <div className="h-3 bg-secondary rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  quota.events_used_this_month / quota.max_events_per_month > 0.9
                    ? 'bg-red-500'
                    : quota.events_used_this_month / quota.max_events_per_month > 0.7
                      ? 'bg-amber-500'
                      : 'bg-primary'
                }`}
                style={{
                  width: `${Math.min((quota.events_used_this_month / quota.max_events_per_month) * 100, 100)}%`,
                }}
              />
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              ใช้ไปแล้ว {Math.round((quota.events_used_this_month / quota.max_events_per_month) * 100)}% ในเดือนนี้
            </p>
          </CardContent>
        </Card>
      )}

      {/* Event Volume Chart */}
      <div className="mb-6">
        <EventVolumeChart />
      </div>

      {/* Middle Row: Recent Activity + Pixel Status */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <RecentActivityFeed events={recentEvents} />
        <PixelStatusList />
      </div>

      {/* Bottom Row: Top Event Types + Recent Replays */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <TopEventTypes events={recentEvents} />
        <RecentReplays />
      </div>
    </div>
  )
}
