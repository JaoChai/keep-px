# Keep-PX Backend

## Running Integration Tests

Integration tests ในโฟลเดอร์ `internal/repository/postgres/` ใช้ Postgres จริง เพื่อ verify SQL queries และ token encryption

### Local

```bash
# 1. รัน Postgres ผ่าน docker compose
docker compose up -d postgres

# 2. รัน integration tests (build tag จำเป็น)
cd backend
TEST_DATABASE_URL="postgres://pixlinks:pixlinks_dev@localhost:5432/pixlinks?sslmode=disable" \
  go test -race -tags=integration -v ./internal/repository/postgres/...
```

### CI

GitHub Actions รันอัตโนมัติผ่าน Postgres service container — ไม่ต้อง config เพิ่ม

### ข้อควรรู้

- ทุก test จะ `TRUNCATE` ตารางก่อน → state isolated
- ถ้า DB ไม่ตอบ test จะถูก `t.Skip` แทนที่จะ fail
- Build tag `//go:build integration` แยก integration test ออกจาก unit test เดิม
