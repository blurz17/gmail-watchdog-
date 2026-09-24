package entities

import (
	"time"

	"github.com/google/uuid"
)

// DetectionMode defines how emails are detected for a provider.
type DetectionMode string

const (
	// DetectionModeUnread notifies only when matching emails are unread (Mode A).
	DetectionModeUnread DetectionMode = "unread"
	// DetectionModeNew notifies on newly received emails regardless of read status (Mode B).
	DetectionModeNew DetectionMode = "new"
)

// Provider represents a configured email sender/provider to monitor.
type Provider struct {
	ID            uuid.UUID
	Name          string
	IsActive      bool
	DetectionMode DetectionMode
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// SenderRules are loaded when needed, not always populated.
	SenderRules []SenderRule
}

// NewProvider creates a new Provider entity with a generated UUID.
func NewProvider(name string, mode DetectionMode) *Provider {
	now := time.Now()
	return &Provider{
		ID:            uuid.New(),
		Name:          name,
		IsActive:      true,
		DetectionMode: mode,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
