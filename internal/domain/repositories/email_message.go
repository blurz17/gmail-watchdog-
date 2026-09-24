package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// CountFilter specifies filtering criteria for email counting.
type CountFilter struct {
	// UnreadOnly counts only unread messages when true.
	UnreadOnly bool
	// Since limits the count to messages received after this time.
	Since *time.Time
}

// AccountCount holds the count result for a single Gmail account.
type AccountCount struct {
	AccountID    uuid.UUID
	AccountEmail string
	Count        int
	LatestAt     *time.Time
}

// EmailMessageRepository defines the persistence interface for email messages.
type EmailMessageRepository interface {
	// Exists checks whether a message with the given account, Gmail message ID, and provider
	// has already been recorded. This is the primary duplicate-prevention mechanism.
	Exists(ctx context.Context, accountID uuid.UUID, gmailMessageID string, providerID uuid.UUID) (bool, error)

	// Create persists a new email message record.
	Create(ctx context.Context, msg *entities.EmailMessage) error

	// UpdateProcessingStatus updates the processing status of a message.
	UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status entities.ProcessingStatus) error

	// GetUnreadByProvider retrieves unread messages for a given provider across all accounts.
	GetUnreadByProvider(ctx context.Context, providerID uuid.UUID) ([]entities.EmailMessage, error)

	// GetLatestByProvider retrieves the N most recent messages for a provider.
	GetLatestByProvider(ctx context.Context, providerID uuid.UUID, limit int) ([]entities.EmailMessage, error)

	// CountByProvider counts messages for a provider, grouped by account, with filtering.
	CountByProvider(ctx context.Context, providerID uuid.UUID, filter CountFilter) ([]AccountCount, error)
}
