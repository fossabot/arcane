package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/getarcaneapp/arcane/types/migration"
	"github.com/stretchr/testify/require"
)

type fakeMigrationStore struct {
	statuses      []MigrationStatus
	statusErr     error
	downSteps     int
	downErr       error
	downToVersion int64
	downToErr     error
	redoCalled    bool
	redoErr       error
}

func (f *fakeMigrationStore) Status(ctx context.Context) ([]MigrationStatus, error) {
	return f.statuses, f.statusErr
}

func (f *fakeMigrationStore) Down(ctx context.Context, steps int) error {
	f.downSteps = steps
	return f.downErr
}

func (f *fakeMigrationStore) DownTo(ctx context.Context, version int64) error {
	f.downToVersion = version
	return f.downToErr
}

func (f *fakeMigrationStore) Redo(ctx context.Context) error {
	f.redoCalled = true
	return f.redoErr
}

func TestMigrationService_Status(t *testing.T) {
	fixedTime := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name     string
		statuses []MigrationStatus
		want     []migration.Status
	}{
		{
			name: "formats applied timestamps",
			statuses: []MigrationStatus{
				{Version: 1, State: "applied", AppliedAt: fixedTime, Path: "001_init.sql"},
				{Version: 2, State: "pending", AppliedAt: time.Time{}, Path: "002_next.sql"},
			},
			want: []migration.Status{
				{Version: 1, State: "applied", AppliedAt: "2025-01-02 03:04:05", Path: "001_init.sql"},
				{Version: 2, State: "pending", AppliedAt: "", Path: "002_next.sql"},
			},
		},
		{
			name:     "handles empty status list",
			statuses: []MigrationStatus{},
			want:     []migration.Status{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMigrationStore{statuses: tt.statuses}
			svc := NewMigrationServiceWithStore(store)

			got, err := svc.Status(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	t.Run("returns error", func(t *testing.T) {
		store := &fakeMigrationStore{statusErr: errors.New("boom")}
		svc := NewMigrationServiceWithStore(store)

		_, err := svc.Status(context.Background())
		require.Error(t, err)
	})
}

func TestMigrationService_Down(t *testing.T) {
	tests := []struct {
		name  string
		steps int
		err   error
	}{
		{name: "rolls back", steps: 2},
		{name: "returns error", steps: 1, err: errors.New("fail")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMigrationStore{downErr: tt.err}
			svc := NewMigrationServiceWithStore(store)

			err := svc.Down(context.Background(), tt.steps)
			if tt.err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.steps, store.downSteps)
		})
	}
}

func TestMigrationService_DownTo(t *testing.T) {
	tests := []struct {
		name    string
		version int64
		err     error
	}{
		{name: "rolls back to version", version: 42},
		{name: "returns error", version: 7, err: errors.New("fail")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMigrationStore{downToErr: tt.err}
			svc := NewMigrationServiceWithStore(store)

			err := svc.DownTo(context.Background(), tt.version)
			if tt.err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.version, store.downToVersion)
		})
	}
}

func TestMigrationService_Redo(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "redo latest"},
		{name: "returns error", err: errors.New("fail")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMigrationStore{redoErr: tt.err}
			svc := NewMigrationServiceWithStore(store)

			err := svc.Redo(context.Background())
			if tt.err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.True(t, store.redoCalled)
		})
	}
}
