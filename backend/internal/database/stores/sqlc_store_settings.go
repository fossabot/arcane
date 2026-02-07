package stores

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/settings"
)

func (s *SqlcStore) WithSettingsTx(ctx context.Context, fn func(tx SettingsStoreTx) error) error {
	switch s.driver {
	case "postgres":
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction begin", "driver", "postgres")
		}
		tx, err := s.pgPool.Begin(ctx)
		if err != nil {
			return err
		}
		store := s.withPgTx(tx)
		if err := fn(store); err != nil {
			rollbackErr := tx.Rollback(ctx)
			if slog.Default().Enabled(ctx, slog.LevelDebug) {
				slog.DebugContext(ctx, "Database transaction rollback", "driver", "postgres", "error", err, "rollback_error", rollbackErr)
			}
			if rollbackErr != nil {
				return fmt.Errorf("transaction failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return err
		}
		commitErr := tx.Commit(ctx)
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction commit", "driver", "postgres", "error", commitErr)
		}
		return commitErr
	case "sqlite":
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction begin", "driver", "sqlite")
		}
		tx, err := s.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		store := s.withSQLiteTx(tx)
		if err := fn(store); err != nil {
			rollbackErr := tx.Rollback()
			if slog.Default().Enabled(ctx, slog.LevelDebug) {
				slog.DebugContext(ctx, "Database transaction rollback", "driver", "sqlite", "error", err, "rollback_error", rollbackErr)
			}
			if rollbackErr != nil {
				return fmt.Errorf("transaction failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return err
		}
		commitErr := tx.Commit()
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction commit", "driver", "sqlite", "error", commitErr)
		}
		return commitErr
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListSettings(ctx context.Context) ([]settings.SettingVariable, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListSettings(ctx)
		if err != nil {
			return nil, err
		}
		settingVars := make([]settings.SettingVariable, 0, len(rows))
		for _, row := range rows {
			settingVars = append(settingVars, settings.SettingVariable{Key: row.Key, Value: row.Value})
		}
		return settingVars, nil
	case "sqlite":
		rows, err := s.sqlite.ListSettings(ctx)
		if err != nil {
			return nil, err
		}
		settingVars := make([]settings.SettingVariable, 0, len(rows))
		for _, row := range rows {
			settingVars = append(settingVars, settings.SettingVariable{Key: row.Key, Value: row.Value})
		}
		return settingVars, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetSetting(ctx context.Context, key string) (*settings.SettingVariable, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetSetting(ctx, key)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return &settings.SettingVariable{Key: row.Key, Value: row.Value}, nil
	case "sqlite":
		row, err := s.sqlite.GetSetting(ctx, key)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return &settings.SettingVariable{Key: row.Key, Value: row.Value}, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpsertSetting(ctx context.Context, key, value string) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpsertSetting(ctx, pgdb.UpsertSettingParams{Key: key, Value: value})
	case "sqlite":
		return s.sqlite.UpsertSetting(ctx, sqlitedb.UpsertSettingParams{Key: key, Value: value})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) InsertSettingIfNotExists(ctx context.Context, key, value string) error {
	switch s.driver {
	case "postgres":
		return s.pg.InsertSettingIfNotExists(ctx, pgdb.InsertSettingIfNotExistsParams{Key: key, Value: value})
	case "sqlite":
		return s.sqlite.InsertSettingIfNotExists(ctx, sqlitedb.InsertSettingIfNotExistsParams{Key: key, Value: value})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteSetting(ctx context.Context, key string) error {
	switch s.driver {
	case "postgres":
		_, err := s.pg.DeleteSetting(ctx, key)
		return err
	case "sqlite":
		_, err := s.sqlite.DeleteSetting(ctx, key)
		return err
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateSettingKey(ctx context.Context, oldKey, newKey string) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateSettingKey(ctx, pgdb.UpdateSettingKeyParams{Key: oldKey, Key_2: newKey})
	case "sqlite":
		return s.sqlite.UpdateSettingKey(ctx, sqlitedb.UpdateSettingKeyParams{Key: oldKey, Key_2: newKey})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteSettingsNotIn(ctx context.Context, keys []string) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.DeleteSettingsNotIn(ctx, keys)
	case "sqlite":
		return s.sqlite.DeleteSettingsNotIn(ctx, keys)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
