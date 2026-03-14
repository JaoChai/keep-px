import {
  Zap,
  Crown,
  Loader2,
  ShieldCheck,
  RefreshCw,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn, isUnlimited, timeAgo, daysUntil, PACK_TYPE_NAMES } from '@/lib/utils'
import type { ReplayCredit } from '@/types'

const REPLAY_OPTIONS = [
  {
    type: 'replay_single',
    name: 'ครั้งเดียว',
    price: 299,
    description: '1 รีเพลย์ · 90 วัน · สูงสุด 100K events',
    icon: Zap,
    mode: 'ซื้อขาด',
    popular: false,
  },
  {
    type: 'replay_monthly',
    name: 'ไม่จำกัด',
    price: 1990,
    description: 'รีเพลย์ไม่จำกัดตลอดรอบบิล',
    icon: Crown,
    mode: 'รายเดือน',
    popular: true,
  },
] as const

interface ReplaySectionProps {
  credits: ReplayCredit[]
  pendingCheckoutType: string | null
  isPending: boolean
  onCheckout: (params: { type: string }) => void
}

export function ReplaySection({
  credits,
  pendingCheckoutType,
  isPending,
  onCheckout,
}: ReplaySectionProps) {
  return (
    <Card className="h-full flex flex-col">
      <CardContent className="p-5 flex flex-col flex-1">
        {/* Header with icon */}
        <div className="flex items-center gap-3 mb-4">
          <div className="h-9 w-9 rounded-lg bg-primary/5 flex items-center justify-center">
            <RefreshCw className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-foreground">รีเพลย์</h3>
            <p className="text-xs text-muted-foreground">ส่ง events ไปยัง pixel ใหม่</p>
          </div>
        </div>

        {/* Active credits — compact rows */}
        {credits.length > 0 && (
          <div className="space-y-2 mb-4">
            <h4 className="text-xs font-medium text-muted-foreground">เครดิตที่มีอยู่</h4>
            {credits.map((credit) => {
              const remaining = daysUntil(credit.expires_at)
              const expiringSoon = remaining > 0 && remaining <= 30

              return (
                <div
                  key={credit.id}
                  className={cn(
                    'rounded-lg p-2.5 flex items-center justify-between',
                    expiringSoon
                      ? 'bg-amber-50 border border-amber-200'
                      : 'bg-secondary/50',
                  )}
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="success" className="text-[10px] px-1.5 py-0">ใช้งาน</Badge>
                    <span className="text-sm font-medium text-foreground">
                      {PACK_TYPE_NAMES[credit.pack_type] ?? credit.pack_type}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span>
                      {credit.used_replays}/{isUnlimited(credit.total_replays) ? '∞' : credit.total_replays}
                    </span>
                    <span>
                      {expiringSoon ? `${remaining}วัน` : timeAgo(credit.expires_at)}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        )}

        {credits.length === 0 && (
          <p className="text-sm text-muted-foreground mb-4 flex items-center gap-1.5">
            <RefreshCw className="h-3.5 w-3.5" />
            ยังไม่มีเครดิตรีเพลย์ — ซื้อด้านล่าง
          </p>
        )}

        {/* Purchase options — stacked horizontal rows */}
        <div className="space-y-2 mb-4">
          {REPLAY_OPTIONS.map((opt) => (
            <div
              key={opt.type}
              className={cn(
                'rounded-lg border p-3 flex items-center justify-between',
                opt.popular && 'border-primary bg-primary/[0.02]',
              )}
            >
              <div className="flex items-center gap-2.5">
                <div className="h-8 w-8 rounded-lg bg-muted flex items-center justify-center shrink-0">
                  <opt.icon className="h-4 w-4 text-foreground" />
                </div>
                <div>
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm font-semibold text-foreground">{opt.name}</span>
                    {opt.popular && (
                      <Badge variant="default" className="text-[10px] px-1.5 py-0">คุ้มค่า</Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">{opt.description}</p>
                </div>
              </div>
              <div className="flex items-center gap-2.5 shrink-0 ml-3">
                <div className="text-right">
                  <span className="text-sm font-bold text-foreground">
                    ฿{opt.price.toLocaleString('th-TH')}
                  </span>
                  {opt.type === 'replay_monthly' && (
                    <span className="text-xs text-muted-foreground">/เดือน</span>
                  )}
                </div>
                <Button
                  size="sm"
                  variant={opt.popular ? 'default' : 'outline'}
                  onClick={() => onCheckout({ type: opt.type })}
                  disabled={isPending}
                  className="h-8 px-3"
                >
                  {pendingCheckoutType === opt.type ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    'ซื้อ'
                  )}
                </Button>
              </div>
            </div>
          ))}
        </div>

        {/* Stripe badge — footer */}
        <div className="mt-auto pt-2">
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <ShieldCheck className="h-3.5 w-3.5" />
            ชำระเงินปลอดภัยผ่าน Stripe
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
