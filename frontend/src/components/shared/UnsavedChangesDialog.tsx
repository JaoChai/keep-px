import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'

interface UnsavedChangesDialogProps {
  isBlocked: boolean
  onStay: () => void
  onLeave: () => void
}

export function UnsavedChangesDialog({ isBlocked, onStay, onLeave }: UnsavedChangesDialogProps) {
  return (
    <Dialog open={isBlocked} onOpenChange={() => onStay()}>
      <DialogContent onClose={() => onStay()}>
        <DialogHeader>
          <DialogTitle>มีการเปลี่ยนแปลงที่ยังไม่ได้บันทึก</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground mt-2">
          คุณต้องการออกโดยไม่บันทึกการเปลี่ยนแปลงหรือไม่?
        </p>
        <DialogFooter>
          <Button variant="outline" onClick={() => onStay()}>อยู่ต่อ</Button>
          <Button variant="destructive" onClick={() => onLeave()}>ออกโดยไม่บันทึก</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
