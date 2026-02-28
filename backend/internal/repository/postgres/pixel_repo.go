package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type PixelRepo struct {
	pool *pgxpool.Pool
}

func NewPixelRepo(pool *pgxpool.Pool) *PixelRepo {
	return &PixelRepo{pool: pool}
}

func (r *PixelRepo) Create(ctx context.Context, p *domain.Pixel) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO pixels (customer_id, fb_pixel_id, fb_access_token, name, backup_pixel_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, is_active, status, created_at, updated_at`,
		p.CustomerID, p.FBPixelID, p.FBAccessToken, p.Name, p.BackupPixelID,
	).Scan(&p.ID, &p.IsActive, &p.Status, &p.CreatedAt, &p.UpdatedAt)
}

func (r *PixelRepo) GetByID(ctx context.Context, id string) (*domain.Pixel, error) {
	p := &domain.Pixel{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, fb_pixel_id, fb_access_token, name, is_active, status, backup_pixel_id, created_at, updated_at
		 FROM pixels WHERE id = $1`, id,
	).Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.FBAccessToken, &p.Name, &p.IsActive, &p.Status, &p.BackupPixelID, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (r *PixelRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Pixel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, fb_pixel_id, fb_access_token, name, is_active, status, backup_pixel_id, created_at, updated_at
		 FROM pixels WHERE customer_id = $1 ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pixels []*domain.Pixel
	for rows.Next() {
		p := &domain.Pixel{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.FBAccessToken, &p.Name, &p.IsActive, &p.Status, &p.BackupPixelID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pixels = append(pixels, p)
	}
	return pixels, rows.Err()
}

func (r *PixelRepo) Update(ctx context.Context, p *domain.Pixel) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pixels SET fb_pixel_id = $2, fb_access_token = $3, name = $4, is_active = $5, status = $6, backup_pixel_id = $7, updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.FBPixelID, p.FBAccessToken, p.Name, p.IsActive, p.Status, p.BackupPixelID,
	)
	return err
}

func (r *PixelRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pixels WHERE id = $1`, id)
	return err
}
