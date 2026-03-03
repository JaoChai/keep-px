import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Plus, Pencil, Trash2, Eye } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { useSalePages, useDeleteSalePage } from '@/hooks/use-sale-pages'
import { usePixels } from '@/hooks/use-pixels'
import { QueryErrorAlert } from '@/components/shared/QueryErrorAlert'

export function SalePagesPage() {
  const { data: salePages, isLoading, isError: isSalePagesError, error: salePagesError, refetch: refetchSalePages } = useSalePages()
  const { data: pixels, isError: isPixelsError, error: pixelsError, refetch: refetchPixels } = usePixels()
  const deleteSalePage = useDeleteSalePage()
  const navigate = useNavigate()

  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const getPixelNames = (pixelIds: string[]) => {
    if (!pixelIds?.length || !pixels) return '-'
    return pixelIds
      .map(pid => pixels.find(p => p.id === pid)?.name ?? '?')
      .join(', ')
  }

  const handleDelete = async (id: string) => {
    await deleteSalePage.mutateAsync(id)
    setDeleteConfirm(null)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">หน้าขาย</h1>
          <p className="text-sm text-muted-foreground mt-1">สร้างและจัดการหน้าขายของคุณ</p>
        </div>
        <Button onClick={() => navigate('/sale-pages/new')}>
          <Plus className="h-4 w-4" />
          สร้างหน้าขาย
        </Button>
      </div>

      {isSalePagesError && <QueryErrorAlert error={salePagesError} onRetry={refetchSalePages} className="mb-6" />}
      {isPixelsError && <QueryErrorAlert error={pixelsError} onRetry={refetchPixels} className="mb-6" />}

      {isLoading ? (
        <div className="text-center py-12 text-muted-foreground">กำลังโหลด...</div>
      ) : !salePages || salePages.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-border rounded-lg">
          <p className="text-muted-foreground mb-4">ยังไม่มีหน้าขาย</p>
          <Button variant="outline" onClick={() => navigate('/sale-pages/new')}>
            <Plus className="h-4 w-4" />
            สร้างหน้าขายแรกของคุณ
          </Button>
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-muted">
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">ชื่อ</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">URL</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">พิกเซล</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">เทมเพลต</th>
                <th className="text-left text-sm font-medium text-muted-foreground px-4 py-3">สถานะ</th>
                <th className="text-right text-sm font-medium text-muted-foreground px-4 py-3">การดำเนินการ</th>
              </tr>
            </thead>
            <tbody>
              {salePages.map((page) => (
                <tr key={page.id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3 text-sm font-medium text-foreground">{page.name}</td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    <a
                      href={`/p/${page.slug}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono truncate max-w-[200px] inline-block text-foreground hover:text-foreground/80 hover:underline"
                    >
                      /p/{page.slug}
                    </a>
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">{getPixelNames(page.pixel_ids)}</td>
                  <td className="px-4 py-3">
                    <Badge variant={page.template_name === 'blocks' ? 'default' : 'secondary'}>
                      {page.template_name === 'blocks' ? 'Blocks' : 'Classic'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={page.is_published ? 'success' : 'secondary'}>
                      {page.is_published ? 'เผยแพร่แล้ว' : 'แบบร่าง'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="ดูหน้า"
                        onClick={() => window.open(`/p/${page.slug}`, '_blank')}
                      >
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="แก้ไข"
                        onClick={() => navigate(
                          page.template_name === 'blocks'
                            ? `/sale-pages/${page.id}/edit-blocks`
                            : `/sale-pages/${page.id}/edit`
                        )}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="ลบ"
                        onClick={() => setDeleteConfirm(page.id)}
                      >
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

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>ลบหน้าขาย</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground mt-2">
            คุณแน่ใจหรือไม่ว่าต้องการลบหน้าขายนี้? ไม่สามารถย้อนกลับได้
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
