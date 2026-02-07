package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/volume"
)

func (s *SqlcStore) CreateVolumeBackup(ctx context.Context, backup volume.VolumeBackup) (*volume.VolumeBackup, error) {
	if backup.ID == "" {
		backup.ID = uuid.NewString()
	}
	if backup.CreatedAt.IsZero() {
		backup.CreatedAt = time.Now().UTC()
	}
	if backup.UpdatedAt == nil {
		now := backup.CreatedAt
		backup.UpdatedAt = &now
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateVolumeBackup(ctx, pgdb.CreateVolumeBackupParams{
			ID:         backup.ID,
			VolumeName: backup.VolumeName,
			Size:       backup.Size,
			CreatedAt:  nullableTimestamptz(backup.CreatedAt),
			UpdatedAt:  nullableTimestamptzPtr(backup.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapVolumeBackupFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateVolumeBackup(ctx, sqlitedb.CreateVolumeBackupParams{
			ID:         backup.ID,
			VolumeName: backup.VolumeName,
			Size:       backup.Size,
			CreatedAt:  backup.CreatedAt,
			UpdatedAt:  nullableNullTimePtr(backup.UpdatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapVolumeBackupFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListVolumeBackupsByVolumeName(ctx context.Context, volumeName string) ([]volume.VolumeBackup, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListVolumeBackupsByVolumeName(ctx, volumeName)
		if err != nil {
			return nil, err
		}
		items := make([]volume.VolumeBackup, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapVolumeBackupFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListVolumeBackupsByVolumeName(ctx, volumeName)
		if err != nil {
			return nil, err
		}
		items := make([]volume.VolumeBackup, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapVolumeBackupFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetVolumeBackupByID(ctx context.Context, id string) (*volume.VolumeBackup, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetVolumeBackupByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapVolumeBackupFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetVolumeBackupByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapVolumeBackupFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteVolumeBackupByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.DeleteVolumeBackupByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rows > 0, nil
	case "sqlite":
		rows, err := s.sqlite.DeleteVolumeBackupByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rows > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func mapVolumeBackupFromPG(row *pgdb.VolumeBackup) *volume.VolumeBackup {
	if row == nil {
		return nil
	}
	created := timeFromPgTimestamptz(row.CreatedAt)
	updated := timePtrFromPgTimestamptz(row.UpdatedAt)
	return &volume.VolumeBackup{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: created,
			UpdatedAt: updated,
		},
		VolumeName: row.VolumeName,
		Size:       row.Size,
		CreatedAt:  created,
	}
}

func mapVolumeBackupFromSQLite(row *sqlitedb.VolumeBackup) *volume.VolumeBackup {
	if row == nil {
		return nil
	}
	created := row.CreatedAt
	updated := timePtrFromNull(row.UpdatedAt)
	return &volume.VolumeBackup{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: created,
			UpdatedAt: updated,
		},
		VolumeName: row.VolumeName,
		Size:       row.Size,
		CreatedAt:  created,
	}
}
