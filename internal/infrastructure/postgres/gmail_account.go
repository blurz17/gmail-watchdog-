package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mostaql-notification/internal/domain/entities"
)

// GmailAccountRepo implements repositories.GmailAccountRepository using PostgreSQL.
type GmailAccountRepo struct {
	db *DB
}

// NewGmailAccountRepo creates a new PostgreSQL-backed Gmail account repository.
func NewGmailAccountRepo(db *DB) *GmailAccountRepo {
	return &GmailAccountRepo{db: db}
}

func (r *GmailAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.GmailAccount, error) {
	query := `
		SELECT id, email, display_name, oauth_client_id, oauth_client_secret,
		       refresh_token, access_token, access_token_expiry, status,
		       last_sync_at, last_history_id, created_at, updated_at
		FROM gmail_accounts
		WHERE id = $1`

	return r.scanOne(ctx, query, id)
}

func (r *GmailAccountRepo) GetByEmail(ctx context.Context, email string) (*entities.GmailAccount, error) {
	query := `
		SELECT id, email, display_name, oauth_client_id, oauth_client_secret,
		       refresh_token, access_token, access_token_expiry, status,
		       last_sync_at, last_history_id, created_at, updated_at
		FROM gmail_accounts
		WHERE email = $1`

	return r.scanOne(ctx, query, email)
}

func (r *GmailAccountRepo) GetAll(ctx context.Context) ([]entities.GmailAccount, error) {
	query := `
		SELECT id, email, display_name, oauth_client_id, oauth_client_secret,
		       refresh_token, access_token, access_token_expiry, status,
		       last_sync_at, last_history_id, created_at, updated_at
		FROM gmail_accounts
		ORDER BY email`

	return r.scanMany(ctx, query)
}

func (r *GmailAccountRepo) GetActive(ctx context.Context) ([]entities.GmailAccount, error) {
	query := `
		SELECT id, email, display_name, oauth_client_id, oauth_client_secret,
		       refresh_token, access_token, access_token_expiry, status,
		       last_sync_at, last_history_id, created_at, updated_at
		FROM gmail_accounts
		WHERE status = 'active'
		ORDER BY email`

	return r.scanMany(ctx, query)
}

func (r *GmailAccountRepo) Create(ctx context.Context, account *entities.GmailAccount) error {
	query := `
		INSERT INTO gmail_accounts (
			id, email, display_name, oauth_client_id, oauth_client_secret,
			refresh_token, access_token, access_token_expiry, status,
			last_sync_at, last_history_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.Pool.Exec(ctx, query,
		account.ID, account.Email, account.DisplayName,
		account.OAuthClientID, account.OAuthClientSecret,
		account.RefreshToken, account.AccessToken, account.AccessTokenExpiry,
		account.Status, account.LastSyncAt, account.LastHistoryID,
		account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating gmail account: %w", err)
	}
	return nil
}

func (r *GmailAccountRepo) Update(ctx context.Context, account *entities.GmailAccount) error {
	query := `
		UPDATE gmail_accounts
		SET email = $2, display_name = $3, oauth_client_id = $4,
		    oauth_client_secret = $5, refresh_token = $6,
		    access_token = $7, access_token_expiry = $8,
		    status = $9, last_sync_at = $10, last_history_id = $11,
		    updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query,
		account.ID, account.Email, account.DisplayName,
		account.OAuthClientID, account.OAuthClientSecret,
		account.RefreshToken, account.AccessToken, account.AccessTokenExpiry,
		account.Status, account.LastSyncAt, account.LastHistoryID,
	)
	if err != nil {
		return fmt.Errorf("updating gmail account: %w", err)
	}
	return nil
}

func (r *GmailAccountRepo) UpdateSyncState(ctx context.Context, id uuid.UUID, lastSyncAt time.Time, historyID uint64) error {
	query := `
		UPDATE gmail_accounts
		SET last_sync_at = $2, last_history_id = $3, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, lastSyncAt, historyID)
	if err != nil {
		return fmt.Errorf("updating sync state: %w", err)
	}
	return nil
}

func (r *GmailAccountRepo) UpdateTokens(ctx context.Context, id uuid.UUID, accessToken string, expiry time.Time) error {
	query := `
		UPDATE gmail_accounts
		SET access_token = $2, access_token_expiry = $3, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, accessToken, expiry)
	if err != nil {
		return fmt.Errorf("updating tokens: %w", err)
	}
	return nil
}

func (r *GmailAccountRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.AccountStatus) error {
	query := `
		UPDATE gmail_accounts
		SET status = $2, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("updating account status: %w", err)
	}
	return nil
}

func (r *GmailAccountRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM gmail_accounts WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting gmail account: %w", err)
	}
	return nil
}

// scanOne scans a single GmailAccount from a query result.
func (r *GmailAccountRepo) scanOne(ctx context.Context, query string, args ...interface{}) (*entities.GmailAccount, error) {
	var a entities.GmailAccount
	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(
		&a.ID, &a.Email, &a.DisplayName,
		&a.OAuthClientID, &a.OAuthClientSecret,
		&a.RefreshToken, &a.AccessToken, &a.AccessTokenExpiry,
		&a.Status, &a.LastSyncAt, &a.LastHistoryID,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scanning gmail account: %w", err)
	}
	return &a, nil
}

// scanMany scans multiple GmailAccounts from a query result.
func (r *GmailAccountRepo) scanMany(ctx context.Context, query string, args ...interface{}) ([]entities.GmailAccount, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying gmail accounts: %w", err)
	}
	defer rows.Close()

	var accounts []entities.GmailAccount
	for rows.Next() {
		var a entities.GmailAccount
		err := rows.Scan(
			&a.ID, &a.Email, &a.DisplayName,
			&a.OAuthClientID, &a.OAuthClientSecret,
			&a.RefreshToken, &a.AccessToken, &a.AccessTokenExpiry,
			&a.Status, &a.LastSyncAt, &a.LastHistoryID,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning gmail account row: %w", err)
		}
		accounts = append(accounts, a)
	}

	return accounts, rows.Err()
}
