import { useEffect } from 'react'
import { useSearchParams } from 'react-router'
import {
  CreditCard,
  Package,
  Zap,
  Crown,
  CheckCircle2,
  Loader2,
  ExternalLink,
  ShieldCheck,
  Database,
  CalendarDays,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { useBillingOverview, useQuota, useCreateCheckout, useCreatePortalSession } from '@/hooks/use-billing'
import { formatDistanceToNow } from 'date-fns'

const REPLAY_PACKS = [
  {
    type: 'replay_1',
    name: 'Starter',
    price: 299,
    replays: 1,
    eventsPerReplay: 100_000,
    description: 'เหมาะสำหรับการย้ายพิกเซลครั้งเดียว',
    icon: Zap,
    popular: false,
  },
  {
    type: 'replay_3',
    name: 'Pro',
    price: 699,
    replays: 3,
    eventsPerReplay: 100_000,
    description: 'สำหรับการเปลี่ยนพิกเซลบ่อย',
    icon: Package,
    popular: true,
  },
  {
    type: 'replay_unlimited',
    name: 'Unlimited',
    price: 1490,
    replays: -1,
    eventsPerReplay: 100_000,
    description: 'รีเพลย์ไม่จำกัดเป็นเวลา 365 วัน',
    icon: Crown,
    popular: false,
  },
] as const

const ADDONS = [
  {
    type: 'retention_180',
    name: 'Retention 180d',
    price: 390,
    description: 'เก็บข้อมูลอีเวนต์ 180 วัน แทนที่จะเป็น 60 วัน',
    icon: Database,
  },
  {
    type: 'retention_365',
    name: 'Retention 365d',
    price: 690,
    description: 'เก็บข้อมูลอีเวนต์ครบ 1 ปี',
    icon: CalendarDays,
  },
  {
    type: 'events_1m',
    name: 'Events 1M',
    price: 490,
    description: 'เพิ่มขีดจำกัดอีเวนต์รายเดือนเป็น 1,000,000',
    icon: Zap,
  },
] as const

function formatBaht(satang: number) {
  return `${(satang / 100).toLocaleString('th-TH')}`;
}

export function BillingPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: overview, isLoading: overviewLoading } = useBillingOverview()
  const { data: quota } = useQuota()
  const checkout = useCreateCheckout()
  const portal = useCreatePortalSession()

  useEffect(() => {
    const status = searchParams.get('status')
    if (status === 'success') {
      toast.success('ชำระเงินสำเร็จ!')
      setSearchParams({}, { replace: true })
    } else if (status === 'cancel') {
      toast.info('การชำระเงินถูกยกเลิก')
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams])

  const activeCredits = overview?.credits ?? []

  const activeSubscriptions = overview?.subscriptions?.filter(
    (s) => s.status === 'active'
  ) ?? []

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">การเงิน</h1>
          <p className="text-sm text-muted-foreground mt-1">
            จัดการแพ็กรีเพลย์ ส่วนเสริม และการสมัครใช้งาน
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => portal.mutate()}
          disabled={portal.isPending}
        >
          <ExternalLink className="h-4 w-4" />
          {portal.isPending ? 'กำลังเปิด...' : 'จัดการการชำระเงิน'}
        </Button>
      </div>

      {/* Quota Summary */}
      {quota && (
        <Card className="mb-6">
          <CardContent className="p-6">
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              <div>
                <p className="text-xs text-muted-foreground">อีเวนต์เดือนนี้</p>
                <p className="text-lg font-bold text-foreground">
                  {quota.events_used_this_month.toLocaleString()} / {quota.max_events_per_month.toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">รีเพลย์คงเหลือ</p>
                <p className="text-lg font-bold text-foreground">
                  {quota.remaining_replays === -1 ? 'ไม่จำกัด' : quota.remaining_replays}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">ระยะเวลาเก็บข้อมูล</p>
                <p className="text-lg font-bold text-foreground">{quota.retention_days} วัน</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">พิกเซลสูงสุด</p>
                <p className="text-lg font-bold text-foreground">{quota.max_pixels}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Replay Packs */}
      <div className="mb-8">
        <h2 className="text-lg font-semibold text-foreground mb-4">แพ็กรีเพลย์</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {REPLAY_PACKS.map((pack) => (
            <Card
              key={pack.type}
              className={pack.popular ? 'border-primary ring-1 ring-primary' : ''}
            >
              {pack.popular && (
                <div className="bg-primary text-primary-foreground text-center text-xs font-medium py-1 rounded-t-lg">
                  ยอดนิยม
                </div>
              )}
              <CardContent className="p-6">
                <div className="flex items-center gap-2 mb-3">
                  <div className="h-9 w-9 rounded-lg bg-muted flex items-center justify-center">
                    <pack.icon className="h-5 w-5 text-foreground" />
                  </div>
                  <div>
                    <p className="font-semibold text-foreground">{pack.name}</p>
                    <p className="text-xs text-muted-foreground">{pack.description}</p>
                  </div>
                </div>
                <div className="mb-4">
                  <span className="text-3xl font-bold text-foreground">{pack.price.toLocaleString('th-TH')}</span>
                  <span className="text-sm text-muted-foreground"> THB</span>
                </div>
                <ul className="space-y-2 mb-4 text-sm text-muted-foreground">
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                    {pack.replays === -1 ? 'รีเพลย์ไม่จำกัด' : `${pack.replays} รีเพลย์`}
                  </li>
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                    สูงสุด {pack.eventsPerReplay.toLocaleString()} อีเวนต์/รีเพลย์
                  </li>
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                    ใช้ได้ 30 วัน
                  </li>
                </ul>
                <Button
                  className="w-full"
                  variant={pack.popular ? 'default' : 'outline'}
                  onClick={() => checkout.mutate({ pack_type: pack.type })}
                  disabled={checkout.isPending}
                >
                  {checkout.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <CreditCard className="h-4 w-4" />
                  )}
                  ซื้อเลย
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Active Credits */}
      {activeCredits.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold text-foreground mb-4">เครดิตที่ใช้งานอยู่</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {activeCredits.map((credit) => (
              <Card key={credit.id}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between mb-3">
                    <Badge variant="success">ใช้งาน</Badge>
                    <span className="text-xs text-muted-foreground">
                      หมดอายุ {formatDistanceToNow(new Date(credit.expires_at), { addSuffix: true })}
                    </span>
                  </div>
                  <p className="text-sm font-medium text-foreground capitalize mb-2">แพ็ก {credit.pack_type}</p>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">รีเพลย์ที่ใช้แล้ว</span>
                    <span className="font-semibold text-foreground">
                      {credit.used_replays} / {credit.total_replays === -1 ? 'ไม่จำกัด' : credit.total_replays}
                    </span>
                  </div>
                  <div className="mt-2">
                    <div className="h-2 bg-secondary rounded-full overflow-hidden">
                      <div
                        className="h-full bg-primary rounded-full"
                        style={{
                          width: credit.total_replays === -1
                            ? '10%'
                            : `${Math.min((credit.used_replays / credit.total_replays) * 100, 100)}%`,
                        }}
                      />
                    </div>
                  </div>
                  <p className="text-xs text-muted-foreground mt-2">
                    สูงสุด {credit.max_events_per_replay.toLocaleString()} อีเวนต์ต่อรีเพลย์
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Add-ons */}
      <div className="mb-8">
        <h2 className="text-lg font-semibold text-foreground mb-4">ส่วนเสริม</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {ADDONS.map((addon) => {
            const isActive = activeSubscriptions.some((s) => s.addon_type === addon.type)
            return (
              <Card key={addon.type}>
                <CardContent className="p-6">
                  <div className="flex items-center gap-2 mb-3">
                    <div className="h-9 w-9 rounded-lg bg-muted flex items-center justify-center">
                      <addon.icon className="h-5 w-5 text-foreground" />
                    </div>
                    <div>
                      <p className="font-semibold text-foreground">{addon.name}</p>
                      <p className="text-xs text-muted-foreground">{addon.description}</p>
                    </div>
                  </div>
                  <div className="mb-4">
                    <span className="text-3xl font-bold text-foreground">{addon.price.toLocaleString('th-TH')}</span>
                    <span className="text-sm text-muted-foreground"> THB/เดือน</span>
                  </div>
                  {isActive ? (
                    <Button variant="outline" className="w-full" disabled>
                      <ShieldCheck className="h-4 w-4" />
                      ใช้งานอยู่
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      className="w-full"
                      onClick={() => checkout.mutate({ addon_type: addon.type })}
                      disabled={checkout.isPending}
                    >
                      {checkout.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <CreditCard className="h-4 w-4" />
                      )}
                      สมัครใช้งาน
                    </Button>
                  )}
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>

      {/* Purchase History */}
      <div>
        <h2 className="text-lg font-semibold text-foreground mb-4">ประวัติการซื้อ</h2>
        <Card>
          <CardContent className="p-0">
            {overviewLoading ? (
              <div className="p-6 text-center text-sm text-muted-foreground">กำลังโหลด...</div>
            ) : !overview?.purchases || overview.purchases.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">
                ยังไม่มีประวัติการซื้อ
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-muted">
                    <tr>
                      <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่</th>
                      <th className="text-left px-4 py-3 font-medium text-muted-foreground">แพ็ก</th>
                      <th className="text-left px-4 py-3 font-medium text-muted-foreground">จำนวนเงิน</th>
                      <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {overview.purchases.map((purchase) => (
                      <tr key={purchase.id}>
                        <td className="px-4 py-3 text-foreground">
                          {new Date(purchase.created_at).toLocaleDateString('th-TH')}
                        </td>
                        <td className="px-4 py-3 text-foreground capitalize">{purchase.pack_type}</td>
                        <td className="px-4 py-3 text-foreground">
                          {formatBaht(purchase.amount_satang)} {purchase.currency}
                        </td>
                        <td className="px-4 py-3">
                          <Badge
                            variant={
                              purchase.status === 'completed'
                                ? 'success'
                                : purchase.status === 'pending'
                                  ? 'secondary'
                                  : 'destructive'
                            }
                          >
                            {purchase.status === 'completed' ? 'สำเร็จ' : purchase.status === 'pending' ? 'รอดำเนินการ' : 'ล้มเหลว'}
                          </Badge>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
