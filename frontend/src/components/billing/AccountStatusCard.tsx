import {
  Activity,
  RefreshCw,
  Radio,
  Clock,
  ArrowUpRight,
  Crown,
  ExternalLink,
  Ticket,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { isUnlimited } from '@/lib/utils'
import type { CustomerQuota, ReplayCredit } from '@/types'

interface AccountStatusCardProps {
  quota: CustomerQuota
  credits: ReplayCredit[]
  onUpgrade: () => void
  onManageBilling: () => void
  isPortalPending?: boolean
}

export function AccountStatusCard({
  quota,
  credits,
  onUpgrade,
  onManageBilling,
  isPortalPending,
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

  const PlanIcon = isPaid ? Crown : Radio

  return (
    <Card className="bg-gradient-to-br from-zinc-50 via-white to-zinc-50 border-zinc-200">
      <CardContent className="p-5 space-y-5">
        {/* Plan header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
              <PlanIcon className="h-5 w-5 text-primary" />
            </div>
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
          </div>
          {isPaid ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={onManageBilling}
              disabled={isPortalPending}
            >
              <ExternalLink className="h-3.5 w-3.5" />
              {isPortalPending ? 'กำลังเปิด...' : 'จัดการการชำระเงิน'}
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={onUpgrade}
            >
              อัปเกรด
              <ArrowUpRight className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>

        {/* Metric sub-cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div className="rounded-lg bg-white border border-border p-4">
            <div className="flex items-center gap-1.5 mb-1">
              <Activity className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-xs text-muted-foreground">อีเวนต์เดือนนี้</span>
            </div>
            <p className="text-2xl font-bold text-foreground">
              {quota.events_used_this_month.toLocaleString()}
              <span className="text-sm font-normal text-muted-foreground"> / {quota.max_events_per_month.toLocaleString()}</span>
            </p>
            {eventsRatio > 0 && (
              <div className="h-1.5 bg-secondary rounded-full overflow-hidden mt-2">
                <div
                  className={`h-full rounded-full transition-all ${eventsRatio >= 0.9 ? 'bg-red-500' : eventsRatio >= 0.7 ? 'bg-amber-500' : 'bg-primary'}`}
                  style={{ width: `${Math.min(eventsRatio * 100, 100)}%` }}
                />
              </div>
            )}
          </div>

          <div className="rounded-lg bg-white border border-border p-4">
            <div className="flex items-center gap-1.5 mb-1">
              <Radio className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-xs text-muted-foreground">พิกเซล</span>
            </div>
            <p className="text-2xl font-bold text-foreground">
              สูงสุด {quota.max_pixels}
            </p>
          </div>

          <div className="rounded-lg bg-white border border-border p-4">
            <div className="flex items-center gap-1.5 mb-1">
              <RefreshCw className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-xs text-muted-foreground">รีเพลย์คงเหลือ</span>
            </div>
            <p className="text-2xl font-bold text-foreground">
              {hasUnlimited
                ? 'ไม่จำกัด'
                : isUnlimited(quota.remaining_replays)
                  ? 'ไม่จำกัด'
                  : quota.remaining_replays === 0 && totalCredits === 0
                    ? 'ไม่มี'
                    : String(quota.remaining_replays)}
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center gap-4 text-xs text-muted-foreground pt-1 border-t border-border">
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            เก็บข้อมูล {quota.retention_days} วัน
          </span>
          {credits.length > 0 && (
            <span className="flex items-center gap-1">
              <Ticket className="h-3 w-3" />
              เครดิตรีเพลย์:{' '}
              {hasUnlimited ? 'ไม่จำกัด' : `${totalCredits} ครั้ง`}
            </span>
          )}
          {!isPaid && (
            <span className="ml-auto text-primary font-medium">
              อัปเกรดเพื่อปลดล็อกทุกฟีเจอร์
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
