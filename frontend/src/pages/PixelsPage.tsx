import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2, Zap, Shield } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { toast } from 'sonner'
import { usePixels, useCreatePixel, useUpdatePixel, useDeletePixel, useTestPixel } from '@/hooks/use-pixels'
import { useQuota } from '@/hooks/use-billing'
import { QueryErrorAlert } from '@/components/shared/QueryErrorAlert'
import { Link } from 'react-router'
import type { Pixel } from '@/types'

const pixelSchema = z.object({
  name: z.string().min(1, 'กรุณากรอกชื่อ'),
  fb_pixel_id: z.string().min(1, 'กรุณากรอก Facebook Pixel ID'),
  fb_access_token: z.string().min(1, 'กรุณากรอก Access Token'),
  test_event_code: z.string().optional(),
})

type PixelForm = z.infer<typeof pixelSchema>

export function PixelsPage() {
  const { data: pixels, isLoading, isError, error, refetch } = usePixels()
  const { data: quota } = useQuota()
  const createPixel = useCreatePixel()
  const updatePixel = useUpdatePixel()
  const deletePixel = useDeletePixel()
  const testPixel = useTestPixel()
  const [showDialog, setShowDialog] = useState(false)
  const [editingPixel, setEditingPixel] = useState<Pixel | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const isAtPixelLimit = !!quota && !!pixels && pixels.length >= quota.max_pixels

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<PixelForm>({
    resolver: zodResolver(pixelSchema),
  })

  const openCreate = () => {
    setEditingPixel(null)
    reset({ name: '', fb_pixel_id: '', fb_access_token: '', test_event_code: '' })
    setShowDialog(true)
  }

  const openEdit = (pixel: Pixel) => {
    setEditingPixel(pixel)
    reset({ name: pixel.name, fb_pixel_id: pixel.fb_pixel_id, fb_access_token: '', test_event_code: pixel.test_event_code || '' })
    setShowDialog(true)
  }

  const onSubmit = async (data: PixelForm) => {
    if (editingPixel) {
      await updatePixel.mutateAsync({
        id: editingPixel.id,
        name: data.name,
        fb_pixel_id: data.fb_pixel_id,
        ...(data.fb_access_token ? { fb_access_token: data.fb_access_token } : {}),
        test_event_code: data.test_event_code || '',
      })
      setShowDialog(false)
    } else {
      await createPixel.mutateAsync({
        ...data,
        test_event_code: data.test_event_code || undefined,
      })
      setShowDialog(false)
    }
  }

  const handleDelete = async (id: string) => {
    await deletePixel.mutateAsync(id)
    setDeleteConfirm(null)
  }

  const handleToggleActive = async (pixel: Pixel) => {
    await updatePixel.mutateAsync({ id: pixel.id, is_active: !pixel.is_active })
  }

  const handleTestConnection = async (pixel: Pixel) => {
    try {
      await testPixel.mutateAsync(pixel.id)
      toast.success('Pixel ทำงานปกติ — Facebook ได้รับ event แล้ว')
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      // Extract error from API response if available
      const apiError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      toast.error(apiError || `ส่งไม่ได้ — ${message}`)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">พิกเซล</h1>
          {quota && pixels && (
            <p className="text-sm text-muted-foreground mt-1">
              จัดการ Facebook Pixel ของคุณ ({pixels.length}/{quota.max_pixels} สล็อต)
            </p>
          )}
          {!quota && (
            <p className="text-sm text-muted-foreground mt-1">จัดการ Facebook Pixel ของคุณ</p>
          )}
        </div>
        <Button onClick={openCreate} disabled={isAtPixelLimit} title={isAtPixelLimit ? 'ถึงขีดจำกัด Pixel Slots แล้ว' : undefined}>
          <Plus className="h-4 w-4" />
          เพิ่มพิกเซล
        </Button>
      </div>

      {isError && <QueryErrorAlert error={error} onRetry={refetch} className="mb-6" />}

      {isLoading ? (
        <div className="text-center py-12 text-muted-foreground">กำลังโหลด...</div>
      ) : !pixels || pixels.length === 0 ? (
        isAtPixelLimit ? (
          <div className="text-center py-12 border border-dashed border-border rounded-lg">
            <p className="text-muted-foreground mb-4">ถึงขีดจำกัด Pixel Slots แล้ว</p>
            <Link to="/billing">
              <Button variant="outline">อัปเกรด</Button>
            </Link>
          </div>
        ) : (
          <div className="text-center py-12 border border-dashed border-border rounded-lg">
            <p className="text-muted-foreground mb-4">ยังไม่มีพิกเซล</p>
            <Button variant="outline" onClick={openCreate}>
              <Plus className="h-4 w-4" />
              เพิ่มพิกเซลแรกของคุณ
            </Button>
          </div>
        )
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-muted">
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">ชื่อ</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">Pixel ID</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">สถานะ</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">สำรอง</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">สร้างเมื่อ</th>
                <th className="text-right text-sm font-medium text-muted-foreground px-4 py-3">การดำเนินการ</th>
              </tr>
            </thead>
            <tbody>
              {pixels.map((pixel) => (
                <tr key={pixel.id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3 text-sm font-medium text-foreground">{pixel.name}</td>
                  <td className="px-4 py-3 text-sm text-muted-foreground font-mono">{pixel.fb_pixel_id}</td>
                  <td className="px-4 py-3">
                    <button onClick={() => handleToggleActive(pixel)}>
                      <Badge variant={pixel.is_active ? 'success' : 'warning'}>
                        {pixel.is_active ? 'ใช้งาน' : 'หยุดชั่วคราว'}
                      </Badge>
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    {pixel.backup_pixel_id ? (
                      <Badge variant="outline" className="gap-1">
                        <Shield className="h-3 w-3" />
                        {pixels?.find(p => p.id === pixel.backup_pixel_id)?.name || 'ไม่ทราบ'}
                      </Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">ไม่มี</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {new Date(pixel.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="ทดสอบการเชื่อมต่อ"
                        onClick={() => handleTestConnection(pixel)}
                        disabled={testPixel.isPending}
                      >
                        {testPixel.isPending && testPixel.variables === pixel.id ? (
                          <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                        ) : (
                          <Zap className="h-4 w-4" />
                        )}
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => openEdit(pixel)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => setDeleteConfirm(pixel.id)}>
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent onClose={() => setShowDialog(false)}>
          <DialogHeader>
            <DialogTitle>{editingPixel ? 'แก้ไขพิกเซล' : 'เพิ่มพิกเซล'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 mt-4">
            <div className="space-y-2">
              <Label htmlFor="name">ชื่อ</Label>
              <Input id="name" placeholder="พิกเซลของฉัน" {...register('name')} />
              {errors.name && <p className="text-sm text-red-500">{errors.name.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="fb_pixel_id">Facebook Pixel ID</Label>
              <Input id="fb_pixel_id" placeholder="123456789012345" {...register('fb_pixel_id')} />
              {errors.fb_pixel_id && <p className="text-sm text-red-500">{errors.fb_pixel_id.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="fb_access_token">
                Access Token {editingPixel && <span className="text-muted-foreground font-normal">(เว้นว่างเพื่อใช้ค่าเดิม)</span>}
              </Label>
              <Input id="fb_access_token" type="password" placeholder="EAAxxxxxxx..." {...register('fb_access_token')} />
              {errors.fb_access_token && <p className="text-sm text-red-500">{errors.fb_access_token.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="test_event_code">
                Test Event Code <span className="text-muted-foreground font-normal">(ไม่บังคับ)</span>
              </Label>
              <Input id="test_event_code" placeholder="TEST12345" {...register('test_event_code')} />
              <p className="text-xs text-muted-foreground">คัดลอกจาก Facebook Events Manager → เหตุการณ์ทดสอบ เพื่อ debug events</p>
            </div>
            {editingPixel && (
              <div className="space-y-2">
                <Label>พิกเซลสำรอง</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-border bg-transparent px-3 py-1 text-sm"
                  defaultValue={editingPixel.backup_pixel_id || ''}
                  onChange={(e) => {
                    updatePixel.mutate({
                      id: editingPixel.id,
                      backup_pixel_id: e.target.value,
                    })
                  }}
                >
                  <option value="">ไม่มีสำรอง</option>
                  {pixels?.filter(p => p.id !== editingPixel.id).map((p) => (
                    <option key={p.id} value={p.id}>{p.name} ({p.fb_pixel_id})</option>
                  ))}
                </select>
                <p className="text-xs text-muted-foreground">อีเวนต์จะถูกส่งต่อไปยังพิกเซลสำรองผ่าน CAPI ด้วย</p>
              </div>
            )}
            <DialogFooter>
              <Button variant="outline" type="button" onClick={() => setShowDialog(false)}>ยกเลิก</Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'กำลังบันทึก...' : editingPixel ? 'บันทึกการเปลี่ยนแปลง' : 'เพิ่มพิกเซล'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>ลบพิกเซล</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground mt-2">
            คุณแน่ใจหรือไม่? การดำเนินการนี้จะลบอีเวนต์และกฎทั้งหมดที่เกี่ยวข้องกับพิกเซลนี้ด้วย ไม่สามารถย้อนกลับได้
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(null)}>ยกเลิก</Button>
            <Button variant="destructive" onClick={() => deleteConfirm && handleDelete(deleteConfirm)}>
              ลบ
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </div>
  )
}
