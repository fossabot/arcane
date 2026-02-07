package stores

import (
	"context"
	"fmt"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/notification"
)

func (s *SqlcStore) GetAppriseSettings(ctx context.Context) (*notification.AppriseSettings, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetAppriseSettings(ctx)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapAppriseSettingFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetAppriseSettings(ctx)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapAppriseSettingFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpsertAppriseSettings(ctx context.Context, apiURL string, enabled bool, imageUpdateTag, containerUpdateTag string) (*notification.AppriseSettings, error) {
	current, err := s.GetAppriseSettings(ctx)
	if err != nil {
		return nil, err
	}

	switch s.driver {
	case "postgres":
		if current == nil {
			row, err := s.pg.CreateAppriseSettings(ctx, pgdb.CreateAppriseSettingsParams{
				ApiUrl:             apiURL,
				Enabled:            boolToPgBool(enabled),
				ImageUpdateTag:     nullableText(imageUpdateTag),
				ContainerUpdateTag: nullableText(containerUpdateTag),
			})
			if err != nil {
				return nil, err
			}
			return mapAppriseSettingFromPG(row), nil
		}
		row, err := s.pg.UpdateAppriseSettings(ctx, pgdb.UpdateAppriseSettingsParams{
			ID:                 int32(current.ID), //nolint:gosec // IDs are non-negative in the database
			ApiUrl:             apiURL,
			Enabled:            boolToPgBool(enabled),
			ImageUpdateTag:     nullableText(imageUpdateTag),
			ContainerUpdateTag: nullableText(containerUpdateTag),
		})
		if err != nil {
			return nil, err
		}
		return mapAppriseSettingFromPG(row), nil
	case "sqlite":
		if current == nil {
			row, err := s.sqlite.CreateAppriseSettings(ctx, sqlitedb.CreateAppriseSettingsParams{
				ApiUrl:             apiURL,
				Enabled:            boolToNullInt(enabled),
				ImageUpdateTag:     nullableString(imageUpdateTag),
				ContainerUpdateTag: nullableString(containerUpdateTag),
			})
			if err != nil {
				return nil, err
			}
			return mapAppriseSettingFromSQLite(row), nil
		}
		row, err := s.sqlite.UpdateAppriseSettings(ctx, sqlitedb.UpdateAppriseSettingsParams{
			ApiUrl:             apiURL,
			Enabled:            boolToNullInt(enabled),
			ImageUpdateTag:     nullableString(imageUpdateTag),
			ContainerUpdateTag: nullableString(containerUpdateTag),
			ID:                 int64(current.ID), //nolint:gosec // IDs are non-negative in the database
		})
		if err != nil {
			return nil, err
		}
		return mapAppriseSettingFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
