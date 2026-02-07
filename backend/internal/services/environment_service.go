package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils/crypto"
	"github.com/getarcaneapp/arcane/backend/internal/utils/edge"
	"github.com/getarcaneapp/arcane/backend/internal/utils/mapper"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/backend/internal/utils/timeouts"
	"github.com/getarcaneapp/arcane/types/containerregistry"
	"github.com/getarcaneapp/arcane/types/environment"
	"github.com/getarcaneapp/arcane/types/event"
	"github.com/getarcaneapp/arcane/types/gitops"
	"github.com/google/uuid"
)

type EnvironmentService struct {
	store           database.Store
	httpClient      *http.Client
	dockerService   *DockerClientService
	eventService    *EventService
	settingsService *SettingsService
}

func NewEnvironmentService(store database.Store, httpClient *http.Client, dockerService *DockerClientService, eventService *EventService, settingsService *SettingsService) *EnvironmentService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &EnvironmentService{
		store:           store,
		httpClient:      httpClient,
		dockerService:   dockerService,
		eventService:    eventService,
		settingsService: settingsService,
	}
}

func (s *EnvironmentService) EnsureLocalEnvironment(ctx context.Context, appUrl string) error {
	const localEnvID = "0"

	existingEnv, err := s.store.GetEnvironmentByID(ctx, localEnvID)
	if err != nil {
		return fmt.Errorf("failed to check for local environment: %w", err)
	}
	if existingEnv != nil {
		// Local environment already exists, ensure ApiUrl matches current appUrl
		if existingEnv.ApiUrl != appUrl {
			now := time.Now()
			updated, err := s.store.PatchEnvironment(ctx, database.EnvironmentPatchInput{
				ID:        localEnvID,
				APIURL:    &appUrl,
				UpdatedAt: &now,
			})
			if err != nil {
				return fmt.Errorf("failed to update local environment api url: %w", err)
			}
			if updated == nil {
				return fmt.Errorf("local environment not found")
			}
			slog.InfoContext(ctx, "updated local environment api url", "id", localEnvID, "url", appUrl)
		}
		return nil
	}

	// Create the local environment
	now := time.Now()
	localEnv, err := s.store.CreateEnvironment(ctx, database.EnvironmentCreateInput{
		ID:        localEnvID,
		Name:      "Local Docker",
		APIURL:    appUrl,
		Status:    string(environment.EnvironmentStatusOnline),
		Enabled:   true,
		IsEdge:    false,
		CreatedAt: now,
		UpdatedAt: &now,
	})
	if err != nil {
		return fmt.Errorf("failed to create local environment: %w", err)
	}
	if localEnv == nil {
		return fmt.Errorf("failed to create local environment")
	}

	slog.InfoContext(ctx, "created local environment record", "id", localEnvID)
	return nil
}

func (s *EnvironmentService) CreateEnvironment(ctx context.Context, env *environment.ModelEnvironment, userID, username *string) (*environment.ModelEnvironment, error) {
	env.ID = uuid.New().String()

	// Only set status to offline if not already set (e.g., API key flow sets it to pending)
	if env.Status == "" {
		env.Status = string(environment.EnvironmentStatusOffline)
	}

	now := time.Now()
	env.CreatedAt = now
	env.UpdatedAt = &now

	created, err := s.store.CreateEnvironment(ctx, database.EnvironmentCreateInput{
		ID:          env.ID,
		Name:        env.Name,
		APIURL:      env.ApiUrl,
		Status:      env.Status,
		Enabled:     env.Enabled,
		IsEdge:      env.IsEdge,
		LastSeen:    env.LastSeen,
		AccessToken: env.AccessToken,
		ApiKeyID:    env.ApiKeyID,
		CreatedAt:   env.CreatedAt,
		UpdatedAt:   env.UpdatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}
	if created == nil {
		return nil, fmt.Errorf("failed to create environment")
	}

	// Create event in background
	go s.createEnvironmentEvent(context.WithoutCancel(ctx), created.ID, created.Name, event.EventTypeEnvironmentCreate, "Environment Created", fmt.Sprintf("Environment '%s' was created", created.Name), event.EventSeveritySuccess, userID, username)

	return created, nil
}

func (s *EnvironmentService) GetEnvironmentByID(ctx context.Context, id string) (*environment.ModelEnvironment, error) {
	env, err := s.store.GetEnvironmentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}
	if env == nil {
		return nil, fmt.Errorf("environment not found")
	}
	return env, nil
}

func (s *EnvironmentService) ListEnvironmentsPaginated(ctx context.Context, params pagination.QueryParams) ([]environment.Environment, pagination.Response, error) {
	if params.Limit != -1 {
		if params.Limit <= 0 {
			params.Limit = 20
		} else if params.Limit > 100 {
			params.Limit = 100
		}
	}
	if params.Start < 0 {
		params.Start = 0
	}

	envs, err := s.store.ListEnvironments(ctx)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list environments: %w", err)
	}

	config := pagination.Config[environment.ModelEnvironment]{
		SearchAccessors: []pagination.SearchAccessor[environment.ModelEnvironment]{
			func(item environment.ModelEnvironment) (string, error) { return item.Name, nil },
			func(item environment.ModelEnvironment) (string, error) { return item.ApiUrl, nil },
		},
		SortBindings: []pagination.SortBinding[environment.ModelEnvironment]{
			{Key: "name", Fn: func(a, b environment.ModelEnvironment) int { return strings.Compare(a.Name, b.Name) }},
			{Key: "apiUrl", Fn: func(a, b environment.ModelEnvironment) int { return strings.Compare(a.ApiUrl, b.ApiUrl) }},
			{Key: "api_url", Fn: func(a, b environment.ModelEnvironment) int { return strings.Compare(a.ApiUrl, b.ApiUrl) }},
			{Key: "status", Fn: func(a, b environment.ModelEnvironment) int { return strings.Compare(a.Status, b.Status) }},
			{Key: "enabled", Fn: func(a, b environment.ModelEnvironment) int { return compareBool(a.Enabled, b.Enabled) }},
			{Key: "lastSeen", Fn: func(a, b environment.ModelEnvironment) int { return compareOptionalTime(a.LastSeen, b.LastSeen) }},
			{Key: "last_seen", Fn: func(a, b environment.ModelEnvironment) int { return compareOptionalTime(a.LastSeen, b.LastSeen) }},
			{Key: "createdAt", Fn: func(a, b environment.ModelEnvironment) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "created_at", Fn: func(a, b environment.ModelEnvironment) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "updatedAt", Fn: func(a, b environment.ModelEnvironment) int { return compareOptionalTime(a.UpdatedAt, b.UpdatedAt) }},
			{Key: "updated_at", Fn: func(a, b environment.ModelEnvironment) int { return compareOptionalTime(a.UpdatedAt, b.UpdatedAt) }},
		},
		FilterAccessors: []pagination.FilterAccessor[environment.ModelEnvironment]{
			{
				Key: "status",
				Fn: func(item environment.ModelEnvironment, filterValue string) bool {
					return matchesStringFilter(item.Status, filterValue)
				},
			},
			{
				Key: "enabled",
				Fn: func(item environment.ModelEnvironment, filterValue string) bool {
					return matchesBooleanFilter(item.Enabled, filterValue)
				},
			},
		},
	}

	result := pagination.SearchOrderAndPaginate(envs, params, config)
	paginationResp := pagination.BuildResponseFromFilterResult(result, params)

	out, mapErr := mapper.MapSlice[environment.ModelEnvironment, environment.Environment](result.Items)
	if mapErr != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to map environments: %w", mapErr)
	}

	return out, paginationResp, nil
}

// ListRemoteEnvironments returns all non-local, enabled environments for syncing purposes.
func (s *EnvironmentService) ListRemoteEnvironments(ctx context.Context) ([]environment.ModelEnvironment, error) {
	envs, err := s.store.ListRemoteEnvironments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote environments: %w", err)
	}
	return envs, nil
}

// ListEnabledEnvironments returns all enabled environments including local environment "0".
func (s *EnvironmentService) ListEnabledEnvironments(ctx context.Context) ([]environment.ModelEnvironment, error) {
	envs, err := s.store.ListEnvironments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	enabled := make([]environment.ModelEnvironment, 0, len(envs))
	for _, env := range envs {
		if env.Enabled {
			enabled = append(enabled, env)
		}
	}

	return enabled, nil
}

func (s *EnvironmentService) UpdateEnvironment(ctx context.Context, id string, updates map[string]interface{}, userID, username *string) (*environment.ModelEnvironment, error) {
	patch, err := buildEnvironmentPatchInput(id, updates)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	patch.UpdatedAt = &now

	updated, err := s.store.PatchEnvironment(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to update environment: %w", err)
	}
	if updated == nil {
		return nil, fmt.Errorf("environment not found")
	}

	// Create event in background (skip for local environment)
	if id != "0" {
		go s.createEnvironmentEvent(context.WithoutCancel(ctx), id, updated.Name, event.EventTypeEnvironmentUpdate, "Environment Updated", fmt.Sprintf("Environment '%s' was updated", updated.Name), event.EventSeverityInfo, userID, username)
	}

	return updated, nil
}

func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, id string, userID, username *string) error {
	// Get environment details before deletion
	env, err := s.GetEnvironmentByID(ctx, id)
	if err != nil {
		return err
	}

	deleted, err := s.store.DeleteEnvironmentByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}
	if !deleted {
		return fmt.Errorf("environment not found")
	}

	// Create event in background
	go s.createEnvironmentEvent(context.WithoutCancel(ctx), id, env.Name, event.EventTypeEnvironmentDelete, "Environment Deleted", fmt.Sprintf("Environment '%s' was deleted", env.Name), event.EventSeverityWarning, userID, username)

	return nil
}

func buildEnvironmentPatchInput(id string, updates map[string]interface{}) (database.EnvironmentPatchInput, error) {
	patch := database.EnvironmentPatchInput{ID: id}

	for key, value := range updates {
		switch key {
		case "name":
			strVal, ok := value.(string)
			if !ok {
				return patch, fmt.Errorf("invalid type for name")
			}
			patch.Name = &strVal
		case "api_url":
			strVal, ok := value.(string)
			if !ok {
				return patch, fmt.Errorf("invalid type for api_url")
			}
			patch.APIURL = &strVal
		case "status":
			strVal, ok := value.(string)
			if !ok {
				return patch, fmt.Errorf("invalid type for status")
			}
			patch.Status = &strVal
		case "enabled":
			boolVal, ok := value.(bool)
			if !ok {
				return patch, fmt.Errorf("invalid type for enabled")
			}
			patch.Enabled = &boolVal
		case "is_edge":
			boolVal, ok := value.(bool)
			if !ok {
				return patch, fmt.Errorf("invalid type for is_edge")
			}
			patch.IsEdge = &boolVal
		case "last_seen":
			if value == nil {
				patch.ClearLastSeen = true
				continue
			}
			switch typed := value.(type) {
			case time.Time:
				t := typed
				patch.LastSeen = &t
			case *time.Time:
				if typed == nil {
					patch.ClearLastSeen = true
				} else {
					patch.LastSeen = typed
				}
			default:
				return patch, fmt.Errorf("invalid type for last_seen")
			}
		case "access_token":
			if value == nil {
				patch.ClearAccessToken = true
				continue
			}
			strVal, ok := value.(string)
			if !ok {
				return patch, fmt.Errorf("invalid type for access_token")
			}
			patch.AccessToken = &strVal
		case "api_key_id":
			if value == nil {
				patch.ClearApiKeyID = true
				continue
			}
			strVal, ok := value.(string)
			if !ok {
				return patch, fmt.Errorf("invalid type for api_key_id")
			}
			patch.ApiKeyID = &strVal
		}
	}

	return patch, nil
}

func matchesStringFilter(value string, filterValue string) bool {
	filterValue = strings.TrimSpace(filterValue)
	if filterValue == "" {
		return true
	}
	for _, token := range strings.Split(filterValue, ",") {
		if strings.EqualFold(strings.TrimSpace(token), value) {
			return true
		}
	}
	return false
}

func matchesBooleanFilter(value bool, filterValue string) bool {
	filterValue = strings.TrimSpace(filterValue)
	if filterValue == "" {
		return true
	}
	for _, token := range strings.Split(filterValue, ",") {
		switch strings.TrimSpace(strings.ToLower(token)) {
		case "true", "1":
			if value {
				return true
			}
		case "false", "0":
			if !value {
				return true
			}
		}
	}
	return false
}

func (s *EnvironmentService) TestConnection(ctx context.Context, id string, customApiUrl *string) (string, error) {
	envModel, err := s.GetEnvironmentByID(ctx, id)
	if err != nil {
		return "error", err
	}

	// Special handling for local Docker environment (ID "0")
	if id == "0" && customApiUrl == nil {
		return s.testLocalDockerConnection(ctx, id)
	}

	// For edge environments, check if there's an active tunnel and route through it
	if envModel.IsEdge && customApiUrl == nil {
		return s.testEdgeConnection(ctx, id)
	}

	apiUrl := envModel.ApiUrl
	if customApiUrl != nil && *customApiUrl != "" {
		apiUrl = *customApiUrl
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	url := strings.TrimRight(apiUrl, "/") + "/api/health"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		if customApiUrl == nil {
			_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOffline))
		}
		return "offline", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		if customApiUrl == nil {
			_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOffline))
		}
		return "offline", fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if customApiUrl == nil {
			_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOnline))
		}
		return "online", nil
	}

	if customApiUrl == nil {
		_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusError))
	}
	return "error", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// testEdgeConnection tests connection to an edge agent via its tunnel
func (s *EnvironmentService) testEdgeConnection(ctx context.Context, id string) (string, error) {
	// Import edge package - this is a circular import issue, but we'll work around it
	// by checking if there's an active tunnel using the registry
	if !edge.HasActiveTunnel(id) {
		_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOffline))
		return "offline", fmt.Errorf("edge agent is not connected")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	statusCode, _, err := edge.DoRequest(reqCtx, id, http.MethodGet, "/api/health", nil)
	if err != nil {
		_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOffline))
		return "offline", fmt.Errorf("health check via tunnel failed: %w", err)
	}

	if statusCode == http.StatusOK {
		_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOnline))
		return "online", nil
	}

	_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusError))
	return "error", fmt.Errorf("unexpected status code: %d", statusCode)
}

func (s *EnvironmentService) testLocalDockerConnection(ctx context.Context, id string) (string, error) {
	// Test local Docker socket by pinging Docker
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dockerClient, err := s.dockerService.GetClient()
	if err != nil {
		_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOffline))
		return "offline", fmt.Errorf("failed to connect to Docker: %w", err)
	}

	_, err = dockerClient.Ping(reqCtx)
	if err != nil {
		_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOffline))
		return "offline", fmt.Errorf("docker ping failed: %w", err)
	}

	_ = s.updateEnvironmentStatusInternal(ctx, id, string(environment.EnvironmentStatusOnline))
	return "online", nil
}

func (s *EnvironmentService) updateEnvironmentStatusInternal(ctx context.Context, id, status string) error {
	// Don't update status for pending environments - they're waiting for agent pairing
	currentEnv, err := s.store.GetEnvironmentByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check environment status: %w", err)
	}
	if currentEnv == nil {
		return fmt.Errorf("environment not found")
	}

	if currentEnv.Status == string(environment.EnvironmentStatusPending) {
		slog.DebugContext(ctx, "skipping status update for pending environment", "environment_id", id)
		return nil
	}

	now := time.Now()
	updated, err := s.store.PatchEnvironment(ctx, database.EnvironmentPatchInput{
		ID:        id,
		Status:    &status,
		LastSeen:  &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return fmt.Errorf("failed to update environment status: %w", err)
	}
	if updated == nil {
		return fmt.Errorf("environment not found")
	}
	return nil
}

func (s *EnvironmentService) UpdateEnvironmentHeartbeat(ctx context.Context, id string) error {
	now := time.Now()
	if _, err := s.store.TouchEnvironmentHeartbeatIfStale(ctx, id, now, now.Add(-30*time.Second)); err != nil {
		return fmt.Errorf("failed to update environment heartbeat: %w", err)
	}

	return nil
}
func (s *EnvironmentService) createEnvironmentEvent(ctx context.Context, envID, envName string, eventType event.EventType, title, description string, severity event.EventSeverity, userID, username *string) {
	resourceType := "environment"
	resourceID := envID
	resourceName := envName
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:          eventType,
		Severity:      severity,
		Title:         title,
		Description:   description,
		ResourceType:  &resourceType,
		ResourceID:    &resourceID,
		ResourceName:  &resourceName,
		UserID:        userID,
		Username:      username,
		EnvironmentID: &envID,
	})
}

func (s *EnvironmentService) RegenerateEnvironmentApiKey(ctx context.Context, envID string, newApiKeyID string, encryptedKey string, userID, username string, envName string) error {
	// Update environment with new API key and set to pending status
	status := string(environment.EnvironmentStatusPending)
	now := time.Now()
	updated, err := s.store.PatchEnvironment(ctx, database.EnvironmentPatchInput{
		ID:            envID,
		ApiKeyID:      &newApiKeyID,
		AccessToken:   &encryptedKey,
		Status:        &status,
		ClearLastSeen: true,
		UpdatedAt:     &now,
	})
	if err != nil {
		return fmt.Errorf("failed to update environment with new API key: %w", err)
	}
	if updated == nil {
		return fmt.Errorf("environment not found")
	}

	// Create event log in background
	go s.createEnvironmentEvent(context.WithoutCancel(ctx), envID, envName, event.EventTypeEnvironmentApiKeyRegenerated, "API Key Regenerated", "Environment API key was regenerated and status set to pending", event.EventSeverityInfo, &userID, &username)

	return nil
}

// Deprecated - Use the Api Key flow
func (s *EnvironmentService) PairAgentWithBootstrap(ctx context.Context, apiUrl, bootstrapToken string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimRight(apiUrl, "/")+"/api/environments/0/agent/pair", nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Arcane-Agent-Bootstrap", bootstrapToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if !parsed.Success || parsed.Data.Token == "" {
		return "", fmt.Errorf("pairing unsuccessful")
	}

	return parsed.Data.Token, nil
}

func (s *EnvironmentService) PairAndPersistAgentToken(ctx context.Context, environmentID, apiUrl, bootstrapToken string) (string, error) {
	token, err := s.PairAgentWithBootstrap(ctx, apiUrl, bootstrapToken)
	if err != nil {
		return "", err
	}
	now := time.Now()
	updated, err := s.store.PatchEnvironment(ctx, database.EnvironmentPatchInput{
		ID:          environmentID,
		AccessToken: &token,
		UpdatedAt:   &now,
	})
	if err != nil {
		return "", fmt.Errorf("failed to persist agent token: %w", err)
	}
	if updated == nil {
		return "", fmt.Errorf("environment not found")
	}
	return token, nil
}

func (s *EnvironmentService) GetEnabledRegistryCredentials(ctx context.Context) ([]containerregistry.Credential, error) {
	registries, err := s.store.ListEnabledContainerRegistries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled container registries: %w", err)
	}

	var creds []containerregistry.Credential
	for _, reg := range registries {
		if !reg.Enabled || reg.Username == "" || reg.Token == "" {
			continue
		}

		decryptedToken, err := crypto.Decrypt(reg.Token)
		if err != nil {
			slog.WarnContext(ctx, "Failed to decrypt registry token", "registryURL", reg.URL, "error", err.Error())
			continue
		}

		creds = append(creds, containerregistry.Credential{
			URL:      reg.URL,
			Username: reg.Username,
			Token:    decryptedToken,
			Enabled:  reg.Enabled,
		})
	}

	return creds, nil
}

// DeploymentSnippets contains deployment configuration snippets for an environment.
type DeploymentSnippets struct {
	DockerRun     string
	DockerCompose string
}

// GenerateDeploymentSnippets generates Docker deployment snippets for an environment.
func (s *EnvironmentService) GenerateDeploymentSnippets(ctx context.Context, envID string, envAddress string, apiKey string) (*DeploymentSnippets, error) {
	managerURL := strings.TrimRight(envAddress, "/")

	dockerRun := fmt.Sprintf(`docker run -d \
  --name arcane-agent \
  --restart unless-stopped \
  -e AGENT_MODE=true \
  -e AGENT_TOKEN=%s \
  -e MANAGER_API_URL=%s \
  -p 3553:3553 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v arcane-data:/data \
  ghcr.io/getarcaneapp/arcane-headless:latest`, apiKey, managerURL)

	dockerCompose := fmt.Sprintf(`services:
  arcane-agent:
    image: ghcr.io/getarcaneapp/arcane-headless:latest
    container_name: arcane-agent
    restart: unless-stopped
    environment:
      - AGENT_MODE=true
      - AGENT_TOKEN=%s
      - MANAGER_API_URL=%s
    ports:
      - "3553:3553"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - arcane-data:/app/data

volumes:
  arcane-data:`, apiKey, managerURL)

	return &DeploymentSnippets{
		DockerRun:     dockerRun,
		DockerCompose: dockerCompose,
	}, nil
}

// GenerateEdgeDeploymentSnippets generates Docker deployment snippets for an edge agent.
// Edge agents connect outbound to the manager and don't require exposed ports.
func (s *EnvironmentService) GenerateEdgeDeploymentSnippets(ctx context.Context, envID string, managerURL string, apiKey string) (*DeploymentSnippets, error) {
	managerURL = strings.TrimRight(managerURL, "/")

	dockerRun := fmt.Sprintf(`docker run -d \
  --name arcane-edge-agent \
  --restart unless-stopped \
  -e EDGE_AGENT=true \
  -e AGENT_TOKEN=%s \
  -e MANAGER_API_URL=%s \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v arcane-data:/app/data \
  ghcr.io/getarcaneapp/arcane-headless:latest`, apiKey, managerURL)

	dockerCompose := fmt.Sprintf(`# Edge agent - connects outbound, no exposed ports required
services:
  arcane-edge-agent:
    image: ghcr.io/getarcaneapp/arcane-headless:latest
    container_name: arcane-edge-agent
    restart: unless-stopped
    environment:
      - EDGE_AGENT=true
      - AGENT_TOKEN=%s
      - MANAGER_API_URL=%s
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - arcane-data:/app/data

volumes:
  arcane-data:`, apiKey, managerURL)

	return &DeploymentSnippets{
		DockerRun:     dockerRun,
		DockerCompose: dockerCompose,
	}, nil
}

// SyncRegistriesToEnvironment syncs all registries from this manager to a remote environment
func (s *EnvironmentService) SyncRegistriesToEnvironment(ctx context.Context, environmentID string) error {
	// Get the environment
	environment, err := s.GetEnvironmentByID(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Don't sync to local environment (ID "0")
	if environmentID == "0" {
		return fmt.Errorf("cannot sync registries to local environment")
	}

	slog.InfoContext(ctx, "Starting registry sync to environment", "environmentID", environmentID, "environmentName", environment.Name, "apiUrl", environment.ApiUrl)

	// Get all registries from this manager
	registries, err := s.store.ListContainerRegistries(ctx)
	if err != nil {
		return fmt.Errorf("failed to get registries: %w", err)
	}

	slog.InfoContext(ctx, "Found registries to sync", "count", len(registries))

	// Prepare sync items with decrypted tokens
	syncItems := make([]containerregistry.Sync, 0, len(registries))
	for _, reg := range registries {
		decryptedToken, err := crypto.Decrypt(reg.Token)
		if err != nil {
			slog.WarnContext(ctx, "Failed to decrypt registry token for sync", "registryID", reg.ID, "registryURL", reg.URL, "error", err.Error())
			continue
		}

		syncItems = append(syncItems, containerregistry.Sync{
			ID:          reg.ID,
			URL:         reg.URL,
			Username:    reg.Username,
			Token:       decryptedToken,
			Description: reg.Description,
			Insecure:    reg.Insecure,
			Enabled:     reg.Enabled,
			CreatedAt:   reg.CreatedAt,
			UpdatedAt:   reg.UpdatedAt,
		})
	}

	// Prepare the sync request
	syncReq := containerregistry.SyncRequest{
		Registries: syncItems,
	}

	// Marshal the request
	reqBody, err := json.Marshal(syncReq)
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	// Send the sync request to the remote environment
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if environment.AccessToken != nil && *environment.AccessToken != "" {
		headers["X-Arcane-Agent-Token"] = *environment.AccessToken
		headers["X-API-Key"] = *environment.AccessToken
		slog.DebugContext(ctx, "Set auth headers for sync request")
	} else {
		slog.WarnContext(ctx, "No access token available for environment sync", "environmentID", environmentID)
	}

	targetURL := strings.TrimRight(environment.ApiUrl, "/") + "/api/container-registries/sync"
	apiPath := "/api/container-registries/sync"

	slog.InfoContext(ctx, "Sending sync request to agent", "url", targetURL, "registryCount", len(syncItems), "isEdge", environment.IsEdge)

	// Use edge-aware client that routes through tunnel for edge environments
	resp, err := edge.DoEdgeAwareRequest(reqCtx, environmentID, environment.IsEdge, http.MethodPost, targetURL, apiPath, headers, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send sync request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Sync request failed", "statusCode", resp.StatusCode, "response", string(resp.Body))
		return fmt.Errorf("sync request failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to decode sync response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("sync failed: %s", result.Data.Message)
	}

	slog.InfoContext(ctx, "Successfully synced registries to environment", "environmentID", environmentID, "environmentName", environment.Name)

	return nil
}

// SyncRepositoriesToEnvironment syncs all git repositories from this manager to a remote environment
func (s *EnvironmentService) SyncRepositoriesToEnvironment(ctx context.Context, environmentID string) error {
	// Get the environment
	environment, err := s.GetEnvironmentByID(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Don't sync to local environment (ID "0")
	if environmentID == "0" {
		return fmt.Errorf("cannot sync repositories to local environment")
	}

	slog.InfoContext(ctx, "Starting git repository sync to environment", "environmentID", environmentID, "environmentName", environment.Name, "apiUrl", environment.ApiUrl)

	// Get all git repositories from this manager
	repositories, err := s.store.ListGitRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to get git repositories: %w", err)
	}

	slog.InfoContext(ctx, "Found git repositories to sync", "count", len(repositories))

	// Prepare sync items with decrypted credentials
	syncItems := make([]gitops.RepositorySync, 0, len(repositories))
	for _, repo := range repositories {
		item := gitops.RepositorySync{
			ID:          repo.ID,
			Name:        repo.Name,
			URL:         repo.URL,
			AuthType:    repo.AuthType,
			Username:    repo.Username,
			Description: repo.Description,
			Enabled:     repo.Enabled,
			CreatedAt:   repo.CreatedAt,
		}
		if repo.UpdatedAt != nil {
			item.UpdatedAt = *repo.UpdatedAt
		}

		// Decrypt token if present
		if repo.Token != "" {
			decryptedToken, err := crypto.Decrypt(repo.Token)
			if err != nil {
				slog.WarnContext(ctx, "Failed to decrypt repository token for sync", "repositoryID", repo.ID, "repositoryName", repo.Name, "error", err.Error())
				continue
			}
			item.Token = decryptedToken
		}

		// Decrypt SSH key if present
		if repo.SSHKey != "" {
			decryptedSSHKey, err := crypto.Decrypt(repo.SSHKey)
			if err != nil {
				slog.WarnContext(ctx, "Failed to decrypt repository SSH key for sync", "repositoryID", repo.ID, "repositoryName", repo.Name, "error", err.Error())
				continue
			}
			item.SSHKey = decryptedSSHKey
		}

		syncItems = append(syncItems, item)
	}

	// Prepare the sync request
	syncReq := gitops.RepositorySyncRequest{
		Repositories: syncItems,
	}

	// Marshal the request
	reqBody, err := json.Marshal(syncReq)
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	// Send the sync request to the remote environment
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if environment.AccessToken != nil && *environment.AccessToken != "" {
		headers["X-Arcane-Agent-Token"] = *environment.AccessToken
		headers["X-API-Key"] = *environment.AccessToken
		slog.DebugContext(ctx, "Set auth headers for git repository sync request")
	} else {
		slog.WarnContext(ctx, "No access token available for environment git repository sync", "environmentID", environmentID)
	}

	targetURL := strings.TrimRight(environment.ApiUrl, "/") + "/api/git-repositories/sync"
	apiPath := "/api/git-repositories/sync"

	slog.InfoContext(ctx, "Sending git repository sync request to agent", "url", targetURL, "repositoryCount", len(syncItems), "isEdge", environment.IsEdge)

	// Use edge-aware client that routes through tunnel for edge environments
	resp, err := edge.DoEdgeAwareRequest(reqCtx, environmentID, environment.IsEdge, http.MethodPost, targetURL, apiPath, headers, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send sync request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Git repository sync request failed", "statusCode", resp.StatusCode, "response", string(resp.Body))
		return fmt.Errorf("sync request failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to decode sync response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("sync failed: %s", result.Data.Message)
	}

	slog.InfoContext(ctx, "Successfully synced git repositories to environment", "environmentID", environmentID, "environmentName", environment.Name)

	return nil
}

// ProxyRequest sends a request to a remote environment's API.
func (s *EnvironmentService) ProxyRequest(ctx context.Context, envID string, method string, path string, body []byte) ([]byte, int, error) {
	environment, err := s.GetEnvironmentByID(ctx, envID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get environment: %w", err)
	}

	if envID == "0" {
		return nil, 0, fmt.Errorf("cannot proxy request to local environment")
	}

	targetURL := strings.TrimRight(environment.ApiUrl, "/") + path

	settings := s.settingsService.GetSettingsConfig()
	proxyCtx, cancel := timeouts.WithTimeout(ctx, settings.ProxyRequestTimeout.AsInt(), timeouts.DefaultProxyRequest)
	defer cancel()

	// Build headers
	headers := make(map[string]string)
	if method != http.MethodGet && len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}

	// Use appropriate auth header
	if environment.AccessToken != nil && *environment.AccessToken != "" {
		headers["X-Arcane-Agent-Token"] = *environment.AccessToken
		headers["X-API-Key"] = *environment.AccessToken
	}

	// Use edge-aware client that routes through tunnel for edge environments
	resp, err := edge.DoEdgeAwareRequest(proxyCtx, envID, environment.IsEdge, method, targetURL, path, headers, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}

	return resp.Body, resp.StatusCode, nil
}
