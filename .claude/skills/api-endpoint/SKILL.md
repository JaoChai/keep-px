---
name: api-endpoint
description: Add a new endpoint method to an existing Keep-PX backend resource (service + handler + route + test)
---

# API Endpoint

## When to Activate

- "Add endpoint to [existing resource]"
- "New route for [resource]"
- "Add [method] to [resource] API"

For a brand new resource, use `go-service-scaffold` instead.

## Workflow

### Step 1: Read Existing Code

Read these files to understand current structure:

1. `backend/internal/service/<resource>_service.go` — existing methods and errors
2. `backend/internal/handler/<resource>_handler.go` — handler patterns
3. `backend/internal/router/router.go` — route registration
4. `backend/internal/repository/interfaces.go` — interface (if new repo method needed)

### Step 2: Implement (follow existing patterns exactly)

| Layer | File | Add |
|-------|------|-----|
| Service | `service/<resource>_service.go` | Input struct + method (ownership check pattern) |
| Repo (if needed) | `repository/interfaces.go` + `postgres/<resource>_repo.go` | Interface method + implementation |
| Mocks (if repo changed) | `service/mocks_test.go` + `handler/testhelpers_test.go` | Mock methods in BOTH files |
| Handler | `handler/<resource>_handler.go` | HTTP handler with error mapping |
| Route | `router/router.go` | Register in correct auth group |
| Tests | `service/<resource>_service_test.go` | Table-driven: success, not found, not owned |

### Error Mapping Reference

| Sentinel Error | HTTP | Message |
|---|---|---|
| `Err*NotFound` | 404 | `<resource> not found` |
| `Err*NotOwned` | 403 | `<resource> not owned by you` |
| `Err*Invalid*` | 400 | descriptive |
| all others | 500 | `failed to <action> <resource>` |

### Route Patterns

- CRUD: `GET /`, `POST /`, `PUT /{id}`, `DELETE /{id}`
- Action: `POST /{id}/<action>`
- Sub-resource: `GET /{id}/<sub-resource>s`

## Verification

```bash
cd backend && go build ./cmd/server && go vet ./... && go test ./internal/service/...
```

## Related

- `go-service-scaffold` — create a brand new resource from scratch
- `db-migration` — if the endpoint requires schema changes
