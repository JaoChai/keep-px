import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

interface TimeseriesPoint {
  timestamp: string
  event_count: number
  capi_success: number
  capi_failure: number
}

interface AdminEventsTimeseriesChartProps {
  data: TimeseriesPoint[]
  height?: number
}

export default function AdminEventsTimeseriesChart({
  data,
  height = 300,
}: AdminEventsTimeseriesChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data}>
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
  )
}
