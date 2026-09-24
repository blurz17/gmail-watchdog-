package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// SenderRuleRepo implements repositories.SenderRuleRepository using PostgreSQL.
type SenderRuleRepo struct {
	db *DB
}

// NewSenderRuleRepo creates a new PostgreSQL-backed sender rule repository.
func NewSenderRuleRepo(db *DB) *SenderRuleRepo {
	return &SenderRuleRepo{db: db}
}

func (r *SenderRuleRepo) GetByProviderID(ctx context.Context, providerID uuid.UUID) ([]entities.SenderRule, error) {
	query := `
		SELECT id, provider_id, sender_email, sender_domain, match_type, is_active, created_at
		FROM sender_rules
		WHERE provider_id = $1
		ORDER BY created_at`

	return r.scanMany(ctx, query, providerID)
}

func (r *SenderRuleRepo) GetAllActive(ctx context.Context) ([]entities.SenderRule, error) {
	query := `
		SELECT id, provider_id, sender_email, sender_domain, match_type, is_active, created_at
		FROM sender_rules
		WHERE is_active = TRUE
		ORDER BY provider_id, created_at`

	return r.scanMany(ctx, query)
}

func (r *SenderRuleRepo) Create(ctx context.Context, rule *entities.SenderRule) error {
	query := `
		INSERT INTO sender_rules (id, provider_id, sender_email, sender_domain, match_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Pool.Exec(ctx, query,
		rule.ID, rule.ProviderID, rule.SenderEmail,
		rule.SenderDomain, rule.MatchType, rule.IsActive, rule.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating sender rule: %w", err)
	}
	return nil
}

func (r *SenderRuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sender_rules WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting sender rule: %w", err)
	}
	return nil
}

func (r *SenderRuleRepo) scanMany(ctx context.Context, query string, args ...interface{}) ([]entities.SenderRule, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying sender rules: %w", err)
	}
	defer rows.Close()

	var rules []entities.SenderRule
	for rows.Next() {
		var s entities.SenderRule
		var senderEmail, senderDomain *string
		err := rows.Scan(
			&s.ID, &s.ProviderID, &senderEmail, &senderDomain,
			&s.MatchType, &s.IsActive, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning sender rule row: %w", err)
		}
		if senderEmail != nil {
			s.SenderEmail = *senderEmail
		}
		if senderDomain != nil {
			s.SenderDomain = *senderDomain
		}
		rules = append(rules, s)
	}
	return rules, rows.Err()
}
