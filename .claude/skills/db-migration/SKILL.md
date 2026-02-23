---
name: db-migration
description: Create numbered SQL migration files following Keep-PX conventions, with optional sqlc query generation
---

# Database Migration

## When to Activate

Activate this skill when the user says:
- "Add table/column/index"
- "Create migration for [feature]"
- "Write SQL for [resource]"
- "Add a [column] to [table]"
- "New migration"

## Step-by-Step Workflow

### Step 1: Determine Next Migration Number

Read existing migrations in `backend/db/migrations/` and find the highest number:

```bash
ls backend/db/migrations/ | sort | tail -2
```

Increment the number and zero-pad to 6 digits: `000001` → `000002` → `000003` etc.

### Step 2: Create `.up.sql`

**File:** `backend/db/migrations/<NNNNNN>_<descriptive_name>.up.sql`

Follow these conventions:

```sql
CREATE TABLE <table_name> (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    -- resource-specific columns
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    content JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_<table>_customer ON <table_name>(customer_id);
CREATE INDEX idx_<table>_<column> ON <table_name>(<column>);
```

**Rules:**
- Primary key: `UUID DEFAULT gen_random_uuid()`
- Foreign keys: `REFERENCES <parent>(id) ON DELETE CASCADE`
- Timestamps: `TIMESTAMPTZ DEFAULT NOW()` (not `TIMESTAMP`)
- Boolean defaults: explicit `DEFAULT true` or `DEFAULT false`
- JSONB defaults: `DEFAULT '{}'`
- Index naming: `idx_<table>_<column>`
- Indexes at bottom of file, after CREATE TABLE
- For ALTER TABLE migrations: one statement per logical change

**For column additions:**

```sql
ALTER TABLE <table_name> ADD COLUMN <column_name> <type> <constraints>;
CREATE INDEX idx_<table>_<column> ON <table_name>(<column>);
```

**For index-only migrations:**

```sql
CREATE INDEX idx_<table>_<column> ON <table_name>(<column>);
```

### Step 3: Create `.down.sql`

**File:** `backend/db/migrations/<NNNNNN>_<descriptive_name>.down.sql`

```sql
DROP TABLE IF EXISTS <table_name>;
```

**For column additions:**

```sql
ALTER TABLE <table_name> DROP COLUMN IF EXISTS <column_name>;
```

**For index-only migrations:**

```sql
DROP INDEX IF EXISTS idx_<table>_<column>;
```

**Rules:**
- Always use `IF EXISTS` to make idempotent
- Drop in reverse order of creation (indexes first if table also dropped)
- For tables with dependencies, drop dependent tables first

### Step 4: (Optional) Create sqlc Query File

Only if the user requests sqlc integration or if adding a new table that will use sqlc.

**File:** `backend/db/queries/<table>.sql`

```sql
-- name: Get<Resource>ByID :one
SELECT * FROM <table> WHERE id = $1;

-- name: List<Resource>sByCustomerID :many
SELECT * FROM <table> WHERE customer_id = $1 ORDER BY created_at DESC;

-- name: Create<Resource> :one
INSERT INTO <table> (customer_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: Update<Resource> :exec
UPDATE <table> SET name = $2, updated_at = NOW() WHERE id = $1;

-- name: Delete<Resource> :exec
DELETE FROM <table> WHERE id = $1;
```

**sqlc annotation rules:**
- `:one` for single row return
- `:many` for multiple rows
- `:exec` for no return value
- `:execresult` if you need rows affected
- Function names use PascalCase: `Get<Resource>ByID`, `List<Resource>s`

**Note:** The actual repository implementations in this project use hand-written SQL with pgx directly. sqlc queries serve as documentation and can optionally generate Go code.

### Step 5: (If Step 4 done) Run sqlc Generate

```bash
cd backend && sqlc generate
```

This regenerates files in `backend/db/generated/`. Do NOT manually edit those files.

## Verification

```bash
# Check files are created correctly
ls -la backend/db/migrations/ | tail -4

# If sqlc was used, verify generated code compiles
cd backend && go build ./...
```

## Database Schema Reference

Current tables in the database:
- `customers` — User accounts (id, email, password_hash, name, api_key, plan)
- `pixels` — Facebook Pixel configs (id, customer_id, fb_pixel_id, fb_access_token, name, is_active, status)
- `pixel_events` — Captured events (id, pixel_id, event_name, event_data, user_data, source_url, event_time)
- `event_rules` — Visual event setup rules (id, pixel_id, page_url, event_name, trigger_type, css_selector)
- `replay_sessions` — Event replay jobs (id, customer_id, source_pixel_id, target_pixel_id, status)
- `refresh_tokens` — JWT refresh tokens (id, customer_id, token_hash, expires_at)
- `sale_pages` — Landing page builder (id, customer_id, pixel_id, name, slug, template_name, content)

## Related

- `go-service-scaffold` skill to create the Go code that uses this table
- Built-in `postgres-patterns` skill for advanced PostgreSQL patterns
