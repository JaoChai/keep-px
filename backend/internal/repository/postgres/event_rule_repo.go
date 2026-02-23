package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type EventRuleRepo struct {
	pool *pgxpool.Pool
}

func NewEventRuleRepo(pool *pgxpool.Pool) *EventRuleRepo {
	return &EventRuleRepo{pool: pool}
}

func (r *EventRuleRepo) Create(ctx context.Context, rule *domain.EventRule) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO event_rules (pixel_id, page_url, event_name, trigger_type, css_selector, xpath, element_text, conditions, parameters, fire_once, delay_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, is_active, created_at, updated_at`,
		rule.PixelID, rule.PageURL, rule.EventName, rule.TriggerType, rule.CSSSelector, rule.XPath, rule.ElementText, rule.Conditions, rule.Parameters, rule.FireOnce, rule.DelayMs,
	).Scan(&rule.ID, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *EventRuleRepo) GetByID(ctx context.Context, id string) (*domain.EventRule, error) {
	rule := &domain.EventRule{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, pixel_id, page_url, event_name, trigger_type, css_selector, xpath, element_text, conditions, parameters, fire_once, delay_ms, is_active, created_at, updated_at
		 FROM event_rules WHERE id = $1`, id,
	).Scan(&rule.ID, &rule.PixelID, &rule.PageURL, &rule.EventName, &rule.TriggerType, &rule.CSSSelector, &rule.XPath, &rule.ElementText, &rule.Conditions, &rule.Parameters, &rule.FireOnce, &rule.DelayMs, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return rule, err
}

func (r *EventRuleRepo) ListByPixelID(ctx context.Context, pixelID string) ([]*domain.EventRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pixel_id, page_url, event_name, trigger_type, css_selector, xpath, element_text, conditions, parameters, fire_once, delay_ms, is_active, created_at, updated_at
		 FROM event_rules WHERE pixel_id = $1 ORDER BY created_at DESC`, pixelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.EventRule
	for rows.Next() {
		rule := &domain.EventRule{}
		if err := rows.Scan(&rule.ID, &rule.PixelID, &rule.PageURL, &rule.EventName, &rule.TriggerType, &rule.CSSSelector, &rule.XPath, &rule.ElementText, &rule.Conditions, &rule.Parameters, &rule.FireOnce, &rule.DelayMs, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *EventRuleRepo) ListActiveByPixelID(ctx context.Context, pixelID string) ([]*domain.EventRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pixel_id, page_url, event_name, trigger_type, css_selector, xpath, element_text, conditions, parameters, fire_once, delay_ms, is_active, created_at, updated_at
		 FROM event_rules WHERE pixel_id = $1 AND is_active = true ORDER BY created_at DESC`, pixelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.EventRule
	for rows.Next() {
		rule := &domain.EventRule{}
		if err := rows.Scan(&rule.ID, &rule.PixelID, &rule.PageURL, &rule.EventName, &rule.TriggerType, &rule.CSSSelector, &rule.XPath, &rule.ElementText, &rule.Conditions, &rule.Parameters, &rule.FireOnce, &rule.DelayMs, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *EventRuleRepo) Update(ctx context.Context, rule *domain.EventRule) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE event_rules SET page_url=$2, event_name=$3, trigger_type=$4, css_selector=$5, xpath=$6, element_text=$7, conditions=$8, parameters=$9, fire_once=$10, delay_ms=$11, is_active=$12, updated_at=NOW()
		 WHERE id = $1`,
		rule.ID, rule.PageURL, rule.EventName, rule.TriggerType, rule.CSSSelector, rule.XPath, rule.ElementText, rule.Conditions, rule.Parameters, rule.FireOnce, rule.DelayMs, rule.IsActive,
	)
	return err
}

func (r *EventRuleRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM event_rules WHERE id = $1`, id)
	return err
}
