import { useState, useEffect } from 'react'
import { Activity, Zap, CheckCircle, XCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { StatCard } from '@/components/shared/StatCard'
import { useAdminEvents, useAdminEventStats } from '@/hooks/use-admin'
import { timeAgo } from '@/lib/utils'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

export function AdminEventsPage() {
  const [customerID, setCustomerID] = useState('')
  const [pixelID, setPixelID] = useState('')
  const [eventName, setEventName] = useState('')
  const [debouncedEventName, setDebouncedEventName] = useState('')
  const [page, setPage] = useState(1)
  const [expandedRow, setExpandedRow] = useState<string | null>(null)
  const perPage = 20

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedEventName(eventName)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [eventName])

  const { data: stats } = useAdminEventStats()
  const { data, isLoading } = useAdminEvents(customerID, pixelID, debouncedEventName, page, perPage)

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-foreground">อีเวนต์</h1>
        <p className="text-sm text-muted-foreground mt-1">ตรวจสอบอีเวนต์ทั้งหมดในระบบ</p>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          title="อีเวนต์วันนี้"
          value={(stats?.total_today ?? 0).toLocaleString()}
          icon={<Zap className="h-5 w-5" />}
        />
        <StatCard
          title="ชั่วโมงนี้"
          value={(stats?.total_this_hour ?? 0).toLocaleString()}
          icon={<Activity className="h-5 w-5" />}
        />
        <StatCard
          title="CAPI Success Rate"
          value={stats ? `${stats.capi_success_rate.toFixed(1)}%` : '-'}
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <StatCard
          title="CAPI Failures"
          value={(stats?.capi_failure_count ?? 0).toLocaleString()}
          icon={<XCircle className="h-5 w-5" />}
        />
      </div>

      {/* Timeseries Chart */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-base">อีเวนต์ 24 ชั่วโมงล่าสุด</CardTitle>
        </CardHeader>
        <CardContent>
          {stats && stats.timeseries.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={stats.timeseries}>
                <defs>
                  <linearGradient id="colorEvents" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#18181B" stopOpacity={0.1} />
                    <stop offset="95%" stopColor="#18181B" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorCapiSuccess" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#16a34a" stopOpacity={0.1} />
                    <stop offset="95%" stopColor="#16a34a" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#E4E4E7" />
                <XAxis
                  dataKey="timestamp"
                  tick={{ fontSize: 12, fill: '#737373' }}
                  tickFormatter={(val: string) => {
                    const d = new Date(val)
                    return `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
                  }}
                />
                <YAxis tick={{ fontSize: 12, fill: '#737373' }} />
                <Tooltip
                  contentStyle={{ borderRadius: '8px', border: '1px solid #e5e5e5', fontSize: '12px' }}
                  labelFormatter={(val) => {
                    const d = new Date(String(val))
                    return d.toLocaleString('th-TH')
                  }}
                />
                <Area type="monotone" dataKey="event_count" stroke="#18181B" fillOpacity={1} fill="url(#colorEvents)" name="อีเวนต์ทั้งหมด" />
                <Area type="monotone" dataKey="capi_success" stroke="#16a34a" fillOpacity={1} fill="url(#colorCapiSuccess)" name="CAPI สำเร็จ" />
                <Area type="monotone" dataKey="capi_failure" stroke="#dc2626" fillOpacity={0} name="CAPI ล้มเหลว" />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-[300px] flex items-center justify-center text-muted-foreground">
              ยังไม่มีข้อมูล
            </div>
          )}
        </CardContent>
      </Card>

      {/* Filters */}
      <div className="flex gap-3 mb-4">
        <Input
          placeholder="Customer ID"
          value={customerID}
          onChange={(e) => { setCustomerID(e.target.value); setPage(1) }}
          className="max-w-[200px]"
        />
        <Input
          placeholder="Pixel ID"
          value={pixelID}
          onChange={(e) => { setPixelID(e.target.value); setPage(1) }}
          className="max-w-[200px]"
        />
        <Input
          placeholder="ชื่ออีเวนต์ (PageView, Purchase...)"
          value={eventName}
          onChange={(e) => setEventName(e.target.value)}
          className="max-w-xs"
        />
      </div>

      {/* Event Table */}
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">อีเวนต์</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">Pixel</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">CAPI</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">เวลา</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                      กำลังโหลด...
                    </td>
                  </tr>
                ) : data && data.data.length > 0 ? (
                  data.data.flatMap((ev) => {
                    const rows = [
                      <tr
                        key={ev.id}
                        onClick={() => setExpandedRow(expandedRow === ev.id ? null : ev.id)}
                        className="border-b border-border hover:bg-muted/50 cursor-pointer transition-colors"
                      >
                        <td className="px-4 py-3">
                          <Badge variant="secondary" className="text-xs">{ev.event_name}</Badge>
                        </td>
                        <td className="px-4 py-3 text-muted-foreground text-xs">{ev.pixel_name}</td>
                        <td className="px-4 py-3 text-foreground">{ev.customer_email}</td>
                        <td className="px-4 py-3">
                          {ev.forwarded_to_capi ? (
                            <Badge variant="success" className="text-xs">{ev.capi_response_code ?? 'OK'}</Badge>
                          ) : (
                            <Badge variant="secondary" className="text-xs">-</Badge>
                          )}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">{timeAgo(ev.event_time)}</td>
                      </tr>,
                    ]
                    if (expandedRow === ev.id) {
                      rows.push(
                        <tr key={`${ev.id}-detail`} className="bg-muted/30">
                          <td colSpan={5} className="px-4 py-3">
                            <pre className="text-xs text-muted-foreground overflow-auto max-h-48 whitespace-pre-wrap">
                              {JSON.stringify(ev.event_data, null, 2)}
                            </pre>
                          </td>
                        </tr>
                      )
                    }
                    return rows
                  })
                ) : (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                      ไม่พบอีเวนต์
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {data && data.total_pages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-muted-foreground">
            หน้า {data.page} จาก {data.total_pages}
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              ก่อนหน้า
            </Button>
            <Button variant="outline" size="sm" disabled={page >= data.total_pages} onClick={() => setPage((p) => p + 1)}>
              ถัดไป
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
