package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/types/apikey"
	"github.com/getarcaneapp/arcane/types/user"
)

var (
	ErrApiKeyNotFound = errors.New("API key not found")
	ErrApiKeyExpired  = errors.New("API key has expired")
	ErrApiKeyInvalid  = errors.New("invalid API key")
)

const (
	apiKeyPrefix    = "arc_"
	apiKeyLength    = 32
	apiKeyPrefixLen = 8
)

type ApiKeyService struct {
	store        database.ApiKeyStore
	userService  *UserService
	argon2Params *Argon2Params
}

func NewApiKeyService(store database.ApiKeyStore, userService *UserService) *ApiKeyService {
	return &ApiKeyService{
		store:        store,
		userService:  userService,
		argon2Params: DefaultArgon2Params(),
	}
}

func (s *ApiKeyService) generateApiKey() (string, error) {
	bytes := make([]byte, apiKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return apiKeyPrefix + hex.EncodeToString(bytes), nil
}

func (s *ApiKeyService) hashApiKey(key string) (string, error) {
	return s.userService.HashPassword(key)
}

func (s *ApiKeyService) validateApiKeyHash(hash, key string) error {
	return s.userService.ValidatePassword(hash, key)
}

func (s *ApiKeyService) CreateApiKey(ctx context.Context, userID string, req apikey.CreateApiKey) (*apikey.ApiKeyCreatedDto, error) {
	rawKey, err := s.generateApiKey()
	if err != nil {
		return nil, err
	}

	keyHash, err := s.hashApiKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	keyPrefix := rawKey[:len(apiKeyPrefix)+apiKeyPrefixLen]

	created, err := s.store.CreateApiKey(ctx, database.ApiKeyCreateInput{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		UserID:      userID,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &apikey.ApiKeyCreatedDto{
		ApiKey: s.toApiKeyDTO(created),
		Key:    rawKey,
	}, nil
}

func (s *ApiKeyService) CreateEnvironmentApiKey(ctx context.Context, environmentID string, userID string) (*apikey.ApiKeyCreatedDto, error) {
	rawKey, err := s.generateApiKey()
	if err != nil {
		return nil, err
	}

	keyHash, err := s.hashApiKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	keyPrefix := rawKey[:len(apiKeyPrefix)+apiKeyPrefixLen]

	envIDShort := environmentID
	if len(environmentID) > 8 {
		envIDShort = environmentID[:8]
	}
	name := fmt.Sprintf("Environment Bootstrap Key - %s", envIDShort)
	description := "Auto-generated key for environment pairing"

	created, err := s.store.CreateApiKey(ctx, database.ApiKeyCreateInput{
		ID:            uuid.NewString(),
		Name:          name,
		Description:   &description,
		KeyHash:       keyHash,
		KeyPrefix:     keyPrefix,
		UserID:        userID,
		EnvironmentID: &environmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create environment API key: %w", err)
	}

	return &apikey.ApiKeyCreatedDto{
		ApiKey: s.toApiKeyDTO(created),
		Key:    rawKey,
	}, nil
}

func (s *ApiKeyService) GetApiKey(ctx context.Context, id string) (*apikey.ApiKey, error) {
	ak, err := s.store.GetApiKeyByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if ak == nil {
		return nil, ErrApiKeyNotFound
	}

	out := s.toApiKeyDTO(ak)
	return &out, nil
}

func (s *ApiKeyService) ListApiKeys(ctx context.Context, params pagination.QueryParams) ([]apikey.ApiKey, pagination.Response, error) {
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

	apiKeys, err := s.store.ListApiKeys(ctx)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list API keys: %w", err)
	}

	config := pagination.Config[apikey.ModelApiKey]{
		SearchAccessors: []pagination.SearchAccessor[apikey.ModelApiKey]{
			func(k apikey.ModelApiKey) (string, error) { return k.Name, nil },
			func(k apikey.ModelApiKey) (string, error) {
				if k.Description == nil {
					return "", nil
				}
				return *k.Description, nil
			},
		},
		SortBindings: []pagination.SortBinding[apikey.ModelApiKey]{
			{Key: "name", Fn: func(a, b apikey.ModelApiKey) int { return strings.Compare(a.Name, b.Name) }},
			{Key: "expiresAt", Fn: func(a, b apikey.ModelApiKey) int { return compareOptionalTime(a.ExpiresAt, b.ExpiresAt) }},
			{Key: "lastUsedAt", Fn: func(a, b apikey.ModelApiKey) int { return compareOptionalTime(a.LastUsedAt, b.LastUsedAt) }},
		},
	}

	result := pagination.SearchOrderAndPaginate(apiKeys, params, config)
	response := pagination.BuildResponseFromFilterResult(result, params)

	out := make([]apikey.ApiKey, len(result.Items))
	for i := range result.Items {
		out[i] = s.toApiKeyDTO(&result.Items[i])
	}

	return out, response, nil
}

func (s *ApiKeyService) UpdateApiKey(ctx context.Context, id string, req apikey.UpdateApiKey) (*apikey.ApiKey, error) {
	ak, err := s.store.GetApiKeyByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if ak == nil {
		return nil, ErrApiKeyNotFound
	}

	if req.Name != nil {
		ak.Name = *req.Name
	}
	if req.Description != nil {
		ak.Description = req.Description
	}
	if req.ExpiresAt != nil {
		ak.ExpiresAt = req.ExpiresAt
	}

	updated, err := s.store.UpdateApiKey(ctx, database.ApiKeyUpdateInput{
		ID:          ak.ID,
		Name:        ak.Name,
		Description: ak.Description,
		ExpiresAt:   ak.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	out := s.toApiKeyDTO(updated)
	return &out, nil
}

func (s *ApiKeyService) DeleteApiKey(ctx context.Context, id string) error {
	deleted, err := s.store.DeleteApiKeyByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	if !deleted {
		return ErrApiKeyNotFound
	}
	return nil
}

func (s *ApiKeyService) ValidateApiKey(ctx context.Context, rawKey string) (*user.ModelUser, error) {
	if !strings.HasPrefix(rawKey, apiKeyPrefix) {
		return nil, ErrApiKeyInvalid
	}

	keyPrefix := rawKey[:len(apiKeyPrefix)+apiKeyPrefixLen]

	apiKeys, err := s.store.ListApiKeysByPrefix(ctx, keyPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to find API keys: %w", err)
	}

	for _, apiKey := range apiKeys {
		if err := s.validateApiKeyHash(apiKey.KeyHash, rawKey); err == nil {
			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				return nil, ErrApiKeyExpired
			}

			// Update last_used_at asynchronously to avoid blocking auth flow.
			go func(keyID string) {
				bgCtx := context.WithoutCancel(ctx)
				now := time.Now()
				_ = s.store.TouchApiKeyLastUsed(bgCtx, keyID, now)
			}(apiKey.ID)

			user, err := s.userService.GetUserByID(ctx, apiKey.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to get user for API key: %w", err)
			}

			return user, nil
		}
	}

	return nil, ErrApiKeyInvalid
}

func (s *ApiKeyService) GetEnvironmentByApiKey(ctx context.Context, rawKey string) (*string, error) {
	if !strings.HasPrefix(rawKey, apiKeyPrefix) {
		return nil, ErrApiKeyInvalid
	}

	keyPrefix := rawKey[:len(apiKeyPrefix)+apiKeyPrefixLen]

	apiKeys, err := s.store.ListApiKeysByPrefix(ctx, keyPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to find API keys: %w", err)
	}

	for _, apiKey := range apiKeys {
		if err := s.validateApiKeyHash(apiKey.KeyHash, rawKey); err == nil {
			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				return nil, ErrApiKeyExpired
			}

			return apiKey.EnvironmentID, nil
		}
	}

	return nil, ErrApiKeyInvalid
}

func (s *ApiKeyService) toApiKeyDTO(ak *apikey.ModelApiKey) apikey.ApiKey {
	return apikey.ApiKey{
		ID:          ak.ID,
		Name:        ak.Name,
		Description: ak.Description,
		KeyPrefix:   ak.KeyPrefix,
		UserID:      ak.UserID,
		ExpiresAt:   ak.ExpiresAt,
		LastUsedAt:  ak.LastUsedAt,
		CreatedAt:   ak.CreatedAt,
		UpdatedAt:   ak.UpdatedAt,
	}
}

func compareOptionalTime(a, b *time.Time) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	case a.Before(*b):
		return -1
	case a.After(*b):
		return 1
	default:
		return 0
	}
}
