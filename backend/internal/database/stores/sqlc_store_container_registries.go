package stores

import (
	"context"
	"fmt"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/containerregistry"
)

func (s *SqlcStore) CreateContainerRegistry(ctx context.Context, input ContainerRegistryCreateInput) (*containerregistry.ModelContainerRegistry, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateContainerRegistry(ctx, pgdb.CreateContainerRegistryParams{
			ID:          input.ID,
			Url:         input.URL,
			Username:    input.Username,
			Token:       input.Token,
			Description: nullableTextPtrKeepEmpty(input.Description),
			Insecure:    input.Insecure,
			Enabled:     input.Enabled,
		})
		if err != nil {
			return nil, err
		}
		return mapContainerRegistryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateContainerRegistry(ctx, sqlitedb.CreateContainerRegistryParams{
			ID:          input.ID,
			Url:         input.URL,
			Username:    input.Username,
			Token:       input.Token,
			Description: nullableNullStringPtrKeepEmpty(input.Description),
			Insecure:    input.Insecure,
			Enabled:     input.Enabled,
		})
		if err != nil {
			return nil, err
		}
		return mapContainerRegistryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetContainerRegistryByID(ctx context.Context, id string) (*containerregistry.ModelContainerRegistry, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetContainerRegistryByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapContainerRegistryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetContainerRegistryByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapContainerRegistryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListContainerRegistries(ctx context.Context) ([]containerregistry.ModelContainerRegistry, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListContainerRegistries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]containerregistry.ModelContainerRegistry, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapContainerRegistryFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListContainerRegistries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]containerregistry.ModelContainerRegistry, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapContainerRegistryFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListEnabledContainerRegistries(ctx context.Context) ([]containerregistry.ModelContainerRegistry, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListEnabledContainerRegistries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]containerregistry.ModelContainerRegistry, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapContainerRegistryFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListEnabledContainerRegistries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]containerregistry.ModelContainerRegistry, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapContainerRegistryFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateContainerRegistry(ctx context.Context, input ContainerRegistryUpdateInput) (*containerregistry.ModelContainerRegistry, error) {
	switch s.driver {
	case "postgres":
		if err := s.pg.UpdateContainerRegistry(ctx, pgdb.UpdateContainerRegistryParams{
			ID:          input.ID,
			Url:         input.URL,
			Username:    input.Username,
			Token:       input.Token,
			Description: nullableTextPtrKeepEmpty(input.Description),
			Insecure:    input.Insecure,
			Enabled:     input.Enabled,
		}); err != nil {
			return nil, err
		}
	case "sqlite":
		if err := s.sqlite.UpdateContainerRegistry(ctx, sqlitedb.UpdateContainerRegistryParams{
			Url:         input.URL,
			Username:    input.Username,
			Token:       input.Token,
			Description: nullableNullStringPtrKeepEmpty(input.Description),
			Insecure:    input.Insecure,
			Enabled:     input.Enabled,
			ID:          input.ID,
		}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}

	return s.GetContainerRegistryByID(ctx, input.ID)
}

func (s *SqlcStore) DeleteContainerRegistryByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteContainerRegistryByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteContainerRegistryByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
