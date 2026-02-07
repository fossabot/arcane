package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/project"
)

func setupProjectTestDB(t *testing.T) *database.DB {
	t.Helper()
	ctx := context.Background()
	db, err := database.Initialize(ctx, testProjectSQLiteDSN(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func setupProjectSettingsStore(t *testing.T, db *database.DB) database.SettingsStore {
	t.Helper()
	store, err := database.NewSqlcStore(db)
	require.NoError(t, err)
	return store
}

func setupProjectStore(t *testing.T, db *database.DB) database.Store {
	t.Helper()
	store, err := database.NewSqlcStore(db)
	require.NoError(t, err)
	return store
}

func testProjectSQLiteDSN(t *testing.T) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}

func TestProjectService_GetProjectFromDatabaseByID(t *testing.T) {
	db := setupProjectTestDB(t)
	ctx := context.Background()

	// Setup dependencies
	store := setupProjectSettingsStore(t, db)
	projectStore := setupProjectStore(t, db)
	settingsService, _ := NewSettingsService(ctx, store)
	svc := NewProjectService(projectStore, settingsService, nil, nil, nil)

	// Create test project
	proj := &project.Project{
		BaseModel: base.BaseModel{
			ID: "p1",
		},
		Name: "test-project",
		Path: "/tmp/test-project",
	}
	_, err := projectStore.CreateProject(ctx, *proj)
	require.NoError(t, err)

	// Test success
	found, err := svc.GetProjectFromDatabaseByID(ctx, "p1")
	require.NoError(t, err)
	assert.Equal(t, "test-project", found.Name)

	// Test not found
	_, err = svc.GetProjectFromDatabaseByID(ctx, "non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

func TestProjectService_GetServiceCounts(t *testing.T) {
	svc := &ProjectService{}

	tests := []struct {
		name        string
		services    []ProjectServiceInfo
		wantTotal   int
		wantRunning int
	}{
		{
			name: "mixed status",
			services: []ProjectServiceInfo{
				{Name: "s1", Status: "running"},
				{Name: "s2", Status: "exited"},
				{Name: "s3", Status: "up"},
			},
			wantTotal:   3,
			wantRunning: 2,
		},
		{
			name: "all stopped",
			services: []ProjectServiceInfo{
				{Name: "s1", Status: "exited"},
			},
			wantTotal:   1,
			wantRunning: 0,
		},
		{
			name:        "empty",
			services:    []ProjectServiceInfo{},
			wantTotal:   0,
			wantRunning: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, running := svc.getServiceCounts(tt.services)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantRunning, running)
		})
	}
}

func TestProjectService_CalculateProjectStatus(t *testing.T) {
	svc := &ProjectService{}

	tests := []struct {
		name     string
		services []ProjectServiceInfo
		want     project.ProjectStatus
	}{
		{
			name:     "empty",
			services: []ProjectServiceInfo{},
			want:     project.ProjectStatusUnknown,
		},
		{
			name: "all running",
			services: []ProjectServiceInfo{
				{Status: "running"},
				{Status: "up"},
			},
			want: project.ProjectStatusRunning,
		},
		{
			name: "all stopped",
			services: []ProjectServiceInfo{
				{Status: "exited"},
				{Status: "stopped"},
			},
			want: project.ProjectStatusStopped,
		},
		{
			name: "partial",
			services: []ProjectServiceInfo{
				{Status: "running"},
				{Status: "exited"},
			},
			want: project.ProjectStatusPartiallyRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.calculateProjectStatus(tt.services)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProjectService_UpdateProjectStatusInternal(t *testing.T) {
	db := setupProjectTestDB(t)
	ctx := context.Background()
	projectStore := setupProjectStore(t, db)
	svc := NewProjectService(projectStore, nil, nil, nil, nil)

	proj := &project.Project{
		BaseModel: base.BaseModel{
			ID: "p1",
		},
		Status: project.ProjectStatusUnknown,
	}
	_, err := projectStore.CreateProject(ctx, *proj)
	require.NoError(t, err)

	err = svc.updateProjectStatusInternal(ctx, "p1", project.ProjectStatusRunning)
	require.NoError(t, err)

	updated, err := projectStore.GetProjectByID(ctx, "p1")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, project.ProjectStatusRunning, updated.Status)
	if updated.UpdatedAt != nil {
		assert.WithinDuration(t, time.Now(), *updated.UpdatedAt, time.Second)
	} else {
		t.Error("UpdatedAt should not be nil")
	}
}

func TestProjectService_IncrementStatusCounts(t *testing.T) {
	svc := &ProjectService{}
	running := 0
	stopped := 0

	svc.incrementStatusCounts(project.ProjectStatusRunning, &running, &stopped)
	assert.Equal(t, 1, running)
	assert.Equal(t, 0, stopped)

	svc.incrementStatusCounts(project.ProjectStatusStopped, &running, &stopped)
	assert.Equal(t, 1, running)
	assert.Equal(t, 1, stopped)

	svc.incrementStatusCounts(project.ProjectStatusUnknown, &running, &stopped)
	assert.Equal(t, 1, running)
	assert.Equal(t, 1, stopped)
}

func TestProjectService_FormatDockerPorts(t *testing.T) {
	tests := []struct {
		name     string
		input    []container.Port
		expected []string
	}{
		{
			name: "public port",
			input: []container.Port{
				{PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
			},
			expected: []string{"8080:80/tcp"},
		},
		{
			name: "private only",
			input: []container.Port{
				{PrivatePort: 80, Type: "tcp"},
			},
			expected: []string{"80/tcp"},
		},
		{
			name:     "empty",
			input:    []container.Port{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDockerPorts(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestProjectService_NormalizeComposeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "with special chars",
			input:    "My Project!",
			expected: "myproject",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeComposeProjectName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
