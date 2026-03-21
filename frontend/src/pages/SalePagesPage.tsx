import { useState, useRef } from 'react'
import { useNavigate } from 'react-router'
import { Plus, Pencil, Trash2, Copy, Check, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { useSalePages, useDeleteSalePage, useCreateSalePage, useUpdateSalePage } from '@/hooks/use-sale-pages'
import { usePixels } from '@/hooks/use-pixels'
import { usePixelNameMap } from '@/hooks/use-pixel-name-map'
import { QueryErrorAlert } from '@/components/shared/QueryErrorAlert'
import { toast } from 'sonner'
import type { SalePage } from '@/types'

export function SalePagesPage() {
  const { data: salePages, isLoading, isError: isSalePagesError, error: salePagesError, refetch: refetchSalePages } = useSalePages()
  const { data: pixels, isError: isPixelsError, error: pixelsError, refetch: refetchPixels } = usePixels()
  const deleteSalePage = useDeleteSalePage()
  const createSalePage = useCreateSalePage()
  const updateSalePage = useUpdateSalePage()
  const navigate = useNavigate()

  const pixelNameMap = usePixelNameMap()
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [duplicatingId, setDuplicatingId] = useState<string | null>(null)
  const [pixelSwitchPage, setPixelSwitchPage] = useState<SalePage | null>(null)
  const [pixelSelections, setPixelSelections] = useState<string[]>([])
  const copyTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const getPixelNames = (pixelIds: string[]) => {
    if (!pixelIds?.length) return '-'
    return pixelIds
      .map(pid => pixelNameMap.get(pid) ?? '?')
      .join(', ')
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteSalePage.mutateAsync(id)
      setDeleteConfirm(null)
    } catch { /* hook onError shows toast */ }
  }

  const copyLink = async (page: SalePage) => {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/p/${page.slug}`)
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
      setCopiedId(page.id)
      toast.success('คัดลอกลิงก์แล้ว')
      copyTimerRef.current = setTimeout(() => setCopiedId(null), 2000)
    } catch {
      toast.error('คัดลอกลิงก์ไม่สำเร็จ')
    }
  }

  const handleDuplicate = async (page: SalePage) => {
    setDuplicatingId(page.id)
    try {
      await createSalePage.mutateAsync({
        name: `สำเนา - ${page.name}`,
        pixel_ids: page.pixel_ids,
        template_name: page.template_name,
        content: page.content,
        is_published: false,
      })
    } catch { /* hook onError shows toast */ }
    setDuplicatingId(null)
  }

  const openPixelSwitch = (page: SalePage) => {
    setPixelSwitchPage(page)
    setPixelSelections(page.pixel_ids || [])
  }

  const handlePixelSave = async () => {
    if (!pixelSwitchPage) return
    try {
      await updateSalePage.mutateAsync({
        id: pixelSwitchPage.id,
        name: pixelSwitchPage.name,
        slug: pixelSwitchPage.slug,
        pixel_ids: pixelSelections,
        template_name: pixelSwitchPage.template_name,
        content: pixelSwitchPage.content,
        is_published: pixelSwitchPage.is_published,
      })
      setPixelSwitchPage(null)
    } catch { /* hook onError shows toast, dialog stays open for retry */ }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">เซลเพจ</h1>
          <p className="text-sm text-muted-foreground mt-1">สร้างและจัดการเซลเพจของคุณ</p>
        </div>
        <Button onClick={() => navigate('/sale-pages/new')}>
          <Plus className="h-4 w-4" />
          สร้างเซลเพจ
        </Button>
      </div>

      {isSalePagesError && <QueryErrorAlert error={salePagesError} onRetry={refetchSalePages} className="mb-6" />}
      {isPixelsError && <QueryErrorAlert error={pixelsError} onRetry={refetchPixels} className="mb-6" />}

      {isLoading ? (
        <div data-testid="sale-page-grid" className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="border border-border rounded-lg overflow-hidden bg-card">
              <Skeleton className="h-1 w-full" />
              <div className="p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <Skeleton className="h-5 w-16" />
                  <Skeleton className="h-5 w-14" />
                </div>
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-28" />
                <div className="flex flex-wrap gap-2 pt-1">
                  <Skeleton className="h-8 w-24" />
                  <Skeleton className="h-8 w-16" />
                  <Skeleton className="h-8 w-16" />
                  <Skeleton className="h-8 w-24" />
                  <Skeleton className="h-8 w-12" />
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : !salePages || salePages.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-border rounded-lg">
          <p className="text-muted-foreground mb-4">ยังไม่มีเซลเพจ</p>
          <Button variant="outline" onClick={() => navigate('/sale-pages/new')}>
            <Plus className="h-4 w-4" />
            สร้างเซลเพจแรกของคุณ
          </Button>
        </div>
      ) : (
        <div data-testid="sale-page-grid" className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {salePages.map((page) => {
            const accent = page.content?.style?.accent_color || '#667eea'
            return (
              <div key={page.id} data-testid="sale-page-card" className="border border-border rounded-lg overflow-hidden bg-card">
                <div style={{ height: '4px', backgroundColor: accent }} />
                <div className="p-4">
                  <div className="flex items-center justify-between mb-2">
                    <Badge variant={page.is_published ? 'success' : 'secondary'}>
                      {page.is_published ? 'เผยแพร่แล้ว' : 'แบบร่าง'}
                    </Badge>
                    <Badge variant={page.template_name === 'blocks' ? 'default' : 'secondary'}>
                      {page.template_name === 'blocks' ? 'Blocks' : 'Classic'}
                    </Badge>
                  </div>

                  <p data-testid="sale-page-name" className="text-base font-medium text-foreground mb-1">
                    {page.name}
                  </p>

                  <p className="text-sm text-muted-foreground mb-1">
                    {getPixelNames(page.pixel_ids)}
                  </p>

                  <p data-testid="sale-page-url" className="font-mono text-sm text-muted-foreground mb-3">
                    /p/{page.slug}
                  </p>

                  <div className="flex flex-wrap gap-2">
                    <Button size="sm" onClick={() => copyLink(page)}>
                      {copiedId === page.id ? (
                        <Check className="h-4 w-4 text-green-500" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                      คัดลอกลิงก์
                    </Button>

                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => navigate(
                        page.template_name === 'blocks'
                          ? `/sale-pages/${page.id}/edit-blocks`
                          : `/sale-pages/${page.id}/edit`
                      )}
                    >
                      <Pencil className="h-4 w-4" />
                      แก้ไข
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={duplicatingId === page.id}
                      onClick={() => handleDuplicate(page)}
                    >
                      <Copy className="h-4 w-4" />
                      ทำซ้ำ
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => openPixelSwitch(page)}
                    >
                      <RefreshCw className="h-4 w-4" />
                      เปลี่ยน Pixel
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-red-500 hover:text-red-600"
                      onClick={() => setDeleteConfirm(page.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                      ลบ
                    </Button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>ลบเซลเพจ</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground mt-2">
            คุณแน่ใจหรือไม่ว่าต้องการลบเซลเพจนี้? ไม่สามารถย้อนกลับได้
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(null)}>ยกเลิก</Button>
            <Button variant="destructive" onClick={() => deleteConfirm && handleDelete(deleteConfirm)}>
              ลบ
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Pixel Switch Dialog */}
      <Dialog open={!!pixelSwitchPage} onOpenChange={() => setPixelSwitchPage(null)}>
        <DialogContent onClose={() => setPixelSwitchPage(null)}>
          <DialogHeader>
            <DialogTitle>เปลี่ยน Pixel</DialogTitle>
          </DialogHeader>
          <div className="mt-4 space-y-2">
            {pixels && pixels.length > 0 ? (
              pixels.map((pixel) => (
                <label key={pixel.id} className="flex items-center gap-2 text-sm py-1 px-1 rounded hover:bg-accent cursor-pointer">
                  <input
                    type="checkbox"
                    checked={pixelSelections.includes(pixel.id)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setPixelSelections(prev => [...prev, pixel.id])
                      } else {
                        setPixelSelections(prev => prev.filter(id => id !== pixel.id))
                      }
                    }}
                  />
                  <span>{pixel.name}</span>
                  <span className="text-muted-foreground">({pixel.fb_pixel_id})</span>
                </label>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">ไม่มี Pixel ที่สามารถเลือกได้</p>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPixelSwitchPage(null)}>ยกเลิก</Button>
            <Button onClick={handlePixelSave} disabled={updateSalePage.isPending}>
              บันทึก
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
