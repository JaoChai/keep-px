import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2, Code, Copy, Check, Zap } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { toast } from 'sonner'
import { usePixels, useCreatePixel, useUpdatePixel, useDeletePixel, useTestPixel } from '@/hooks/use-pixels'
import { useAuthStore } from '@/stores/auth-store'
import type { Pixel } from '@/types'

const pixelSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  fb_pixel_id: z.string().min(1, 'Facebook Pixel ID is required'),
  fb_access_token: z.string().min(1, 'Access Token is required'),
})

type PixelForm = z.infer<typeof pixelSchema>

function getSnippet(pixelId: string, apiKey: string): string {
  const sdkUrl = `${window.location.origin}/sdk/pixlinks.min.js`
  return `<script src="${sdkUrl}"
        data-pixlinks-key="${apiKey}"
        data-pixlinks-pixel-id="${pixelId}"
        data-endpoint="${window.location.origin}">
</script>`
}

export function PixelsPage() {
  const { data: pixels, isLoading } = usePixels()
  const createPixel = useCreatePixel()
  const updatePixel = useUpdatePixel()
  const deletePixel = useDeletePixel()
  const testPixel = useTestPixel()
  const customer = useAuthStore((s) => s.customer)

  const [showDialog, setShowDialog] = useState(false)
  const [editingPixel, setEditingPixel] = useState<Pixel | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [snippetPixel, setSnippetPixel] = useState<Pixel | null>(null)
  const [copiedSnippet, setCopiedSnippet] = useState(false)

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
    reset({ name: '', fb_pixel_id: '', fb_access_token: '' })
    setShowDialog(true)
  }

  const openEdit = (pixel: Pixel) => {
    setEditingPixel(pixel)
    reset({ name: pixel.name, fb_pixel_id: pixel.fb_pixel_id, fb_access_token: '' })
    setShowDialog(true)
  }

  const onSubmit = async (data: PixelForm) => {
    if (editingPixel) {
      await updatePixel.mutateAsync({
        id: editingPixel.id,
        name: data.name,
        fb_pixel_id: data.fb_pixel_id,
        ...(data.fb_access_token ? { fb_access_token: data.fb_access_token } : {}),
      })
      setShowDialog(false)
    } else {
      const pixel = await createPixel.mutateAsync(data)
      setShowDialog(false)
      setSnippetPixel(pixel)
    }
  }

  const copySnippet = () => {
    if (snippetPixel && customer?.api_key) {
      navigator.clipboard.writeText(getSnippet(snippetPixel.id, customer.api_key))
      setCopiedSnippet(true)
      setTimeout(() => setCopiedSnippet(false), 2000)
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
          <h1 className="text-2xl font-bold text-neutral-900">Pixels</h1>
          <p className="text-sm text-neutral-500 mt-1">Manage your Facebook Pixels</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4" />
          Add Pixel
        </Button>
      </div>

      {isLoading ? (
        <div className="text-center py-12 text-neutral-500">Loading...</div>
      ) : !pixels || pixels.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-neutral-300 rounded-lg">
          <p className="text-neutral-500 mb-4">No pixels yet</p>
          <Button variant="outline" onClick={openCreate}>
            <Plus className="h-4 w-4" />
            Add your first Pixel
          </Button>
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-200 bg-neutral-50">
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Name</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Pixel ID</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Status</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Created</th>
                <th className="text-right text-sm font-medium text-neutral-500 px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pixels.map((pixel) => (
                <tr key={pixel.id} className="border-b border-neutral-200 last:border-0">
                  <td className="px-4 py-3 text-sm font-medium text-neutral-900">{pixel.name}</td>
                  <td className="px-4 py-3 text-sm text-neutral-600 font-mono">{pixel.fb_pixel_id}</td>
                  <td className="px-4 py-3">
                    <button onClick={() => handleToggleActive(pixel)}>
                      <Badge variant={pixel.is_active ? 'success' : 'warning'}>
                        {pixel.is_active ? 'Active' : 'Paused'}
                      </Badge>
                    </button>
                  </td>
                  <td className="px-4 py-3 text-sm text-neutral-500">
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
                      <Button variant="ghost" size="icon" title="Get Code" onClick={() => setSnippetPixel(pixel)}>
                        <Code className="h-4 w-4" />
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
                Access Token {editingPixel && <span className="text-neutral-400 font-normal">(leave blank to keep current)</span>}
              </Label>
              <Input id="fb_access_token" type="password" placeholder="EAAxxxxxxx..." {...register('fb_access_token')} />
              {errors.fb_access_token && <p className="text-sm text-red-500">{errors.fb_access_token.message}</p>}
            </div>
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
          <p className="text-sm text-neutral-500 mt-2">
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

      {/* Snippet Dialog */}
      <Dialog open={!!snippetPixel} onOpenChange={() => { setSnippetPixel(null); setCopiedSnippet(false) }}>
        <DialogContent onClose={() => { setSnippetPixel(null); setCopiedSnippet(false) }}>
          <DialogHeader>
            <DialogTitle>
              <div className="flex items-center gap-2">
                <Code className="h-5 w-5" />
                Get Code — {snippetPixel?.name}
              </div>
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-neutral-500 mt-2">
            Add this snippet to your website's <code className="bg-neutral-100 px-1 rounded">&lt;head&gt;</code> tag to start tracking events.
          </p>
          <div className="relative mt-3">
            <pre className="bg-neutral-900 text-neutral-100 text-sm p-4 rounded-lg overflow-x-auto">
              <code>{snippetPixel && customer?.api_key ? getSnippet(snippetPixel.id, customer.api_key) : ''}</code>
            </pre>
            <Button
              variant="outline"
              size="icon"
              className="absolute top-2 right-2 h-8 w-8 bg-neutral-800 border-neutral-700 hover:bg-neutral-700 text-neutral-300"
              onClick={copySnippet}
            >
              {copiedSnippet ? <Check className="h-4 w-4 text-green-400" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setSnippetPixel(null); setCopiedSnippet(false) }}>
              Done
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
