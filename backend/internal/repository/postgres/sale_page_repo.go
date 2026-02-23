package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type SalePageRepo struct {
	pool *pgxpool.Pool
}

func NewSalePageRepo(pool *pgxpool.Pool) *SalePageRepo {
	return &SalePageRepo{pool: pool}
}

func (r *SalePageRepo) Create(ctx context.Context, p *domain.SalePage) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO sale_pages (customer_id, pixel_id, name, slug, template_name, content, is_published)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		p.CustomerID, p.PixelID, p.Name, p.Slug, p.TemplateName, p.Content, p.IsPublished,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *SalePageRepo) GetByID(ctx context.Context, id string) (*domain.SalePage, error) {
	p := &domain.SalePage{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, pixel_id, name, slug, template_name, content, is_published, created_at, updated_at
		 FROM sale_pages WHERE id = $1`, id,
	).Scan(&p.ID, &p.CustomerID, &p.PixelID, &p.Name, &p.Slug, &p.TemplateName, &p.Content, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (r *SalePageRepo) GetBySlug(ctx context.Context, slug string) (*domain.SalePage, error) {
	p := &domain.SalePage{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, pixel_id, name, slug, template_name, content, is_published, created_at, updated_at
		 FROM sale_pages WHERE slug = $1`, slug,
	).Scan(&p.ID, &p.CustomerID, &p.PixelID, &p.Name, &p.Slug, &p.TemplateName, &p.Content, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (r *SalePageRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.SalePage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, pixel_id, name, slug, template_name, content, is_published, created_at, updated_at
		 FROM sale_pages WHERE customer_id = $1 ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*domain.SalePage
	for rows.Next() {
		p := &domain.SalePage{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.PixelID, &p.Name, &p.Slug, &p.TemplateName, &p.Content, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (r *SalePageRepo) Update(ctx context.Context, p *domain.SalePage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sale_pages SET pixel_id = $2, name = $3, slug = $4, template_name = $5, content = $6, is_published = $7, updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.PixelID, p.Name, p.Slug, p.TemplateName, p.Content, p.IsPublished,
	)
	return err
}

func (r *SalePageRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sale_pages WHERE id = $1`, id)
	return err
}

func (r *SalePageRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sale_pages WHERE slug = $1)`, slug,
	).Scan(&exists)
	return exists, err
}
