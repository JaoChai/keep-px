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
import type { Pixel } from '@/types'

const pixelSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  fb_pixel_id: z.string().min(1, 'Facebook Pixel ID is required'),
  fb_access_token: z.string().min(1, 'Access Token is required'),
  test_event_code: z.string().optional(),
})

type PixelForm = z.infer<typeof pixelSchema>

export function PixelsPage() {
  const { data: pixels, isLoading } = usePixels()
  const createPixel = useCreatePixel()
  const updatePixel = useUpdatePixel()
  const deletePixel = useDeletePixel()
  const testPixel = useTestPixel()
  const [showDialog, setShowDialog] = useState(false)
  const [editingPixel, setEditingPixel] = useState<Pixel | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

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
          <h1 className="text-2xl font-bold text-foreground">Pixels</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage your Facebook Pixels</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4" />
          Add Pixel
        </Button>
      </div>

      {isLoading ? (
        <div className="text-center py-12 text-muted-foreground">Loading...</div>
      ) : !pixels || pixels.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-border rounded-lg">
          <p className="text-muted-foreground mb-4">No pixels yet</p>
          <Button variant="outline" onClick={openCreate}>
            <Plus className="h-4 w-4" />
            Add your first Pixel
          </Button>
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-muted">
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">Name</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">Pixel ID</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">Status</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">Backup</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">Created</th>
                <th className="text-right text-sm font-medium text-muted-foreground px-4 py-3">Actions</th>
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
                        {pixel.is_active ? 'Active' : 'Paused'}
                      </Badge>
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    {pixel.backup_pixel_id ? (
                      <Badge variant="outline" className="gap-1">
                        <Shield className="h-3 w-3" />
                        {pixels?.find(p => p.id === pixel.backup_pixel_id)?.name || 'Unknown'}
                      </Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">None</span>
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
            <DialogTitle>{editingPixel ? 'Edit Pixel' : 'Add Pixel'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 mt-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input id="name" placeholder="My Pixel" {...register('name')} />
              {errors.name && <p className="text-sm text-red-500">{errors.name.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="fb_pixel_id">Facebook Pixel ID</Label>
              <Input id="fb_pixel_id" placeholder="123456789012345" {...register('fb_pixel_id')} />
              {errors.fb_pixel_id && <p className="text-sm text-red-500">{errors.fb_pixel_id.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="fb_access_token">
                Access Token {editingPixel && <span className="text-muted-foreground font-normal">(leave blank to keep current)</span>}
              </Label>
              <Input id="fb_access_token" type="password" placeholder="EAAxxxxxxx..." {...register('fb_access_token')} />
              {errors.fb_access_token && <p className="text-sm text-red-500">{errors.fb_access_token.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="test_event_code">
                Test Event Code <span className="text-muted-foreground font-normal">(optional)</span>
              </Label>
              <Input id="test_event_code" placeholder="TEST12345" {...register('test_event_code')} />
              <p className="text-xs text-muted-foreground">คัดลอกจาก Facebook Events Manager → เหตุการณ์ทดสอบ เพื่อ debug events</p>
            </div>
            {editingPixel && (
              <div className="space-y-2">
                <Label>Backup Pixel</Label>
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
                  <option value="">No backup</option>
                  {pixels?.filter(p => p.id !== editingPixel.id).map((p) => (
                    <option key={p.id} value={p.id}>{p.name} ({p.fb_pixel_id})</option>
                  ))}
                </select>
                <p className="text-xs text-muted-foreground">Events will also be forwarded to the backup pixel via CAPI</p>
              </div>
            )}
            <DialogFooter>
              <Button variant="outline" type="button" onClick={() => setShowDialog(false)}>Cancel</Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : editingPixel ? 'Save Changes' : 'Add Pixel'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>Delete Pixel</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground mt-2">
            Are you sure? This will also delete all events and rules associated with this pixel. This action cannot be undone.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(null)}>Cancel</Button>
            <Button variant="destructive" onClick={() => deleteConfirm && handleDelete(deleteConfirm)}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </div>
  )
}
