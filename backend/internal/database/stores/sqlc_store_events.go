package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/event"
)

func (s *SqlcStore) CreateEvent(ctx context.Context, event event.ModelEvent) (*event.ModelEvent, error) {
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := event.UpdatedAt
	if updatedAt == nil {
		updatedAt = &createdAt
	}
	timestamp := event.Timestamp
	if timestamp.IsZero() {
		timestamp = createdAt
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateEvent(ctx, pgdb.CreateEventParams{
			ID:            event.ID,
			Type:          string(event.Type),
			Severity:      string(event.Severity),
			Title:         event.Title,
			Description:   nullableTextPtrKeepEmpty(&event.Description),
			ResourceType:  nullableTextPtrKeepEmpty(event.ResourceType),
			ResourceID:    nullableTextPtrKeepEmpty(event.ResourceID),
			ResourceName:  nullableTextPtrKeepEmpty(event.ResourceName),
			UserID:        nullableTextPtrKeepEmpty(event.UserID),
			Username:      nullableTextPtrKeepEmpty(event.Username),
			EnvironmentID: nullableTextPtrKeepEmpty(event.EnvironmentID),
			Metadata:      event.Metadata,
			Timestamp:     nullableTimestamptz(timestamp),
			CreatedAt:     nullableTimestamptz(createdAt),
			UpdatedAt:     nullableTimestamptzPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapEventFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateEvent(ctx, sqlitedb.CreateEventParams{
			ID:            event.ID,
			Type:          string(event.Type),
			Severity:      string(event.Severity),
			Title:         event.Title,
			Description:   nullableNullStringPtrKeepEmpty(&event.Description),
			ResourceType:  nullableNullStringPtrKeepEmpty(event.ResourceType),
			ResourceID:    nullableNullStringPtrKeepEmpty(event.ResourceID),
			ResourceName:  nullableNullStringPtrKeepEmpty(event.ResourceName),
			UserID:        nullableNullStringPtrKeepEmpty(event.UserID),
			Username:      nullableNullStringPtrKeepEmpty(event.Username),
			EnvironmentID: nullableNullStringPtrKeepEmpty(event.EnvironmentID),
			Metadata:      event.Metadata,
			Timestamp:     timestamp,
			CreatedAt:     createdAt,
			UpdatedAt:     nullableNullTimePtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapEventFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListEvents(ctx context.Context) ([]event.ModelEvent, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListEvents(ctx)
		if err != nil {
			return nil, err
		}
		events := make([]event.ModelEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, *mapEventFromPG(row))
		}
		return events, nil
	case "sqlite":
		rows, err := s.sqlite.ListEvents(ctx)
		if err != nil {
			return nil, err
		}
		events := make([]event.ModelEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, *mapEventFromSQLite(row))
		}
		return events, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteEventByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteEventByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteEventByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteEventsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.DeleteEventsOlderThan(ctx, nullableTimestamptz(cutoff))
	case "sqlite":
		return s.sqlite.DeleteEventsOlderThan(ctx, cutoff)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
