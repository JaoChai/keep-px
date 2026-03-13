import {
  Zap,
  Crown,
  CreditCard,
  Loader2,
  ShieldCheck,
  Ticket,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { isUnlimited, timeAgo, daysUntil, PACK_TYPE_NAMES } from '@/lib/utils'
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
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-bold text-foreground">รีเพลย์</h3>
        <p className="text-sm text-muted-foreground mt-1">
          ส่ง events ไปยัง pixel ใหม่เมื่อ pixel เดิมโดนแบน
        </p>
      </div>

      {/* Active credits */}
      {credits.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-muted-foreground mb-3">เครดิตที่มีอยู่</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {credits.map((credit) => {
              const remaining = daysUntil(credit.expires_at)
              const expiringSoon = remaining > 0 && remaining <= 30
              const ratio = isUnlimited(credit.total_replays)
                ? 0.1
                : credit.total_replays > 0
                  ? credit.used_replays / credit.total_replays
                  : 0

              return (
                <Card key={credit.id} className={expiringSoon ? 'border-amber-400' : ''}>
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between mb-2">
                      <Badge variant="success">ใช้งาน</Badge>
                      <span className="text-xs text-muted-foreground">
                        หมดอายุ {timeAgo(credit.expires_at)}
                      </span>
                    </div>
                    {expiringSoon && (
                      <p className="text-xs text-amber-600 mb-2">หมดอายุใน {remaining} วัน</p>
                    )}
                    <p className="text-sm font-medium text-foreground mb-2">
                      {PACK_TYPE_NAMES[credit.pack_type] ?? credit.pack_type}
                    </p>
                    <div className="flex items-center justify-between text-sm mb-2">
                      <span className="text-muted-foreground">ใช้แล้ว</span>
                      <span className="font-semibold text-foreground">
                        {credit.used_replays} / {isUnlimited(credit.total_replays) ? 'ไม่จำกัด' : credit.total_replays}
                      </span>
                    </div>
                    <div className="h-1.5 bg-secondary rounded-full overflow-hidden">
                      <div
                        className="h-full bg-primary rounded-full"
                        style={{ width: `${Math.min(ratio * 100, 100)}%` }}
                      />
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </div>
      )}

      {credits.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="p-6 text-center">
            <Ticket className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
            <p className="text-sm text-muted-foreground">ยังไม่มีเครดิตรีเพลย์</p>
            <p className="text-xs text-muted-foreground">ซื้อด้านล่างเพื่อเริ่มต้น</p>
          </CardContent>
        </Card>
      )}

      {/* Purchase options */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {REPLAY_OPTIONS.map((opt) => (
          <Card
            key={opt.type}
            className={opt.popular ? 'border-primary ring-1 ring-primary relative' : ''}
          >
            {opt.popular && (
              <div className="bg-primary text-primary-foreground text-center text-xs font-medium py-1 rounded-t-lg">
                คุ้มค่า
              </div>
            )}
            <CardContent className="p-5">
              <div className="flex items-center gap-2 mb-3">
                <div className="h-9 w-9 rounded-lg bg-muted flex items-center justify-center">
                  <opt.icon className="h-5 w-5 text-foreground" />
                </div>
                <div>
                  <p className="font-semibold text-foreground">{opt.name}</p>
                  <p className="text-xs text-muted-foreground">{opt.mode}</p>
                </div>
              </div>

              <div className="mb-3">
                <span className="text-2xl font-bold text-foreground">
                  ฿{opt.price.toLocaleString('th-TH')}
                </span>
                {opt.type === 'replay_monthly' && (
                  <span className="text-sm text-muted-foreground">/เดือน</span>
                )}
              </div>

              <p className="text-sm text-muted-foreground mb-4">{opt.description}</p>

              <Button
                className="w-full"
                variant={opt.popular ? 'default' : 'outline'}
                onClick={() => onCheckout({ type: opt.type })}
                disabled={isPending}
              >
                {pendingCheckoutType === opt.type ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <CreditCard className="h-4 w-4" />
                )}
                {opt.type === 'replay_monthly' ? 'สมัครสมาชิก' : 'ซื้อเลย'}
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>

      <p className="text-xs text-muted-foreground text-center flex items-center justify-center gap-1">
        <ShieldCheck className="h-3.5 w-3.5" />
        ชำระเงินปลอดภัยผ่าน Stripe
      </p>
    </div>
  )
}
