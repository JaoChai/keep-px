import { Ticket } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { isUnlimited, timeAgo, daysUntil } from '@/lib/utils'
import type { ReplayCredit } from '@/types'

interface ActiveCreditsSectionProps {
  credits: ReplayCredit[]
  onViewPlans?: () => void
}

export function ActiveCreditsSection({ credits, onViewPlans }: ActiveCreditsSectionProps) {
  if (credits.length === 0) {
    return (
      <section>
        <h2 className="text-lg font-semibold text-foreground mb-4">เครดิตรีเพลย์</h2>
        <Card className="border-dashed">
          <CardContent className="p-8 text-center">
            <Ticket className="h-8 w-8 text-muted-foreground mx-auto mb-3" />
            <p className="text-sm text-muted-foreground mb-3">
              ยังไม่มีเครดิตรีเพลย์ — ซื้อแพ็กด้านบนเพื่อเริ่มต้น
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={onViewPlans}
            >
              ดูแพ็กรีเพลย์
            </Button>
          </CardContent>
        </Card>
      </section>
    )
  }

  return (
    <section>
      <h2 className="text-lg font-semibold text-foreground mb-4">เครดิตรีเพลย์</h2>
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

                <p className="text-sm font-medium text-foreground capitalize mb-2">
                  แพ็ก {credit.pack_type}
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
    </section>
  )
}
