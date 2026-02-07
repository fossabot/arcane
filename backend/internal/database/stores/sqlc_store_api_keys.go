package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/apikey"
)

func (s *SqlcStore) CreateApiKey(ctx context.Context, input ApiKeyCreateInput) (*apikey.ModelApiKey, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateApiKey(ctx, pgdb.CreateApiKeyParams{
			ID:            input.ID,
			Name:          input.Name,
			Description:   nullableTextPtrKeepEmpty(input.Description),
			KeyHash:       input.KeyHash,
			KeyPrefix:     input.KeyPrefix,
			UserID:        input.UserID,
			EnvironmentID: nullableTextPtrKeepEmpty(input.EnvironmentID),
			ExpiresAt:     nullableTimestamptzPtr(input.ExpiresAt),
			LastUsedAt:    nullableTimestamptzPtr(input.LastUsedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapApiKeyFromPGValues(
			row.ID,
			row.Name,
			row.Description,
			row.KeyHash,
			row.KeyPrefix,
			row.UserID,
			row.EnvironmentID,
			row.ExpiresAt,
			row.LastUsedAt,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.CreateApiKey(ctx, sqlitedb.CreateApiKeyParams{
			ID:            input.ID,
			Name:          input.Name,
			Description:   nullableNullStringPtrKeepEmpty(input.Description),
			KeyHash:       input.KeyHash,
			KeyPrefix:     input.KeyPrefix,
			UserID:        input.UserID,
			EnvironmentID: nullableNullStringPtrKeepEmpty(input.EnvironmentID),
			ExpiresAt:     nullableNullTimePtr(input.ExpiresAt),
			LastUsedAt:    nullableNullTimePtr(input.LastUsedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapApiKeyFromSQLiteValues(
			row.ID,
			row.Name,
			row.Description,
			row.KeyHash,
			row.KeyPrefix,
			row.UserID,
			row.EnvironmentID,
			row.ExpiresAt,
			row.LastUsedAt,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetApiKeyByID(ctx context.Context, id string) (*apikey.ModelApiKey, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetApiKeyByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapApiKeyFromPGValues(
			row.ID,
			row.Name,
			row.Description,
			row.KeyHash,
			row.KeyPrefix,
			row.UserID,
			row.EnvironmentID,
			row.ExpiresAt,
			row.LastUsedAt,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.GetApiKeyByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapApiKeyFromSQLiteValues(
			row.ID,
			row.Name,
			row.Description,
			row.KeyHash,
			row.KeyPrefix,
			row.UserID,
			row.EnvironmentID,
			row.ExpiresAt,
			row.LastUsedAt,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListApiKeys(ctx context.Context) ([]apikey.ModelApiKey, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListApiKeys(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]apikey.ModelApiKey, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapApiKeyFromPGValues(
				row.ID,
				row.Name,
				row.Description,
				row.KeyHash,
				row.KeyPrefix,
				row.UserID,
				row.EnvironmentID,
				row.ExpiresAt,
				row.LastUsedAt,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListApiKeys(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]apikey.ModelApiKey, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapApiKeyFromSQLiteValues(
				row.ID,
				row.Name,
				row.Description,
				row.KeyHash,
				row.KeyPrefix,
				row.UserID,
				row.EnvironmentID,
				row.ExpiresAt,
				row.LastUsedAt,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListApiKeysByPrefix(ctx context.Context, keyPrefix string) ([]apikey.ModelApiKey, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListApiKeysByPrefix(ctx, keyPrefix)
		if err != nil {
			return nil, err
		}
		items := make([]apikey.ModelApiKey, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapApiKeyFromPGValues(
				row.ID,
				row.Name,
				row.Description,
				row.KeyHash,
				row.KeyPrefix,
				row.UserID,
				row.EnvironmentID,
				row.ExpiresAt,
				row.LastUsedAt,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListApiKeysByPrefix(ctx, keyPrefix)
		if err != nil {
			return nil, err
		}
		items := make([]apikey.ModelApiKey, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapApiKeyFromSQLiteValues(
				row.ID,
				row.Name,
				row.Description,
				row.KeyHash,
				row.KeyPrefix,
				row.UserID,
				row.EnvironmentID,
				row.ExpiresAt,
				row.LastUsedAt,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateApiKey(ctx context.Context, input ApiKeyUpdateInput) (*apikey.ModelApiKey, error) {
	switch s.driver {
	case "postgres":
		if err := s.pg.UpdateApiKey(ctx, pgdb.UpdateApiKeyParams{
			ID:          input.ID,
			Name:        input.Name,
			Description: nullableTextPtrKeepEmpty(input.Description),
			ExpiresAt:   nullableTimestamptzPtr(input.ExpiresAt),
		}); err != nil {
			return nil, err
		}
	case "sqlite":
		if err := s.sqlite.UpdateApiKey(ctx, sqlitedb.UpdateApiKeyParams{
			Name:        input.Name,
			Description: nullableNullStringPtrKeepEmpty(input.Description),
			ExpiresAt:   nullableNullTimePtr(input.ExpiresAt),
			ID:          input.ID,
		}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
	return s.GetApiKeyByID(ctx, input.ID)
}

func (s *SqlcStore) DeleteApiKeyByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteApiKeyByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteApiKeyByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) TouchApiKeyLastUsed(ctx context.Context, id string, lastUsedAt time.Time) error {
	switch s.driver {
	case "postgres":
		return s.pg.TouchApiKeyLastUsed(ctx, pgdb.TouchApiKeyLastUsedParams{
			ID:         id,
			LastUsedAt: nullableTimestamptz(lastUsedAt),
		})
	case "sqlite":
		return s.sqlite.TouchApiKeyLastUsed(ctx, sqlitedb.TouchApiKeyLastUsedParams{
			LastUsedAt: nullableNullTime(lastUsedAt),
			ID:         id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
