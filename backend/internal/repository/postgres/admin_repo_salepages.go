package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

func (r *AdminRepo) ListAllSalePages(ctx context.Context, search, customerID string, published *bool, limit, offset int) ([]*domain.AdminSalePage, int, error) {
	baseWhere := baseWhereTrue
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (sp.name ILIKE $%d OR sp.slug ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if customerID != "" {
		baseWhere += fmt.Sprintf(" AND sp.customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}
	if published != nil {
		baseWhere += fmt.Sprintf(" AND sp.is_published = $%d", argIdx)
		args = append(args, *published)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM sale_pages sp " + baseWhere
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sale pages: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT sp.id, sp.customer_id, sp.name, sp.slug, sp.template_name, sp.content, sp.is_published, sp.created_at, sp.updated_at,
		        c.email, c.name,
		        COALESCE(ec.cnt, 0) AS event_count
		 FROM sale_pages sp
		 JOIN customers c ON c.id = sp.customer_id
		 LEFT JOIN (
		   SELECT spp.sale_page_id, COUNT(*) AS cnt
		   FROM sale_page_pixels spp
		   JOIN pixel_events pe ON pe.pixel_id = spp.pixel_id
		   GROUP BY spp.sale_page_id
		 ) ec ON ec.sale_page_id = sp.id
		 %s ORDER BY sp.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sale pages: %w", err)
	}
	defer rows.Close()

	var pages []*domain.AdminSalePage
	for rows.Next() {
		sp := &domain.AdminSalePage{}
		if err := rows.Scan(
			&sp.ID, &sp.CustomerID, &sp.Name, &sp.Slug, &sp.TemplateName, &sp.Content, &sp.IsPublished, &sp.CreatedAt, &sp.UpdatedAt,
			&sp.CustomerEmail, &sp.CustomerName,
			&sp.EventCount,
		); err != nil {
			return nil, 0, err
		}
		pages = append(pages, sp)
	}
	if pages == nil {
		pages = []*domain.AdminSalePage{}
	}
	return pages, total, rows.Err()
}

func (r *AdminRepo) GetSalePageAdminDetail(ctx context.Context, id string) (*domain.AdminSalePageDetail, error) {
	detail := &domain.AdminSalePageDetail{SalePage: &domain.SalePage{}}

	err := r.pool.QueryRow(ctx,
		`SELECT sp.id, sp.customer_id, sp.name, sp.slug, sp.template_name, sp.content, sp.is_published, sp.created_at, sp.updated_at,
		        c.email, c.name
		 FROM sale_pages sp JOIN customers c ON c.id = sp.customer_id
		 WHERE sp.id = $1`, id,
	).Scan(
		&detail.SalePage.ID, &detail.SalePage.CustomerID, &detail.SalePage.Name, &detail.SalePage.Slug,
		&detail.SalePage.TemplateName, &detail.SalePage.Content, &detail.SalePage.IsPublished, &detail.SalePage.CreatedAt, &detail.SalePage.UpdatedAt,
		&detail.CustomerEmail, &detail.CustomerName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sale page detail: %w", err)
	}

	var linkedPixels []*domain.Pixel
	var eventCount int64

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT p.id, p.customer_id, p.fb_pixel_id, p.name, p.is_active, p.backup_pixel_id, p.test_event_code, p.created_at, p.updated_at
			 FROM pixels p JOIN sale_page_pixels spp ON spp.pixel_id = p.id
			 WHERE spp.sale_page_id = $1 ORDER BY spp.position`, id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p := &domain.Pixel{}
			if err := rows.Scan(&p.ID, &p.CustomerID, &p.FBPixelID, &p.Name, &p.IsActive, &p.BackupPixelID, &p.TestEventCode, &p.CreatedAt, &p.UpdatedAt); err != nil {
				return err
			}
			linkedPixels = append(linkedPixels, p)
		}
		return rows.Err()
	})

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events pe
			 JOIN sale_page_pixels spp ON spp.pixel_id = pe.pixel_id
			 WHERE spp.sale_page_id = $1`, id,
		).Scan(&eventCount)
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get sale page detail sub-queries: %w", err)
	}

	detail.EventCount = eventCount
	if linkedPixels == nil {
		linkedPixels = []*domain.Pixel{}
	}
	detail.LinkedPixels = linkedPixels

	return detail, nil
}

func (r *AdminRepo) SetSalePagePublished(ctx context.Context, id string, published bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE sale_pages SET is_published = $2, updated_at = NOW() WHERE id = $1`, id, published,
	)
	if err != nil {
		return fmt.Errorf("set sale page published: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *AdminRepo) DeleteSalePageByAdmin(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sale_pages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sale page: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
