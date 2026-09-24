package entities

import (
	"time"

	"github.com/google/uuid"
)

// AccountStatus represents the state of a Gmail account.
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusAuthError AccountStatus = "auth_error"
	AccountStatusDisabled  AccountStatus = "disabled"
)

// GmailAccount represents a monitored Gmail account with its OAuth credentials.
type GmailAccount struct {
	ID                uuid.UUID
	Email             string
	DisplayName       string
	OAuthClientID     string
	OAuthClientSecret string
	RefreshToken      string
	AccessToken       string
	AccessTokenExpiry time.Time
	Status            AccountStatus
	LastSyncAt        *time.Time
	LastHistoryID     uint64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsActive returns true if the account is in active status.
func (a *GmailAccount) IsActive() bool {
	return a.Status == AccountStatusActive
}

// IsTokenExpired returns true if the access token has expired or will expire within the given margin.
func (a *GmailAccount) IsTokenExpired(margin time.Duration) bool {
	if a.AccessToken == "" {
		return true
	}
	return time.Now().Add(margin).After(a.AccessTokenExpiry)
}

// NewGmailAccount creates a new GmailAccount entity with a generated UUID.
func NewGmailAccount(email, displayName, clientID, clientSecret, refreshToken string) *GmailAccount {
	now := time.Now()
	return &GmailAccount{
		ID:                uuid.New(),
		Email:             email,
		DisplayName:       displayName,
		OAuthClientID:     clientID,
		OAuthClientSecret: clientSecret,
		RefreshToken:      refreshToken,
		Status:            AccountStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}
