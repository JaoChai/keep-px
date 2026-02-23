---
name: api-endpoint
description: Add a new endpoint method to an existing Keep-PX backend resource (service + handler + route + test)
---

# API Endpoint

## When to Activate

Activate this skill when the user says:
- "Add endpoint to [existing resource]"
- "New route for [action] on [resource]"
- "Add [method] to [resource] API"
- "I need an endpoint that [does something] for [resource]"

This skill is for adding a method to an **existing** resource. For creating a brand new resource from scratch, use `go-service-scaffold` instead.

## Step-by-Step Workflow

### Step 1: Read Existing Code

Before making any changes, read the existing files to understand current structure:

1. `backend/internal/service/<resource>_service.go` — existing service methods and sentinel errors
2. `backend/internal/handler/<resource>_handler.go` — existing handler methods
3. `backend/internal/router/router.go` — current route registration
4. `backend/internal/repository/interfaces.go` — repository interface (may need new method)

### Step 2: Add Service Method

**File:** `backend/internal/service/<resource>_service.go`

Add new input struct (if needed) and method:

```go
type <Action><Resource>Input struct {
	Field string `json:"field" validate:"required"`
}

func (s *<Resource>Service) <Action>(ctx context.Context, customerID string, input <Action><Resource>Input) (*domain.<Resource>, error) {
	// 1. Fetch the resource
	item, err := s.<resource>Repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("<action> <resource>: %w", err)
	}

	// 2. Check existence
	if item == nil {
		return nil, Err<Resource>NotFound
	}

	// 3. Check ownership
	if item.CustomerID != customerID {
		return nil, Err<Resource>NotOwned
	}

	// 4. Business logic
	// ... modify item ...

	// 5. Persist
	if err := s.<resource>Repo.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("<action> <resource>: %w", err)
	}
	return item, nil
}
```

**Rules:**
- Follow existing sentinel error pattern — reuse `Err<Resource>NotFound` / `Err<Resource>NotOwned`
- Add new sentinel errors only if needed (e.g., `Err<Resource>AlreadyActive`)
- Ownership check pattern: nil → NotFound, CustomerID mismatch → NotOwned
- Wrap all repo errors: `fmt.Errorf("verb context: %w", err)`
- If new repo method needed, add to interface first (Step 2b)

**Step 2b (if needed): Add Repository Method**

If the endpoint needs a new query, add to the repository interface in `backend/internal/repository/interfaces.go`:

```go
// Add to existing <Resource>Repository interface:
<NewMethod>(ctx context.Context, params...) (returnType, error)
```

Then implement in `backend/internal/repository/postgres/<resource>_repo.go`:

```go
func (r *<Resource>Repo) <NewMethod>(ctx context.Context, params...) (returnType, error) {
	// pgx query implementation
}
```

And add to mock in `backend/internal/service/mocks_test.go`:

```go
func (m *Mock<Resource>Repo) <NewMethod>(ctx context.Context, params...) (returnType, error) {
	args := m.Called(ctx, params...)
	// nil guard for pointer returns
	return args.Get(0).(returnType), args.Error(1)
}
```

### Step 3: Add Handler Method

**File:** `backend/internal/handler/<resource>_handler.go`

```go
func (h *<Resource>Handler) <Action>(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	// For URL params:
	id := chi.URLParam(r, "id")

	// For request body:
	var input service.<Action><Resource>Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	// Call service
	result, err := h.<resource>Service.<Action>(r.Context(), customerID, id, input)
	if err != nil {
		// Error mapping
		if errors.Is(err, service.Err<Resource>NotFound) {
			ErrorJSON(w, http.StatusNotFound, "<resource> not found")
			return
		}
		if errors.Is(err, service.Err<Resource>NotOwned) {
			ErrorJSON(w, http.StatusForbidden, "<resource> not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to <action> <resource>")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: result})
}
```

**Error mapping reference:**
| Sentinel Error | HTTP Status | Message |
|---|---|---|
| `Err*NotFound` | 404 | `<resource> not found` |
| `Err*NotOwned` | 403 | `<resource> not owned by you` |
| `Err*Invalid*` | 400 | descriptive message |
| all others | 500 | `failed to <action> <resource>` |

**Status codes for successful responses:**
| Action | Status | Response |
|---|---|---|
| GET (single) | 200 | `APIResponse{Data: item}` |
| GET (list) | 200 | `APIResponse{Data: items}` or `PaginatedResponse{...}` |
| POST (create) | 201 | `APIResponse{Data: item}` |
| PUT (update) | 200 | `APIResponse{Data: item}` |
| DELETE | 200 | `APIResponse{Message: "<resource> deleted"}` |
| POST (action) | 200 | `APIResponse{Data: result}` |

### Step 4: Register Route

**File:** `backend/internal/router/router.go`

Add the route in the correct auth group:

**Public routes** (no auth):
```go
r.Route("/auth", func(r chi.Router) {
    r.Post("/new-endpoint", handler.Method)
})
```

**API Key routes** (SDK):
```go
r.Route("/events", func(r chi.Router) {
    r.Use(middleware.APIKeyAuth(customerRepo))
    r.Post("/new-endpoint", handler.Method)
})
```

**JWT routes** (Dashboard — most common):
```go
// Inside the existing r.Group with middleware.JWTAuth:
r.Route("/<resource>s", func(r chi.Router) {
    // ... existing routes ...
    r.Post("/{id}/<action>", <resource>Handler.<Action>)  // new route
})
```

**Route patterns:**
- CRUD: `GET /`, `POST /`, `PUT /{id}`, `DELETE /{id}`
- Action on resource: `POST /{id}/<action>` (e.g., `POST /{id}/activate`)
- Sub-resource: `GET /{id}/<sub-resource>s` (e.g., `GET /{pixelId}/rules`)
- URL param name: `{id}` for primary resource, `{pixelId}` etc. for parent

### Step 5: Add Tests

**File:** `backend/internal/service/<resource>_service_test.go` (append to existing file)

```go
func Test<Resource>Service_<Action>(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		input      <Action><Resource>Input
		setup      func(*Mock<Resource>Repo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			input:      <Action><Resource>Input{...},
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, "item-1").Return(&domain.<Resource>{
					ID: "item-1", CustomerID: "cust-1",
				}, nil)
				// ... additional mock expectations ...
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			input:      <Action><Resource>Input{...},
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantErr: Err<Resource>NotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			input:      <Action><Resource>Input{...},
			setup: func(r *Mock<Resource>Repo) {
				r.On("GetByID", mock.Anything, mock.Anything).Return(&domain.<Resource>{
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

			result, err := svc.<Action>(context.Background(), tt.customerID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
			repo.AssertExpectations(t)
		})
	}
}
```

**Rules:**
- Follow existing test patterns in the file
- Always test: success, not found, not owned
- Fresh mocks per subtest
- `assert.ErrorIs` for sentinel errors
- `repo.AssertExpectations(t)` at end

## Verification

```bash
cd backend && go build ./cmd/server && go vet ./... && go test ./internal/service/...
```

All three commands must pass.

## Related

- `go-service-scaffold` skill for creating a brand new resource from scratch
- `db-migration` skill if the endpoint requires schema changes
- Built-in `golang-patterns` skill for general Go best practices
- Built-in `/go-review` command for post-implementation code review
