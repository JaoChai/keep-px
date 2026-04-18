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

func (r *AdminRepo) ListAllPixels(ctx context.Context, search, customerID string, active *bool, limit, offset int) ([]*domain.AdminPixel, int, error) {
	baseWhere := baseWhereTrue
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.fb_pixel_id ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if customerID != "" {
		baseWhere += fmt.Sprintf(" AND p.customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}
	if active != nil {
		baseWhere += fmt.Sprintf(" AND p.is_active = $%d", argIdx)
		args = append(args, *active)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM pixels p "+baseWhere, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pixels: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT p.id, p.customer_id, p.fb_pixel_id, p.name, p.is_active, p.backup_pixel_id, p.test_event_code, p.created_at, p.updated_at,
		        c.email, c.name,
		        COALESCE(ec.cnt, 0) AS event_count,
		        COALESCE(sc.cnt, 0) AS sale_page_count
		 FROM pixels p
		 JOIN customers c ON c.id = p.customer_id
		 LEFT JOIN (
		   SELECT pixel_id, COUNT(*) AS cnt FROM pixel_events GROUP BY pixel_id
		 ) ec ON ec.pixel_id = p.id
		 LEFT JOIN (
		   SELECT pixel_id, COUNT(*) AS cnt FROM sale_page_pixels GROUP BY pixel_id
		 ) sc ON sc.pixel_id = p.id
		 %s ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pixels: %w", err)
	}
	defer rows.Close()

	var pixels []*domain.AdminPixel
	for rows.Next() {
		ap := &domain.AdminPixel{}
		if err := rows.Scan(
			&ap.ID, &ap.CustomerID, &ap.FBPixelID, &ap.Name, &ap.IsActive, &ap.BackupPixelID, &ap.TestEventCode, &ap.CreatedAt, &ap.UpdatedAt,
			&ap.CustomerEmail, &ap.CustomerName,
			&ap.EventCount, &ap.SalePageCount,
		); err != nil {
			return nil, 0, err
		}
		pixels = append(pixels, ap)
	}
	if pixels == nil {
		pixels = []*domain.AdminPixel{}
	}
	return pixels, total, rows.Err()
}

func (r *AdminRepo) GetPixelAdminDetail(ctx context.Context, id string) (*domain.AdminPixelDetail, error) {
	detail := &domain.AdminPixelDetail{Pixel: &domain.Pixel{}}

	err := r.pool.QueryRow(ctx,
		`SELECT p.id, p.customer_id, p.fb_pixel_id, p.name, p.is_active, p.backup_pixel_id, p.test_event_code, p.created_at, p.updated_at,
		        c.email, c.name
		 FROM pixels p JOIN customers c ON c.id = p.customer_id
		 WHERE p.id = $1`, id,
	).Scan(
		&detail.Pixel.ID, &detail.Pixel.CustomerID, &detail.Pixel.FBPixelID, &detail.Pixel.Name,
		&detail.Pixel.IsActive, &detail.Pixel.BackupPixelID, &detail.Pixel.TestEventCode,
		&detail.Pixel.CreatedAt, &detail.Pixel.UpdatedAt,
		&detail.CustomerEmail, &detail.CustomerName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get pixel detail: %w", err)
	}

	var eventCount int64
	var linkedSalePages []*domain.SalePage

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.pool.QueryRow(gCtx,
			`SELECT COUNT(*) FROM pixel_events WHERE pixel_id = $1`, id,
		).Scan(&eventCount)
	})

	g.Go(func() error {
		rows, err := r.pool.Query(gCtx,
			`SELECT sp.id, sp.customer_id, sp.name, sp.slug, sp.template_name, sp.content, sp.is_published, sp.created_at, sp.updated_at
			 FROM sale_pages sp JOIN sale_page_pixels spp ON spp.sale_page_id = sp.id
			 WHERE spp.pixel_id = $1`, id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			sp := &domain.SalePage{}
			if err := rows.Scan(&sp.ID, &sp.CustomerID, &sp.Name, &sp.Slug, &sp.TemplateName, &sp.Content, &sp.IsPublished, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
				return err
			}
			linkedSalePages = append(linkedSalePages, sp)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get pixel detail sub-queries: %w", err)
	}

	detail.EventCount = eventCount
	if linkedSalePages == nil {
		linkedSalePages = []*domain.SalePage{}
	}
	detail.LinkedSalePages = linkedSalePages

	return detail, nil
}

func (r *AdminRepo) SetPixelActive(ctx context.Context, id string, active bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE pixels SET is_active = $2, updated_at = NOW() WHERE id = $1`, id, active,
	)
	if err != nil {
		return fmt.Errorf("set pixel active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
