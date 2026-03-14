package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/crypto"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type PixelRepo struct {
	pool      *pgxpool.Pool
	encryptor *crypto.TokenEncryptor
}

func NewPixelRepo(pool *pgxpool.Pool, encryptor *crypto.TokenEncryptor) *PixelRepo {
	return &PixelRepo{pool: pool, encryptor: encryptor}
}

func (r *PixelRepo) encryptToken(token string) (string, error) {
	if r.encryptor == nil || token == "" {
		if r.encryptor == nil && token != "" {
			slog.Warn("storing FB access token without encryption — set TOKEN_ENCRYPTION_KEY")
		}
		return token, nil
	}
	return r.encryptor.Encrypt(token)
}

func (r *PixelRepo) decryptToken(token string) (string, error) {
	if r.encryptor == nil || token == "" {
		return token, nil
	}
	return r.encryptor.Decrypt(token)
}

func (r *PixelRepo) Create(ctx context.Context, p *domain.Pixel) error {
	encToken, err := r.encryptToken(p.FBAccessToken)
	if err != nil {
		return fmt.Errorf("encrypt token: %w", err)
	}
	return r.pool.QueryRow(ctx,
		`INSERT INTO pixels (customer_id, fb_pixel_id, fb_access_token, name, backup_pixel_id, test_event_code)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, is_active, created_at, updated_at`,
		p.CustomerID, p.FBPixelID, encToken, p.Name, p.BackupPixelID, p.TestEventCode,
	).Scan(&p.ID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
}

func (r *PixelRepo) GetByID(ctx context.Context, id string) (*domain.Pixel, error) {
	p := &domain.Pixel{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, fb_pixel_id, fb_access_token, name, is_active, backup_pixel_id, test_event_code, created_at, updated_at
		 FROM pixels WHERE id = $1`, id,
	).Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.FBAccessToken, &p.Name, &p.IsActive, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dec, err := r.decryptToken(p.FBAccessToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt token for pixel %s: %w", p.ID, err)
	}
	p.FBAccessToken = dec
	return p, nil
}

func (r *PixelRepo) GetByIDs(ctx context.Context, ids []string) ([]*domain.Pixel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, fb_pixel_id, fb_access_token, name, is_active, backup_pixel_id, test_event_code, created_at, updated_at
		 FROM pixels WHERE id = ANY($1)`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pixels []*domain.Pixel
	for rows.Next() {
		p := &domain.Pixel{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.FBAccessToken, &p.Name, &p.IsActive, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		dec, err := r.decryptToken(p.FBAccessToken)
		if err != nil {
			return nil, fmt.Errorf("decrypt token for pixel %s: %w", p.ID, err)
		}
		p.FBAccessToken = dec
		pixels = append(pixels, p)
	}
	return pixels, rows.Err()
}

func (r *PixelRepo) CountByCustomerID(ctx context.Context, customerID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pixels WHERE customer_id = $1`, customerID,
	).Scan(&count)
	return count, err
}

func (r *PixelRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.Pixel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, fb_pixel_id, fb_access_token, name, is_active, backup_pixel_id, test_event_code, created_at, updated_at
		 FROM pixels WHERE customer_id = $1 ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pixels []*domain.Pixel
	for rows.Next() {
		p := &domain.Pixel{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.FBAccessToken, &p.Name, &p.IsActive, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		dec, err := r.decryptToken(p.FBAccessToken)
		if err != nil {
			return nil, fmt.Errorf("decrypt token for pixel %s: %w", p.ID, err)
		}
		p.FBAccessToken = dec
		pixels = append(pixels, p)
	}
	return pixels, rows.Err()
}

func (r *PixelRepo) Update(ctx context.Context, p *domain.Pixel) error {
	encToken, err := r.encryptToken(p.FBAccessToken)
	if err != nil {
		return fmt.Errorf("encrypt token: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE pixels SET fb_pixel_id = $2, fb_access_token = $3, name = $4, is_active = $5, backup_pixel_id = $6, test_event_code = $7, updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.FBPixelID, encToken, p.Name, p.IsActive, p.BackupPixelID, p.TestEventCode,
	)
	return err
}

func (r *PixelRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pixels WHERE id = $1`, id)
	return err
}
