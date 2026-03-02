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

func scanCustomer(row pgx.Row) (*domain.Customer, error) {
	c := &domain.Customer{}
	var passwordHash *string
	err := row.Scan(
		&c.ID, &c.Email, &passwordHash, &c.GoogleID,
		&c.Name, &c.APIKey, &c.Plan, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if passwordHash != nil {
		c.PasswordHash = *passwordHash
	}
	return c, nil
}

func (r *CustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	var passwordHash *string
	if c.PasswordHash != "" {
		passwordHash = &c.PasswordHash
	}
	return r.pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, google_id, name, api_key, plan)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		c.Email, passwordHash, c.GoogleID, c.Name, c.APIKey, c.Plan,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepo) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	return scanCustomer(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, google_id, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE id = $1`, id,
	))
}

func (r *CustomerRepo) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	return scanCustomer(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, google_id, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE email = $1`, email,
	))
}

func (r *CustomerRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.Customer, error) {
	return scanCustomer(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, google_id, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE google_id = $1`, googleID,
	))
}

func (r *CustomerRepo) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Customer, error) {
	return scanCustomer(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, google_id, name, api_key, plan, created_at, updated_at
		 FROM customers WHERE api_key = $1`, apiKey,
	))
}

func (r *CustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE customers SET email = $2, google_id = $3, name = $4, plan = $5, updated_at = NOW()
		 WHERE id = $1`,
		c.ID, c.Email, c.GoogleID, c.Name, c.Plan,
	)
	return err
}
