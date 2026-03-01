import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Plus, Pencil, Trash2, Eye } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { useSalePages, useDeleteSalePage } from '@/hooks/use-sale-pages'
import { usePixels } from '@/hooks/use-pixels'

export function SalePagesPage() {
  const { data: salePages, isLoading } = useSalePages()
  const { data: pixels } = usePixels()
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
          <h1 className="text-2xl font-bold text-neutral-900">Sale Pages</h1>
          <p className="text-sm text-neutral-500 mt-1">Create and manage your sale pages</p>
        </div>
        <Button onClick={() => navigate('/sale-pages/new')}>
          <Plus className="h-4 w-4" />
          Create Sale Page
        </Button>
      </div>

      {isLoading ? (
        <div className="text-center py-12 text-neutral-500">Loading...</div>
      ) : !salePages || salePages.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-neutral-300 rounded-lg">
          <p className="text-neutral-500 mb-4">No sale pages yet</p>
          <Button variant="outline" onClick={() => navigate('/sale-pages/new')}>
            <Plus className="h-4 w-4" />
            Create your first Sale Page
          </Button>
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-200 bg-neutral-50">
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Name</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">URL</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Pixel</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Template</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Status</th>
                <th className="text-right text-sm font-medium text-neutral-500 px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {salePages.map((page) => (
                <tr key={page.id} className="border-b border-neutral-200 last:border-0">
                  <td className="px-4 py-3 text-sm font-medium text-neutral-900">{page.name}</td>
                  <td className="px-4 py-3 text-sm text-neutral-600">
                    <a
                      href={`/p/${page.slug}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono truncate max-w-[200px] inline-block text-indigo-600 hover:text-indigo-800 hover:underline"
                    >
                      /p/{page.slug}
                    </a>
                  </td>
                  <td className="px-4 py-3 text-sm text-neutral-600">{getPixelNames(page.pixel_ids)}</td>
                  <td className="px-4 py-3">
                    <Badge variant={page.template_name === 'blocks' ? 'default' : 'secondary'}>
                      {page.template_name === 'blocks' ? 'Blocks' : 'Classic'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={page.is_published ? 'success' : 'secondary'}>
                      {page.is_published ? 'Published' : 'Draft'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="View page"
                        onClick={() => window.open(`/p/${page.slug}`, '_blank')}
                      >
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Edit"
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
                        title="Delete"
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
            <DialogTitle>Delete Sale Page</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-neutral-500 mt-2">
            Are you sure you want to delete this sale page? This action cannot be undone.
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
