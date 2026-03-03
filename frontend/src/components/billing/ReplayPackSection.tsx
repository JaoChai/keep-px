import {
  Zap,
  Package,
  Crown,
  CheckCircle2,
  CreditCard,
  Loader2,
  ShieldCheck,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { isUnlimited } from '@/lib/utils'

const REPLAY_PACKS = [
  {
    type: 'replay_1',
    name: 'Starter',
    price: 299,
    replays: 1,
    eventsPerReplay: 100_000,
    expiryDays: 90,
    pricePerReplay: 299,
    icon: Zap,
    popular: false,
  },
  {
    type: 'replay_3',
    name: 'Pro',
    price: 699,
    replays: 3,
    eventsPerReplay: 100_000,
    expiryDays: 180,
    pricePerReplay: 233,
    icon: Package,
    popular: true,
  },
  {
    type: 'replay_unlimited',
    name: 'Unlimited',
    price: 1490,
    replays: -1,
    eventsPerReplay: 100_000,
    expiryDays: 365,
    icon: Crown,
    popular: false,
  },
] as const

interface ReplayPackSectionProps {
  pendingCheckoutType: string | null
  isPending: boolean
  onCheckout: (params: { pack_type: string }) => void
}

export function ReplayPackSection({ pendingCheckoutType, isPending, onCheckout }: ReplayPackSectionProps) {
  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <h2 className="text-lg font-semibold text-foreground">แพ็กรีเพลย์</h2>
        <Badge variant="secondary">ซื้อครั้งเดียว</Badge>
      </div>

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
                  {pack.price.toLocaleString('th-TH')}
                </span>
                <span className="text-sm text-muted-foreground"> THB</span>
              </div>
              {'pricePerReplay' in pack && (
                <p className="text-xs text-muted-foreground mb-4">
                  {pack.pricePerReplay} ฿/รีเพลย์
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
    </section>
  )
}
