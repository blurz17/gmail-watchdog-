package repositories

import (
	"context"

	"github.com/mostaql-notification/internal/domain/entities"
)

// SystemEventRepository defines the persistence interface for system events.
type SystemEventRepository interface {
	// Create persists a new system event.
	Create(ctx context.Context, event *entities.SystemEvent) error

	// GetRecent retrieves the most recent system events.
	GetRecent(ctx context.Context, limit int) ([]entities.SystemEvent, error)
}
