package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func (r *AdminRepo) CreateAuditLog(ctx context.Context, entry *domain.AuditLogEntry) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, target_customer_id, details)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		entry.AdminID, entry.Action, entry.TargetType, entry.TargetID, entry.TargetCustomerID, entry.Details,
	).Scan(&entry.ID, &entry.CreatedAt)
}

func (r *AdminRepo) ListAuditLogs(ctx context.Context, adminID, action, targetCustomerID string, from, to *time.Time, limit, offset int) ([]*domain.AuditLogEntry, int, error) {
	baseWhere := baseWhereTrue
	args := []interface{}{}
	argIdx := 1

	if adminID != "" {
		baseWhere += fmt.Sprintf(" AND al.admin_id = $%d", argIdx)
		args = append(args, adminID)
		argIdx++
	}
	if action != "" {
		baseWhere += fmt.Sprintf(" AND al.action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	if targetCustomerID != "" {
		baseWhere += fmt.Sprintf(" AND al.target_customer_id = $%d", argIdx)
		args = append(args, targetCustomerID)
		argIdx++
	}
	if from != nil {
		baseWhere += fmt.Sprintf(" AND al.created_at >= $%d", argIdx)
		args = append(args, *from)
		argIdx++
	}
	if to != nil {
		baseWhere += fmt.Sprintf(" AND al.created_at <= $%d", argIdx)
		args = append(args, *to)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_logs al "+baseWhere, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	selectQuery := fmt.Sprintf(
		`SELECT al.id, al.admin_id, admin_c.email, al.action, al.target_type, al.target_id, al.target_customer_id,
		        COALESCE(target_c.email, ''), al.details, al.created_at
		 FROM admin_audit_logs al
		 JOIN customers admin_c ON admin_c.id = al.admin_id
		 LEFT JOIN customers target_c ON target_c.id = al.target_customer_id
		 %s ORDER BY al.created_at DESC LIMIT $%d OFFSET $%d`,
		baseWhere, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditLogEntry
	for rows.Next() {
		e := &domain.AuditLogEntry{}
		var details json.RawMessage
		if err := rows.Scan(
			&e.ID, &e.AdminID, &e.AdminEmail, &e.Action, &e.TargetType, &e.TargetID, &e.TargetCustomerID,
			&e.CustomerEmail, &details, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		e.Details = details
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []*domain.AuditLogEntry{}
	}
	return entries, total, rows.Err()
}
