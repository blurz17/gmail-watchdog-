package entities

import (
	"time"

	"github.com/google/uuid"
)

// NotificationStatus tracks the state of a notification in the outbox.
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSending   NotificationStatus = "sending"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed    NotificationStatus = "failed"
)

// ChannelType identifies the notification delivery channel.
type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelWhatsApp ChannelType = "whatsapp"
)

// DeliveryStatus tracks the outcome of a single delivery attempt.
type DeliveryStatus string

const (
	DeliveryStatusSuccess DeliveryStatus = "success"
	DeliveryStatusFailed  DeliveryStatus = "failed"
)

// Notification represents a pending or processed notification in the outbox.
type Notification struct {
	ID               uuid.UUID
	EmailMessageID   uuid.UUID
	ProviderID       uuid.UUID
	AccountID        uuid.UUID
	FormattedContent string
	Status           NotificationStatus
	RetryCount       int
	NextRetryAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Populated via joins when needed.
	AccountEmail string
	ProviderName string
}

// NewNotification creates a new pending Notification in the outbox.
func NewNotification(
	emailMessageID uuid.UUID,
	providerID uuid.UUID,
	accountID uuid.UUID,
	formattedContent string,
) *Notification {
	now := time.Now()
	return &Notification{
		ID:               uuid.New(),
		EmailMessageID:   emailMessageID,
		ProviderID:       providerID,
		AccountID:        accountID,
		FormattedContent: formattedContent,
		Status:           NotificationStatusPending,
		RetryCount:       0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// NotificationDelivery records the outcome of a delivery attempt to a specific channel.
type NotificationDelivery struct {
	ID             uuid.UUID
	NotificationID uuid.UUID
	Channel        ChannelType
	Status         DeliveryStatus
	ExternalID     string // e.g., Telegram message_id
	ErrorMessage   string
	DeliveredAt    *time.Time
	CreatedAt      time.Time
}

// NewNotificationDelivery creates a new delivery record.
func NewNotificationDelivery(
	notificationID uuid.UUID,
	channel ChannelType,
	status DeliveryStatus,
	externalID string,
	errorMessage string,
) *NotificationDelivery {
	now := time.Now()
	d := &NotificationDelivery{
		ID:             uuid.New(),
		NotificationID: notificationID,
		Channel:        channel,
		Status:         status,
		ExternalID:     externalID,
		ErrorMessage:   errorMessage,
		CreatedAt:      now,
	}
	if status == DeliveryStatusSuccess {
		d.DeliveredAt = &now
	}
	return d
}
