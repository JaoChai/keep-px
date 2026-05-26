//go:build integration

// Package testutil provides shared helpers for integration tests that exercise
// real Postgres. It is only compiled when the `integration` build tag is set.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDatabaseURL = "postgres://pixlinks:pixlinks_dev@localhost:5432/pixlinks?sslmode=disable"

var (
	migrationsOnce sync.Once
	migrationsErr  error
)

// NewTestPool returns a pgxpool connected to the integration test database.
// It runs migrations once per process and registers a cleanup that closes the
// pool when the test finishes.
//
// Set TEST_DATABASE_URL to point at a Postgres instance (default:
// `postgres://pixlinks:pixlinks_dev@localhost:5432/pixlinks?sslmode=disable`).
// The test is skipped when the DB is unreachable so unit-only environments do
// not fail.
func NewTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultTestDatabaseURL
	}

	migrationsOnce.Do(func() {
		migrationsErr = runMigrations(dbURL)
	})
	if migrationsErr != nil {
		t.Skipf("skipping integration test — migrations failed (DB unreachable?): %v", migrationsErr)
	}

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parse test DB config: %v", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Skipf("skipping integration test — DB unreachable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test — DB ping failed: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

// TruncateAll wipes every table affected by integration tests. Call at the
// start of each test to guarantee isolation.
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Discover tables at runtime — schema may drift over time.
	// Excludes schema_migrations so golang-migrate keeps its bookkeeping.
	rows, err := pool.Query(ctx,
		`SELECT tablename FROM pg_tables
		 WHERE schemaname = 'public' AND tablename != 'schema_migrations'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}
	if len(tables) == 0 {
		return
	}
	// CASCADE handles FK chains. Order doesn't matter with CASCADE.
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinTables(tables))
	if _, err := pool.Exec(ctx, stmt); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func joinTables(tables []string) string {
	out := ""
	for i, name := range tables {
		if i > 0 {
			out += ", "
		}
		out += name
	}
	return out
}

// runMigrations applies all up migrations from backend/db/migrations.
// Path is resolved relative to this file so tests work from any working dir.
func runMigrations(databaseURL string) error {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("cannot resolve testutil file path")
	}
	// thisFile = .../backend/internal/testutil/pgtest.go
	// migrations  = .../backend/db/migrations
	backendRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	migrationsPath := filepath.Join(backendRoot, "db", "migrations")
	if _, err := os.Stat(migrationsPath); err != nil {
		return fmt.Errorf("stat migrations dir %q: %w", migrationsPath, err)
	}

	m, err := migrate.New("file://"+migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
