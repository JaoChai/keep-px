import {
  Zap,
  Package,
  Crown,
  CheckCircle2,
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

const REPLAY_PACKS = [
  {
    type: 'replay_1',
    name: 'Single',
    price: 499,
    replays: 1,
    eventsPerReplay: 100_000,
    expiryDays: 90,
    pricePerReplay: 499,
    icon: Zap,
    popular: false,
  },
  {
    type: 'replay_3',
    name: 'Triple',
    price: 999,
    replays: 3,
    eventsPerReplay: 100_000,
    expiryDays: 180,
    pricePerReplay: 333,
    icon: Package,
    popular: true,
  },
  {
    type: 'replay_unlimited',
    name: 'Unlimited',
    price: 2990,
    replays: -1,
    eventsPerReplay: 100_000,
    expiryDays: 365,
    icon: Crown,
    popular: false,
  },
] as const

interface ReplayTabProps {
  credits: ReplayCredit[]
  pendingCheckoutType: string | null
  isPending: boolean
  onCheckout: (params: { pack_type: string }) => void
}

export function ReplayTab({
  credits,
  pendingCheckoutType,
  isPending,
  onCheckout,
}: ReplayTabProps) {
  return (
    <div className="space-y-6">
      {/* Active credits */}
      {credits.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-muted-foreground mb-3">เครดิตที่มีอยู่</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {credits.map((credit) => {
              const remaining = daysUntil(credit.expires_at)
              const expiringSoon = remaining > 0 && remaining <= 30
              const ratio = isUnlimited(credit.total_replays)
                ? 0.1
                : credit.total_replays > 0
                  ? credit.used_replays / credit.total_replays
                  : 0

              return (
                <Card
                  key={credit.id}
                  className={expiringSoon ? 'border-amber-400' : ''}
                >
                  <CardContent className="p-5">
                    <div className="flex items-center justify-between mb-3">
                      <Badge variant="success">ใช้งาน</Badge>
                      <span className="text-xs text-muted-foreground">
                        หมดอายุ {timeAgo(credit.expires_at)}
                      </span>
                    </div>

                    {expiringSoon && (
                      <p className="text-xs text-amber-600 mb-2">
                        หมดอายุใน {remaining} วัน
                      </p>
                    )}

                    <p className="text-sm font-medium text-foreground mb-2">
                      {PACK_TYPE_NAMES[credit.pack_type] ?? credit.pack_type}
                    </p>

                    <div className="flex items-center justify-between text-sm mb-2">
                      <span className="text-muted-foreground">รีเพลย์ที่ใช้แล้ว</span>
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

                    <p className="text-xs text-muted-foreground mt-2">
                      สูงสุด {credit.max_events_per_replay.toLocaleString()} อีเวนต์ต่อรีเพลย์
                    </p>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </div>
      )}

      {credits.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="p-8 text-center">
            <Ticket className="h-8 w-8 text-muted-foreground mx-auto mb-3" />
            <p className="text-sm text-muted-foreground mb-1">
              ยังไม่มีเครดิตรีเพลย์
            </p>
            <p className="text-xs text-muted-foreground">
              ซื้อแพ็กด้านล่างเพื่อเริ่มต้น
            </p>
          </CardContent>
        </Card>
      )}

      {/* Packs for purchase */}
      <div>
        <h3 className="text-sm font-medium text-muted-foreground mb-3">แพ็กรีเพลย์</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {REPLAY_PACKS.map((pack) => (
            <Card
              key={pack.type}
              className={
                pack.popular
                  ? 'border-primary ring-1 ring-primary md:scale-[1.02] relative'
                  : ''
              }
            >
              {pack.popular && (
                <div className="bg-primary text-primary-foreground text-center text-xs font-medium py-1 rounded-t-lg">
                  ยอดนิยม
                </div>
              )}
              <CardContent className="p-6">
                <div className="flex items-center gap-2 mb-4">
                  <div className="h-9 w-9 rounded-lg bg-muted flex items-center justify-center">
                    <pack.icon className="h-5 w-5 text-foreground" />
                  </div>
                  <p className="font-semibold text-foreground">{pack.name}</p>
                </div>

                <div className="mb-1">
                  <span className="text-3xl font-bold text-foreground">
                    ฿{pack.price.toLocaleString('th-TH')}
                  </span>
                </div>
                {'pricePerReplay' in pack && (
                  <p className="text-xs text-muted-foreground mb-4">
                    ฿{pack.pricePerReplay}/รีเพลย์
                  </p>
                )}
                {!('pricePerReplay' in pack) && <div className="mb-4" />}

                <ul className="space-y-2 mb-5 text-sm text-muted-foreground">
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                    {isUnlimited(pack.replays) ? 'รีเพลย์ไม่จำกัด' : `${pack.replays} รีเพลย์`}
                  </li>
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                    สูงสุด {pack.eventsPerReplay.toLocaleString()} อีเวนต์/รีเพลย์
                  </li>
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                    ใช้ได้ {pack.expiryDays} วัน
                  </li>
                </ul>

                <Button
                  className="w-full"
                  variant={pack.popular ? 'default' : 'outline'}
                  onClick={() => onCheckout({ pack_type: pack.type })}
                  disabled={isPending}
                >
                  {pendingCheckoutType === pack.type ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <CreditCard className="h-4 w-4" />
                  )}
                  {pack.popular ? 'เลือกแพ็กนี้' : 'ซื้อเลย'}
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>

        <p className="text-xs text-muted-foreground text-center mt-3 flex items-center justify-center gap-1">
          <ShieldCheck className="h-3.5 w-3.5" />
          ชำระเงินปลอดภัยผ่าน Stripe
        </p>
      </div>
    </div>
  )
}
