package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type CustomDomainRepo struct {
	pool *pgxpool.Pool
}

func NewCustomDomainRepo(pool *pgxpool.Pool) *CustomDomainRepo {
	return &CustomDomainRepo{pool: pool}
}

func (r *CustomDomainRepo) Create(ctx context.Context, d *domain.CustomDomain) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO custom_domains (customer_id, sale_page_id, domain, cf_hostname_id, verification_token, dns_verified, ssl_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		d.CustomerID, d.SalePageID, d.Domain, d.CFHostnameID, d.VerificationToken, d.DNSVerified, d.SSLActive,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *CustomDomainRepo) GetByID(ctx context.Context, id string) (*domain.CustomDomain, error) {
	d := &domain.CustomDomain{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, sale_page_id, domain, cf_hostname_id, verification_token, dns_verified, ssl_active, verified_at, created_at, updated_at
		 FROM custom_domains WHERE id = $1`, id,
	).Scan(&d.ID, &d.CustomerID, &d.SalePageID, &d.Domain, &d.CFHostnameID, &d.VerificationToken, &d.DNSVerified, &d.SSLActive, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *CustomDomainRepo) GetByDomain(ctx context.Context, domainName string) (*domain.CustomDomain, error) {
	d := &domain.CustomDomain{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, sale_page_id, domain, cf_hostname_id, verification_token, dns_verified, ssl_active, verified_at, created_at, updated_at
		 FROM custom_domains WHERE domain = $1`, domainName,
	).Scan(&d.ID, &d.CustomerID, &d.SalePageID, &d.Domain, &d.CFHostnameID, &d.VerificationToken, &d.DNSVerified, &d.SSLActive, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *CustomDomainRepo) ListByCustomerID(ctx context.Context, customerID string) ([]*domain.CustomDomain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, sale_page_id, domain, cf_hostname_id, verification_token, dns_verified, ssl_active, verified_at, created_at, updated_at
		 FROM custom_domains WHERE customer_id = $1 ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*domain.CustomDomain
	for rows.Next() {
		d := &domain.CustomDomain{}
		if err := rows.Scan(&d.ID, &d.CustomerID, &d.SalePageID, &d.Domain, &d.CFHostnameID, &d.VerificationToken, &d.DNSVerified, &d.SSLActive, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func (r *CustomDomainRepo) ListBySalePageID(ctx context.Context, salePageID string) ([]*domain.CustomDomain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, sale_page_id, domain, cf_hostname_id, verification_token, dns_verified, ssl_active, verified_at, created_at, updated_at
		 FROM custom_domains WHERE sale_page_id = $1 ORDER BY created_at DESC`, salePageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*domain.CustomDomain
	for rows.Next() {
		d := &domain.CustomDomain{}
		if err := rows.Scan(&d.ID, &d.CustomerID, &d.SalePageID, &d.Domain, &d.CFHostnameID, &d.VerificationToken, &d.DNSVerified, &d.SSLActive, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func (r *CustomDomainRepo) Update(ctx context.Context, d *domain.CustomDomain) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE custom_domains SET domain = $2, cf_hostname_id = $3, dns_verified = $4, ssl_active = $5, verified_at = $6, updated_at = NOW()
		 WHERE id = $1`,
		d.ID, d.Domain, d.CFHostnameID, d.DNSVerified, d.SSLActive, d.VerifiedAt,
	)
	return err
}

func (r *CustomDomainRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM custom_domains WHERE id = $1`, id)
	return err
}
