---
name: go-service-scaffold
description: Scaffold complete Go resource with all 7 clean architecture layers — ใช้เมื่อ: สร้าง resource ใหม่, CRUD ใหม่, backend resource ใหม่, scaffold service, สร้าง service ใหม่, เพิ่ม resource
---

# Go Service Scaffold

## When to Activate

- "Add a [resource] to the backend"
- "Create CRUD for [resource]"
- "New [resource] API"
- "Scaffold [resource] service"

## Workflow

Use the **pixel** resource as the canonical reference. Read these files first:

| Layer | Reference File | What to Copy |
|-------|---------------|-------------|
| Domain | `backend/internal/domain/pixel.go` | Struct pattern, field types, json tags |
| Interface | `backend/internal/repository/interfaces.go` | `PixelRepository` interface |
| Postgres | `backend/internal/repository/postgres/pixel_repo.go` | CRUD query patterns |
| Service | `backend/internal/service/pixel_service.go` | Ownership check, sentinel errors, input structs |
| Mocks | `backend/internal/service/mocks_test.go` | `MockPixelRepo` pattern |
| Tests | `backend/internal/service/pixel_service_test.go` | Table-driven tests, helper constructor |
| Handler | `backend/internal/handler/pixel_handler.go` | Error mapping, validation, response shape |
| Router | `backend/internal/router/router.go` | DI wiring order: repo → service → handler → routes |

Follow the same patterns exactly. Replace `Pixel` with your resource name.

## Critical Rules

### Domain
- Package `domain`, no imports except `time`
- All IDs are `string` (UUIDs), timestamps are `time.Time`
- Pointer types (`*string`) for optional fields, no methods on structs

### Repository
- `GetByID` returns `(nil, nil)` when not found (NOT an error)
- `Create` uses `QueryRow` + `RETURNING` to fill generated columns
- `ListBy*` uses `defer rows.Close()` and checks `rows.Err()`
- Use `$1, $2` placeholders (not `?`)

### Service
- Sentinel errors: `Err<Resource>NotFound`, `Err<Resource>NotOwned`
- Ownership check: nil → NotFound, CustomerID mismatch → NotOwned
- Input structs with `validate` tags, pointer fields for partial updates
- Wrap errors: `fmt.Errorf("verb context: %w", err)`
- Convert nil slices: `if items == nil { items = []*T{} }`

### Mocks
- Update BOTH `service/mocks_test.go` AND `handler/testhelpers_test.go`
- `args.Get(0) == nil` guard before type assertion for pointer returns

### Handler
- Concrete service pointer (not interface)
- `middleware.GetCustomerID(r.Context())` for customer ID
- Error mapping: NotFound→404, NotOwned→403, else→500
- Use `JSON()` / `ErrorJSON()` from `handler/response.go`

### Router
- DI wiring in `router.go`: repo → service → handler → routes
- Place in JWT-protected group (default)

### Tests
- Table-driven with `setup func(*MockRepo)` per case
- Fresh mocks per subtest
- `assert.ErrorIs` for sentinel errors
- `repo.AssertExpectations(t)` at end

## Verification

```bash
cd backend && go build ./cmd/server && go vet ./... && go test ./internal/service/...
```

## Related

- `db-migration` — create the database table first
- `api-endpoint` — add methods to an existing resource
