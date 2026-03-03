import {
  Activity,
  RefreshCw,
  Clock,
  LayoutGrid,
  Radio,
  Sparkles,
  Rocket,
  Shield,
  Crown,
  ArrowUpRight,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { isUnlimited, PLAN_LABELS } from '@/lib/utils'
import type { CustomerQuota, ReplayCredit } from '@/types'

const PLAN_ICONS: Record<string, React.ElementType> = {
  sandbox: Sparkles,
  launch: Rocket,
  shield: Shield,
  vault: Crown,
}

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
  currentPlan: string
  onUpgrade: () => void
  onManageBilling: () => void
}

export function AccountStatusCard({
  quota,
  credits,
  currentPlan,
  onUpgrade,
  onManageBilling,
}: AccountStatusCardProps) {
  const PlanIcon = PLAN_ICONS[currentPlan] ?? Sparkles
  const planLabel = PLAN_LABELS[currentPlan] ?? currentPlan
  const isPaid = currentPlan !== 'sandbox'

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
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center">
              <PlanIcon className="h-5 w-5 text-foreground" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-bold text-foreground">{planLabel}</h2>
                <Badge variant="outline" className="text-xs">
                  แผนปัจจุบัน
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground">
                {isPaid ? 'แผนรายเดือน' : 'ทดลองใช้ฟรี'}
              </p>
            </div>
          </div>
          <Button
            variant={isPaid ? 'outline' : 'default'}
            size="sm"
            onClick={isPaid ? onManageBilling : onUpgrade}
          >
            {isPaid ? 'จัดการแผน' : 'อัปเกรด'}
            <ArrowUpRight className="h-3.5 w-3.5" />
          </Button>
        </div>

        {/* Quota grid */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <QuotaBar
            icon={Activity}
            label="อีเวนต์เดือนนี้"
            current={quota.events_used_this_month.toLocaleString()}
            max={quota.max_events_per_month.toLocaleString()}
            ratio={eventsRatio}
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
          <QuotaBar
            icon={LayoutGrid}
            label="Sale Pages"
            current={`สูงสุด ${quota.max_sale_pages}`}
          />
          <QuotaBar
            icon={Radio}
            label="พิกเซล"
            current={`สูงสุด ${quota.max_pixels}`}
          />
        </div>

        {/* Footer info */}
        <div className="flex items-center gap-4 text-xs text-muted-foreground pt-1 border-t border-border">
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            เก็บข้อมูล {quota.retention_days} วัน
          </span>
          {credits.length > 0 && (
            <span>
              เครดิตรีเพลย์:{' '}
              {hasUnlimited ? 'ไม่จำกัด' : `${totalCredits} ครั้ง`}
              {' '}({credits.length} แพ็ก)
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
