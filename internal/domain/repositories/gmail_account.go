package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// GmailAccountRepository defines the persistence interface for Gmail accounts.
type GmailAccountRepository interface {
	// GetByID retrieves a Gmail account by its unique identifier.
	GetByID(ctx context.Context, id uuid.UUID) (*entities.GmailAccount, error)

	// GetByEmail retrieves a Gmail account by its email address.
	GetByEmail(ctx context.Context, email string) (*entities.GmailAccount, error)

	// GetAll retrieves all Gmail accounts.
	GetAll(ctx context.Context) ([]entities.GmailAccount, error)

	// GetActive retrieves only active Gmail accounts.
	GetActive(ctx context.Context) ([]entities.GmailAccount, error)

	// Create persists a new Gmail account.
	Create(ctx context.Context, account *entities.GmailAccount) error

	// Update persists changes to an existing Gmail account.
	Update(ctx context.Context, account *entities.GmailAccount) error

	// UpdateSyncState updates the last sync timestamp and history ID for an account.
	UpdateSyncState(ctx context.Context, id uuid.UUID, lastSyncAt time.Time, historyID uint64) error

	// UpdateTokens updates the OAuth access token and its expiry for an account.
	UpdateTokens(ctx context.Context, id uuid.UUID, accessToken string, expiry time.Time) error

	// UpdateStatus changes the status of a Gmail account.
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.AccountStatus) error

	// Delete removes a Gmail account by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
