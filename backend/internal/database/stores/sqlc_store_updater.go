package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/updater"
)

func (s *SqlcStore) CreateAutoUpdateRecord(ctx context.Context, record updater.AutoUpdateRecord) error {
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.StartTime.IsZero() {
		record.StartTime = time.Now().UTC()
	}

	switch s.driver {
	case "postgres":
		return s.pg.CreateAutoUpdateRecord(ctx, pgdb.CreateAutoUpdateRecordParams{
			ID:               record.ID,
			ResourceID:       record.ResourceID,
			ResourceType:     record.ResourceType,
			ResourceName:     record.ResourceName,
			Status:           string(record.Status),
			StartTime:        nullableTimestamptz(record.StartTime),
			EndTime:          nullableTimestamptzPtr(record.EndTime),
			UpdateAvailable:  record.UpdateAvailable,
			UpdateApplied:    record.UpdateApplied,
			OldImageVersions: record.OldImageVersions,
			NewImageVersions: record.NewImageVersions,
			Error:            nullableTextPtrKeepEmpty(record.Error),
			Details:          record.Details,
		})
	case "sqlite":
		return s.sqlite.CreateAutoUpdateRecord(ctx, sqlitedb.CreateAutoUpdateRecordParams{
			ID:               record.ID,
			ResourceID:       record.ResourceID,
			ResourceType:     record.ResourceType,
			ResourceName:     record.ResourceName,
			Status:           string(record.Status),
			StartTime:        record.StartTime,
			EndTime:          nullableNullTimePtr(record.EndTime),
			UpdateAvailable:  record.UpdateAvailable,
			UpdateApplied:    record.UpdateApplied,
			OldImageVersions: record.OldImageVersions,
			NewImageVersions: record.NewImageVersions,
			Error:            nullableNullStringPtrKeepEmpty(record.Error),
			Details:          record.Details,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListAutoUpdateRecords(ctx context.Context, limit int) ([]updater.AutoUpdateRecord, error) {
	switch s.driver {
	case "postgres":
		var (
			rows []*pgdb.AutoUpdateRecord
			err  error
		)
		if limit > 0 {
			rows, err = s.pg.ListAutoUpdateRecordsLimited(ctx, int32(limit))
		} else {
			rows, err = s.pg.ListAutoUpdateRecords(ctx)
		}
		if err != nil {
			return nil, err
		}
		items := make([]updater.AutoUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapAutoUpdateRecordFromPG(row))
		}
		return items, nil
	case "sqlite":
		var (
			rows []*sqlitedb.AutoUpdateRecord
			err  error
		)
		if limit > 0 {
			rows, err = s.sqlite.ListAutoUpdateRecordsLimited(ctx, int64(limit))
		} else {
			rows, err = s.sqlite.ListAutoUpdateRecords(ctx)
		}
		if err != nil {
			return nil, err
		}
		items := make([]updater.AutoUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapAutoUpdateRecordFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func mapAutoUpdateRecordFromPG(row *pgdb.AutoUpdateRecord) *updater.AutoUpdateRecord {
	if row == nil {
		return nil
	}
	return &updater.AutoUpdateRecord{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
		ResourceID:       row.ResourceID,
		ResourceType:     row.ResourceType,
		ResourceName:     row.ResourceName,
		Status:           updater.AutoUpdateStatus(row.Status),
		StartTime:        timeFromPgTimestamptz(row.StartTime),
		EndTime:          timePtrFromPgTimestamptz(row.EndTime),
		UpdateAvailable:  row.UpdateAvailable,
		UpdateApplied:    row.UpdateApplied,
		OldImageVersions: row.OldImageVersions,
		NewImageVersions: row.NewImageVersions,
		Error:            stringPtrFromPgText(row.Error),
		Details:          row.Details,
	}
}

func mapAutoUpdateRecordFromSQLite(row *sqlitedb.AutoUpdateRecord) *updater.AutoUpdateRecord {
	if row == nil {
		return nil
	}
	return &updater.AutoUpdateRecord{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
		ResourceID:       row.ResourceID,
		ResourceType:     row.ResourceType,
		ResourceName:     row.ResourceName,
		Status:           updater.AutoUpdateStatus(row.Status),
		StartTime:        row.StartTime,
		EndTime:          timePtrFromNull(row.EndTime),
		UpdateAvailable:  row.UpdateAvailable,
		UpdateApplied:    row.UpdateApplied,
		OldImageVersions: row.OldImageVersions,
		NewImageVersions: row.NewImageVersions,
		Error:            stringPtrFromNull(row.Error),
		Details:          row.Details,
	}
}
