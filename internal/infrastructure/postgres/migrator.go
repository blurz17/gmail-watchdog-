package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migrator handles database schema migrations.
type Migrator struct {
	db     *DB
	logger *slog.Logger
}

// NewMigrator creates a new database migrator.
func NewMigrator(db *DB, logger *slog.Logger) *Migrator {
	return &Migrator{db: db, logger: logger}
}

// Up runs all pending up migrations from the given directory.
func (m *Migrator) Up(ctx context.Context, migrationsDir string) error {
	// Ensure the migrations tracking table exists.
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("ensuring migrations table: %w", err)
	}

	// Find all .up.sql files.
	files, err := m.findMigrations(migrationsDir, ".up.sql")
	if err != nil {
		return fmt.Errorf("finding migrations: %w", err)
	}

	if len(files) == 0 {
		m.logger.Info("no migration files found", "dir", migrationsDir)
		return nil
	}

	// Get already-applied migrations.
	applied, err := m.getApplied(ctx)
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	for _, file := range files {
		name := filepath.Base(file)
		if applied[name] {
			m.logger.Debug("migration already applied, skipping", "migration", name)
			continue
		}

		m.logger.Info("applying migration", "migration", name)

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		tx, err := m.db.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning transaction for %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("executing migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("recording migration %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing migration %s: %w", name, err)
		}

		m.logger.Info("migration applied successfully", "migration", name)
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`
	_, err := m.db.Pool.Exec(ctx, query)
	return err
}

func (m *Migrator) getApplied(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.Pool.Query(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) findMigrations(dir string, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}
