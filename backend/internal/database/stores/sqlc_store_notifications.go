package stores

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/notification"
)

func (s *SqlcStore) ListNotificationSettings(ctx context.Context) ([]notification.NotificationSettings, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListNotificationSettings(ctx)
		if err != nil {
			return nil, err
		}
		settings := make([]notification.NotificationSettings, 0, len(rows))
		for _, row := range rows {
			settings = append(settings, mapNotificationSettingFromPG(row))
		}
		return settings, nil
	case "sqlite":
		rows, err := s.sqlite.ListNotificationSettings(ctx)
		if err != nil {
			return nil, err
		}
		settings := make([]notification.NotificationSettings, 0, len(rows))
		for _, row := range rows {
			settings = append(settings, mapNotificationSettingFromSQLite(row))
		}
		return settings, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetNotificationSettingByProvider(ctx context.Context, provider notification.NotificationProvider) (*notification.NotificationSettings, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetNotificationSettingByProvider(ctx, string(provider))
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		setting := mapNotificationSettingFromPG(row)
		return &setting, nil
	case "sqlite":
		row, err := s.sqlite.GetNotificationSettingByProvider(ctx, string(provider))
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		setting := mapNotificationSettingFromSQLite(row)
		return &setting, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpsertNotificationSetting(ctx context.Context, provider notification.NotificationProvider, enabled bool, config base.JSON) (*notification.NotificationSettings, error) {
	current, err := s.GetNotificationSettingByProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	switch s.driver {
	case "postgres":
		if current == nil {
			row, err := s.pg.CreateNotificationSetting(ctx, pgdb.CreateNotificationSettingParams{
				Provider: string(provider),
				Enabled:  boolToPgBool(enabled),
				Config:   config,
			})
			if err != nil {
				return nil, err
			}
			setting := mapNotificationSettingFromPG(row)
			return &setting, nil
		}
		row, err := s.pg.UpdateNotificationSetting(ctx, pgdb.UpdateNotificationSettingParams{
			ID:      int32(current.ID), //nolint:gosec // IDs are non-negative in the database
			Enabled: boolToPgBool(enabled),
			Config:  config,
		})
		if err != nil {
			return nil, err
		}
		setting := mapNotificationSettingFromPG(row)
		return &setting, nil
	case "sqlite":
		if current == nil {
			row, err := s.sqlite.CreateNotificationSetting(ctx, sqlitedb.CreateNotificationSettingParams{
				Provider: string(provider),
				Enabled:  sql.NullBool{Bool: enabled, Valid: true},
				Config:   config,
			})
			if err != nil {
				return nil, err
			}
			setting := mapNotificationSettingFromSQLite(row)
			return &setting, nil
		}
		row, err := s.sqlite.UpdateNotificationSetting(ctx, sqlitedb.UpdateNotificationSettingParams{
			Enabled: sql.NullBool{Bool: enabled, Valid: true},
			Config:  config,
			ID:      int64(current.ID), //nolint:gosec // IDs are non-negative in the database
		})
		if err != nil {
			return nil, err
		}
		setting := mapNotificationSettingFromSQLite(row)
		return &setting, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteNotificationSetting(ctx context.Context, provider notification.NotificationProvider) error {
	switch s.driver {
	case "postgres":
		_, err := s.pg.DeleteNotificationSettingByProvider(ctx, string(provider))
		return err
	case "sqlite":
		_, err := s.sqlite.DeleteNotificationSettingByProvider(ctx, string(provider))
		return err
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CreateNotificationLog(ctx context.Context, log notification.NotificationLog) error {
	switch s.driver {
	case "postgres":
		return s.pg.CreateNotificationLog(ctx, pgdb.CreateNotificationLogParams{
			Provider: string(log.Provider),
			ImageRef: log.ImageRef,
			Status:   log.Status,
			Error:    nullableTextPtr(log.Error),
			Metadata: log.Metadata,
			SentAt:   timeToPgTimestamp(log.SentAt),
		})
	case "sqlite":
		return s.sqlite.CreateNotificationLog(ctx, sqlitedb.CreateNotificationLogParams{
			Provider: string(log.Provider),
			ImageRef: log.ImageRef,
			Status:   log.Status,
			Error:    nullableStringPtr(log.Error),
			Metadata: log.Metadata,
			SentAt:   log.SentAt,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
