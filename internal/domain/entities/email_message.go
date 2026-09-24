package entities

import (
	"time"

	"github.com/google/uuid"
)

// ProcessingStatus tracks the processing state of a detected email.
type ProcessingStatus string

const (
	ProcessingStatusDetected ProcessingStatus = "detected"
	ProcessingStatusNotified ProcessingStatus = "notified"
	ProcessingStatusSkipped  ProcessingStatus = "skipped"
)

// EmailMessage represents a detected email message from Gmail.
type EmailMessage struct {
	ID               uuid.UUID
	AccountID        uuid.UUID
	GmailMessageID   string
	ThreadID         string
	ProviderID       uuid.UUID
	From             string
	Subject          string
	Snippet          string
	ReceivedAt       time.Time
	IsUnread         bool
	ProcessingStatus ProcessingStatus
	DetectedAt       time.Time
	CreatedAt        time.Time

	// Populated via joins when needed, not always set.
	AccountEmail string
	ProviderName string
}

// NewEmailMessage creates a new EmailMessage entity with a generated UUID.
func NewEmailMessage(
	accountID uuid.UUID,
	gmailMessageID string,
	threadID string,
	providerID uuid.UUID,
	from string,
	subject string,
	snippet string,
	receivedAt time.Time,
	isUnread bool,
) *EmailMessage {
	now := time.Now()
	return &EmailMessage{
		ID:               uuid.New(),
		AccountID:        accountID,
		GmailMessageID:   gmailMessageID,
		ThreadID:         threadID,
		ProviderID:       providerID,
		From:             from,
		Subject:          subject,
		Snippet:          snippet,
		ReceivedAt:       receivedAt,
		IsUnread:         isUnread,
		ProcessingStatus: ProcessingStatusDetected,
		DetectedAt:       now,
		CreatedAt:        now,
	}
}
