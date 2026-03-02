import { CheckCircle2, Loader2, XCircle, Clock, StopCircle } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface ReplayStatusBadgeProps {
  status: string
  className?: string
}

export function ReplayStatusBadge({ status, className }: ReplayStatusBadgeProps) {
  switch (status) {
    case 'completed':
      return <Badge variant="success" className={className}><CheckCircle2 className="h-3 w-3 mr-1" />เสร็จสิ้น</Badge>
    case 'running':
      return <Badge variant="warning" className={className}><Loader2 className="h-3 w-3 mr-1 animate-spin" />กำลังทำงาน</Badge>
    case 'failed':
      return <Badge variant="destructive" className={className}><XCircle className="h-3 w-3 mr-1" />ล้มเหลว</Badge>
    case 'cancelled':
      return <Badge variant="outline" className={className}><StopCircle className="h-3 w-3 mr-1" />ยกเลิกแล้ว</Badge>
    default:
      return <Badge variant="secondary" className={className}><Clock className="h-3 w-3 mr-1" />รอดำเนินการ</Badge>
  }
}
