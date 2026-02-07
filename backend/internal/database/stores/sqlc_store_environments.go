package stores

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/environment"
)

func (s *SqlcStore) CreateEnvironment(ctx context.Context, input EnvironmentCreateInput) (*environment.ModelEnvironment, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateEnvironment(ctx, pgdb.CreateEnvironmentParams{
			ID:          input.ID,
			Name:        pgtype.Text{String: input.Name, Valid: true},
			ApiUrl:      input.APIURL,
			Status:      input.Status,
			Enabled:     input.Enabled,
			IsEdge:      input.IsEdge,
			LastSeen:    nullableTimestamptzPtr(input.LastSeen),
			AccessToken: nullableTextPtrKeepEmpty(input.AccessToken),
			ApiKeyID:    nullableTextPtrKeepEmpty(input.ApiKeyID),
			CreatedAt:   nullableTimestamptz(input.CreatedAt),
			UpdatedAt:   nullableTimestamptzPtr(input.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapEnvironmentFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateEnvironment(ctx, sqlitedb.CreateEnvironmentParams{
			ID:          input.ID,
			Name:        sql.NullString{String: input.Name, Valid: true},
			ApiUrl:      input.APIURL,
			Status:      input.Status,
			Enabled:     input.Enabled,
			IsEdge:      boolToInt64(input.IsEdge),
			LastSeen:    nullableNullTimePtr(input.LastSeen),
			AccessToken: nullableNullStringPtrKeepEmpty(input.AccessToken),
			ApiKeyID:    nullableNullStringPtrKeepEmpty(input.ApiKeyID),
			CreatedAt:   input.CreatedAt,
			UpdatedAt:   nullableNullTimePtr(input.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapEnvironmentFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetEnvironmentByID(ctx context.Context, id string) (*environment.ModelEnvironment, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetEnvironmentByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapEnvironmentFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetEnvironmentByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapEnvironmentFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListEnvironments(ctx context.Context) ([]environment.ModelEnvironment, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListEnvironments(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]environment.ModelEnvironment, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapEnvironmentFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListEnvironments(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]environment.ModelEnvironment, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapEnvironmentFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListRemoteEnvironments(ctx context.Context) ([]environment.ModelEnvironment, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListRemoteEnvironments(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]environment.ModelEnvironment, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapEnvironmentFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListRemoteEnvironments(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]environment.ModelEnvironment, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapEnvironmentFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) FindEnvironmentIDByApiKeyHash(ctx context.Context, keyHash string) (string, error) {
	switch s.driver {
	case "postgres":
		id, err := s.pg.FindEnvironmentIDByApiKeyHash(ctx, keyHash)
		if err != nil {
			if isNotFound(err) {
				return "", nil
			}
			return "", err
		}
		return id, nil
	case "sqlite":
		id, err := s.sqlite.FindEnvironmentIDByApiKeyHash(ctx, keyHash)
		if err != nil {
			if isNotFound(err) {
				return "", nil
			}
			return "", err
		}
		return id, nil
	default:
		return "", fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

//nolint:gocognit // TODO: refactor patch logic to reduce complexity.
func (s *SqlcStore) PatchEnvironment(ctx context.Context, input EnvironmentPatchInput) (*environment.ModelEnvironment, error) {
	switch s.driver {
	case "postgres":
		params := pgdb.PatchEnvironmentParams{
			ID:               input.ID,
			ClearLastSeen:    input.ClearLastSeen,
			ClearAccessToken: input.ClearAccessToken,
			ClearApiKeyID:    input.ClearApiKeyID,
			LastSeen:         nullableTimestamptzPtr(input.LastSeen),
			AccessToken:      nullableTextPtrKeepEmpty(input.AccessToken),
			ApiKeyID:         nullableTextPtrKeepEmpty(input.ApiKeyID),
			UpdatedAt:        nullableTimestamptzPtr(input.UpdatedAt),
		}
		if input.Name != nil {
			params.Name = pgtype.Text{String: *input.Name, Valid: true}
		}
		if input.APIURL != nil {
			params.ApiUrl = pgtype.Text{String: *input.APIURL, Valid: true}
		}
		if input.Status != nil {
			params.Status = pgtype.Text{String: *input.Status, Valid: true}
		}
		if input.Enabled != nil {
			params.Enabled = pgtype.Bool{Bool: *input.Enabled, Valid: true}
		}
		if input.IsEdge != nil {
			params.IsEdge = pgtype.Bool{Bool: *input.IsEdge, Valid: true}
		}

		row, err := s.pg.PatchEnvironment(ctx, params)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapEnvironmentFromPG(row), nil
	case "sqlite":
		params := sqlitedb.PatchEnvironmentParams{
			ID:               input.ID,
			ClearLastSeen:    boolToInt64(input.ClearLastSeen),
			ClearAccessToken: boolToInt64(input.ClearAccessToken),
			ClearApiKeyID:    boolToInt64(input.ClearApiKeyID),
			LastSeen:         nullableNullTimePtr(input.LastSeen),
			AccessToken:      nullableNullStringPtrKeepEmpty(input.AccessToken),
			ApiKeyID:         nullableNullStringPtrKeepEmpty(input.ApiKeyID),
			UpdatedAt:        nullableNullTimePtr(input.UpdatedAt),
		}
		if input.Name != nil {
			params.Name = sql.NullString{String: *input.Name, Valid: true}
		}
		if input.APIURL != nil {
			params.ApiUrl = sql.NullString{String: *input.APIURL, Valid: true}
		}
		if input.Status != nil {
			params.Status = sql.NullString{String: *input.Status, Valid: true}
		}
		if input.Enabled != nil {
			params.Enabled = sql.NullBool{Bool: *input.Enabled, Valid: true}
		}
		if input.IsEdge != nil {
			params.IsEdge = sql.NullInt64{Int64: boolToInt64(*input.IsEdge), Valid: true}
		}

		row, err := s.sqlite.PatchEnvironment(ctx, params)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapEnvironmentFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteEnvironmentByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteEnvironmentByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteEnvironmentByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) TouchEnvironmentHeartbeatIfStale(ctx context.Context, id string, now time.Time, staleBefore time.Time) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.TouchEnvironmentHeartbeatIfStale(ctx, pgdb.TouchEnvironmentHeartbeatIfStaleParams{
			ID:         id,
			LastSeen:   nullableTimestamptz(now),
			Status:     string(environment.EnvironmentStatusOnline),
			LastSeen_2: nullableTimestamptz(staleBefore),
		})
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.TouchEnvironmentHeartbeatIfStale(ctx, sqlitedb.TouchEnvironmentHeartbeatIfStaleParams{
			LastSeen:   nullableNullTime(now),
			Status:     string(environment.EnvironmentStatusOnline),
			UpdatedAt:  nullableNullTime(now),
			ID:         id,
			LastSeen_2: nullableNullTime(staleBefore),
		})
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
