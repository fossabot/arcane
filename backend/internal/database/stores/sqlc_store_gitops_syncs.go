package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/gitops"
)

func (s *SqlcStore) CreateGitOpsSync(ctx context.Context, sync gitops.ModelGitOpsSync) (*gitops.ModelGitOpsSync, error) {
	createdAt := sync.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := sync.UpdatedAt
	if updatedAt == nil {
		updatedAt = &createdAt
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateGitOpsSync(ctx, pgdb.CreateGitOpsSyncParams{
			ID:             sync.ID,
			Name:           sync.Name,
			EnvironmentID:  sync.EnvironmentID,
			RepositoryID:   sync.RepositoryID,
			Branch:         sync.Branch,
			ComposePath:    sync.ComposePath,
			ProjectName:    sync.ProjectName,
			ProjectID:      nullableTextPtrKeepEmpty(sync.ProjectID),
			AutoSync:       sync.AutoSync,
			SyncInterval:   int32(sync.SyncInterval),
			LastSyncAt:     nullableTimestamptzPtr(sync.LastSyncAt),
			LastSyncStatus: nullableTextPtrKeepEmpty(sync.LastSyncStatus),
			LastSyncError:  nullableTextPtrKeepEmpty(sync.LastSyncError),
			LastSyncCommit: nullableTextPtrKeepEmpty(sync.LastSyncCommit),
			CreatedAt:      nullableTimestamptz(createdAt),
			UpdatedAt:      nullableTimestamptzPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapGitOpsSyncFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateGitOpsSync(ctx, sqlitedb.CreateGitOpsSyncParams{
			ID:             sync.ID,
			Name:           sync.Name,
			EnvironmentID:  sync.EnvironmentID,
			RepositoryID:   sync.RepositoryID,
			Branch:         sync.Branch,
			ComposePath:    sync.ComposePath,
			ProjectName:    sync.ProjectName,
			ProjectID:      nullableNullStringPtrKeepEmpty(sync.ProjectID),
			AutoSync:       sync.AutoSync,
			SyncInterval:   int64(sync.SyncInterval),
			LastSyncAt:     nullableNullTimePtr(sync.LastSyncAt),
			LastSyncStatus: nullableNullStringPtrKeepEmpty(sync.LastSyncStatus),
			LastSyncError:  nullableNullStringPtrKeepEmpty(sync.LastSyncError),
			LastSyncCommit: nullableNullStringPtrKeepEmpty(sync.LastSyncCommit),
			CreatedAt:      createdAt,
			UpdatedAt:      *updatedAt,
		})
		if err != nil {
			return nil, err
		}
		return mapGitOpsSyncFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetGitOpsSyncByID(ctx context.Context, id string) (*gitops.ModelGitOpsSync, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetGitOpsSyncByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitOpsSyncFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetGitOpsSyncByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitOpsSyncFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListGitOpsSyncs(ctx context.Context) ([]gitops.ModelGitOpsSync, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListGitOpsSyncs(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitOpsSync, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitOpsSyncFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListGitOpsSyncs(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitOpsSync, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitOpsSyncFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListGitOpsSyncsByEnvironment(ctx context.Context, environmentID string) ([]gitops.ModelGitOpsSync, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListGitOpsSyncsByEnvironment(ctx, environmentID)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitOpsSync, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitOpsSyncFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListGitOpsSyncsByEnvironment(ctx, environmentID)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitOpsSync, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitOpsSyncFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListAutoSyncGitOpsSyncs(ctx context.Context) ([]gitops.ModelGitOpsSync, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListAutoSyncGitOpsSyncs(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitOpsSync, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitOpsSyncFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListAutoSyncGitOpsSyncs(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitOpsSync, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitOpsSyncFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveGitOpsSync(ctx context.Context, sync gitops.ModelGitOpsSync) (*gitops.ModelGitOpsSync, error) {
	updatedAt := sync.UpdatedAt
	if updatedAt == nil {
		now := time.Now()
		updatedAt = &now
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.SaveGitOpsSync(ctx, pgdb.SaveGitOpsSyncParams{
			Name:           sync.Name,
			EnvironmentID:  sync.EnvironmentID,
			RepositoryID:   sync.RepositoryID,
			Branch:         sync.Branch,
			ComposePath:    sync.ComposePath,
			ProjectName:    sync.ProjectName,
			ProjectID:      nullableTextPtrKeepEmpty(sync.ProjectID),
			AutoSync:       sync.AutoSync,
			SyncInterval:   int32(sync.SyncInterval),
			LastSyncAt:     nullableTimestamptzPtr(sync.LastSyncAt),
			LastSyncStatus: nullableTextPtrKeepEmpty(sync.LastSyncStatus),
			LastSyncError:  nullableTextPtrKeepEmpty(sync.LastSyncError),
			LastSyncCommit: nullableTextPtrKeepEmpty(sync.LastSyncCommit),
			UpdatedAt:      nullableTimestamptzPtr(updatedAt),
			ID:             sync.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitOpsSyncFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.SaveGitOpsSync(ctx, sqlitedb.SaveGitOpsSyncParams{
			Name:           sync.Name,
			EnvironmentID:  sync.EnvironmentID,
			RepositoryID:   sync.RepositoryID,
			Branch:         sync.Branch,
			ComposePath:    sync.ComposePath,
			ProjectName:    sync.ProjectName,
			ProjectID:      nullableNullStringPtrKeepEmpty(sync.ProjectID),
			AutoSync:       sync.AutoSync,
			SyncInterval:   int64(sync.SyncInterval),
			LastSyncAt:     nullableNullTimePtr(sync.LastSyncAt),
			LastSyncStatus: nullableNullStringPtrKeepEmpty(sync.LastSyncStatus),
			LastSyncError:  nullableNullStringPtrKeepEmpty(sync.LastSyncError),
			LastSyncCommit: nullableNullStringPtrKeepEmpty(sync.LastSyncCommit),
			UpdatedAt:      *updatedAt,
			ID:             sync.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitOpsSyncFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteGitOpsSyncByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteGitOpsSyncByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteGitOpsSyncByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateGitOpsSyncInterval(ctx context.Context, id string, minutes int) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateGitOpsSyncInterval(ctx, pgdb.UpdateGitOpsSyncIntervalParams{
			SyncInterval: int32(minutes),
			ID:           id,
		})
	case "sqlite":
		return s.sqlite.UpdateGitOpsSyncInterval(ctx, sqlitedb.UpdateGitOpsSyncIntervalParams{
			SyncInterval: int64(minutes),
			ID:           id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateGitOpsSyncProjectID(ctx context.Context, id string, projectID *string) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateGitOpsSyncProjectID(ctx, pgdb.UpdateGitOpsSyncProjectIDParams{
			ProjectID: nullableTextPtrKeepEmpty(projectID),
			ID:        id,
		})
	case "sqlite":
		return s.sqlite.UpdateGitOpsSyncProjectID(ctx, sqlitedb.UpdateGitOpsSyncProjectIDParams{
			ProjectID: nullableNullStringPtrKeepEmpty(projectID),
			ID:        id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateGitOpsSyncStatus(ctx context.Context, id string, lastSyncAt time.Time, status string, errorMsg *string, commitHash *string) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateGitOpsSyncStatus(ctx, pgdb.UpdateGitOpsSyncStatusParams{
			LastSyncAt:     nullableTimestamptz(lastSyncAt),
			LastSyncStatus: nullableText(status),
			LastSyncError:  nullableTextPtrKeepEmpty(errorMsg),
			LastSyncCommit: nullableTextPtrKeepEmpty(commitHash),
			ID:             id,
		})
	case "sqlite":
		return s.sqlite.UpdateGitOpsSyncStatus(ctx, sqlitedb.UpdateGitOpsSyncStatusParams{
			LastSyncAt:     nullableNullTime(lastSyncAt),
			LastSyncStatus: nullableString(status),
			LastSyncError:  nullableNullStringPtrKeepEmpty(errorMsg),
			LastSyncCommit: nullableNullStringPtrKeepEmpty(commitHash),
			UpdatedAt:      lastSyncAt,
			ID:             id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SetProjectGitOpsManagedBy(ctx context.Context, projectID string, syncID *string) error {
	switch s.driver {
	case "postgres":
		return s.pg.SetProjectGitOpsManagedBy(ctx, pgdb.SetProjectGitOpsManagedByParams{
			GitopsManagedBy: nullableTextPtrKeepEmpty(syncID),
			ID:              projectID,
		})
	case "sqlite":
		return s.sqlite.SetProjectGitOpsManagedBy(ctx, sqlitedb.SetProjectGitOpsManagedByParams{
			GitopsManagedBy: nullableNullStringPtrKeepEmpty(syncID),
			ID:              projectID,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ClearProjectGitOpsManagedByIfMatches(ctx context.Context, projectID string, syncID string) error {
	switch s.driver {
	case "postgres":
		return s.pg.ClearProjectGitOpsManagedByIfMatches(ctx, pgdb.ClearProjectGitOpsManagedByIfMatchesParams{
			ID:              projectID,
			GitopsManagedBy: nullableText(syncID),
		})
	case "sqlite":
		return s.sqlite.ClearProjectGitOpsManagedByIfMatches(ctx, sqlitedb.ClearProjectGitOpsManagedByIfMatchesParams{
			ID:              projectID,
			GitopsManagedBy: nullableString(syncID),
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
