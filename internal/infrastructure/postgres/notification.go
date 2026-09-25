package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// NotificationRepo implements repositories.NotificationRepository using PostgreSQL.
type NotificationRepo struct {
	db *DB
}

// NewNotificationRepo creates a new PostgreSQL-backed notification repository.
func NewNotificationRepo(db *DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, notification *entities.Notification) error {
	query := `
		INSERT INTO notifications (
			id, email_message_id, provider_id, account_id,
			formatted_content, status, retry_count, next_retry_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Pool.Exec(ctx, query,
		notification.ID, notification.EmailMessageID, notification.ProviderID,
		notification.AccountID, notification.FormattedContent, notification.Status,
		notification.RetryCount, notification.NextRetryAt,
		notification.CreatedAt, notification.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating notification: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetPending(ctx context.Context, limit int) ([]entities.Notification, error) {
	query := `
		SELECT n.id, n.email_message_id, n.provider_id, n.account_id,
		       n.formatted_content, n.status, n.retry_count, n.next_retry_at,
		       n.created_at, n.updated_at,
		       ga.email AS account_email, p.name AS provider_name
		FROM notifications n
		JOIN gmail_accounts ga ON ga.id = n.account_id
		JOIN providers p ON p.id = n.provider_id
		WHERE n.status IN ('pending', 'failed')
		  AND (n.next_retry_at IS NULL OR n.next_retry_at <= NOW())
		ORDER BY n.created_at ASC
		LIMIT $1`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("getting pending notifications: %w", err)
	}
	defer rows.Close()

	var notifications []entities.Notification
	for rows.Next() {
		var n entities.Notification
		err := rows.Scan(
			&n.ID, &n.EmailMessageID, &n.ProviderID, &n.AccountID,
			&n.FormattedContent, &n.Status, &n.RetryCount, &n.NextRetryAt,
			&n.CreatedAt, &n.UpdatedAt,
			&n.AccountEmail, &n.ProviderName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning notification row: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.NotificationStatus, retryCount int, nextRetry *time.Time) error {
	query := `
		UPDATE notifications
		SET status = $2, retry_count = $3, next_retry_at = $4, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, status, retryCount, nextRetry)
	if err != nil {
		return fmt.Errorf("updating notification status: %w", err)
	}
	return nil
}

func (r *NotificationRepo) MarkDelivered(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications
		SET status = 'delivered', updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("marking notification delivered: %w", err)
	}
	return nil
}

func (r *NotificationRepo) CreateDelivery(ctx context.Context, delivery *entities.NotificationDelivery) error {
	query := `
		INSERT INTO notification_deliveries (
			id, notification_id, channel, status,
			external_id, error_message, delivered_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Pool.Exec(ctx, query,
		delivery.ID, delivery.NotificationID, delivery.Channel,
		delivery.Status, delivery.ExternalID, delivery.ErrorMessage,
		delivery.DeliveredAt, delivery.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating notification delivery: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetPendingCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE status IN ('pending', 'failed')`

	var count int
	err := r.db.Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("getting pending count: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) GetDeliveredCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE status = 'delivered'`

	var count int
	err := r.db.Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("getting delivered count: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) GetRecentDeliveries(ctx context.Context, limit int) ([]entities.NotificationDelivery, error) {
	query := `
		SELECT id, notification_id, channel, status, external_id,
		       error_message, delivered_at, created_at
		FROM notification_deliveries
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("getting recent deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []entities.NotificationDelivery
	for rows.Next() {
		var d entities.NotificationDelivery
		err := rows.Scan(
			&d.ID, &d.NotificationID, &d.Channel, &d.Status,
			&d.ExternalID, &d.ErrorMessage, &d.DeliveredAt, &d.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning delivery row: %w", err)
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}
