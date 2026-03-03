import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { useBillingOverview, useQuota, useCreateCheckout, useCreatePortalSession } from '@/hooks/use-billing'
import { PlanSection } from '@/components/billing/PlanSection'
import { UsageDashboard } from '@/components/billing/UsageDashboard'
import { ReplayPackSection } from '@/components/billing/ReplayPackSection'
import { ActiveCreditsSection } from '@/components/billing/ActiveCreditsSection'
import { AddonToggleSection } from '@/components/billing/AddonToggleSection'
import { PurchaseHistorySection } from '@/components/billing/PurchaseHistorySection'

export function BillingPage() {
  const [searchParams, setSearchParams] = useSearchParams()
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

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">การเงิน</h1>
          <p className="text-sm text-muted-foreground mt-1">
            จัดการแผน ส่วนเสริม และแพ็กรีเพลย์
          </p>
        </div>
      </div>

      <PlanSection
        currentPlan={currentPlan}
        pendingCheckoutType={pendingCheckoutType}
        isPending={checkout.isPending}
        onCheckout={handleCheckout}
        onManageBilling={() => portal.mutate()}
      />

      {quota && <UsageDashboard quota={quota} />}

      <div id="replay-packs">
        <ReplayPackSection
          pendingCheckoutType={pendingCheckoutType}
          isPending={checkout.isPending}
          onCheckout={handleCheckout}
        />
      </div>

      <ActiveCreditsSection
        credits={activeCredits}
        onViewPlans={() => {
          document.getElementById('replay-packs')?.scrollIntoView({ behavior: 'smooth' })
        }}
      />

      <AddonToggleSection
        activeSubscriptions={activeSubscriptions}
        pendingCheckoutType={pendingCheckoutType}
        isPending={checkout.isPending}
        onCheckout={handleCheckout}
        onManageBilling={() => portal.mutate()}
      />

      <div className="flex items-center justify-between">
        <div />
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

      <PurchaseHistorySection
        purchases={overview?.purchases ?? []}
        isLoading={overviewLoading}
      />
    </div>
  )
}
