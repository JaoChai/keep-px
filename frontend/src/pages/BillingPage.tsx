import { useCallback, useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useBillingOverview, useQuota, useCreateCheckout, useCreatePortalSession } from '@/hooks/use-billing'
import { AccountStatusCard } from '@/components/billing/AccountStatusCard'
import { PlanSection } from '@/components/billing/PlanSection'
import { ReplayTab } from '@/components/billing/ReplayTab'
import { AddonToggleSection } from '@/components/billing/AddonToggleSection'
import { PurchaseHistorySection } from '@/components/billing/PurchaseHistorySection'

type BillingTab = 'plans' | 'replays' | 'addons'

const TABS: { key: BillingTab; label: string }[] = [
  { key: 'plans', label: 'แผน' },
  { key: 'replays', label: 'รีเพลย์' },
  { key: 'addons', label: 'ส่วนเสริม' },
]

export function BillingPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const rawTab = searchParams.get('tab') as BillingTab | null
  const activeTab: BillingTab = TABS.some((t) => t.key === rawTab) ? rawTab! : 'plans'
  const setActiveTab = useCallback(
    (tab: BillingTab) => setSearchParams((prev) => { prev.set('tab', tab); return prev }, { replace: true }),
    [setSearchParams],
  )
  const { data: overview, isLoading: overviewLoading } = useBillingOverview()
  const { data: quota } = useQuota()
  const checkout = useCreateCheckout()
  const portal = useCreatePortalSession()
  const [pendingCheckoutType, setPendingCheckoutType] = useState<string | null>(null)

  const handleCheckout = (params: { pack_type?: string; addon_type?: string; plan_type?: string }) => {
    const key = params.plan_type ?? params.pack_type ?? params.addon_type ?? null
    setPendingCheckoutType(key)
    checkout.mutate(params, {
      onSettled: () => setPendingCheckoutType(null),
    })
  }

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
  const currentPlan = overview?.plan ?? quota?.plan ?? 'sandbox'

  const handleManageBilling = () => portal.mutate()
  const scrollToPlans = () => setActiveTab('plans')

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-foreground">การเงิน</h1>
        <p className="text-sm text-muted-foreground mt-1">
          จัดการแผน ส่วนเสริม และแพ็กรีเพลย์
        </p>
      </div>

      {/* Zone 1: Account Status Hero */}
      {quota && (
        <AccountStatusCard
          quota={quota}
          credits={activeCredits}
          currentPlan={currentPlan}
          onUpgrade={scrollToPlans}
          onManageBilling={handleManageBilling}
        />
      )}

      {/* Zone 2: Tab Bar + Content */}
      <div className="space-y-4">
        <div className="flex gap-1 bg-muted p-1 rounded-lg w-fit">
          {TABS.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={cn(
                'px-4 py-1.5 text-sm font-medium rounded-md transition-colors',
                activeTab === tab.key
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {activeTab === 'plans' && (
          <PlanSection
            currentPlan={currentPlan}
            pendingCheckoutType={pendingCheckoutType}
            isPending={checkout.isPending}
            onCheckout={handleCheckout}
            onManageBilling={handleManageBilling}
          />
        )}

        {activeTab === 'replays' && (
          <ReplayTab
            credits={activeCredits}
            pendingCheckoutType={pendingCheckoutType}
            isPending={checkout.isPending}
            onCheckout={handleCheckout}
          />
        )}

        {activeTab === 'addons' && (
          <AddonToggleSection
            activeSubscriptions={activeSubscriptions}
            pendingCheckoutType={pendingCheckoutType}
            isPending={checkout.isPending}
            onCheckout={handleCheckout}
            onManageBilling={handleManageBilling}
          />
        )}
      </div>

      {/* Zone 3: Payment management + History */}
      <div className="flex items-center justify-between">
        <div />
        <Button
          variant="outline"
          size="sm"
          onClick={handleManageBilling}
          disabled={portal.isPending}
        >
          <ExternalLink className="h-4 w-4" />
          {portal.isPending ? 'กำลังเปิด...' : 'จัดการการชำระเงิน'}
        </Button>
      </div>

      <PurchaseHistorySection
        purchases={overview?.purchases ?? []}
        isLoading={overviewLoading}
      />
    </div>
  )
}
