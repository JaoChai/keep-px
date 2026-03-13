import {
  Activity,
  RefreshCw,
  Radio,
  Clock,
  ArrowUpRight,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { isUnlimited } from '@/lib/utils'
import type { CustomerQuota, ReplayCredit } from '@/types'

function progressColor(ratio: number): string {
  if (ratio >= 0.9) return 'bg-red-500'
  if (ratio >= 0.7) return 'bg-amber-500'
  return 'bg-primary'
}

interface QuotaBarProps {
  icon: React.ElementType
  label: string
  current: string
  max?: string
  ratio?: number
}

function QuotaBar({ icon: Icon, label, current, max, ratio }: QuotaBarProps) {
  return (
    <div className="space-y-1.5">
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

interface AccountStatusCardProps {
  quota: CustomerQuota
  credits: ReplayCredit[]
  onUpgrade: () => void
  onManageBilling: () => void
}

export function AccountStatusCard({
  quota,
  credits,
  onUpgrade,
  onManageBilling,
}: AccountStatusCardProps) {
  const isPaid = quota.pixel_slots > 0
  const monthlyPrice = quota.pixel_slots * 199

  const eventsRatio = quota.max_events_per_month > 0
    ? quota.events_used_this_month / quota.max_events_per_month
    : 0

  const totalCredits = credits.reduce(
    (sum, c) => sum + (isUnlimited(c.total_replays) ? 0 : c.total_replays - c.used_replays),
    0,
  )
  const hasUnlimited = credits.some((c) => isUnlimited(c.total_replays))

  return (
    <Card>
      <CardContent className="p-5 space-y-5">
        {/* Plan header */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-bold text-foreground">
                {isPaid
                  ? `${quota.pixel_slots} Pixel Slot${quota.pixel_slots > 1 ? 's' : ''}`
                  : 'Free'}
              </h2>
              <Badge variant="outline" className="text-xs">
                {isPaid ? `฿${monthlyPrice.toLocaleString('th-TH')}/เดือน` : 'ฟรี'}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">
              {isPaid ? 'สมาชิกรายเดือน' : '1 Pixel · 5K events · 7 วัน'}
            </p>
          </div>
          <Button
            variant={isPaid ? 'outline' : 'default'}
            size="sm"
            onClick={isPaid ? onManageBilling : onUpgrade}
          >
            {isPaid ? 'จัดการ' : 'อัปเกรด'}
            <ArrowUpRight className="h-3.5 w-3.5" />
          </Button>
        </div>

        {/* Quota grid */}
        <div className="grid grid-cols-3 gap-4">
          <QuotaBar
            icon={Activity}
            label="อีเวนต์เดือนนี้"
            current={quota.events_used_this_month.toLocaleString()}
            max={quota.max_events_per_month.toLocaleString()}
            ratio={eventsRatio}
          />
          <QuotaBar
            icon={Radio}
            label="พิกเซล"
            current={`สูงสุด ${quota.max_pixels}`}
          />
          <QuotaBar
            icon={RefreshCw}
            label="รีเพลย์คงเหลือ"
            current={
              hasUnlimited
                ? 'ไม่จำกัด'
                : isUnlimited(quota.remaining_replays)
                  ? 'ไม่จำกัด'
                  : quota.remaining_replays === 0 && totalCredits === 0
                    ? 'ไม่มี'
                    : String(quota.remaining_replays)
            }
          />
        </div>

        {/* Footer */}
        <div className="flex items-center gap-4 text-xs text-muted-foreground pt-1 border-t border-border">
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            เก็บข้อมูล {quota.retention_days} วัน
          </span>
          {credits.length > 0 && (
            <span>
              เครดิตรีเพลย์:{' '}
              {hasUnlimited ? 'ไม่จำกัด' : `${totalCredits} ครั้ง`}
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
