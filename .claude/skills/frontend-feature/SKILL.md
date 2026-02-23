---
name: frontend-feature
description: Scaffold a React dashboard page with TanStack Query hook, TypeScript types, route, and sidebar entry following Keep-PX conventions
---

# Frontend Feature

## When to Activate

Activate this skill when the user says:
- "Add [feature] page"
- "Create UI for [resource]"
- "Dashboard page for [feature]"
- "Frontend for [resource]"
- "Add [resource] to the dashboard"

## Step-by-Step Workflow

### Step 1: Add TypeScript Interface

**File:** `frontend/src/types/index.ts` (append to existing file)

```typescript
export interface <Resource> {
  id: string
  customer_id: string
  name: string
  // resource-specific fields
  created_at: string
  updated_at: string
}
```

**Rules:**
- All IDs are `string` (UUIDs)
- All dates are `string` (ISO format from API)
- Use `Record<string, unknown>` for JSONB fields
- Optional fields use `?` suffix: `source_url?: string`
- Boolean fields without `?` default to required
- Place new interface after existing ones, before `PaginatedResponse`
- Import from `@/types` (not relative path)

### Step 2: Create TanStack Query Hook

**File:** `frontend/src/hooks/use-<feature>.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import type { APIResponse, <Resource> } from '@/types'

export function use<Resource>s() {
  return useQuery({
    queryKey: ['<resource>s'],
    queryFn: async () => {
      const { data } = await api.get<APIResponse<<Resource>[]>>('/<resource>s')
      return data.data!
    },
  })
}

export function useCreate<Resource>() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: { name: string }) => {
      const { data } = await api.post<APIResponse<<Resource>>>('/<resource>s', input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['<resource>s'] })
    },
  })
}

export function useUpdate<Resource>() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, ...input }: { id: string; name?: string }) => {
      const { data } = await api.put<APIResponse<<Resource>>>(`/<resource>s/${id}`, input)
      return data.data!
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['<resource>s'] })
    },
  })
}

export function useDelete<Resource>() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/<resource>s/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['<resource>s'] })
    },
  })
}
```

**Rules:**
- `api` is imported from `@/lib/api` (Axios instance with auth interceptor)
- Query keys are simple string arrays: `['<resource>s']`
- `queryFn` returns unwrapped data: `data.data!`
- Mutations always call `invalidateQueries` on success
- Type the API response: `api.get<APIResponse<T>>`
- Update mutation destructures `{ id, ...input }` to separate ID from body
- Delete mutation takes just `id: string`

### Step 3: Create Page Component

**File:** `frontend/src/pages/<Feature>Page.tsx`

```tsx
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { use<Resource>s, useCreate<Resource>, useUpdate<Resource>, useDelete<Resource> } from '@/hooks/use-<feature>'
import type { <Resource> } from '@/types'

const <resource>Schema = z.object({
  name: z.string().min(1, 'Name is required'),
})

type <Resource>Form = z.infer<typeof <resource>Schema>

export function <Feature>Page() {
  const { data: items, isLoading } = use<Resource>s()
  const create = useCreate<Resource>()
  const update = useUpdate<Resource>()
  const remove = useDelete<Resource>()

  const [showDialog, setShowDialog] = useState(false)
  const [editing, setEditing] = useState<<Resource> | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<<Resource>Form>({
    resolver: zodResolver(<resource>Schema),
  })

  const openCreate = () => {
    setEditing(null)
    reset({ name: '' })
    setShowDialog(true)
  }

  const openEdit = (item: <Resource>) => {
    setEditing(item)
    reset({ name: item.name })
    setShowDialog(true)
  }

  const onSubmit = async (data: <Resource>Form) => {
    if (editing) {
      await update.mutateAsync({ id: editing.id, ...data })
    } else {
      await create.mutateAsync(data)
    }
    setShowDialog(false)
  }

  const handleDelete = async (id: string) => {
    await remove.mutateAsync(id)
    setDeleteConfirm(null)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-neutral-900"><Feature>s</h1>
          <p className="text-sm text-neutral-500 mt-1">Manage your <feature>s</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4" />
          Add <Feature>
        </Button>
      </div>

      {isLoading ? (
        <div className="text-center py-12 text-neutral-500">Loading...</div>
      ) : !items || items.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-neutral-300 rounded-lg">
          <p className="text-neutral-500 mb-4">No <feature>s yet</p>
          <Button variant="outline" onClick={openCreate}>
            <Plus className="h-4 w-4" />
            Add your first <Feature>
          </Button>
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-200 bg-neutral-50">
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Name</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Created</th>
                <th className="text-right text-sm font-medium text-neutral-500 px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id} className="border-b border-neutral-200 last:border-0">
                  <td className="px-4 py-3 text-sm font-medium text-neutral-900">{item.name}</td>
                  <td className="px-4 py-3 text-sm text-neutral-500">
                    {new Date(item.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="icon" onClick={() => openEdit(item)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => setDeleteConfirm(item.id)}>
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
            <DialogTitle>{editing ? 'Edit <Feature>' : 'Add <Feature>'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 mt-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input id="name" placeholder="Enter name" {...register('name')} />
              {errors.name && <p className="text-sm text-red-500">{errors.name.message}</p>}
            </div>
            <DialogFooter>
              <Button variant="outline" type="button" onClick={() => setShowDialog(false)}>Cancel</Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : editing ? 'Save Changes' : 'Add <Feature>'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>Delete <Feature></DialogTitle>
          </DialogHeader>
          <p className="text-sm text-neutral-500 mt-2">
            Are you sure? This action cannot be undone.
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
```

**Rules:**
- Co-located Zod schema at top of page file
- `useForm` with `zodResolver` for form validation
- Dialog state: `boolean` for create/edit, `string | null` for delete confirm, `<Resource> | null` for editing
- Three states: loading, empty (dashed border), data (table)
- `shadcn/ui` components: Button, Input, Label, Dialog, Badge
- Icons from `lucide-react`
- Neutral color palette: `text-neutral-900`, `text-neutral-500`, `border-neutral-200`
- Named export: `export function <Feature>Page()`

### Step 4: Add Route

**File:** `frontend/src/router.tsx` (edit existing file)

1. Add import at top:
```typescript
import { <Feature>Page } from '@/pages/<Feature>Page'
```

2. Add route inside `children` array of the `'/'` route (the ProtectedRoute wrapper):
```typescript
{ path: '<feature>s', element: <<Feature>Page /> },
```

Place the new route after existing routes, before `settings`.

### Step 5: Add Sidebar Entry

**File:** `frontend/src/components/layout/Sidebar.tsx` (edit existing file)

1. Add icon import from `lucide-react`:
```typescript
import { ..., <Icon> } from 'lucide-react'
```

2. Add entry to `navItems` array:
```typescript
{ to: '/<feature>s', icon: <Icon>, label: '<Feature>s' },
```

Place the new entry in a logical position within the navigation (group related features together).

**Common lucide-react icons:**
- Content: `FileText`, `BookOpen`, `Newspaper`, `Archive`
- Data: `Database`, `BarChart3`, `PieChart`, `TrendingUp`
- Communication: `MessageSquare`, `Mail`, `Bell`, `Send`
- Actions: `Zap`, `Target`, `Crosshair`, `Wand2`
- Commerce: `ShoppingCart`, `CreditCard`, `DollarSign`, `Package`

## Verification

```bash
cd frontend && npm run build
```

Build must pass without TypeScript errors.

## Related

- `go-service-scaffold` skill for the backend API this page connects to
- `api-endpoint` skill to add new endpoints the page consumes
- Built-in `frontend-patterns` skill for general React best practices
