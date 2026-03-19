---
name: frontend-feature
description: Scaffold a React dashboard page with TanStack Query hook, TypeScript types, route, and sidebar entry following Keep-PX conventions
---

# Frontend Feature

## When to Activate

- "Add [feature] page"
- "Create UI for [resource]"
- "Dashboard page for [feature]"
- "Frontend for [resource]"

## Workflow

Use an existing page as reference. Read these files first:

| Layer | Reference File | What to Copy |
|-------|---------------|-------------|
| Types | `frontend/src/types/index.ts` | Existing interfaces, `APIResponse` shape |
| Hook | `frontend/src/hooks/use-pixels.ts` | TanStack Query + mutation patterns |
| Page | `frontend/src/pages/PixelListPage.tsx` | CRUD page with dialog, table, empty state |
| Route | `frontend/src/router.tsx` | Route registration pattern |
| Sidebar | `frontend/src/components/layout/Sidebar.tsx` | Nav entry pattern |

Follow the same patterns. Replace the resource name.

## Files to Create/Edit (in order)

1. **`src/types/index.ts`** — Add interface (all IDs `string`, dates `string`, optional fields use `?`)
2. **`src/hooks/use-<feature>.ts`** — Query hook + CRUD mutations, invalidate on success
3. **`src/pages/<Feature>Page.tsx`** — Three states: loading, empty (dashed border), data (table)
4. **`src/router.tsx`** — Add route inside ProtectedRoute children
5. **`src/components/layout/Sidebar.tsx`** — Add nav entry with lucide-react icon

## Critical Rules

- Import API from `@/lib/api` (Axios with auth interceptor)
- Query keys: simple string arrays `['<resource>s']`
- `queryFn` returns unwrapped: `data.data!`
- Mutations always `invalidateQueries` on success
- Zod schema co-located at top of page file
- `useForm` with `zodResolver` for validation
- Named export: `export function <Feature>Page()`
- shadcn/ui components + lucide-react icons + neutral color palette
- **No `components.json`** — write shadcn components manually

## Verification

```bash
cd frontend && npm run build
```

## Related

- `go-service-scaffold` — backend API this page connects to
- `api-endpoint` — add new endpoints the page consumes
