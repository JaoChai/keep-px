package postgres

import (
	"context"
	"errors"

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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO sale_pages (customer_id, name, slug, template_name, content, is_published)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		p.CustomerID, p.Name, p.Slug, p.TemplateName, p.Content, p.IsPublished,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return err
	}

	if err := r.setPixels(ctx, tx, p.ID, p.PixelIDs); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *SalePageRepo) GetByID(ctx context.Context, id string) (*domain.SalePage, error) {
	p := &domain.SalePage{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, name, slug, template_name, content, is_published, created_at, updated_at
		 FROM sale_pages WHERE id = $1`, id,
	).Scan(&p.ID, &p.CustomerID, &p.Name, &p.Slug, &p.TemplateName, &p.Content, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pixelIDs, err := r.loadPixelIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.PixelIDs = pixelIDs

	return p, nil
}

func (r *SalePageRepo) GetBySlug(ctx context.Context, slug string) (*domain.SalePage, error) {
	p := &domain.SalePage{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, name, slug, template_name, content, is_published, created_at, updated_at
		 FROM sale_pages WHERE slug = $1`, slug,
	).Scan(&p.ID, &p.CustomerID, &p.Name, &p.Slug, &p.TemplateName, &p.Content, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pixelIDs, err := r.loadPixelIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.PixelIDs = pixelIDs

	return p, nil
}

func (r *SalePageRepo) CountByCustomerID(ctx context.Context, customerID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sale_pages WHERE customer_id = $1`, customerID,
	).Scan(&count)
	return count, err
}

func (r *SalePageRepo) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.SalePage, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sale_pages WHERE customer_id = $1`, customerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, name, slug, template_name, content, is_published, created_at, updated_at
		 FROM sale_pages WHERE customer_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		customerID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pages []*domain.SalePage
	var pageIDs []string
	for rows.Next() {
		p := &domain.SalePage{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.Name, &p.Slug, &p.TemplateName, &p.Content, &p.IsPublished, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		pages = append(pages, p)
		pageIDs = append(pageIDs, p.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	pixelMap, err := r.loadPixelIDsForPages(ctx, pageIDs)
	if err != nil {
		return nil, 0, err
	}
	for _, p := range pages {
		p.PixelIDs = pixelMap[p.ID]
	}

	return pages, total, nil
}

func (r *SalePageRepo) Update(ctx context.Context, p *domain.SalePage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`UPDATE sale_pages SET name = $2, slug = $3, template_name = $4, content = $5, is_published = $6, updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.Name, p.Slug, p.TemplateName, p.Content, p.IsPublished,
	)
	if err != nil {
		return err
	}

	if err := r.setPixels(ctx, tx, p.ID, p.PixelIDs); err != nil {
		return err
	}

	return tx.Commit(ctx)
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

func (r *SalePageRepo) setPixels(ctx context.Context, tx pgx.Tx, salePageID string, pixelIDs []string) error {
	_, err := tx.Exec(ctx, `DELETE FROM sale_page_pixels WHERE sale_page_id = $1`, salePageID)
	if err != nil {
		return err
	}
	for i, pid := range pixelIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO sale_page_pixels (sale_page_id, pixel_id, position) VALUES ($1, $2, $3)`,
			salePageID, pid, i,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *SalePageRepo) loadPixelIDsForPages(ctx context.Context, pageIDs []string) (map[string][]string, error) {
	if len(pageIDs) == 0 {
		return map[string][]string{}, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT sale_page_id, pixel_id FROM sale_page_pixels WHERE sale_page_id = ANY($1) ORDER BY sale_page_id, position ASC`,
		pageIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]string, len(pageIDs))
	for _, pid := range pageIDs {
		result[pid] = []string{}
	}
	for rows.Next() {
		var spID, pxID string
		if err := rows.Scan(&spID, &pxID); err != nil {
			return nil, err
		}
		result[spID] = append(result[spID], pxID)
	}
	return result, rows.Err()
}

func (r *SalePageRepo) loadPixelIDs(ctx context.Context, salePageID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pixel_id FROM sale_page_pixels WHERE sale_page_id = $1 ORDER BY position ASC`,
		salePageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, rows.Err()
}
