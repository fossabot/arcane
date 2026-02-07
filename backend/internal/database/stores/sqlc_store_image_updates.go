package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/imageupdate"
)

func (s *SqlcStore) GetImageUpdateByID(ctx context.Context, id string) (*imageupdate.ImageUpdateRecord, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetImageUpdateByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapImageUpdateFromPGValues(
			row.ID,
			row.Repository,
			row.Tag,
			row.HasUpdate,
			row.UpdateType,
			row.CurrentVersion,
			row.LatestVersion,
			row.CurrentDigest,
			row.LatestDigest,
			row.CheckTime,
			row.ResponseTimeMs,
			row.LastError,
			row.AuthMethod,
			row.AuthUsername,
			row.AuthRegistry,
			row.UsedCredential,
			row.NotificationSent,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.GetImageUpdateByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapImageUpdateFromSQLiteValues(
			row.ID,
			row.Repository,
			row.Tag,
			row.HasUpdate,
			row.UpdateType,
			row.CurrentVersion,
			row.LatestVersion,
			row.CurrentDigest,
			row.LatestDigest,
			row.CheckTime,
			row.ResponseTimeMs,
			row.LastError,
			row.AuthMethod,
			row.AuthUsername,
			row.AuthRegistry,
			row.UsedCredential,
			row.NotificationSent,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveImageUpdateRecord(ctx context.Context, record imageupdate.ImageUpdateRecord) (*imageupdate.ImageUpdateRecord, error) {
	checkTime := record.CheckTime
	if checkTime.IsZero() {
		checkTime = time.Now()
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.SaveImageUpdate(ctx, pgdb.SaveImageUpdateParams{
			ID:               record.ID,
			Repository:       record.Repository,
			Tag:              record.Tag,
			HasUpdate:        record.HasUpdate,
			UpdateType:       nullableText(record.UpdateType),
			CurrentVersion:   record.CurrentVersion,
			LatestVersion:    nullableTextPtrKeepEmpty(record.LatestVersion),
			CurrentDigest:    nullableTextPtrKeepEmpty(record.CurrentDigest),
			LatestDigest:     nullableTextPtrKeepEmpty(record.LatestDigest),
			CheckTime:        nullableTimestamptz(checkTime),
			ResponseTimeMs:   int32(record.ResponseTimeMs),
			LastError:        nullableTextPtrKeepEmpty(record.LastError),
			AuthMethod:       nullableTextPtrKeepEmpty(record.AuthMethod),
			AuthUsername:     nullableTextPtrKeepEmpty(record.AuthUsername),
			AuthRegistry:     nullableTextPtrKeepEmpty(record.AuthRegistry),
			UsedCredential:   boolToPgBool(record.UsedCredential),
			NotificationSent: boolToPgBool(record.NotificationSent),
		})
		if err != nil {
			return nil, err
		}
		return mapImageUpdateFromPGValues(
			row.ID,
			row.Repository,
			row.Tag,
			row.HasUpdate,
			row.UpdateType,
			row.CurrentVersion,
			row.LatestVersion,
			row.CurrentDigest,
			row.LatestDigest,
			row.CheckTime,
			row.ResponseTimeMs,
			row.LastError,
			row.AuthMethod,
			row.AuthUsername,
			row.AuthRegistry,
			row.UsedCredential,
			row.NotificationSent,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.SaveImageUpdate(ctx, sqlitedb.SaveImageUpdateParams{
			ID:               record.ID,
			Repository:       record.Repository,
			Tag:              record.Tag,
			HasUpdate:        record.HasUpdate,
			UpdateType:       nullableString(record.UpdateType),
			CurrentVersion:   record.CurrentVersion,
			LatestVersion:    nullableNullStringPtrKeepEmpty(record.LatestVersion),
			CurrentDigest:    nullableNullStringPtrKeepEmpty(record.CurrentDigest),
			LatestDigest:     nullableNullStringPtrKeepEmpty(record.LatestDigest),
			CheckTime:        checkTime,
			ResponseTimeMs:   int64(record.ResponseTimeMs),
			LastError:        nullableNullStringPtrKeepEmpty(record.LastError),
			AuthMethod:       nullableNullStringPtrKeepEmpty(record.AuthMethod),
			AuthUsername:     nullableNullStringPtrKeepEmpty(record.AuthUsername),
			AuthRegistry:     nullableNullStringPtrKeepEmpty(record.AuthRegistry),
			UsedCredential:   boolToNullInt(record.UsedCredential),
			NotificationSent: boolToNullInt(record.NotificationSent),
		})
		if err != nil {
			return nil, err
		}
		return mapImageUpdateFromSQLiteValues(
			row.ID,
			row.Repository,
			row.Tag,
			row.HasUpdate,
			row.UpdateType,
			row.CurrentVersion,
			row.LatestVersion,
			row.CurrentDigest,
			row.LatestDigest,
			row.CheckTime,
			row.ResponseTimeMs,
			row.LastError,
			row.AuthMethod,
			row.AuthUsername,
			row.AuthRegistry,
			row.UsedCredential,
			row.NotificationSent,
			row.CreatedAt,
			row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListImageUpdateRecords(ctx context.Context) ([]imageupdate.ImageUpdateRecord, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListImageUpdates(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromPGValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListImageUpdates(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromSQLiteValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListImageUpdateRecordsByIDs(ctx context.Context, ids []string) ([]imageupdate.ImageUpdateRecord, error) {
	if len(ids) == 0 {
		return []imageupdate.ImageUpdateRecord{}, nil
	}
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListImageUpdatesByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromPGValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListImageUpdatesByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromSQLiteValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListImageUpdateRecordsWithUpdate(ctx context.Context) ([]imageupdate.ImageUpdateRecord, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListImageUpdatesWithUpdate(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromPGValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListImageUpdatesWithUpdate(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromSQLiteValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListUnnotifiedImageUpdates(ctx context.Context) ([]imageupdate.ImageUpdateRecord, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListUnnotifiedImageUpdates(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromPGValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListUnnotifiedImageUpdates(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]imageupdate.ImageUpdateRecord, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapImageUpdateFromSQLiteValues(
				row.ID,
				row.Repository,
				row.Tag,
				row.HasUpdate,
				row.UpdateType,
				row.CurrentVersion,
				row.LatestVersion,
				row.CurrentDigest,
				row.LatestDigest,
				row.CheckTime,
				row.ResponseTimeMs,
				row.LastError,
				row.AuthMethod,
				row.AuthUsername,
				row.AuthRegistry,
				row.UsedCredential,
				row.NotificationSent,
				row.CreatedAt,
				row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) MarkImageUpdatesAsNotified(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	switch s.driver {
	case "postgres":
		return s.pg.MarkImageUpdatesNotified(ctx, ids)
	case "sqlite":
		return s.sqlite.MarkImageUpdatesNotified(ctx, ids)
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteImageUpdatesByIDs(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	switch s.driver {
	case "postgres":
		return s.pg.DeleteImageUpdatesByIDs(ctx, ids)
	case "sqlite":
		return s.sqlite.DeleteImageUpdatesByIDs(ctx, ids)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateImageUpdateHasUpdateByRepositoryTag(ctx context.Context, repository, tag string, hasUpdate bool) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateImageUpdateHasUpdateByRepositoryTag(ctx, pgdb.UpdateImageUpdateHasUpdateByRepositoryTagParams{
			Repository: repository,
			Tag:        tag,
			HasUpdate:  hasUpdate,
		})
	case "sqlite":
		return s.sqlite.UpdateImageUpdateHasUpdateByRepositoryTag(ctx, sqlitedb.UpdateImageUpdateHasUpdateByRepositoryTagParams{
			HasUpdate:  hasUpdate,
			Repository: repository,
			Tag:        tag,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CountImageUpdates(ctx context.Context) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.CountImageUpdates(ctx)
	case "sqlite":
		return s.sqlite.CountImageUpdates(ctx)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CountImageUpdatesWithUpdate(ctx context.Context) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.CountImageUpdatesWithUpdate(ctx)
	case "sqlite":
		return s.sqlite.CountImageUpdatesWithUpdate(ctx)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CountImageUpdatesWithUpdateType(ctx context.Context, updateType string) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.CountImageUpdatesWithUpdateType(ctx, nullableText(updateType))
	case "sqlite":
		return s.sqlite.CountImageUpdatesWithUpdateType(ctx, nullableString(updateType))
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CountImageUpdatesWithErrors(ctx context.Context) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.CountImageUpdatesWithErrors(ctx)
	case "sqlite":
		return s.sqlite.CountImageUpdatesWithErrors(ctx)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
