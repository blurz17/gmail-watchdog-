package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// SenderRuleRepository defines the persistence interface for sender rules.
type SenderRuleRepository interface {
	// GetByProviderID retrieves all sender rules for a given provider.
	GetByProviderID(ctx context.Context, providerID uuid.UUID) ([]entities.SenderRule, error)

	// GetAllActive retrieves all active sender rules across all providers.
	GetAllActive(ctx context.Context) ([]entities.SenderRule, error)

	// Create persists a new sender rule.
	Create(ctx context.Context, rule *entities.SenderRule) error

	// Delete removes a sender rule by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
