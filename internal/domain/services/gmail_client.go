package services

import (
	"context"
	"time"

	"github.com/mostaql-notification/internal/domain/entities"
)

// GmailMessage represents a single email message fetched from the Gmail API.
type GmailMessage struct {
	MessageID  string
	ThreadID   string
	From       string
	Subject    string
	Snippet    string
	ReceivedAt time.Time
	IsUnread   bool
}

// MessageListResult holds the result of a Gmail messages.list call.
type MessageListResult struct {
	Messages      []GmailMessage
	NextPageToken string
	// ResultSizeEstimate is Gmail's estimate of the total result count.
	ResultSizeEstimate int
}

// GmailClient defines the interface for interacting with the Gmail API.
// This is a port — the infrastructure layer provides the concrete implementation.
type GmailClient interface {
	// FetchMessages retrieves messages matching a query for the given account.
	// Use pageToken for pagination (empty string for first page).
	FetchMessages(ctx context.Context, account *entities.GmailAccount, query string, pageToken string) (*MessageListResult, error)

	// GetMessage retrieves the full details of a specific message.
	GetMessage(ctx context.Context, account *entities.GmailAccount, messageID string) (*GmailMessage, error)

	// RefreshAccessToken obtains a new access token using the refresh token.
	// Returns the new access token and its expiry time.
	RefreshAccessToken(ctx context.Context, account *entities.GmailAccount) (accessToken string, expiry time.Time, err error)
}
