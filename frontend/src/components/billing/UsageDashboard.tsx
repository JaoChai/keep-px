import {
  Activity,
  RefreshCw,
  Clock,
  LayoutGrid,
  Radio,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { isUnlimited } from '@/lib/utils'
import type { CustomerQuota } from '@/types'

interface UsageDashboardProps {
  quota: CustomerQuota
}

function progressColor(ratio: number): string {
  if (ratio >= 0.9) return 'bg-red-500'
  if (ratio >= 0.7) return 'bg-amber-500'
  return 'bg-primary'
}

interface StatCardProps {
  icon: React.ElementType
  label: string
  current: string
  max?: string
  ratio?: number
}

function StatCard({ icon: Icon, label, current, max, ratio }: StatCardProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
        <span className="text-xs text-muted-foreground">{label}</span>
      </div>
      <p className="text-lg font-bold text-foreground">
        {current}
        {max && (
          <span className="text-sm font-normal text-muted-foreground"> / {max}</span>
        )}
      </p>
      {ratio !== undefined && (
        <div className="h-1.5 bg-secondary rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${progressColor(ratio)}`}
            style={{ width: `${Math.min(ratio * 100, 100)}%` }}
          />
        </div>
      )}
    </div>
  )
}

export function UsageDashboard({ quota }: UsageDashboardProps) {
  const eventsRatio = quota.max_events_per_month > 0
    ? quota.events_used_this_month / quota.max_events_per_month
    : 0

  return (
    <section>
      <h2 className="text-sm font-medium text-muted-foreground mb-3">บัญชีของคุณ</h2>
      <Card>
        <CardContent className="p-5">
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
            <StatCard
              icon={Activity}
              label="อีเวนต์เดือนนี้"
              current={quota.events_used_this_month.toLocaleString()}
              max={quota.max_events_per_month.toLocaleString()}
              ratio={eventsRatio}
            />
            <StatCard
              icon={RefreshCw}
              label="รีเพลย์คงเหลือ"
              current={
                isUnlimited(quota.remaining_replays)
                  ? 'ไม่จำกัด'
                  : quota.remaining_replays === 0
                    ? 'ไม่มี'
                    : String(quota.remaining_replays)
              }
            />
            <StatCard
              icon={Clock}
              label="เก็บข้อมูล"
              current={`${quota.retention_days} วัน`}
            />
            <StatCard
              icon={LayoutGrid}
              label="Sale Pages"
              current={`สูงสุด ${quota.max_sale_pages}`}
            />
            <StatCard
              icon={Radio}
              label="พิกเซล"
              current={`สูงสุด ${quota.max_pixels}`}
            />
          </div>
        </CardContent>
      </Card>
    </section>
  )
}
