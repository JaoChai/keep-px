package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type CustomerRepo struct {
	pool *pgxpool.Pool
}

func NewCustomerRepo(pool *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{pool: pool}
}

func (r *CustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, name, api_key, plan)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		c.Email, c.PasswordHash, c.Name, c.APIKey, c.Plan,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepo) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	c := &domain.Customer{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Email, &c.PasswordHash, &c.Name, &c.APIKey, &c.Plan, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *CustomerRepo) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	c := &domain.Customer{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE email = $1`, email,
	).Scan(&c.ID, &c.Email, &c.PasswordHash, &c.Name, &c.APIKey, &c.Plan, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *CustomerRepo) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Customer, error) {
	c := &domain.Customer{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE api_key = $1`, apiKey,
	).Scan(&c.ID, &c.Email, &c.PasswordHash, &c.Name, &c.APIKey, &c.Plan, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *CustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE customers SET email = $2, name = $3, plan = $4, updated_at = NOW()
		 WHERE id = $1`,
		c.ID, c.Email, c.Name, c.Plan,
	)
	return err
}
