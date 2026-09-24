package services

import (
	"fmt"
	"time"

	"github.com/mostaql-notification/internal/domain/entities"
	"net/url"
)

// NotificationFormatter builds human-readable notification content from email messages.
// This is a domain service — it contains formatting business rules but no external dependencies.
type NotificationFormatter struct {
	timezone *time.Location
}

// NewNotificationFormatter creates a new formatter with the given timezone for display.
func NewNotificationFormatter(timezone *time.Location) *NotificationFormatter {
	return &NotificationFormatter{
		timezone: timezone,
	}
}

// FormatNewEmail formats a notification for a newly detected email.
func (f *NotificationFormatter) FormatNewEmail(
	msg *entities.EmailMessage,
	accountEmail string,
	providerName string,
) string {
	receivedLocal := msg.ReceivedAt.In(f.timezone)
	detectedLocal := msg.DetectedAt.In(f.timezone)

	status := "Read"
	if msg.IsUnread {
		status = "Unread"
	}

	snippet := msg.Snippet
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}

	gmailLink := GmailMessageURL(accountEmail, msg.GmailMessageID)

	return fmt.Sprintf(`📧 New Gmail Notification

Provider: %s
Gmail Account: %s

From: %s
Subject: %s

Received:
%s
%s

Status: %s

Preview:
%s

🔗 Open in Gmail:
%s

Message ID: %s

────────────────
Detected: %s`,
		providerName,
		accountEmail,
		msg.From,
		msg.Subject,
		receivedLocal.Format("02 January 2006"),
		receivedLocal.Format("03:04 PM (MST)"),
		status,
		snippet,
		gmailLink,
		msg.GmailMessageID,
		detectedLocal.Format("03:04 PM"),
	)
}

// FormatCount formats the response for a /count command.
func (f *NotificationFormatter) FormatCount(
	providerName string,
	counts []CountResult,
	totalUnread int,
) string {
	result := fmt.Sprintf("📊 %s\n\n", providerName)

	for _, c := range counts {
		result += fmt.Sprintf("Gmail account: %s\n", c.AccountEmail)
		result += fmt.Sprintf("Unread matching emails: %d\n", c.Count)
		if c.LatestAt != nil {
			latestLocal := c.LatestAt.In(f.timezone)
			result += fmt.Sprintf("Latest matching email: %s\n", latestLocal.Format("2006-01-02 15:04"))
		}
		result += "\n"
	}

	result += fmt.Sprintf("Total unread: %d", totalUnread)
	return result
}

// FormatLatest formats the response for a /latest command.
func (f *NotificationFormatter) FormatLatest(
	providerName string,
	messages []entities.EmailMessage,
) string {
	if len(messages) == 0 {
		return fmt.Sprintf("📭 %s\n\nNo matching messages found.", providerName)
	}

	result := fmt.Sprintf("📨 %s\n\n", providerName)

	for _, msg := range messages {
		receivedLocal := msg.ReceivedAt.In(f.timezone)
		gmailLink := GmailMessageURL(msg.AccountEmail, msg.GmailMessageID)
		result += fmt.Sprintf("Account: %s\n", msg.AccountEmail)
		result += fmt.Sprintf("Subject: %s\n", msg.Subject)
		result += fmt.Sprintf("From: %s\n", msg.From)
		result += fmt.Sprintf("Date: %s\n", receivedLocal.Format("02 Jan 2006 15:04"))
		result += fmt.Sprintf("🔗 %s\n\n", gmailLink)
	}

	return result
}

// CountResult is used by FormatCount.
type CountResult struct {
	AccountEmail string
	Count        int
	LatestAt     *time.Time
}

// GmailMessageURL builds a direct link to open a specific email in Gmail's web UI.
// It uses the authuser parameter so the correct account opens even when
// the user is signed into multiple Google accounts.
func GmailMessageURL(accountEmail string, gmailMessageID string) string {
	return fmt.Sprintf(
		"https://mail.google.com/mail/u/?authuser=%s#inbox/%s",
		url.QueryEscape(accountEmail),
		gmailMessageID,
	)
}
