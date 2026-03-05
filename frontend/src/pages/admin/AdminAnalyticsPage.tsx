import { useState } from 'react'
import {
  Users,
  Zap,
  RotateCcw,
  DollarSign,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAdminAnalytics, useAdminGrowthChart } from '@/hooks/use-admin'
import { formatBaht, PLAN_LABELS } from '@/lib/utils'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from 'recharts'

const TIME_RANGES = [
  { label: '7d', days: 7 },
  { label: '14d', days: 14 },
  { label: '30d', days: 30 },
  { label: '90d', days: 90 },
] as const

interface StatCardProps {
  title: string
  value: string | number
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

export function AdminAnalyticsPage() {
  const [days, setDays] = useState(30)
  const { data: analytics } = useAdminAnalytics()
  const { data: growthData } = useAdminGrowthChart(days)

  const planDistribution = analytics
    ? Object.entries(analytics.customers_by_plan).map(([plan, count]) => ({
        plan: PLAN_LABELS[plan] ?? plan,
        count,
      }))
    : []

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-foreground">สถิติแพลตฟอร์ม</h1>
        <p className="text-sm text-muted-foreground mt-1">ภาพรวมข้อมูลระบบ</p>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          title="ลูกค้าทั้งหมด"
          value={analytics?.total_customers ?? 0}
          subtitle={`ใช้งาน ${analytics?.active_customers ?? 0} / ระงับ ${analytics?.suspended_customers ?? 0}`}
          icon={<Users className="h-5 w-5" />}
        />
        <StatCard
          title="อีเวนต์เดือนนี้"
          value={(analytics?.events_this_month ?? 0).toLocaleString()}
          subtitle={`วันนี้ ${(analytics?.events_today ?? 0).toLocaleString()}`}
          icon={<Zap className="h-5 w-5" />}
        />
        <StatCard
          title="รีเพลย์"
          value={analytics?.total_replays ?? 0}
          subtitle={`สำเร็จ ${analytics?.successful_replays ?? 0} / ล้มเหลว ${analytics?.failed_replays ?? 0}`}
          icon={<RotateCcw className="h-5 w-5" />}
        />
        <StatCard
          title="รายได้เดือนนี้"
          value={analytics ? `${formatBaht(analytics.revenue_this_month_thb)} THB` : '-'}
          subtitle={analytics ? `รวมทั้งหมด ${formatBaht(analytics.total_revenue_thb)} THB` : undefined}
          icon={<DollarSign className="h-5 w-5" />}
        />
      </div>

      {/* Growth Chart */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">การเติบโตของผู้ใช้</CardTitle>
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
          {growthData && growthData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={growthData}>
                <defs>
                  <linearGradient id="colorGrowth" x1="0" y1="0" x2="0" y2="1">
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
                  dataKey="total_customers"
                  stroke="#18181B"
                  fillOpacity={1}
                  fill="url(#colorGrowth)"
                  name="ผู้ใช้ทั้งหมด"
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-[300px] flex items-center justify-center text-muted-foreground">
              ยังไม่มีข้อมูล
            </div>
          )}
        </CardContent>
      </Card>

      {/* Plan Distribution */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">การกระจายตามแผน</CardTitle>
        </CardHeader>
        <CardContent>
          {planDistribution.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={planDistribution} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#E4E4E7" horizontal={false} />
                <XAxis type="number" tick={{ fontSize: 12, fill: '#737373' }} />
                <YAxis
                  type="category"
                  dataKey="plan"
                  tick={{ fontSize: 12, fill: '#737373' }}
                  width={80}
                />
                <Tooltip
                  contentStyle={{
                    borderRadius: '8px',
                    border: '1px solid #e5e5e5',
                    fontSize: '12px',
                  }}
                />
                <Bar dataKey="count" fill="#18181B" radius={[0, 4, 4, 0]} name="จำนวนลูกค้า" />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-[200px] flex items-center justify-center text-muted-foreground">
              ยังไม่มีข้อมูล
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
