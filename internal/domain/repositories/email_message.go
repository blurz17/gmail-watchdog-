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

// EmailListFilter specifies filtering for the email browser.
type EmailListFilter struct {
	ProviderID *uuid.UUID
	AccountID  *uuid.UUID
	Search     string
	Offset     int
	Limit      int
}

// DailyCount holds the count for a single day.
type DailyCount struct {
	Date  time.Time
	Count int
}

// ProviderCount holds the count for a single provider.
type ProviderCount struct {
	ProviderID   uuid.UUID
	ProviderName string
	Count        int
}

// HourCount holds the count for a single hour of the day.
type HourCount struct {
	Hour  int
	Count int
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

	// GetAll retrieves email messages with filtering and pagination.
	GetAll(ctx context.Context, filter EmailListFilter) ([]entities.EmailMessage, int, error)

	// CountByDay returns email counts grouped by day for the last N days.
	CountByDay(ctx context.Context, days int) ([]DailyCount, error)

	// CountByProviderGrouped returns total email counts grouped by provider.
	CountByProviderGrouped(ctx context.Context) ([]ProviderCount, error)

	// CountByHour returns email counts grouped by hour of day.
	CountByHour(ctx context.Context, days int) ([]HourCount, error)

	// CountTotal returns total email count.
	CountTotal(ctx context.Context) (int, error)
}
