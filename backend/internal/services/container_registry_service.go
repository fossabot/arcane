package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils"
	"github.com/getarcaneapp/arcane/backend/internal/utils/cache"
	"github.com/getarcaneapp/arcane/backend/internal/utils/crypto"
	"github.com/getarcaneapp/arcane/backend/internal/utils/mapper"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/backend/internal/utils/registry"
	"github.com/getarcaneapp/arcane/types/containerregistry"
	ref "go.podman.io/image/v5/docker/reference"
)

const (
	registryCheckTimeout = 10 * time.Second
	registryCacheTTL     = 30 * time.Minute
)

func getHeaderCaseInsensitive(h http.Header, key string) string {
	for k, v := range h {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a && b:
		return -1
	default:
		return 1
	}
}

func compareTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func boolFilterMatches(value bool, filterValue string) bool {
	switch strings.TrimSpace(strings.ToLower(filterValue)) {
	case "true", "1":
		return value
	case "false", "0":
		return !value
	default:
		return false
	}
}

type ContainerRegistryService struct {
	store      database.ContainerRegistryStore
	httpClient *http.Client
	cache      map[string]*cache.Cache[string] // imageRef -> digest cache
	cacheMu    sync.RWMutex
}

func NewContainerRegistryService(store database.ContainerRegistryStore) *ContainerRegistryService {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment

	return &ContainerRegistryService{
		store: store,
		httpClient: &http.Client{
			Timeout:   registryCheckTimeout,
			Transport: transport,
		},
		cache: make(map[string]*cache.Cache[string]),
	}
}

func (s *ContainerRegistryService) GetAllRegistries(ctx context.Context) ([]containerregistry.ModelContainerRegistry, error) {
	registries, err := s.store.ListContainerRegistries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container registries: %w", err)
	}
	return registries, nil
}

func (s *ContainerRegistryService) GetRegistriesPaginated(ctx context.Context, params pagination.QueryParams) ([]containerregistry.ContainerRegistry, pagination.Response, error) {
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

	registries, err := s.store.ListContainerRegistries(ctx)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list container registries: %w", err)
	}

	config := pagination.Config[containerregistry.ModelContainerRegistry]{
		SearchAccessors: []pagination.SearchAccessor[containerregistry.ModelContainerRegistry]{
			func(r containerregistry.ModelContainerRegistry) (string, error) { return r.URL, nil },
			func(r containerregistry.ModelContainerRegistry) (string, error) { return r.Username, nil },
			func(r containerregistry.ModelContainerRegistry) (string, error) {
				if r.Description == nil {
					return "", nil
				}
				return *r.Description, nil
			},
		},
		SortBindings: []pagination.SortBinding[containerregistry.ModelContainerRegistry]{
			{Key: "url", Fn: func(a, b containerregistry.ModelContainerRegistry) int { return strings.Compare(a.URL, b.URL) }},
			{Key: "username", Fn: func(a, b containerregistry.ModelContainerRegistry) int {
				return strings.Compare(a.Username, b.Username)
			}},
			{Key: "description", Fn: func(a, b containerregistry.ModelContainerRegistry) int {
				return strings.Compare(utils.DerefString(a.Description), utils.DerefString(b.Description))
			}},
			{Key: "insecure", Fn: func(a, b containerregistry.ModelContainerRegistry) int { return compareBool(a.Insecure, b.Insecure) }},
			{Key: "enabled", Fn: func(a, b containerregistry.ModelContainerRegistry) int { return compareBool(a.Enabled, b.Enabled) }},
			{Key: "createdAt", Fn: func(a, b containerregistry.ModelContainerRegistry) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "updatedAt", Fn: func(a, b containerregistry.ModelContainerRegistry) int { return compareTime(a.UpdatedAt, b.UpdatedAt) }},
		},
		FilterAccessors: []pagination.FilterAccessor[containerregistry.ModelContainerRegistry]{
			{
				Key: "enabled",
				Fn: func(r containerregistry.ModelContainerRegistry, filterValue string) bool {
					return boolFilterMatches(r.Enabled, filterValue)
				},
			},
			{
				Key: "insecure",
				Fn: func(r containerregistry.ModelContainerRegistry, filterValue string) bool {
					return boolFilterMatches(r.Insecure, filterValue)
				},
			},
		},
	}

	result := pagination.SearchOrderAndPaginate(registries, params, config)
	paginationResp := pagination.BuildResponseFromFilterResult(result, params)

	out, mapErr := mapper.MapSlice[containerregistry.ModelContainerRegistry, containerregistry.ContainerRegistry](result.Items)
	if mapErr != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to map registries: %w", mapErr)
	}

	return out, paginationResp, nil
}

func (s *ContainerRegistryService) GetRegistryByID(ctx context.Context, id string) (*containerregistry.ModelContainerRegistry, error) {
	registry, err := s.store.GetContainerRegistryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get container registry: %w", err)
	}
	if registry == nil {
		return nil, fmt.Errorf("failed to get container registry: not found")
	}
	return registry, nil
}

func (s *ContainerRegistryService) CreateRegistry(ctx context.Context, req containerregistry.CreateContainerRegistryRequest) (*containerregistry.ModelContainerRegistry, error) {
	// Encrypt the token before storing
	encryptedToken, err := crypto.Encrypt(req.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	registry := &containerregistry.ModelContainerRegistry{
		URL:         req.URL,
		Username:    req.Username,
		Token:       encryptedToken,
		Description: req.Description,
		Insecure:    req.Insecure != nil && *req.Insecure,
		Enabled:     req.Enabled == nil || *req.Enabled,
	}

	created, err := s.store.CreateContainerRegistry(ctx, database.ContainerRegistryCreateInput{
		ID:          uuid.NewString(),
		URL:         registry.URL,
		Username:    registry.Username,
		Token:       registry.Token,
		Description: registry.Description,
		Insecure:    registry.Insecure,
		Enabled:     registry.Enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	return created, nil
}

func (s *ContainerRegistryService) UpdateRegistry(ctx context.Context, id string, req containerregistry.UpdateContainerRegistryRequest) (*containerregistry.ModelContainerRegistry, error) {
	registry, err := s.GetRegistryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	utils.UpdateIfChanged(&registry.URL, req.URL)
	utils.UpdateIfChanged(&registry.Username, req.Username)

	if req.Token != nil && *req.Token != "" {
		// Encrypt the new token
		encryptedToken, err := crypto.Encrypt(*req.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		utils.UpdateIfChanged(&registry.Token, encryptedToken)
	}

	utils.UpdateIfChanged(&registry.Description, req.Description)
	utils.UpdateIfChanged(&registry.Insecure, req.Insecure)
	utils.UpdateIfChanged(&registry.Enabled, req.Enabled)

	updated, err := s.store.UpdateContainerRegistry(ctx, database.ContainerRegistryUpdateInput{
		ID:          registry.ID,
		URL:         registry.URL,
		Username:    registry.Username,
		Token:       registry.Token,
		Description: registry.Description,
		Insecure:    registry.Insecure,
		Enabled:     registry.Enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update registry: %w", err)
	}

	return updated, nil
}

func (s *ContainerRegistryService) DeleteRegistry(ctx context.Context, id string) error {
	_, err := s.store.DeleteContainerRegistryByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete container registry: %w", err)
	}
	return nil
}

// GetDecryptedToken returns the decrypted token for a registry
func (s *ContainerRegistryService) GetDecryptedToken(ctx context.Context, id string) (string, error) {
	registry, err := s.GetRegistryByID(ctx, id)
	if err != nil {
		return "", err
	}

	decryptedToken, err := crypto.Decrypt(registry.Token)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return decryptedToken, nil
}

// GetEnabledRegistries returns all enabled registries
func (s *ContainerRegistryService) GetEnabledRegistries(ctx context.Context) ([]containerregistry.ModelContainerRegistry, error) {
	registries, err := s.store.ListEnabledContainerRegistries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled container registries: %w", err)
	}
	return registries, nil
}

// GetImageDigest fetches the current digest for an image:tag from the registry
// This is used for digest-based update detection for non-semver tags
func (s *ContainerRegistryService) GetImageDigest(ctx context.Context, imageRef string) (string, error) {
	repository, tag := parseImageReference(imageRef)
	if repository == "" || tag == "" {
		return "", fmt.Errorf("invalid image reference: %s", imageRef)
	}

	// Build a cache key from the full image reference
	cacheKey := fmt.Sprintf("%s:%s", repository, tag)

	// Get or create a cache for this specific image reference
	s.cacheMu.RLock()
	imageCache, exists := s.cache[cacheKey]
	s.cacheMu.RUnlock()

	if !exists {
		s.cacheMu.Lock()
		if imageCache, exists = s.cache[cacheKey]; !exists {
			imageCache = cache.New[string](registryCacheTTL)
			s.cache[cacheKey] = imageCache
		}
		s.cacheMu.Unlock()
	}

	digest, err := imageCache.GetOrFetch(ctx, func(ctx context.Context) (string, error) {
		return s.fetchDigestFromRegistry(ctx, repository, tag)
	})

	var staleErr *cache.ErrStale
	if err != nil && !errors.As(err, &staleErr) {
		return "", err
	}

	return digest, nil
}

// fetchDigestFromRegistry queries the Docker registry API for the image digest
func (s *ContainerRegistryService) fetchDigestFromRegistry(ctx context.Context, repository, tag string) (string, error) {
	registryURL, repoPath := parseRegistryAndRepo(repository)
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registryURL, repoPath, tag)

	reqCtx, cancel := context.WithTimeout(ctx, registryCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", fmt.Errorf("create registry request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")

	// Try to find stored credentials for this registry
	creds := s.findCredentialsForRegistry(ctx, registryURL)
	if creds != nil {
		req.SetBasicAuth(creds.Username, creds.Token)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return s.fetchWithTokenAuth(ctx, repository, tag, getHeaderCaseInsensitive(resp.Header, "WWW-Authenticate"), creds)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		digest = strings.Trim(resp.Header.Get("Etag"), "\"")
	}

	if digest == "" {
		return "", fmt.Errorf("no digest found in registry response")
	}

	return digest, nil
}

// findCredentialsForRegistry finds stored credentials for a registry URL
func (s *ContainerRegistryService) findCredentialsForRegistry(ctx context.Context, registryURL string) *struct{ Username, Token string } {
	registries, err := s.GetEnabledRegistries(ctx)
	if err != nil {
		return nil
	}

	// Normalize registry URL for comparison
	normalizedURL := strings.TrimPrefix(registryURL, "https://")
	normalizedURL = strings.TrimPrefix(normalizedURL, "http://")

	for _, reg := range registries {
		regURL := strings.TrimPrefix(reg.URL, "https://")
		regURL = strings.TrimPrefix(regURL, "http://")

		if strings.Contains(normalizedURL, regURL) || strings.Contains(regURL, normalizedURL) {
			token, err := crypto.Decrypt(reg.Token)
			if err != nil {
				slog.WarnContext(ctx, "Failed to decrypt registry token", "registry", reg.URL, "error", err)
				continue
			}
			return &struct{ Username, Token string }{Username: reg.Username, Token: token}
		}
	}

	return nil
}

// fetchWithTokenAuth handles token-based authentication for registries
func (s *ContainerRegistryService) fetchWithTokenAuth(ctx context.Context, repository, tag, wwwAuth string, creds *struct{ Username, Token string }) (string, error) {
	realm, service := parseWWWAuth(wwwAuth)
	if realm == "" {
		return "", fmt.Errorf("no auth realm found")
	}

	registryURL, repoPath := parseRegistryAndRepo(repository)

	tokenURL := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", realm, service, repoPath)

	reqCtx, cancel := context.WithTimeout(ctx, registryCheckTimeout)
	defer cancel()

	tokenReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}

	if creds != nil {
		tokenReq.SetBasicAuth(creds.Username, creds.Token)
	}

	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d", tokenResp.StatusCode)
	}

	var tokenData struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	token := tokenData.Token
	if token == "" {
		token = tokenData.AccessToken
	}
	if token == "" {
		return "", fmt.Errorf("no token in response")
	}

	// Retry with token
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registryURL, repoPath, tag)

	reqCtx2, cancel2 := context.WithTimeout(ctx, registryCheckTimeout)
	defer cancel2()

	req, err := http.NewRequestWithContext(reqCtx2, http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", fmt.Errorf("create authenticated request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("authenticated request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authenticated request returned status %d", resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		digest = strings.Trim(resp.Header.Get("Etag"), "\"")
	}

	if digest == "" {
		return "", fmt.Errorf("no digest found in authenticated response")
	}

	return digest, nil
}

// SyncRegistries syncs registries from a manager to this agent instance
// It creates, updates, or deletes registries to match the provided list
func (s *ContainerRegistryService) SyncRegistries(ctx context.Context, syncItems []containerregistry.Sync) error {
	existingMap, err := s.getExistingRegistriesMapInternal(ctx)
	if err != nil {
		return err
	}

	syncedIDs := make(map[string]bool)

	// Process each sync item
	for _, item := range syncItems {
		syncedIDs[item.ID] = true

		if err := s.processSyncItemInternal(ctx, item, existingMap); err != nil {
			return err
		}
	}

	// Delete registries that are not in the sync list
	return s.deleteUnsyncedInternal(ctx, existingMap, syncedIDs)
}

func (s *ContainerRegistryService) getExistingRegistriesMapInternal(ctx context.Context) (map[string]*containerregistry.ModelContainerRegistry, error) {
	existingRegistries, err := s.store.ListContainerRegistries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing registries: %w", err)
	}

	existingMap := make(map[string]*containerregistry.ModelContainerRegistry)
	for i := range existingRegistries {
		existingMap[existingRegistries[i].ID] = &existingRegistries[i]
	}

	return existingMap, nil
}

func (s *ContainerRegistryService) processSyncItemInternal(ctx context.Context, item containerregistry.Sync, existingMap map[string]*containerregistry.ModelContainerRegistry) error {
	existing, exists := existingMap[item.ID]
	if exists {
		return s.updateExistingRegistryInternal(ctx, item, existing)
	}
	return s.createNewRegistryInternal(ctx, item)
}

func (s *ContainerRegistryService) updateExistingRegistryInternal(ctx context.Context, item containerregistry.Sync, existing *containerregistry.ModelContainerRegistry) error {
	needsUpdate := s.checkRegistryNeedsUpdateInternal(item, existing)

	if needsUpdate {
		_, err := s.store.UpdateContainerRegistry(ctx, database.ContainerRegistryUpdateInput{
			ID:          existing.ID,
			URL:         existing.URL,
			Username:    existing.Username,
			Token:       existing.Token,
			Description: existing.Description,
			Insecure:    existing.Insecure,
			Enabled:     existing.Enabled,
		})
		if err != nil {
			return fmt.Errorf("failed to update registry %s: %w", item.ID, err)
		}
	}

	return nil
}

func (s *ContainerRegistryService) checkRegistryNeedsUpdateInternal(item containerregistry.Sync, existing *containerregistry.ModelContainerRegistry) bool {
	needsUpdate := utils.UpdateIfChanged(&existing.URL, item.URL)
	needsUpdate = utils.UpdateIfChanged(&existing.Username, item.Username) || needsUpdate

	// Always update token as it comes decrypted from manager
	encryptedToken, err := crypto.Encrypt(item.Token)
	if err == nil {
		needsUpdate = utils.UpdateIfChanged(&existing.Token, encryptedToken) || needsUpdate
	}

	needsUpdate = utils.UpdateIfChanged(&existing.Description, item.Description) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.Insecure, item.Insecure) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.Enabled, item.Enabled) || needsUpdate

	return needsUpdate
}

func (s *ContainerRegistryService) createNewRegistryInternal(ctx context.Context, item containerregistry.Sync) error {
	encryptedToken, err := crypto.Encrypt(item.Token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token for new registry %s: %w", item.ID, err)
	}

	_, err = s.store.CreateContainerRegistry(ctx, database.ContainerRegistryCreateInput{
		ID:          item.ID,
		URL:         item.URL,
		Username:    item.Username,
		Token:       encryptedToken,
		Description: item.Description,
		Insecure:    item.Insecure,
		Enabled:     item.Enabled,
	})
	if err != nil {
		return fmt.Errorf("failed to create registry %s: %w", item.ID, err)
	}

	return nil
}

func (s *ContainerRegistryService) deleteUnsyncedInternal(ctx context.Context, existingMap map[string]*containerregistry.ModelContainerRegistry, syncedIDs map[string]bool) error {
	for id := range existingMap {
		if !syncedIDs[id] {
			_, err := s.store.DeleteContainerRegistryByID(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to delete registry %s: %w", id, err)
			}
		}
	}
	return nil
}

// parseImageReference splits an image reference into repository and tag using distribution/reference
func parseImageReference(imageRef string) (repository, tag string) {
	named, err := ref.ParseNormalizedNamed(imageRef)
	if err != nil {
		return imageRef, "latest"
	}

	tagged, ok := named.(ref.Tagged)
	if ok {
		tag = tagged.Tag()
	} else {
		tag = "latest"
	}

	return named.Name(), tag
}

// parseRegistryAndRepo splits a repository into registry URL and repo path using distribution/reference
func parseRegistryAndRepo(repository string) (registryURL, repoPath string) {
	named, err := ref.ParseNormalizedNamed(repository)
	if err != nil {
		return "https://registry-1.docker.io", "library/" + repository
	}

	domain := ref.Domain(named)
	repoPath = ref.Path(named)

	registryURL, err = registry.GetRegistryAddress(named.Name())
	if err != nil {
		registryURL = "https://" + domain
	} else {
		registryURL = "https://" + registryURL
	}

	return registryURL, repoPath
}

// parseWWWAuth parses the WWW-Authenticate header using the registry client
func parseWWWAuth(header string) (realm, service string) {
	c := registry.NewClient()
	return c.ParseAuthChallenge(header)
}
