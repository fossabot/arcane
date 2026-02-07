package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/getarcaneapp/arcane/backend/internal/config"
	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/services"
)

func setupAnalyticsSettingsService(t *testing.T) *services.SettingsService {
	t.Helper()
	ctx := context.Background()
	db, err := database.Initialize(ctx, testAnalyticsSQLiteDSN(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store, err := database.NewSqlcStore(db)
	require.NoError(t, err)

	svc, err := services.NewSettingsService(ctx, store)
	require.NoError(t, err)
	require.NoError(t, svc.SetStringSetting(ctx, "instanceId", "test-instance"))

	return svc
}

func testAnalyticsSQLiteDSN(t *testing.T) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}

func newHeartbeatServer(t *testing.T) (*httptest.Server, <-chan []byte) {
	t.Helper()
	bodyCh := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read heartbeat body: %v", err)
		}
		bodyCh <- body
		w.WriteHeader(http.StatusOK)
	}))

	return server, bodyCh
}

func TestAnalyticsJob_Run_ManagerPayload(t *testing.T) {
	ctx := context.Background()
	settingsService := setupAnalyticsSettingsService(t)
	server, bodyCh := newHeartbeatServer(t)
	defer server.Close()

	cfg := &config.Config{Environment: config.AppEnvironmentProduction}
	job := NewAnalyticsJob(settingsService, server.Client(), cfg)
	job.heartbeatURL = server.URL

	job.Run(ctx)

	var body []byte
	select {
	case body = <-bodyCh:
	default:
		t.Fatal("expected heartbeat request")
	}

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, getAnalyticsVersion(), payload["version"])
	require.Equal(t, "test-instance", payload["instance_id"])
	require.Equal(t, "manager", payload["server_type"])
}

func TestAnalyticsJob_Run_AgentPayload(t *testing.T) {
	ctx := context.Background()
	settingsService := setupAnalyticsSettingsService(t)
	server, bodyCh := newHeartbeatServer(t)
	defer server.Close()

	cfg := &config.Config{AgentMode: true, Environment: config.AppEnvironmentProduction}
	job := NewAnalyticsJob(settingsService, server.Client(), cfg)
	job.heartbeatURL = server.URL

	job.Run(ctx)

	var body []byte
	select {
	case body = <-bodyCh:
	default:
		t.Fatal("expected heartbeat request")
	}

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "agent", payload["server_type"])
}

func TestAnalyticsJob_Run_SkipsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	settingsService := setupAnalyticsSettingsService(t)
	server, bodyCh := newHeartbeatServer(t)
	defer server.Close()

	cfg := &config.Config{AnalyticsDisabled: true, Environment: config.AppEnvironmentProduction}
	job := NewAnalyticsJob(settingsService, server.Client(), cfg)
	job.heartbeatURL = server.URL

	job.Run(ctx)

	select {
	case <-bodyCh:
		t.Fatal("unexpected heartbeat request")
	default:
	}
}

func TestAnalyticsJob_Run_SkipsWhenTestEnv(t *testing.T) {
	ctx := context.Background()
	settingsService := setupAnalyticsSettingsService(t)
	server, bodyCh := newHeartbeatServer(t)
	defer server.Close()

	cfg := &config.Config{Environment: config.AppEnvironmentTest}
	job := NewAnalyticsJob(settingsService, server.Client(), cfg)
	job.heartbeatURL = server.URL

	job.Run(ctx)

	select {
	case <-bodyCh:
		t.Fatal("unexpected heartbeat request")
	default:
	}
}
