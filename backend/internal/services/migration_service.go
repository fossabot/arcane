package services

import (
	"context"
	"fmt"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/types/migration"
)

const migrationTimeLayout = "2006-01-02 15:04:05"

// MigrationStatus represents a simplified migration status from the database layer.
type MigrationStatus struct {
	Version   int64
	State     string
	AppliedAt time.Time
	Path      string
}

// MigrationStore defines the persistence operations needed for migration management.
type MigrationStore interface {
	Status(ctx context.Context) ([]MigrationStatus, error)
	Down(ctx context.Context, steps int) error
	DownTo(ctx context.Context, version int64) error
	Redo(ctx context.Context) error
}

// MigrationService provides migration management operations for the API.
type MigrationService struct {
	store MigrationStore
}

// NewMigrationService constructs a MigrationService backed by the database.
func NewMigrationService(db *database.DB) *MigrationService {
	return NewMigrationServiceWithStore(&dbMigrationStore{db: db})
}

// NewMigrationServiceWithStore constructs a MigrationService backed by a custom store.
func NewMigrationServiceWithStore(store MigrationStore) *MigrationService {
	return &MigrationService{store: store}
}

// Status returns the current migration status list.
func (s *MigrationService) Status(ctx context.Context) ([]migration.Status, error) {
	statuses, err := s.store.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	result := make([]migration.Status, len(statuses))
	for i, status := range statuses {
		appliedAt := ""
		if !status.AppliedAt.IsZero() {
			appliedAt = status.AppliedAt.Format(migrationTimeLayout)
		}
		result[i] = migration.Status{
			Version:   status.Version,
			State:     status.State,
			AppliedAt: appliedAt,
			Path:      status.Path,
		}
	}

	return result, nil
}

// Down rolls back migrations by the specified number of steps.
func (s *MigrationService) Down(ctx context.Context, steps int) error {
	if err := s.store.Down(ctx, steps); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

// DownTo rolls back migrations down to a specific version.
func (s *MigrationService) DownTo(ctx context.Context, version int64) error {
	if err := s.store.DownTo(ctx, version); err != nil {
		return fmt.Errorf("failed to rollback migrations to version %d: %w", version, err)
	}
	return nil
}

// Redo rolls back and re-applies the most recent migration.
func (s *MigrationService) Redo(ctx context.Context) error {
	if err := s.store.Redo(ctx); err != nil {
		return fmt.Errorf("failed to redo latest migration: %w", err)
	}
	return nil
}

type dbMigrationStore struct {
	db *database.DB
}

func (s *dbMigrationStore) Status(ctx context.Context) ([]MigrationStatus, error) {
	statuses, err := s.db.MigrateStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration status: %w", err)
	}

	result := make([]MigrationStatus, len(statuses))
	for i, status := range statuses {
		result[i] = MigrationStatus{
			Version:   status.Source.Version,
			State:     fmt.Sprint(status.State),
			AppliedAt: status.AppliedAt,
			Path:      status.Source.Path,
		}
	}

	return result, nil
}

func (s *dbMigrationStore) Down(ctx context.Context, steps int) error {
	if err := s.db.MigrateDown(ctx, steps); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

func (s *dbMigrationStore) DownTo(ctx context.Context, version int64) error {
	if err := s.db.MigrateDownTo(ctx, version); err != nil {
		return fmt.Errorf("failed to rollback migrations to version %d: %w", version, err)
	}
	return nil
}

func (s *dbMigrationStore) Redo(ctx context.Context) error {
	if err := s.db.MigrateRedo(ctx); err != nil {
		return fmt.Errorf("failed to redo latest migration: %w", err)
	}
	return nil
}

var _ MigrationStore = (*dbMigrationStore)(nil)
