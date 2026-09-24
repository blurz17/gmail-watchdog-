package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// NotificationRepository defines the persistence interface for the notification outbox.
type NotificationRepository interface {
	// Create persists a new notification in the outbox.
	Create(ctx context.Context, notification *entities.Notification) error

	// GetPending retrieves notifications that are ready for delivery (pending or failed with
	// next_retry_at in the past). Results are ordered by creation time.
	GetPending(ctx context.Context, limit int) ([]entities.Notification, error)

	// UpdateStatus updates the delivery status, retry count, and next retry time of a notification.
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.NotificationStatus, retryCount int, nextRetry *time.Time) error

	// MarkDelivered marks a notification as delivered and records the delivery details.
	MarkDelivered(ctx context.Context, id uuid.UUID) error

	// CreateDelivery records a delivery attempt for a notification.
	CreateDelivery(ctx context.Context, delivery *entities.NotificationDelivery) error

	// GetPendingCount returns the number of notifications in pending or failed state.
	GetPendingCount(ctx context.Context) (int, error)

	// GetRecentDeliveries retrieves the most recent notification deliveries.
	GetRecentDeliveries(ctx context.Context, limit int) ([]entities.NotificationDelivery, error)
}
