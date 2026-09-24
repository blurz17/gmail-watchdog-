package services

import "context"

// NotificationSender defines the interface for sending notifications to a messaging channel.
// This is a port — each channel (Telegram, WhatsApp, etc.) implements this interface.
type NotificationSender interface {
	// Send delivers a notification message to the configured channel.
	// Returns the external message ID (e.g., Telegram message_id) on success.
	Send(ctx context.Context, content string) (externalID string, err error)

	// Channel returns the channel type identifier (e.g., "telegram", "whatsapp").
	Channel() string
}
