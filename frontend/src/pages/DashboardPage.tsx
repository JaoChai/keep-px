import { Radio, Zap, RotateCcw, Send } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useOverviewStats, useEventChart } from '@/hooks/use-analytics'
import {
  AreaChart,
  Area,
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

export function DashboardPage() {
  const { data: stats } = useOverviewStats()
  const { data: chartData } = useEventChart(30)

  return (
    <div>
      <h1 className="text-2xl font-bold text-foreground mb-6">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          title="Active Pixels"
          value={stats?.active_pixels ?? 0}
          subtitle={`${stats?.total_pixels ?? 0} total`}
          icon={<Radio className="h-5 w-5" />}
        />
        <StatCard
          title="Events Today"
          value={stats?.events_today ?? 0}
          subtitle={`${stats?.events_this_week ?? 0} this week`}
          icon={<Zap className="h-5 w-5" />}
        />
        <StatCard
          title="Total Events"
          value={(stats?.total_events ?? 0).toLocaleString()}
          icon={<Send className="h-5 w-5" />}
        />
        <StatCard
          title="Replays"
          value={stats?.total_replays ?? 0}
          subtitle={`${stats?.forwarded_events ?? 0} forwarded`}
          icon={<RotateCcw className="h-5 w-5" />}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Events (Last 30 Days)</CardTitle>
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
                <Tooltip />
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
              No event data yet
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
