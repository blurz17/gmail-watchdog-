package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/domain/repositories"
)

// EmailMessageRepo implements repositories.EmailMessageRepository using PostgreSQL.
type EmailMessageRepo struct {
	db *DB
}

// NewEmailMessageRepo creates a new PostgreSQL-backed email message repository.
func NewEmailMessageRepo(db *DB) *EmailMessageRepo {
	return &EmailMessageRepo{db: db}
}

func (r *EmailMessageRepo) Exists(ctx context.Context, accountID uuid.UUID, gmailMessageID string, providerID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM email_messages
			WHERE account_id = $1 AND gmail_message_id = $2 AND provider_id = $3
		)`

	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, accountID, gmailMessageID, providerID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking email message existence: %w", err)
	}
	return exists, nil
}

func (r *EmailMessageRepo) Create(ctx context.Context, msg *entities.EmailMessage) error {
	query := `
		INSERT INTO email_messages (
			id, account_id, gmail_message_id, thread_id, provider_id,
			sender_from, subject, snippet, received_at, is_unread,
			processing_status, detected_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (account_id, gmail_message_id, provider_id) DO NOTHING`

	_, err := r.db.Pool.Exec(ctx, query,
		msg.ID, msg.AccountID, msg.GmailMessageID, msg.ThreadID, msg.ProviderID,
		msg.From, msg.Subject, msg.Snippet, msg.ReceivedAt, msg.IsUnread,
		msg.ProcessingStatus, msg.DetectedAt, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating email message: %w", err)
	}
	return nil
}

func (r *EmailMessageRepo) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status entities.ProcessingStatus) error {
	query := `UPDATE email_messages SET processing_status = $2 WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("updating processing status: %w", err)
	}
	return nil
}

func (r *EmailMessageRepo) GetUnreadByProvider(ctx context.Context, providerID uuid.UUID) ([]entities.EmailMessage, error) {
	query := `
		SELECT em.id, em.account_id, em.gmail_message_id, em.thread_id, em.provider_id,
		       em.sender_from, em.subject, em.snippet, em.received_at, em.is_unread,
		       em.processing_status, em.detected_at, em.created_at,
		       ga.email AS account_email, p.name AS provider_name
		FROM email_messages em
		JOIN gmail_accounts ga ON ga.id = em.account_id
		JOIN providers p ON p.id = em.provider_id
		WHERE em.provider_id = $1 AND em.is_unread = TRUE
		ORDER BY em.received_at DESC`

	return r.scanManyWithJoins(ctx, query, providerID)
}

func (r *EmailMessageRepo) GetLatestByProvider(ctx context.Context, providerID uuid.UUID, limit int) ([]entities.EmailMessage, error) {
	query := `
		SELECT em.id, em.account_id, em.gmail_message_id, em.thread_id, em.provider_id,
		       em.sender_from, em.subject, em.snippet, em.received_at, em.is_unread,
		       em.processing_status, em.detected_at, em.created_at,
		       ga.email AS account_email, p.name AS provider_name
		FROM email_messages em
		JOIN gmail_accounts ga ON ga.id = em.account_id
		JOIN providers p ON p.id = em.provider_id
		WHERE em.provider_id = $1
		ORDER BY em.received_at DESC
		LIMIT $2`

	return r.scanManyWithJoins(ctx, query, providerID, limit)
}

func (r *EmailMessageRepo) CountByProvider(ctx context.Context, providerID uuid.UUID, filter repositories.CountFilter) ([]repositories.AccountCount, error) {
	query := `
		SELECT em.account_id, ga.email, COUNT(*) AS cnt, MAX(em.received_at) AS latest_at
		FROM email_messages em
		JOIN gmail_accounts ga ON ga.id = em.account_id
		WHERE em.provider_id = $1`

	args := []interface{}{providerID}
	argIdx := 2

	if filter.UnreadOnly {
		query += fmt.Sprintf(" AND em.is_unread = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if filter.Since != nil {
		query += fmt.Sprintf(" AND em.received_at >= $%d", argIdx)
		args = append(args, *filter.Since)
		argIdx++
	}

	query += " GROUP BY em.account_id, ga.email ORDER BY ga.email"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("counting by provider: %w", err)
	}
	defer rows.Close()

	var results []repositories.AccountCount
	for rows.Next() {
		var ac repositories.AccountCount
		err := rows.Scan(&ac.AccountID, &ac.AccountEmail, &ac.Count, &ac.LatestAt)
		if err != nil {
			return nil, fmt.Errorf("scanning count row: %w", err)
		}
		results = append(results, ac)
	}
	return results, rows.Err()
}

func (r *EmailMessageRepo) scanManyWithJoins(ctx context.Context, query string, args ...interface{}) ([]entities.EmailMessage, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying email messages: %w", err)
	}
	defer rows.Close()

	var messages []entities.EmailMessage
	for rows.Next() {
		var m entities.EmailMessage
		err := rows.Scan(
			&m.ID, &m.AccountID, &m.GmailMessageID, &m.ThreadID, &m.ProviderID,
			&m.From, &m.Subject, &m.Snippet, &m.ReceivedAt, &m.IsUnread,
			&m.ProcessingStatus, &m.DetectedAt, &m.CreatedAt,
			&m.AccountEmail, &m.ProviderName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning email message row: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
