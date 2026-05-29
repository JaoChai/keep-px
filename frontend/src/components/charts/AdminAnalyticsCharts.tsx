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
import { formatBaht } from '@/lib/utils'

interface GrowthPoint {
  date: string
  new_customers: number
  total_customers: number
}

interface RevenuePoint {
  date: string
  amount_satang: number
  purchase_count: number
}

interface PlanDistributionPoint {
  plan: string
  count: number
}

interface GrowthChartProps {
  data: GrowthPoint[]
  height?: number
}

export function GrowthChart({ data, height = 300 }: GrowthChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data}>
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
  )
}

interface RevenueChartProps {
  data: RevenuePoint[]
  height?: number
}

export function RevenueChart({ data, height = 300 }: RevenueChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#16a34a" stopOpacity={0.1} />
            <stop offset="95%" stopColor="#16a34a" stopOpacity={0} />
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
        <YAxis
          tick={{ fontSize: 12, fill: '#737373' }}
          tickFormatter={(val) => formatBaht(val)}
        />
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
          formatter={(value) => [formatBaht(value as number), 'รายได้ (THB)']}
        />
        <Area
          type="monotone"
          dataKey="amount_satang"
          stroke="#16a34a"
          fillOpacity={1}
          fill="url(#colorRevenue)"
          name="รายได้ (THB)"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}

interface PlanDistributionChartProps {
  data: PlanDistributionPoint[]
  height?: number
}

export function PlanDistributionChart({ data, height = 200 }: PlanDistributionChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} layout="vertical">
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
  )
}
