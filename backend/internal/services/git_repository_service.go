package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils"
	"github.com/getarcaneapp/arcane/backend/internal/utils/crypto"
	"github.com/getarcaneapp/arcane/backend/internal/utils/git"
	"github.com/getarcaneapp/arcane/backend/internal/utils/mapper"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/backend/internal/utils/timeouts"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/event"
	"github.com/getarcaneapp/arcane/types/gitops"
)

var ErrGitRepositoryNotFound = errors.New("repository not found")

type GitRepositoryService struct {
	store           database.GitRepositoryStore
	gitClient       *git.Client
	eventService    *EventService
	settingsService *SettingsService
}

func NewGitRepositoryService(store database.GitRepositoryStore, workDir string, eventService *EventService, settingsService *SettingsService) *GitRepositoryService {
	return &GitRepositoryService{
		store:           store,
		gitClient:       git.NewClient(workDir),
		eventService:    eventService,
		settingsService: settingsService,
	}
}

func (s *GitRepositoryService) GetRepositoriesPaginated(ctx context.Context, params pagination.QueryParams) ([]gitops.GitRepository, pagination.Response, error) {
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

	repositories, err := s.store.ListGitRepositories(ctx)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list git repositories: %w", err)
	}

	config := pagination.Config[gitops.ModelGitRepository]{
		SearchAccessors: []pagination.SearchAccessor[gitops.ModelGitRepository]{
			func(r gitops.ModelGitRepository) (string, error) { return r.Name, nil },
			func(r gitops.ModelGitRepository) (string, error) { return r.URL, nil },
			func(r gitops.ModelGitRepository) (string, error) {
				if r.Description == nil {
					return "", nil
				}
				return *r.Description, nil
			},
		},
		SortBindings: []pagination.SortBinding[gitops.ModelGitRepository]{
			{Key: "name", Fn: func(a, b gitops.ModelGitRepository) int { return strings.Compare(a.Name, b.Name) }},
			{Key: "url", Fn: func(a, b gitops.ModelGitRepository) int { return strings.Compare(a.URL, b.URL) }},
			{Key: "authType", Fn: func(a, b gitops.ModelGitRepository) int { return strings.Compare(a.AuthType, b.AuthType) }},
			{Key: "username", Fn: func(a, b gitops.ModelGitRepository) int { return strings.Compare(a.Username, b.Username) }},
			{Key: "description", Fn: func(a, b gitops.ModelGitRepository) int {
				return strings.Compare(utils.DerefString(a.Description), utils.DerefString(b.Description))
			}},
			{Key: "enabled", Fn: func(a, b gitops.ModelGitRepository) int { return compareBool(a.Enabled, b.Enabled) }},
			{Key: "createdAt", Fn: func(a, b gitops.ModelGitRepository) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "updatedAt", Fn: func(a, b gitops.ModelGitRepository) int { return compareOptionalTime(a.UpdatedAt, b.UpdatedAt) }},
		},
		FilterAccessors: []pagination.FilterAccessor[gitops.ModelGitRepository]{
			{
				Key: "enabled",
				Fn: func(r gitops.ModelGitRepository, filterValue string) bool {
					return boolFilterMatches(r.Enabled, filterValue)
				},
			},
			{
				Key: "authType",
				Fn: func(r gitops.ModelGitRepository, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(r.AuthType), strings.TrimSpace(filterValue))
				},
			},
		},
	}

	result := pagination.SearchOrderAndPaginate(repositories, params, config)
	paginationResp := pagination.BuildResponseFromFilterResult(result, params)

	out, mapErr := mapper.MapSlice[gitops.ModelGitRepository, gitops.GitRepository](result.Items)
	if mapErr != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to map repositories: %w", mapErr)
	}

	return out, paginationResp, nil
}

func (s *GitRepositoryService) GetRepositoryByID(ctx context.Context, id string) (*gitops.ModelGitRepository, error) {
	repository, err := s.store.GetGitRepositoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repository == nil {
		return nil, ErrGitRepositoryNotFound
	}
	return repository, nil
}

func (s *GitRepositoryService) GetRepositoryByName(ctx context.Context, name string) (*gitops.ModelGitRepository, error) {
	repository, err := s.store.GetGitRepositoryByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repository == nil {
		return nil, ErrGitRepositoryNotFound
	}
	return repository, nil
}

func (s *GitRepositoryService) CreateRepository(ctx context.Context, req gitops.CreateGitRepositoryRequest) (*gitops.ModelGitRepository, error) {
	repository := gitops.ModelGitRepository{
		BaseModel: base.BaseModel{
			ID: uuid.NewString(),
		},
		Name:                   req.Name,
		URL:                    req.URL,
		AuthType:               req.AuthType,
		Username:               req.Username,
		SSHHostKeyVerification: req.SSHHostKeyVerification,
		Description:            req.Description,
		Enabled:                true,
	}

	// Default to accept_new if not specified
	if repository.SSHHostKeyVerification == "" {
		repository.SSHHostKeyVerification = "accept_new"
	}

	if req.Enabled != nil {
		repository.Enabled = *req.Enabled
	}

	// Encrypt sensitive fields
	if req.Token != "" {
		encrypted, err := crypto.Encrypt(req.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		repository.Token = encrypted
	}

	if req.SSHKey != "" {
		encrypted, err := crypto.Encrypt(req.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt SSH key: %w", err)
		}
		repository.SSHKey = encrypted
	}

	created, err := s.store.CreateGitRepository(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Log event
	resourceType := "git_repository"
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:         event.EventTypeGitRepositoryCreate,
		Severity:     event.EventSeveritySuccess,
		Title:        "Git repository created",
		Description:  fmt.Sprintf("Created git repository '%s' (%s)", created.Name, created.URL),
		ResourceType: &resourceType,
		ResourceID:   &created.ID,
		ResourceName: &created.Name,
	})

	return created, nil
}

func (s *GitRepositoryService) UpdateRepository(ctx context.Context, id string, req gitops.UpdateGitRepositoryRequest) (*gitops.ModelGitRepository, error) {
	repository, err := s.GetRepositoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated := false

	if req.Name != nil {
		updated = utils.UpdateIfChanged(&repository.Name, req.Name) || updated
	}
	if req.URL != nil {
		updated = utils.UpdateIfChanged(&repository.URL, req.URL) || updated
	}
	if req.AuthType != nil {
		updated = utils.UpdateIfChanged(&repository.AuthType, req.AuthType) || updated
	}
	if req.Username != nil {
		updated = utils.UpdateIfChanged(&repository.Username, req.Username) || updated
	}
	if req.Description != nil {
		updated = utils.UpdateIfChanged(&repository.Description, req.Description) || updated
	}
	if req.Enabled != nil {
		updated = utils.UpdateIfChanged(&repository.Enabled, req.Enabled) || updated
	}
	if req.SSHHostKeyVerification != nil {
		updated = utils.UpdateIfChanged(&repository.SSHHostKeyVerification, req.SSHHostKeyVerification) || updated
	}

	if req.Token != nil {
		if *req.Token == "" {
			updated = utils.UpdateIfChanged(&repository.Token, "") || updated
		} else {
			encrypted, err := crypto.Encrypt(*req.Token)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt token: %w", err)
			}
			updated = utils.UpdateIfChanged(&repository.Token, encrypted) || updated
		}
	}

	if req.SSHKey != nil {
		if *req.SSHKey == "" {
			updated = utils.UpdateIfChanged(&repository.SSHKey, "") || updated
		} else {
			encrypted, err := crypto.Encrypt(*req.SSHKey)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt SSH key: %w", err)
			}
			updated = utils.UpdateIfChanged(&repository.SSHKey, encrypted) || updated
		}
	}

	if updated {
		saved, err := s.store.SaveGitRepository(ctx, *repository)
		if err != nil {
			return nil, fmt.Errorf("failed to update repository: %w", err)
		}
		if saved != nil {
			repository = saved
		}

		// Log event
		resourceType := "git_repository"
		_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
			Type:         event.EventTypeGitRepositoryUpdate,
			Severity:     event.EventSeveritySuccess,
			Title:        "Git repository updated",
			Description:  fmt.Sprintf("Updated git repository '%s'", repository.Name),
			ResourceType: &resourceType,
			ResourceID:   &repository.ID,
			ResourceName: &repository.Name,
		})
	}

	return repository, nil
}

func (s *GitRepositoryService) DeleteRepository(ctx context.Context, id string) error {
	// Check if repository is used by any syncs
	count, err := s.store.CountGitOpsSyncsByRepositoryID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check repository usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("repository is used by %d sync configuration(s)", count)
	}

	// Get repository info before deleting
	repository, err := s.GetRepositoryByID(ctx, id)
	if err != nil {
		return err
	}

	deleted, err := s.store.DeleteGitRepositoryByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}
	if !deleted {
		return ErrGitRepositoryNotFound
	}

	// Log event
	resourceType := "git_repository"
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:         event.EventTypeGitRepositoryDelete,
		Severity:     event.EventSeverityInfo,
		Title:        "Git repository deleted",
		Description:  fmt.Sprintf("Deleted git repository '%s'", repository.Name),
		ResourceType: &resourceType,
		ResourceID:   &repository.ID,
		ResourceName: &repository.Name,
	})

	return nil
}
func (s *GitRepositoryService) TestConnection(ctx context.Context, id string, branch string) error {
	settings := s.settingsService.GetSettingsConfig()
	ctx, cancel := timeouts.WithTimeout(ctx, settings.GitOperationTimeout.AsInt(), timeouts.DefaultGitOperation)
	defer cancel()

	repository, err := s.GetRepositoryByID(ctx, id)
	if err != nil {
		return err
	}

	authConfig, err := s.GetAuthConfig(ctx, repository)
	if err != nil {
		return err
	}

	if branch == "" {
		branch = "main"
	}

	err = s.gitClient.TestConnection(ctx, repository.URL, branch, authConfig)
	if err != nil {
		// Log error event
		resourceType := "git_repository"
		_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
			Type:         event.EventTypeGitRepositoryError,
			Severity:     event.EventSeverityError,
			Title:        "Git repository connection test failed",
			Description:  fmt.Sprintf("Failed to connect to repository '%s': %s", repository.Name, err.Error()),
			ResourceType: &resourceType,
			ResourceID:   &repository.ID,
			ResourceName: &repository.Name,
		})
		return err
	}

	// Log success event
	resourceType := "git_repository"
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:         event.EventTypeGitRepositoryTest,
		Severity:     event.EventSeveritySuccess,
		Title:        "Git repository connection successful",
		Description:  fmt.Sprintf("Successfully connected to repository '%s'", repository.Name),
		ResourceType: &resourceType,
		ResourceID:   &repository.ID,
		ResourceName: &repository.Name,
	})

	return nil
}

func (s *GitRepositoryService) GetAuthConfig(ctx context.Context, repository *gitops.ModelGitRepository) (git.AuthConfig, error) {
	authConfig := git.AuthConfig{
		AuthType:               repository.AuthType,
		Username:               repository.Username,
		SSHHostKeyVerification: repository.SSHHostKeyVerification,
	}

	if repository.Token != "" {
		token, err := crypto.Decrypt(repository.Token)
		if err != nil {
			return authConfig, fmt.Errorf("failed to decrypt token: %w", err)
		}
		authConfig.Token = token
	}

	if repository.SSHKey != "" {
		sshKey, err := crypto.Decrypt(repository.SSHKey)
		if err != nil {
			return authConfig, fmt.Errorf("failed to decrypt SSH key: %w", err)
		}
		authConfig.SSHKey = sshKey
	}

	return authConfig, nil
}

func (s *GitRepositoryService) ListBranches(ctx context.Context, id string) ([]gitops.BranchInfo, error) {
	settings := s.settingsService.GetSettingsConfig()
	listCtx, cancel := timeouts.WithTimeout(ctx, settings.GitOperationTimeout.AsInt(), timeouts.DefaultGitOperation)
	defer cancel()

	repository, err := s.GetRepositoryByID(listCtx, id)
	if err != nil {
		return nil, err
	}

	authConfig, err := s.GetAuthConfig(listCtx, repository)
	if err != nil {
		return nil, err
	}

	branches, err := s.gitClient.ListBranches(listCtx, repository.URL, authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var result []gitops.BranchInfo
	for _, branch := range branches {
		result = append(result, gitops.BranchInfo{
			Name:      branch.Name,
			IsDefault: branch.IsDefault,
		})
	}

	return result, nil
}

func (s *GitRepositoryService) BrowseFiles(ctx context.Context, id, branch, path string) (*gitops.BrowseResponse, error) {
	settings := s.settingsService.GetSettingsConfig()
	ctx, cancel := timeouts.WithTimeout(ctx, settings.GitOperationTimeout.AsInt(), timeouts.DefaultGitOperation)
	defer cancel()

	repository, err := s.GetRepositoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	authConfig, err := s.GetAuthConfig(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Clone the repository
	repoPath, err := s.gitClient.Clone(ctx, repository.URL, branch, authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer func() {
		if cleanupErr := s.gitClient.Cleanup(repoPath); cleanupErr != nil {
			// Log cleanup error but don't fail the operation
			_ = cleanupErr
		}
	}()

	// Browse the tree
	files, err := s.gitClient.BrowseTree(ctx, repoPath, path)
	if err != nil {
		return nil, err
	}

	return &gitops.BrowseResponse{
		Path:  path,
		Files: files,
	}, nil
}

// SyncRepositories syncs repositories from a manager to this agent instance.
// It creates, updates, or deletes repositories to match the provided list.
func (s *GitRepositoryService) SyncRepositories(ctx context.Context, syncItems []gitops.RepositorySync) error {
	existingMap, err := s.getExistingRepositoriesMap(ctx)
	if err != nil {
		return err
	}

	syncedIDs := make(map[string]bool)

	// Process each sync item
	for _, item := range syncItems {
		syncedIDs[item.ID] = true

		if err := s.processSyncItem(ctx, item, existingMap); err != nil {
			return err
		}
	}

	// Delete repositories that are not in the sync list
	return s.deleteUnsynced(ctx, existingMap, syncedIDs)
}

func (s *GitRepositoryService) getExistingRepositoriesMap(ctx context.Context) (map[string]*gitops.ModelGitRepository, error) {
	existing, err := s.store.ListGitRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing repositories: %w", err)
	}

	existingMap := make(map[string]*gitops.ModelGitRepository)
	for i := range existing {
		existingMap[existing[i].ID] = &existing[i]
	}
	return existingMap, nil
}

func (s *GitRepositoryService) processSyncItem(ctx context.Context, item gitops.RepositorySync, existingMap map[string]*gitops.ModelGitRepository) error {
	existing, exists := existingMap[item.ID]
	if exists {
		return s.updateExistingRepository(ctx, item, existing)
	}
	return s.createNewRepository(ctx, item)
}

func (s *GitRepositoryService) updateExistingRepository(ctx context.Context, item gitops.RepositorySync, existing *gitops.ModelGitRepository) error {
	needsUpdate := s.checkRepositoryNeedsUpdate(item, existing)

	if needsUpdate {
		if _, err := s.store.SaveGitRepository(ctx, *existing); err != nil {
			return fmt.Errorf("failed to update repository %s: %w", item.ID, err)
		}
	}

	return nil
}

func (s *GitRepositoryService) checkRepositoryNeedsUpdate(item gitops.RepositorySync, existing *gitops.ModelGitRepository) bool {
	needsUpdate := utils.UpdateIfChanged(&existing.Name, item.Name)
	needsUpdate = utils.UpdateIfChanged(&existing.URL, item.URL) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.AuthType, item.AuthType) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.Username, item.Username) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.SSHHostKeyVerification, item.SSHHostKeyVerification) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.Description, item.Description) || needsUpdate
	needsUpdate = utils.UpdateIfChanged(&existing.Enabled, item.Enabled) || needsUpdate

	// Handle Token update
	if item.Token != "" {
		encryptedToken, err := crypto.Encrypt(item.Token)
		if err == nil {
			needsUpdate = utils.UpdateIfChanged(&existing.Token, encryptedToken) || needsUpdate
		}
	} else if existing.Token != "" {
		existing.Token = ""
		needsUpdate = true
	}

	// Handle SSH Key update
	if item.SSHKey != "" {
		encryptedSSHKey, err := crypto.Encrypt(item.SSHKey)
		if err == nil {
			needsUpdate = utils.UpdateIfChanged(&existing.SSHKey, encryptedSSHKey) || needsUpdate
		}
	} else if existing.SSHKey != "" {
		existing.SSHKey = ""
		needsUpdate = true
	}

	return needsUpdate
}

func (s *GitRepositoryService) createNewRepository(ctx context.Context, item gitops.RepositorySync) error {
	var encryptedToken, encryptedSSHKey string
	var err error

	if item.Token != "" {
		encryptedToken, err = crypto.Encrypt(item.Token)
		if err != nil {
			return fmt.Errorf("failed to encrypt token for repository %s: %w", item.ID, err)
		}
	}

	if item.SSHKey != "" {
		encryptedSSHKey, err = crypto.Encrypt(item.SSHKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt SSH key for repository %s: %w", item.ID, err)
		}
	}

	sshHostKeyVerification := item.SSHHostKeyVerification
	if sshHostKeyVerification == "" {
		sshHostKeyVerification = "accept_new"
	}

	repo := gitops.ModelGitRepository{
		BaseModel: base.BaseModel{
			ID: item.ID,
		},
		Name:                   item.Name,
		URL:                    item.URL,
		AuthType:               item.AuthType,
		Username:               item.Username,
		Token:                  encryptedToken,
		SSHKey:                 encryptedSSHKey,
		SSHHostKeyVerification: sshHostKeyVerification,
		Description:            item.Description,
		Enabled:                item.Enabled,
	}
	if _, err := s.store.CreateGitRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to create repository %s: %w", item.ID, err)
	}

	return nil
}

func (s *GitRepositoryService) deleteUnsynced(ctx context.Context, existingMap map[string]*gitops.ModelGitRepository, syncedIDs map[string]bool) error {
	for id := range existingMap {
		if !syncedIDs[id] {
			if _, err := s.store.DeleteGitRepositoryByID(ctx, id); err != nil {
				return fmt.Errorf("failed to delete repository %s: %w", id, err)
			}
		}
	}
	return nil
}
