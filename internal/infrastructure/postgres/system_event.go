package postgres

import (
	"context"
	"fmt"

	"github.com/mostaql-notification/internal/domain/entities"
)

// SystemEventRepo implements repositories.SystemEventRepository using PostgreSQL.
type SystemEventRepo struct {
	db *DB
}

// NewSystemEventRepo creates a new PostgreSQL-backed system event repository.
func NewSystemEventRepo(db *DB) *SystemEventRepo {
	return &SystemEventRepo{db: db}
}

func (r *SystemEventRepo) Create(ctx context.Context, event *entities.SystemEvent) error {
	query := `
		INSERT INTO system_events (id, event_type, details, severity, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Pool.Exec(ctx, query,
		event.ID, event.Type, event.Details, event.Severity, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating system event: %w", err)
	}
	return nil
}

func (r *SystemEventRepo) GetRecent(ctx context.Context, limit int) ([]entities.SystemEvent, error) {
	query := `
		SELECT id, event_type, details, severity, created_at
		FROM system_events
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("getting recent events: %w", err)
	}
	defer rows.Close()

	var events []entities.SystemEvent
	for rows.Next() {
		var e entities.SystemEvent
		err := rows.Scan(&e.ID, &e.Type, &e.Details, &e.Severity, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning event row: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
