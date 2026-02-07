package stores

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/gitops"
)

func (s *SqlcStore) CreateGitRepository(ctx context.Context, repository gitops.ModelGitRepository) (*gitops.ModelGitRepository, error) {
	createdAt := repository.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := repository.UpdatedAt
	if updatedAt == nil {
		updatedAt = &createdAt
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateGitRepository(ctx, pgdb.CreateGitRepositoryParams{
			ID:                     repository.ID,
			Name:                   repository.Name,
			Url:                    repository.URL,
			AuthType:               repository.AuthType,
			Username:               pgtype.Text{String: repository.Username, Valid: true},
			Token:                  pgtype.Text{String: repository.Token, Valid: true},
			SshKey:                 pgtype.Text{String: repository.SSHKey, Valid: true},
			Description:            nullableTextPtrKeepEmpty(repository.Description),
			Enabled:                repository.Enabled,
			SshHostKeyVerification: repository.SSHHostKeyVerification,
			CreatedAt:              nullableTimestamptz(createdAt),
			UpdatedAt:              nullableTimestamptzPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapGitRepositoryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateGitRepository(ctx, sqlitedb.CreateGitRepositoryParams{
			ID:                     repository.ID,
			Name:                   repository.Name,
			Url:                    repository.URL,
			AuthType:               repository.AuthType,
			Username:               sql.NullString{String: repository.Username, Valid: true},
			Token:                  sql.NullString{String: repository.Token, Valid: true},
			SshKey:                 sql.NullString{String: repository.SSHKey, Valid: true},
			Description:            nullableNullStringPtrKeepEmpty(repository.Description),
			Enabled:                boolToInt64(repository.Enabled),
			SshHostKeyVerification: repository.SSHHostKeyVerification,
			CreatedAt:              createdAt,
			UpdatedAt:              nullableNullTimePtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapGitRepositoryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetGitRepositoryByID(ctx context.Context, id string) (*gitops.ModelGitRepository, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetGitRepositoryByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitRepositoryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetGitRepositoryByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitRepositoryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetGitRepositoryByName(ctx context.Context, name string) (*gitops.ModelGitRepository, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetGitRepositoryByName(ctx, name)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitRepositoryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetGitRepositoryByName(ctx, name)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitRepositoryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListGitRepositories(ctx context.Context) ([]gitops.ModelGitRepository, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListGitRepositories(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitRepository, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitRepositoryFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListGitRepositories(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]gitops.ModelGitRepository, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapGitRepositoryFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveGitRepository(ctx context.Context, repository gitops.ModelGitRepository) (*gitops.ModelGitRepository, error) {
	updatedAt := repository.UpdatedAt
	if updatedAt == nil {
		now := time.Now()
		updatedAt = &now
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.SaveGitRepository(ctx, pgdb.SaveGitRepositoryParams{
			Name:                   repository.Name,
			Url:                    repository.URL,
			AuthType:               repository.AuthType,
			Username:               pgtype.Text{String: repository.Username, Valid: true},
			Token:                  pgtype.Text{String: repository.Token, Valid: true},
			SshKey:                 pgtype.Text{String: repository.SSHKey, Valid: true},
			Description:            nullableTextPtrKeepEmpty(repository.Description),
			Enabled:                repository.Enabled,
			SshHostKeyVerification: repository.SSHHostKeyVerification,
			UpdatedAt:              nullableTimestamptzPtr(updatedAt),
			ID:                     repository.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitRepositoryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.SaveGitRepository(ctx, sqlitedb.SaveGitRepositoryParams{
			Name:                   repository.Name,
			Url:                    repository.URL,
			AuthType:               repository.AuthType,
			Username:               sql.NullString{String: repository.Username, Valid: true},
			Token:                  sql.NullString{String: repository.Token, Valid: true},
			SshKey:                 sql.NullString{String: repository.SSHKey, Valid: true},
			Description:            nullableNullStringPtrKeepEmpty(repository.Description),
			Enabled:                boolToInt64(repository.Enabled),
			SshHostKeyVerification: repository.SSHHostKeyVerification,
			UpdatedAt:              nullableNullTimePtr(updatedAt),
			ID:                     repository.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapGitRepositoryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteGitRepositoryByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteGitRepositoryByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteGitRepositoryByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CountGitOpsSyncsByRepositoryID(ctx context.Context, repositoryID string) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.CountGitOpsSyncsByRepositoryID(ctx, repositoryID)
	case "sqlite":
		return s.sqlite.CountGitOpsSyncsByRepositoryID(ctx, repositoryID)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
