package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// ProviderRepository defines the persistence interface for providers.
type ProviderRepository interface {
	// GetByID retrieves a provider by its unique identifier.
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Provider, error)

	// GetByName retrieves a provider by its human-readable name (case-insensitive).
	GetByName(ctx context.Context, name string) (*entities.Provider, error)

	// GetAll retrieves all providers.
	GetAll(ctx context.Context) ([]entities.Provider, error)

	// GetActive retrieves only active providers.
	GetActive(ctx context.Context) ([]entities.Provider, error)

	// Create persists a new provider.
	Create(ctx context.Context, provider *entities.Provider) error

	// Update persists changes to an existing provider.
	Update(ctx context.Context, provider *entities.Provider) error

	// Delete removes a provider by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
