import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

interface EventChartPoint {
  date: string
  count: number
}

interface DashboardEventVolumeChartProps {
  data: EventChartPoint[]
  height?: number
}

export default function DashboardEventVolumeChart({
  data,
  height = 300,
}: DashboardEventVolumeChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data}>
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
  )
}
