package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mostaql-notification/internal/domain/entities"
)

// ProviderRepo implements repositories.ProviderRepository using PostgreSQL.
type ProviderRepo struct {
	db *DB
}

// NewProviderRepo creates a new PostgreSQL-backed provider repository.
func NewProviderRepo(db *DB) *ProviderRepo {
	return &ProviderRepo{db: db}
}

func (r *ProviderRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Provider, error) {
	query := `
		SELECT id, name, is_active, detection_mode, created_at, updated_at
		FROM providers
		WHERE id = $1`

	var p entities.Provider
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.IsActive, &p.DetectionMode,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting provider by ID: %w", err)
	}
	return &p, nil
}

func (r *ProviderRepo) GetByName(ctx context.Context, name string) (*entities.Provider, error) {
	query := `
		SELECT id, name, is_active, detection_mode, created_at, updated_at
		FROM providers
		WHERE LOWER(name) = LOWER($1)`

	var p entities.Provider
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.IsActive, &p.DetectionMode,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting provider by name: %w", err)
	}
	return &p, nil
}

func (r *ProviderRepo) GetAll(ctx context.Context) ([]entities.Provider, error) {
	query := `
		SELECT id, name, is_active, detection_mode, created_at, updated_at
		FROM providers
		ORDER BY name`

	return r.scanMany(ctx, query)
}

func (r *ProviderRepo) GetActive(ctx context.Context) ([]entities.Provider, error) {
	query := `
		SELECT id, name, is_active, detection_mode, created_at, updated_at
		FROM providers
		WHERE is_active = TRUE
		ORDER BY name`

	return r.scanMany(ctx, query)
}

func (r *ProviderRepo) Create(ctx context.Context, provider *entities.Provider) error {
	query := `
		INSERT INTO providers (id, name, is_active, detection_mode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Pool.Exec(ctx, query,
		provider.ID, provider.Name, provider.IsActive,
		provider.DetectionMode, provider.CreatedAt, provider.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating provider: %w", err)
	}
	return nil
}

func (r *ProviderRepo) Update(ctx context.Context, provider *entities.Provider) error {
	query := `
		UPDATE providers
		SET name = $2, is_active = $3, detection_mode = $4, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query,
		provider.ID, provider.Name, provider.IsActive, provider.DetectionMode,
	)
	if err != nil {
		return fmt.Errorf("updating provider: %w", err)
	}
	return nil
}

func (r *ProviderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM providers WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting provider: %w", err)
	}
	return nil
}

func (r *ProviderRepo) scanMany(ctx context.Context, query string, args ...interface{}) ([]entities.Provider, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying providers: %w", err)
	}
	defer rows.Close()

	var providers []entities.Provider
	for rows.Next() {
		var p entities.Provider
		err := rows.Scan(
			&p.ID, &p.Name, &p.IsActive, &p.DetectionMode,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning provider row: %w", err)
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}
