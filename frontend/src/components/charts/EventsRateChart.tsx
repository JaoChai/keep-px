import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

interface TimeBucket {
  label: string
  count: number
}

interface EventsRateChartProps {
  data: TimeBucket[]
  height?: number
}

export default function EventsRateChart({ data, height = 200 }: EventsRateChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data}>
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
  )
}
