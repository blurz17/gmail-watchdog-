package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/domain/repositories"
	"github.com/mostaql-notification/internal/domain/services"
)

// RetryPolicy defines the retry behavior for failed notification deliveries.
type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
	JitterFactor   float64
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: 30 * time.Second,
		MaxBackoff:     15 * time.Minute,
		BackoffFactor:  2.0,
		JitterFactor:   0.1,
	}
}

// DeliverNotificationsUseCase processes the notification outbox.
// It picks up pending/failed notifications, delivers them via the configured
// notification sender, and updates their status.
type DeliverNotificationsUseCase struct {
	notifications repositories.NotificationRepository
	messages      repositories.EmailMessageRepository
	sender        services.NotificationSender
	retryPolicy   RetryPolicy
	logger        *slog.Logger
}

// NewDeliverNotificationsUseCase creates a new DeliverNotificationsUseCase.
func NewDeliverNotificationsUseCase(
	notifications repositories.NotificationRepository,
	messages repositories.EmailMessageRepository,
	sender services.NotificationSender,
	retryPolicy RetryPolicy,
	logger *slog.Logger,
) *DeliverNotificationsUseCase {
	return &DeliverNotificationsUseCase{
		notifications: notifications,
		messages:      messages,
		sender:        sender,
		retryPolicy:   retryPolicy,
		logger:        logger,
	}
}

// Execute processes one batch of pending notifications from the outbox.
func (uc *DeliverNotificationsUseCase) Execute(ctx context.Context) error {
	// Pick up pending notifications (pending or failed past their retry time).
	pending, err := uc.notifications.GetPending(ctx, 20)
	if err != nil {
		return fmt.Errorf("fetching pending notifications: %w", err)
	}

	if len(pending) == 0 {
		return nil
	}

	uc.logger.Debug("processing notification batch", "count", len(pending))

	var delivered, failed int

	for i := range pending {
		notification := &pending[i]

		if err := uc.deliverOne(ctx, notification); err != nil {
			uc.logger.Error("notification delivery failed",
				"notification_id", notification.ID,
				"provider", notification.ProviderName,
				"attempt", notification.RetryCount+1,
				"error", err,
			)
			failed++
		} else {
			delivered++
		}

		// Prevent Telegram 429 Too Many Requests by rate limiting our outgoing messages
		// Telegram allows ~1 message per second to a single chat ID.
		time.Sleep(1500 * time.Millisecond)
	}

	if delivered > 0 || failed > 0 {
		uc.logger.Info("notification batch processed",
			"delivered", delivered,
			"failed", failed,
		)
	}

	return nil
}

// deliverOne attempts to deliver a single notification.
func (uc *DeliverNotificationsUseCase) deliverOne(ctx context.Context, notification *entities.Notification) error {
	// Mark as sending (prevents other workers from picking it up).
	if err := uc.notifications.UpdateStatus(ctx, notification.ID,
		entities.NotificationStatusSending, notification.RetryCount, notification.NextRetryAt); err != nil {
		return fmt.Errorf("marking as sending: %w", err)
	}

	// Attempt delivery.
	externalID, err := uc.sender.Send(ctx, notification.FormattedContent)

	if err != nil {
		// Delivery failed — apply retry policy.
		return uc.handleFailure(ctx, notification, err)
	}

	// Delivery succeeded.
	return uc.handleSuccess(ctx, notification, externalID)
}

// handleSuccess records a successful delivery.
func (uc *DeliverNotificationsUseCase) handleSuccess(ctx context.Context, notification *entities.Notification, externalID string) error {
	// Mark notification as delivered.
	if err := uc.notifications.MarkDelivered(ctx, notification.ID); err != nil {
		return fmt.Errorf("marking as delivered: %w", err)
	}

	// Record the delivery.
	delivery := entities.NewNotificationDelivery(
		notification.ID,
		entities.ChannelType(uc.sender.Channel()),
		entities.DeliveryStatusSuccess,
		externalID,
		"",
	)

	if err := uc.notifications.CreateDelivery(ctx, delivery); err != nil {
		// Non-fatal: the notification was delivered, just the tracking failed.
		uc.logger.Warn("failed to record delivery", "notification_id", notification.ID, "error", err)
	}

	// Update the email message processing status.
	if err := uc.messages.UpdateProcessingStatus(ctx, notification.EmailMessageID, entities.ProcessingStatusNotified); err != nil {
		uc.logger.Warn("failed to update message status", "notification_id", notification.ID, "error", err)
	}

	uc.logger.Info("notification delivered",
		"notification_id", notification.ID,
		"channel", uc.sender.Channel(),
		"external_id", externalID,
		"provider", notification.ProviderName,
	)

	return nil
}

// handleFailure applies the retry policy after a failed delivery attempt.
func (uc *DeliverNotificationsUseCase) handleFailure(ctx context.Context, notification *entities.Notification, deliveryErr error) error {
	newRetryCount := notification.RetryCount + 1

	// Record the failed delivery attempt.
	delivery := entities.NewNotificationDelivery(
		notification.ID,
		entities.ChannelType(uc.sender.Channel()),
		entities.DeliveryStatusFailed,
		"",
		deliveryErr.Error(),
	)

	if err := uc.notifications.CreateDelivery(ctx, delivery); err != nil {
		uc.logger.Warn("failed to record failed delivery", "notification_id", notification.ID, "error", err)
	}

	// Check if max retries exceeded.
	if newRetryCount >= uc.retryPolicy.MaxAttempts {
		uc.logger.Error("notification permanently failed, max retries exceeded",
			"notification_id", notification.ID,
			"provider", notification.ProviderName,
			"attempts", newRetryCount,
		)

		if err := uc.notifications.UpdateStatus(ctx, notification.ID,
			entities.NotificationStatusFailed, newRetryCount, nil); err != nil {
			return fmt.Errorf("marking as permanently failed: %w", err)
		}
		return fmt.Errorf("max retries exceeded: %w", deliveryErr)
	}

	// Calculate next retry time with exponential backoff + jitter.
	nextRetry := uc.calculateNextRetry(newRetryCount)

	if err := uc.notifications.UpdateStatus(ctx, notification.ID,
		entities.NotificationStatusFailed, newRetryCount, &nextRetry); err != nil {
		return fmt.Errorf("scheduling retry: %w", err)
	}

	uc.logger.Warn("notification delivery retrying",
		"notification_id", notification.ID,
		"attempt", newRetryCount,
		"next_retry", nextRetry.Format(time.RFC3339),
	)

	return fmt.Errorf("delivery failed (attempt %d/%d): %w",
		newRetryCount, uc.retryPolicy.MaxAttempts, deliveryErr)
}

// calculateNextRetry computes the next retry time using exponential backoff with jitter.
// Formula: min(initial * factor^attempt + jitter, max)
func (uc *DeliverNotificationsUseCase) calculateNextRetry(attempt int) time.Time {
	backoff := float64(uc.retryPolicy.InitialBackoff) * math.Pow(uc.retryPolicy.BackoffFactor, float64(attempt-1))

	// Apply jitter: ±10% of the backoff.
	jitter := backoff * uc.retryPolicy.JitterFactor * (2*rand.Float64() - 1)
	backoff += jitter

	// Cap at max backoff.
	if backoff > float64(uc.retryPolicy.MaxBackoff) {
		backoff = float64(uc.retryPolicy.MaxBackoff)
	}

	return time.Now().Add(time.Duration(backoff))
}
