import {
  Loader2,
  Crown,
  Shield,
  Rocket,
  Sparkles,
  CheckCircle2,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

interface PlanDef {
  key: string
  planType?: string
  name: string
  price: number
  icon: React.ElementType
  popular?: boolean
  limits: {
    salePages: number
    pixels: number
    events: string
    retention: string
    replay: string
  }
}

const PLANS: PlanDef[] = [
  {
    key: 'sandbox',
    name: 'Sandbox',
    price: 0,
    icon: Sparkles,
    limits: {
      salePages: 1,
      pixels: 2,
      events: '5,000',
      retention: '7 วัน',
      replay: 'ไม่ได้',
    },
  },
  {
    key: 'launch',
    planType: 'plan_launch',
    name: 'Launch',
    price: 790,
    icon: Rocket,
    limits: {
      salePages: 5,
      pixels: 10,
      events: '200,000',
      retention: '60 วัน',
      replay: 'ซื้อแยก',
    },
  },
  {
    key: 'shield',
    planType: 'plan_shield',
    name: 'Shield',
    price: 2490,
    icon: Shield,
    popular: true,
    limits: {
      salePages: 15,
      pixels: 25,
      events: '1,000,000',
      retention: '180 วัน',
      replay: '3 credits/เดือน',
    },
  },
  {
    key: 'vault',
    planType: 'plan_vault',
    name: 'Vault',
    price: 4990,
    icon: Crown,
    limits: {
      salePages: 30,
      pixels: 50,
      events: '5,000,000',
      retention: '365 วัน',
      replay: 'ไม่จำกัด',
    },
  },
]

interface PlanSectionProps {
  currentPlan: string
  pendingCheckoutType: string | null
  isPending: boolean
  onCheckout: (params: { plan_type: string }) => void
  onManageBilling: () => void
}

export function PlanSection({
  currentPlan,
  pendingCheckoutType,
  isPending,
  onCheckout,
  onManageBilling,
}: PlanSectionProps) {
  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <h2 className="text-lg font-semibold text-foreground">แผนการใช้งาน</h2>
        <Badge variant="secondary">รายเดือน</Badge>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {PLANS.map((plan) => {
          const isCurrent = currentPlan === plan.key
          const isPaidPlan = plan.price > 0
          const isLoading = pendingCheckoutType === plan.planType

          return (
            <Card
              key={plan.key}
              className={
                plan.popular
                  ? 'border-primary ring-1 ring-primary relative'
                  : isCurrent
                    ? 'border-emerald-500 ring-1 ring-emerald-500'
                    : ''
              }
            >
              {plan.popular && (
                <div className="bg-primary text-primary-foreground text-center text-xs font-medium py-1 rounded-t-lg">
                  แนะนำ
                </div>
              )}
              <CardContent className="p-5">
                <div className="flex items-center gap-2 mb-3">
                  <div className="h-8 w-8 rounded-lg bg-muted flex items-center justify-center">
                    <plan.icon className="h-4 w-4 text-foreground" />
                  </div>
                  <span className="font-semibold text-foreground">{plan.name}</span>
                  {isCurrent && (
                    <Badge variant="outline" className="text-emerald-600 border-emerald-600 text-xs ml-auto">
                      แผนปัจจุบัน
                    </Badge>
                  )}
                </div>

                <div className="mb-4">
                  {plan.price > 0 ? (
                    <>
                      <span className="text-2xl font-bold text-foreground">
                        {plan.price.toLocaleString('th-TH')}
                      </span>
                      <span className="text-sm text-muted-foreground"> ฿/เดือน</span>
                    </>
                  ) : (
                    <span className="text-2xl font-bold text-foreground">ฟรี</span>
                  )}
                </div>

                <ul className="space-y-1.5 mb-4 text-xs text-muted-foreground">
                  <li className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                    {plan.limits.salePages} Sale Pages
                  </li>
                  <li className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                    {plan.limits.pixels} Pixels
                  </li>
                  <li className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                    {plan.limits.events} events/เดือน
                  </li>
                  <li className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                    เก็บข้อมูล {plan.limits.retention}
                  </li>
                  <li className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                    รีเพลย์: {plan.limits.replay}
                  </li>
                </ul>

                {isCurrent ? (
                  isPaidPlan ? (
                    <Button
                      className="w-full"
                      variant="outline"
                      size="sm"
                      onClick={onManageBilling}
                    >
                      จัดการแผน
                    </Button>
                  ) : null
                ) : isPaidPlan && plan.planType ? (
                  <Button
                    className="w-full"
                    variant={plan.popular ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => onCheckout({ plan_type: plan.planType! })}
                    disabled={isPending}
                  >
                    {isLoading ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : null}
                    {currentPlan !== 'sandbox' ? 'เปลี่ยนแผน' : 'เลือกแผนนี้'}
                  </Button>
                ) : null}
              </CardContent>
            </Card>
          )
        })}
      </div>
    </section>
  )
}
