---
name: go-service-scaffold
description: Scaffold a complete Go resource with all 7 clean architecture layers following Keep-PX conventions
---

# Go Service Scaffold

## When to Activate

Activate this skill when the user says:
- "Add a [resource] to the backend"
- "Create CRUD for [resource]"
- "New [resource] API"
- "Scaffold [resource] service"

## Step-by-Step Workflow

### Step 1: Create Domain Struct

**File:** `backend/internal/domain/<resource>.go`

```go
package domain

import "time"

type <Resource> struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	// resource-specific fields here
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
```

**Rules:**
- Package `domain`, no imports except `time` (and `encoding/json` only if using `json.RawMessage`)
- All IDs are `string` (UUIDs stored as strings)
- Timestamps are `time.Time`
- Use `json:"-"` for sensitive fields (tokens, secrets)
- Use pointer types (`*string`, `*bool`) for optional fields
- No methods on domain structs

### Step 2: Add Repository Interface

**File:** `backend/internal/repository/interfaces.go` (append to existing file)

```go
type <Resource>Repository interface {
	Create(ctx context.Context, item *domain.<Resource>) error
	GetByID(ctx context.Context, id string) (*domain.<Resource>, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]*domain.<Resource>, error)
	Update(ctx context.Context, item *domain.<Resource>) error
	Delete(ctx context.Context, id string) error
}
```

**Rules:**
- First parameter is always `context.Context`
- `Create` mutates the input struct in-place (fills ID, timestamps via RETURNING)
- `GetByID` returns `(*T, error)` — returns `(nil, nil)` when not found
- `ListBy*` returns `([]*T, error)` — never `([]T, error)`
- `Delete` takes only the `id` string
- Import `domain` package: `"github.com/jaochai/pixlinks/backend/internal/domain"`

### Step 3: Create PostgreSQL Repository

**File:** `backend/internal/repository/postgres/<resource>_repo.go`

```go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type <Resource>Repo struct {
	pool *pgxpool.Pool
}

func New<Resource>Repo(pool *pgxpool.Pool) *<Resource>Repo {
	return &<Resource>Repo{pool: pool}
}

func (r *<Resource>Repo) Create(ctx context.Context, item *domain.<Resource>) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO <table> (customer_id, name)
		 VALUES ($1, $2)
		 RETURNING id, created_at, updated_at`,
		item.CustomerID, item.Name,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (r *<Resource>Repo) GetByID(ctx context.Context, id string) (*domain.<Resource>, error) {
	item := &domain.<Resource>{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, name, created_at, updated_at
		 FROM <table> WHERE id = $1`, id,
	).Scan(&item.ID, &item.CustomerID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func (r *<Resource>Repo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.<Resource>, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, name, created_at, updated_at
		 FROM <table> WHERE customer_id = $1 ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.<Resource>
	for rows.Next() {
		item := &domain.<Resource>{}
		if err := rows.Scan(&item.ID, &item.CustomerID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *<Resource>Repo) Update(ctx context.Context, item *domain.<Resource>) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE <table> SET name = $2, updated_at = NOW()
		 WHERE id = $1`,
		item.ID, item.Name,
	)
	return err
}

func (r *<Resource>Repo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM <table> WHERE id = $1`, id)
	return err
}
```

**Rules:**
- Use `pgx.ErrNoRows` → return `nil, nil` (NOT an error)
- `Create` uses `QueryRow` + `RETURNING` to fill generated columns
- `ListBy*` uses `defer rows.Close()` and `rows.Err()` at end
- `Update` sets `updated_at = NOW()` in SQL
- `Delete` uses `Exec` (no return value needed)
- ORDER BY `created_at DESC` for lists
- Use `$1, $2` placeholders (not `?`)

### Step 4: Create Service Layer

**File:** `backend/internal/service/<resource>_service.go`

```go
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	Err<Resource>NotFound = errors.New("<resource> not found")
	Err<Resource>NotOwned = errors.New("<resource> not owned by customer")
)

type <Resource>Service struct {
	<resource>Repo repository.<Resource>Repository
}

func New<Resource>Service(<resource>Repo repository.<Resource>Repository) *<Resource>Service {
	return &<Resource>Service{<resource>Repo: <resource>Repo}
}

type Create<Resource>Input struct {
	Name string `json:"name" validate:"required"`
}

type Update<Resource>Input struct {
	Name *string `json:"name,omitempty"`
}

func (s *<Resource>Service) Create(ctx context.Context, customerID string, input Create<Resource>Input) (*domain.<Resource>, error) {
	item := &domain.<Resource>{
		CustomerID: customerID,
		Name:       input.Name,
	}
	if err := s.<resource>Repo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create <resource>: %w", err)
	}
	return item, nil
}

func (s *<Resource>Service) GetByID(ctx context.Context, customerID, id string) (*domain.<Resource>, error) {
	item, err := s.<resource>Repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get <resource>: %w", err)
	}
	if item == nil {
		return nil, Err<Resource>NotFound
	}
	if item.CustomerID != customerID {
		return nil, Err<Resource>NotOwned
	}
	return item, nil
}

func (s *<Resource>Service) List(ctx context.Context, customerID string) ([]*domain.<Resource>, error) {
	items, err := s.<resource>Repo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list <resource>s: %w", err)
	}
	if items == nil {
		items = []*domain.<Resource>{}
	}
	return items, nil
}

func (s *<Resource>Service) Update(ctx context.Context, customerID, id string, input Update<Resource>Input) (*domain.<Resource>, error) {
	item, err := s.<resource>Repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get <resource>: %w", err)
	}
	if item == nil {
		return nil, Err<Resource>NotFound
	}
	if item.CustomerID != customerID {
		return nil, Err<Resource>NotOwned
	}

	if input.Name != nil {
		item.Name = *input.Name
	}

	if err := s.<resource>Repo.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update <resource>: %w", err)
	}
	return item, nil
}

func (s *<Resource>Service) Delete(ctx context.Context, customerID, id string) error {
	item, err := s.<resource>Repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get <resource>: %w", err)
	}
	if item == nil {
		return Err<Resource>NotFound
	}
	if item.CustomerID != customerID {
		return Err<Resource>NotOwned
	}
	return s.<resource>Repo.Delete(ctx, id)
}
```

**Rules:**
- Sentinel errors at package level: `Err<Resource>NotFound`, `Err<Resource>NotOwned`
- Take repository interfaces (not concrete types)
- Input structs defined in service package with `validate` tags
- Update input uses pointer fields (`*string`) for partial updates
- Every mutating method checks ownership: `nil` check → `NotFound`, `CustomerID` mismatch → `NotOwned`
- Wrap errors with `fmt.Errorf("verb context: %w", err)` to preserve sentinel errors
- Convert nil list results to empty slices: `if items == nil { items = []*T{} }`

### Step 5: Add Mock Repository

**File:** `backend/internal/service/mocks_test.go` (append to existing file)

```go
// Mock<Resource>Repo
type Mock<Resource>Repo struct{ mock.Mock }

func (m *Mock<Resource>Repo) Create(ctx context.Context, item *domain.<Resource>) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}
func (m *Mock<Resource>Repo) GetByID(ctx context.Context, id string) (*domain.<Resource>, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.<Resource>), args.Error(1)
}
func (m *Mock<Resource>Repo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.<Resource>, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.<Resource>), args.Error(1)
}
func (m *Mock<Resource>Repo) Update(ctx context.Context, item *domain.<Resource>) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}
func (m *Mock<Resource>Repo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
```

**Rules:**
- Mocks live in `service` package (NOT `_test.go` file — they are in `mocks_test.go` which is test-only)
- `args.Get(0) == nil` guard before type assertion for pointer returns
- Every mock method must match the interface signature exactly
- Use `testify/mock`

### Step 6: Create Service Tests

**File:** `backend/internal/service/<resource>_service_test.go`

```go
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTest<Resource>Service() (*<Resource>Service, *Mock<Resource>Repo) {
	repo := new(Mock<Resource>Repo)
	svc := New<Resource>Service(repo)
	return svc, repo
}

func Test<Resource>Service_Create(t *testing.T) {
	svc, repo := newTest<Resource>Service()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.<Resource>")).Return(nil)

	item, err := svc.Create(context.Background(), "cust-1", Create<Resource>Input{
		Name: "Test Item",
	})

	assert.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "cust-1", item.CustomerID)
	repo.AssertExpectations(t)
}

func Test<Resource>Service_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		itemID     string
		setup      func(*Mock<Resource>Repo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			itemID:     "item-1",
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "item-1").Return(&domain.<Resource>{
					ID: "item-1", CustomerID: "cust-1", Name: "Test",
				}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			itemID:     "nonexistent",
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: Err<Resource>NotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			itemID:     "item-1",
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "item-1").Return(&domain.<Resource>{
					ID: "item-1", CustomerID: "cust-1",
				}, nil)
			},
			wantErr: Err<Resource>NotOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTest<Resource>Service()
			tt.setup(repo)

			item, err := svc.GetByID(context.Background(), tt.customerID, tt.itemID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, item)
			}
			repo.AssertExpectations(t)
		})
	}
}

func Test<Resource>Service_Delete(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		itemID     string
		setup      func(*Mock<Resource>Repo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			itemID:     "item-1",
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "item-1").Return(&domain.<Resource>{
					ID: "item-1", CustomerID: "cust-1",
				}, nil)
				r.On("Delete", mock.Anything, "item-1").Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			itemID:     "nonexistent",
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: Err<Resource>NotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			itemID:     "item-1",
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "item-1").Return(&domain.<Resource>{
					ID: "item-1", CustomerID: "cust-1",
				}, nil)
			},
			wantErr: Err<Resource>NotOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTest<Resource>Service()
			tt.setup(repo)

			err := svc.Delete(context.Background(), tt.customerID, tt.itemID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
```

**Rules:**
- Table-driven tests with `setup func` per case
- Fresh mocks per subtest: `svc, repo := newTest<Resource>Service()` inside `t.Run`
- Use `assert.ErrorIs` for sentinel error checks
- `repo.AssertExpectations(t)` at end of every test/subtest
- Helper constructor: `newTest<Resource>Service()` returns `(*Service, *MockRepo)`

### Step 7: Create Handler + Wire Router

**File:** `backend/internal/handler/<resource>_handler.go`

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type <Resource>Handler struct {
	<resource>Service *service.<Resource>Service
	validate          *validator.Validate
}

func New<Resource>Handler(<resource>Service *service.<Resource>Service) *<Resource>Handler {
	return &<Resource>Handler{
		<resource>Service: <resource>Service,
		validate:          validator.New(),
	}
}

func (h *<Resource>Handler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	items, err := h.<resource>Service.List(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list <resource>s")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: items})
}

func (h *<Resource>Handler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.Create<Resource>Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.<resource>Service.Create(r.Context(), customerID, input)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to create <resource>")
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: item})
}

func (h *<Resource>Handler) Update(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	var input service.Update<Resource>Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.<resource>Service.Update(r.Context(), customerID, id, input)
	if err != nil {
		if errors.Is(err, service.Err<Resource>NotFound) {
			ErrorJSON(w, http.StatusNotFound, "<resource> not found")
			return
		}
		if errors.Is(err, service.Err<Resource>NotOwned) {
			ErrorJSON(w, http.StatusForbidden, "<resource> not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to update <resource>")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: item})
}

func (h *<Resource>Handler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	id := chi.URLParam(r, "id")

	err := h.<resource>Service.Delete(r.Context(), customerID, id)
	if err != nil {
		if errors.Is(err, service.Err<Resource>NotFound) {
			ErrorJSON(w, http.StatusNotFound, "<resource> not found")
			return
		}
		if errors.Is(err, service.Err<Resource>NotOwned) {
			ErrorJSON(w, http.StatusForbidden, "<resource> not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete <resource>")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "<resource> deleted"})
}
```

**Router wiring** — add to `backend/internal/router/router.go` inside the `New()` function:

```go
// In Repositories section:
<resource>Repo := postgres.New<Resource>Repo(pool)

// In Services section:
<resource>Service := service.New<Resource>Service(<resource>Repo)

// In Handlers section:
<resource>Handler := handler.New<Resource>Handler(<resource>Service)

// In JWT-protected routes group:
r.Route("/<resource>s", func(r chi.Router) {
    r.Get("/", <resource>Handler.List)
    r.Post("/", <resource>Handler.Create)
    r.Put("/{id}", <resource>Handler.Update)
    r.Delete("/{id}", <resource>Handler.Delete)
})
```

**Rules:**
- Handlers take concrete service pointers (not interfaces)
- Own `validator.Validate` instance
- Extract customerID via `middleware.GetCustomerID(r.Context())`
- Error mapping: `NotFound` → 404, `NotOwned` → 403, everything else → 500
- Use `JSON()` / `ErrorJSON()` from `handler/response.go`
- DI wiring order in router: repo → service → handler → routes
- Place routes in JWT-protected group (default)

## Verification

After scaffolding all files, run:

```bash
cd backend && go build ./cmd/server && go vet ./... && go test ./internal/service/...
```

All three commands must pass.

## Related

- Built-in `golang-patterns` skill for general Go best practices
- Built-in `golang-testing` skill for testing patterns
- `db-migration` skill to create the database table first
- `api-endpoint` skill to add methods to an existing resource
