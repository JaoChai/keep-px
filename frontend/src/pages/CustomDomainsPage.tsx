import { useState } from 'react'
import { Plus, Trash2, RefreshCw, Copy, CheckCircle2, Clock, Globe } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'
import {
  useCustomDomains,
  useCreateCustomDomain,
  useVerifyCustomDomain,
  useDeleteCustomDomain,
} from '@/hooks/use-custom-domains'
import { useSalePages } from '@/hooks/use-sale-pages'

export function CustomDomainsPage() {
  const { data: domains, isLoading } = useCustomDomains()
  const { data: salePages } = useSalePages()
  const createDomain = useCreateCustomDomain()
  const verifyDomain = useVerifyCustomDomain()
  const deleteDomain = useDeleteCustomDomain()

  const [showAddDialog, setShowAddDialog] = useState(false)
  const [showDnsDialog, setShowDnsDialog] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const [newDomain, setNewDomain] = useState('')
  const [selectedSalePageId, setSelectedSalePageId] = useState('')
  const [cnameTarget, setCnameTarget] = useState('')
  const [createdDomain, setCreatedDomain] = useState('')

  const publishedSalePages = salePages?.filter((p) => p.is_published)

  const getSalePageName = (salePageId: string) => {
    if (!salePages) return '-'
    const page = salePages.find((p) => p.id === salePageId)
    return page?.name ?? '-'
  }

  const handleCreate = async () => {
    if (!newDomain || !selectedSalePageId) {
      toast.error('Please fill in all fields')
      return
    }
    try {
      const result = await createDomain.mutateAsync({
        domain: newDomain,
        sale_page_id: selectedSalePageId,
      })
      setCreatedDomain(newDomain)
      setCnameTarget(result.cname_target)
      setShowAddDialog(false)
      setShowDnsDialog(true)
      setNewDomain('')
      setSelectedSalePageId('')
      toast.success('Domain added successfully')
    } catch {
      toast.error('Failed to add domain')
    }
  }

  const handleVerify = async (id: string) => {
    try {
      await verifyDomain.mutateAsync(id)
      toast.success('Domain verification initiated')
    } catch {
      toast.error('Failed to verify domain')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteDomain.mutateAsync(id)
      setDeleteConfirm(null)
      toast.success('Domain deleted successfully')
    } catch {
      toast.error('Failed to delete domain')
    }
  }

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success('Copied to clipboard')
    } catch {
      toast.error('Failed to copy')
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-neutral-900">Custom Domains</h1>
          <p className="text-sm text-neutral-500 mt-1">
            Manage custom domains for your sale pages
          </p>
        </div>
        <Button onClick={() => setShowAddDialog(true)}>
          <Plus className="h-4 w-4" />
          Add Domain
        </Button>
      </div>

      {isLoading ? (
        <div className="text-center py-12 text-neutral-500">Loading...</div>
      ) : !domains || domains.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-neutral-300 rounded-lg">
          <Globe className="h-10 w-10 text-neutral-400 mx-auto mb-3" />
          <p className="text-neutral-500 mb-4">No custom domains yet</p>
          <Button variant="outline" onClick={() => setShowAddDialog(true)}>
            <Plus className="h-4 w-4" />
            Add your first domain
          </Button>
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-200 bg-neutral-50">
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">
                  Domain
                </th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">
                  Sale Page
                </th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">
                  DNS Status
                </th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">
                  SSL Status
                </th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">
                  Created
                </th>
                <th className="text-right text-sm font-medium text-neutral-500 px-4 py-3">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {domains.map((domain) => (
                <tr key={domain.id} className="border-b border-neutral-200 last:border-0">
                  <td className="px-4 py-3 text-sm font-medium text-neutral-900">
                    {domain.domain}
                  </td>
                  <td className="px-4 py-3 text-sm text-neutral-600">
                    {getSalePageName(domain.sale_page_id)}
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={domain.dns_verified ? 'success' : 'secondary'}>
                      {domain.dns_verified ? (
                        <span className="flex items-center gap-1">
                          <CheckCircle2 className="h-3 w-3" />
                          Verified
                        </span>
                      ) : (
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          Pending
                        </span>
                      )}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={domain.ssl_active ? 'success' : 'secondary'}>
                      {domain.ssl_active ? (
                        <span className="flex items-center gap-1">
                          <CheckCircle2 className="h-3 w-3" />
                          Active
                        </span>
                      ) : (
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          Pending
                        </span>
                      )}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-sm text-neutral-600">
                    {new Date(domain.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Verify DNS"
                        onClick={() => handleVerify(domain.id)}
                      >
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Delete"
                        onClick={() => setDeleteConfirm(domain.id)}
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

      {/* Add Domain Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent onClose={() => setShowAddDialog(false)}>
          <DialogHeader>
            <DialogTitle>Add Custom Domain</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 mt-4">
            <div className="space-y-2">
              <Label htmlFor="domain">Domain</Label>
              <Input
                id="domain"
                placeholder="shop.example.com"
                value={newDomain}
                onChange={(e) => setNewDomain(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="sale-page">Sale Page</Label>
              <select
                id="sale-page"
                className="flex h-10 w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm"
                value={selectedSalePageId}
                onChange={(e) => setSelectedSalePageId(e.target.value)}
              >
                <option value="">Select a sale page</option>
                {publishedSalePages?.map((page) => (
                  <option key={page.id} value={page.id}>
                    {page.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createDomain.isPending}>
              {createDomain.isPending ? 'Adding...' : 'Add Domain'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* DNS Instructions Dialog */}
      <Dialog open={showDnsDialog} onOpenChange={setShowDnsDialog}>
        <DialogContent onClose={() => setShowDnsDialog(false)}>
          <DialogHeader>
            <DialogTitle>DNS Configuration Required</DialogTitle>
          </DialogHeader>
          <div className="mt-4 space-y-4">
            <p className="text-sm text-neutral-600">
              To connect your domain <span className="font-semibold">{createdDomain}</span>, add
              the following CNAME record in your DNS provider:
            </p>
            <div className="rounded-lg border border-neutral-200 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-neutral-200 bg-neutral-50">
                    <th className="text-left px-4 py-2 font-medium text-neutral-500">Type</th>
                    <th className="text-left px-4 py-2 font-medium text-neutral-500">Name</th>
                    <th className="text-left px-4 py-2 font-medium text-neutral-500">Value</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td className="px-4 py-2 font-mono">CNAME</td>
                    <td className="px-4 py-2 font-mono">{createdDomain}</td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-2">
                        <span className="font-mono truncate">{cnameTarget}</span>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="shrink-0 h-7 w-7"
                          onClick={() => handleCopy(cnameTarget)}
                        >
                          <Copy className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className="text-sm text-neutral-500">
              DNS changes may take up to 24 hours to propagate. You can click the verify button to
              check the status at any time.
            </p>
          </div>
          <DialogFooter>
            <Button onClick={() => setShowDnsDialog(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>Delete Custom Domain</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-neutral-500 mt-2">
            Are you sure you want to delete this domain? This action cannot be undone.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
