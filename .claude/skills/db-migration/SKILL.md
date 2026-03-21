---
name: db-migration
description: Create SQL migration files, sqlc query generation — ใช้เมื่อ: เพิ่ม table, เพิ่ม column, สร้าง migration, แก้ schema, เพิ่มฟิลด์, สร้างตารางใหม่, แก้ database, เปลี่ยน index
---

# Database Migration

## When to Activate

- "Add table/column/index"
- "Create migration for [feature]"
- "New migration"

## Workflow

### Step 1: Find Next Number

```bash
ls backend/db/migrations/ | sort | tail -2
```

Increment, zero-pad to 6 digits.

### Step 2: Create Files

**`backend/db/migrations/<NNNNNN>_<name>.up.sql`** and **`<NNNNNN>_<name>.down.sql`**

Reference existing migrations in `backend/db/migrations/` for patterns.

### SQL Conventions

| Element | Convention |
|---------|-----------|
| Primary key | `UUID DEFAULT gen_random_uuid()` |
| Foreign key | `REFERENCES <parent>(id) ON DELETE CASCADE` |
| Timestamps | `TIMESTAMPTZ DEFAULT NOW()` (not `TIMESTAMP`) |
| Boolean | explicit `DEFAULT true/false` |
| JSONB | `DEFAULT '{}'` |
| Index naming | `idx_<table>_<column>` |
| M:M tables | Composite `PRIMARY KEY (a_id, b_id)` |

### Critical Rules

- **Idempotency**: Use `IF NOT EXISTS` / `IF EXISTS` (migrations may re-run on restart)
- **Down file**: Always use `IF EXISTS`, drop in reverse order
- **Column additions**: `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`

### Optional: sqlc

If using sqlc, add queries to `backend/db/queries/<table>.sql` then:

```bash
cd backend && sqlc generate
```

NEVER edit `db/generated/` manually.

## Current Schema

16 active tables, 24 migrations. See `backend/db/migrations/` for full schema.

## Related

- `go-service-scaffold` — create the Go code that uses this table
