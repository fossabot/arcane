package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils"
	bootstraputils "github.com/getarcaneapp/arcane/backend/internal/utils"
	"github.com/getarcaneapp/arcane/backend/internal/utils/mapper"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/types/event"
	"github.com/getarcaneapp/arcane/types/gitops"
	"github.com/getarcaneapp/arcane/types/project"
)

type GitOpsSyncService struct {
	store          database.GitOpsSyncStore
	repoService    *GitRepositoryService
	projectService *ProjectService
	eventService   *EventService
}

const defaultGitSyncTimeout = 5 * time.Minute

func NewGitOpsSyncService(store database.GitOpsSyncStore, repoService *GitRepositoryService, projectService *ProjectService, eventService *EventService) *GitOpsSyncService {
	return &GitOpsSyncService{
		store:          store,
		repoService:    repoService,
		projectService: projectService,
		eventService:   eventService,
	}
}

func (s *GitOpsSyncService) enrichSyncRepository(ctx context.Context, sync *gitops.ModelGitOpsSync) error {
	if sync == nil || sync.RepositoryID == "" {
		return nil
	}

	repository, err := s.repoService.GetRepositoryByID(ctx, sync.RepositoryID)
	if err != nil {
		if errors.Is(err, ErrGitRepositoryNotFound) {
			sync.Repository = nil
			return nil
		}
		return fmt.Errorf("failed to load repository for sync %s: %w", sync.ID, err)
	}

	sync.Repository = repository
	return nil
}

func (s *GitOpsSyncService) enrichSyncRepositories(ctx context.Context, syncs []gitops.ModelGitOpsSync) error {
	repositoryCache := make(map[string]*gitops.ModelGitRepository)
	missing := make(map[string]bool)

	for i := range syncs {
		repositoryID := syncs[i].RepositoryID
		if repositoryID == "" {
			continue
		}
		if repository, ok := repositoryCache[repositoryID]; ok {
			syncs[i].Repository = repository
			continue
		}
		if missing[repositoryID] {
			continue
		}

		repository, err := s.repoService.GetRepositoryByID(ctx, repositoryID)
		if err != nil {
			if errors.Is(err, ErrGitRepositoryNotFound) {
				missing[repositoryID] = true
				continue
			}
			return fmt.Errorf("failed to load repository for sync %s: %w", syncs[i].ID, err)
		}

		repositoryCache[repositoryID] = repository
		syncs[i].Repository = repository
	}

	return nil
}

func (s *GitOpsSyncService) ListSyncIntervalsRaw(ctx context.Context) ([]bootstraputils.IntervalMigrationItem, error) {
	syncs, err := s.store.ListGitOpsSyncs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load git sync intervals: %w", err)
	}

	items := make([]bootstraputils.IntervalMigrationItem, 0, len(syncs))
	for _, sync := range syncs {
		items = append(items, bootstraputils.IntervalMigrationItem{
			ID:       sync.ID,
			RawValue: strconv.Itoa(sync.SyncInterval),
		})
	}

	return items, nil
}

func (s *GitOpsSyncService) UpdateSyncIntervalMinutes(ctx context.Context, id string, minutes int) error {
	if minutes <= 0 {
		return fmt.Errorf("sync interval must be positive")
	}
	return s.store.UpdateGitOpsSyncInterval(ctx, id, minutes)
}

func (s *GitOpsSyncService) GetSyncsPaginated(ctx context.Context, environmentID string, params pagination.QueryParams) ([]gitops.GitOpsSync, pagination.Response, error) {
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

	syncs, err := s.store.ListGitOpsSyncsByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list gitops syncs: %w", err)
	}

	config := pagination.Config[gitops.ModelGitOpsSync]{
		SearchAccessors: []pagination.SearchAccessor[gitops.ModelGitOpsSync]{
			func(sync gitops.ModelGitOpsSync) (string, error) { return sync.Name, nil },
			func(sync gitops.ModelGitOpsSync) (string, error) { return sync.Branch, nil },
			func(sync gitops.ModelGitOpsSync) (string, error) { return sync.ComposePath, nil },
		},
		SortBindings: []pagination.SortBinding[gitops.ModelGitOpsSync]{
			{Key: "name", Fn: func(a, b gitops.ModelGitOpsSync) int { return strings.Compare(a.Name, b.Name) }},
			{Key: "repositoryId", Fn: func(a, b gitops.ModelGitOpsSync) int { return strings.Compare(a.RepositoryID, b.RepositoryID) }},
			{Key: "branch", Fn: func(a, b gitops.ModelGitOpsSync) int { return strings.Compare(a.Branch, b.Branch) }},
			{Key: "composePath", Fn: func(a, b gitops.ModelGitOpsSync) int { return strings.Compare(a.ComposePath, b.ComposePath) }},
			{Key: "projectName", Fn: func(a, b gitops.ModelGitOpsSync) int { return strings.Compare(a.ProjectName, b.ProjectName) }},
			{Key: "autoSync", Fn: func(a, b gitops.ModelGitOpsSync) int { return compareBool(a.AutoSync, b.AutoSync) }},
			{Key: "syncInterval", Fn: func(a, b gitops.ModelGitOpsSync) int { return compareInt(a.SyncInterval, b.SyncInterval) }},
			{Key: "lastSyncAt", Fn: func(a, b gitops.ModelGitOpsSync) int { return compareOptionalTime(a.LastSyncAt, b.LastSyncAt) }},
			{Key: "createdAt", Fn: func(a, b gitops.ModelGitOpsSync) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "updatedAt", Fn: func(a, b gitops.ModelGitOpsSync) int { return compareOptionalTime(a.UpdatedAt, b.UpdatedAt) }},
		},
		FilterAccessors: []pagination.FilterAccessor[gitops.ModelGitOpsSync]{
			{
				Key: "autoSync",
				Fn: func(sync gitops.ModelGitOpsSync, filterValue string) bool {
					return boolFilterMatches(sync.AutoSync, filterValue)
				},
			},
			{
				Key: "repositoryId",
				Fn: func(sync gitops.ModelGitOpsSync, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(sync.RepositoryID), strings.TrimSpace(filterValue))
				},
			},
			{
				Key: "projectId",
				Fn: func(sync gitops.ModelGitOpsSync, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(utils.DerefString(sync.ProjectID)), strings.TrimSpace(filterValue))
				},
			},
		},
	}

	result := pagination.SearchOrderAndPaginate(syncs, params, config)
	paginationResp := pagination.BuildResponseFromFilterResult(result, params)

	if err := s.enrichSyncRepositories(ctx, result.Items); err != nil {
		return nil, pagination.Response{}, err
	}

	out, mapErr := mapper.MapSlice[gitops.ModelGitOpsSync, gitops.GitOpsSync](result.Items)
	if mapErr != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to map syncs: %w", mapErr)
	}

	return out, paginationResp, nil
}

func (s *GitOpsSyncService) GetSyncByID(ctx context.Context, environmentID, id string) (*gitops.ModelGitOpsSync, error) {
	sync, err := s.store.GetGitOpsSyncByID(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get GitOps sync", "syncID", id, "environmentID", environmentID, "error", err)
		return nil, fmt.Errorf("failed to get sync: %w", err)
	}
	if sync == nil || (environmentID != "" && sync.EnvironmentID != environmentID) {
		slog.WarnContext(ctx, "GitOps sync not found", "syncID", id, "environmentID", environmentID)
		return nil, fmt.Errorf("sync not found")
	}

	if err := s.enrichSyncRepository(ctx, sync); err != nil {
		return nil, err
	}

	return sync, nil
}

func (s *GitOpsSyncService) CreateSync(ctx context.Context, environmentID string, req gitops.CreateSyncRequest) (*gitops.ModelGitOpsSync, error) {
	slog.InfoContext(ctx, "Creating GitOps sync", "environmentID", environmentID, "name", req.Name, "repositoryID", req.RepositoryID)

	// Validate repository exists
	repo, err := s.repoService.GetRepositoryByID(ctx, req.RepositoryID)
	if err != nil {
		slog.ErrorContext(ctx, "Repository not found for GitOps sync", "repositoryID", req.RepositoryID, "error", err)
		return nil, fmt.Errorf("repository not found: %w", err)
	}
	slog.InfoContext(ctx, "Found repository for GitOps sync", "repositoryID", req.RepositoryID, "repositoryName", repo.Name)

	// Store the project name - use sync name if project name not provided
	projectName := req.ProjectName
	if projectName == "" {
		projectName = req.Name
	}

	sync := gitops.ModelGitOpsSync{
		Name:          req.Name,
		EnvironmentID: environmentID,
		RepositoryID:  req.RepositoryID,
		Branch:        req.Branch,
		ComposePath:   req.ComposePath,
		ProjectName:   projectName,
		ProjectID:     nil, // Will be set during first sync
		AutoSync:      false,
		SyncInterval:  60,
	}

	if req.AutoSync != nil {
		sync.AutoSync = *req.AutoSync
	}
	if req.SyncInterval != nil {
		sync.SyncInterval = *req.SyncInterval
	}

	sync.ID = uuid.NewString()
	created, err := s.store.CreateGitOpsSync(ctx, sync)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create GitOps sync in database", "name", req.Name, "repositoryID", req.RepositoryID, "environmentID", environmentID, "error", err)
		return nil, fmt.Errorf("failed to create sync: %w", err)
	}
	slog.InfoContext(ctx, "GitOps sync created successfully", "syncID", created.ID, "name", created.Name)

	// Log event
	resourceType := "git_sync"
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:         event.EventTypeGitSyncCreate,
		Severity:     event.EventSeveritySuccess,
		Title:        "Git sync created",
		Description:  fmt.Sprintf("Created git sync configuration '%s'", created.Name),
		ResourceType: &resourceType,
		ResourceID:   &created.ID,
		ResourceName: &created.Name,
		UserID:       &systemUser.ID,
		Username:     &systemUser.Username,
	})

	if _, err := s.PerformSync(ctx, created.EnvironmentID, created.ID); err != nil {
		slog.ErrorContext(ctx, "Failed to perform initial sync after creation", "syncId", created.ID, "error", err)
		// Don't fail the entire creation - the sync config exists and can be retried
	}

	return s.GetSyncByID(ctx, "", created.ID)
}

func (s *GitOpsSyncService) UpdateSync(ctx context.Context, environmentID, id string, req gitops.UpdateSyncRequest) (*gitops.ModelGitOpsSync, error) {
	sync, err := s.GetSyncByID(ctx, environmentID, id)
	if err != nil {
		return nil, err
	}

	updated := false

	if req.Name != nil {
		updated = utils.UpdateIfChanged(&sync.Name, req.Name) || updated
	}
	if req.RepositoryID != nil {
		// Validate repository exists
		_, err := s.repoService.GetRepositoryByID(ctx, *req.RepositoryID)
		if err != nil {
			return nil, fmt.Errorf("repository not found: %w", err)
		}
		updated = utils.UpdateIfChanged(&sync.RepositoryID, req.RepositoryID) || updated
	}
	if req.Branch != nil {
		updated = utils.UpdateIfChanged(&sync.Branch, req.Branch) || updated
	}
	if req.ComposePath != nil {
		updated = utils.UpdateIfChanged(&sync.ComposePath, req.ComposePath) || updated
	}
	if req.ProjectName != nil {
		updated = utils.UpdateIfChanged(&sync.ProjectName, req.ProjectName) || updated
	}
	if req.AutoSync != nil {
		updated = utils.UpdateIfChanged(&sync.AutoSync, req.AutoSync) || updated
	}
	if req.SyncInterval != nil {
		if sync.SyncInterval != *req.SyncInterval {
			sync.SyncInterval = *req.SyncInterval
			updated = true
		}
	}

	if updated {
		saved, err := s.store.SaveGitOpsSync(ctx, *sync)
		if err != nil {
			return nil, fmt.Errorf("failed to update sync: %w", err)
		}
		if saved != nil {
			sync = saved
		}

		// Log event
		resourceType := "git_sync"
		_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
			Type:         event.EventTypeGitSyncUpdate,
			Severity:     event.EventSeveritySuccess,
			Title:        "Git sync updated",
			Description:  fmt.Sprintf("Updated git sync configuration '%s'", sync.Name),
			ResourceType: &resourceType,
			ResourceID:   &sync.ID,
			ResourceName: &sync.Name,
		})
	}

	return sync, nil
}

func (s *GitOpsSyncService) DeleteSync(ctx context.Context, environmentID, id string) error {
	// Get sync info before deleting
	sync, err := s.GetSyncByID(ctx, environmentID, id)
	if err != nil {
		return err
	}

	// Clear gitops_managed_by for the associated project, if any.
	if sync.ProjectID != nil && *sync.ProjectID != "" {
		if err := s.store.ClearProjectGitOpsManagedByIfMatches(ctx, *sync.ProjectID, id); err != nil {
			return fmt.Errorf("failed to clear gitops_managed_by: %w", err)
		}
	}

	deleted, err := s.store.DeleteGitOpsSyncByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete sync: %w", err)
	}
	if !deleted {
		return fmt.Errorf("sync not found")
	}

	// Log event
	resourceType := "git_sync"
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:         event.EventTypeGitSyncDelete,
		Severity:     event.EventSeverityInfo,
		Title:        "Git sync deleted",
		Description:  fmt.Sprintf("Deleted git sync configuration '%s'", sync.Name),
		ResourceType: &resourceType,
		ResourceID:   &sync.ID,
		ResourceName: &sync.Name, UserID: &systemUser.ID,
		Username: &systemUser.Username})

	return nil
}

func (s *GitOpsSyncService) PerformSync(ctx context.Context, environmentID, id string) (*gitops.SyncResult, error) {
	syncCtx, cancel := context.WithTimeout(ctx, defaultGitSyncTimeout)
	defer cancel()

	sync, err := s.GetSyncByID(syncCtx, environmentID, id)
	if err != nil {
		return nil, err
	}

	result := &gitops.SyncResult{
		Success:  false,
		SyncedAt: time.Now(),
	}

	// Get repository and auth config
	repository := sync.Repository
	if repository == nil {
		return result, s.failSync(syncCtx, id, result, sync, "Repository not found", "repository not found")
	}

	authConfig, err := s.repoService.GetAuthConfig(syncCtx, repository)
	if err != nil {
		return result, s.failSync(syncCtx, id, result, sync, "Failed to get authentication config", err.Error())
	}

	// Clone the repository
	repoPath, err := s.repoService.gitClient.Clone(syncCtx, repository.URL, sync.Branch, authConfig)
	if err != nil {
		return result, s.failSync(syncCtx, id, result, sync, "Failed to clone repository", err.Error())
	}
	defer func() {
		if cleanupErr := s.repoService.gitClient.Cleanup(repoPath); cleanupErr != nil {
			slog.WarnContext(syncCtx, "Failed to cleanup repository", "path", repoPath, "error", cleanupErr)
		}
	}()

	// Get the current commit hash
	commitHash, err := s.repoService.gitClient.GetCurrentCommit(syncCtx, repoPath)
	if err != nil {
		slog.WarnContext(syncCtx, "Failed to get commit hash", "error", err)
		commitHash = ""
	}

	// Check if compose file exists
	if !s.repoService.gitClient.FileExists(syncCtx, repoPath, sync.ComposePath) {
		errMsg := fmt.Sprintf("compose file not found: %s", sync.ComposePath)
		return result, s.failSync(syncCtx, id, result, sync, fmt.Sprintf("Compose file not found at %s", sync.ComposePath), errMsg)
	}

	// Read compose file content
	composeContent, err := s.repoService.gitClient.ReadFile(syncCtx, repoPath, sync.ComposePath)
	if err != nil {
		return result, s.failSync(syncCtx, id, result, sync, "Failed to read compose file", err.Error())
	}

	// Try to read .env file from the same directory as the compose file
	var envContent *string
	envPath := filepath.Join(filepath.Dir(sync.ComposePath), ".env")
	if s.repoService.gitClient.FileExists(syncCtx, repoPath, envPath) {
		content, err := s.repoService.gitClient.ReadFile(syncCtx, repoPath, envPath)
		if err != nil {
			slog.WarnContext(syncCtx, "Failed to read .env file", "path", envPath, "error", err)
		} else {
			envContent = &content
		}
	}

	// Get or create project
	project, err := s.getOrCreateProjectInternal(syncCtx, sync, id, composeContent, envContent, result)
	if err != nil {
		return result, err
	}

	// Update sync status
	s.updateSyncStatus(syncCtx, id, "success", "", commitHash)

	result.Success = true
	result.Message = fmt.Sprintf("Successfully synced compose file from %s to project %s", sync.ComposePath, project.Name)

	// Log success event
	resourceType := "git_sync"
	_, _ = s.eventService.CreateEvent(syncCtx, CreateEventRequest{
		Type:         event.EventTypeGitSyncRun,
		Severity:     event.EventSeveritySuccess,
		Title:        "Git sync completed",
		Description:  fmt.Sprintf("Successfully synced '%s' to project '%s'", sync.Name, project.Name),
		ResourceType: &resourceType,
		ResourceID:   &sync.ID,
		ResourceName: &sync.Name,
		UserID:       &systemUser.ID,
		Username:     &systemUser.Username,
	})

	slog.InfoContext(syncCtx, "GitOps sync completed", "syncId", id, "project", project.Name)

	return result, nil
}

func (s *GitOpsSyncService) updateSyncStatus(ctx context.Context, id, status, errorMsg, commitHash string) {
	now := time.Now()
	var errorPtr *string
	if errorMsg != "" {
		errorPtr = &errorMsg
	}
	var commitPtr *string
	if commitHash != "" {
		commitPtr = &commitHash
	}

	if err := s.store.UpdateGitOpsSyncStatus(ctx, id, now, status, errorPtr, commitPtr); err != nil {
		slog.ErrorContext(ctx, "Failed to update sync status", "error", err, "syncId", id)
	}
}

func (s *GitOpsSyncService) GetSyncStatus(ctx context.Context, environmentID, id string) (*gitops.SyncStatus, error) {
	sync, err := s.GetSyncByID(ctx, environmentID, id)
	if err != nil {
		return nil, err
	}

	status := &gitops.SyncStatus{
		ID:             sync.ID,
		AutoSync:       sync.AutoSync,
		LastSyncAt:     sync.LastSyncAt,
		LastSyncStatus: sync.LastSyncStatus,
		LastSyncError:  sync.LastSyncError,
		LastSyncCommit: sync.LastSyncCommit,
	}

	// Calculate next sync time
	if sync.AutoSync && sync.LastSyncAt != nil {
		nextSync := sync.LastSyncAt.Add(time.Duration(sync.SyncInterval) * time.Minute)
		status.NextSyncAt = &nextSync
	}

	return status, nil
}

func (s *GitOpsSyncService) SyncAllEnabled(ctx context.Context) error {
	syncs, err := s.store.ListAutoSyncGitOpsSyncs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auto-sync enabled syncs: %w", err)
	}

	for _, sync := range syncs {
		// Check if sync is due
		if sync.LastSyncAt != nil {
			nextSync := sync.LastSyncAt.Add(time.Duration(sync.SyncInterval) * time.Minute)
			// Use a 30-second buffer to account for execution time drift
			if time.Now().Add(30 * time.Second).Before(nextSync) {
				continue
			}
		}

		// Perform sync
		result, err := s.PerformSync(ctx, sync.EnvironmentID, sync.ID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to sync", "syncId", sync.ID, "error", err)
			continue
		}

		if result.Success {
			slog.InfoContext(ctx, "Sync completed", "syncId", sync.ID, "message", result.Message)
		}
	}

	return nil
}

func (s *GitOpsSyncService) BrowseFiles(ctx context.Context, environmentID, id string, path string) (*gitops.BrowseResponse, error) {
	browseCtx, cancel := context.WithTimeout(ctx, defaultGitSyncTimeout)
	defer cancel()

	sync, err := s.GetSyncByID(browseCtx, environmentID, id)
	if err != nil {
		return nil, err
	}

	repository := sync.Repository
	if repository == nil {
		return nil, fmt.Errorf("repository not found")
	}

	authConfig, err := s.repoService.GetAuthConfig(browseCtx, repository)
	if err != nil {
		return nil, err
	}

	// Clone the repository
	repoPath, err := s.repoService.gitClient.Clone(browseCtx, repository.URL, sync.Branch, authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer func() {
		if cleanupErr := s.repoService.gitClient.Cleanup(repoPath); cleanupErr != nil {
			slog.WarnContext(browseCtx, "Failed to cleanup repository", "path", repoPath, "error", cleanupErr)
		}
	}()

	// Browse the tree
	files, err := s.repoService.gitClient.BrowseTree(browseCtx, repoPath, path)
	if err != nil {
		return nil, err
	}

	return &gitops.BrowseResponse{
		Path:  path,
		Files: files,
	}, nil
}

func (s *GitOpsSyncService) ImportSyncs(ctx context.Context, environmentID string, req []gitops.ImportGitOpsSyncRequest) (*gitops.ImportGitOpsSyncResponse, error) {
	response := &gitops.ImportGitOpsSyncResponse{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       []string{},
	}

	for _, importItem := range req {
		// Find repository by name
		repo, err := s.repoService.GetRepositoryByName(ctx, importItem.GitRepo)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Stack '%s': Repository '%s' not found (%v)", importItem.SyncName, importItem.GitRepo, err))
			continue
		}

		createReq := gitops.CreateSyncRequest{
			Name:         importItem.SyncName,
			RepositoryID: repo.ID,
			Branch:       importItem.Branch,
			ComposePath:  importItem.DockerComposePath,
			ProjectName:  importItem.SyncName,
			AutoSync:     &importItem.AutoSync,
			SyncInterval: &importItem.SyncInterval,
		}

		_, err = s.CreateSync(ctx, environmentID, createReq)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Stack '%s': %v", importItem.SyncName, err))
		} else {
			response.SuccessCount++
		}
	}

	return response, nil
}

func (s *GitOpsSyncService) logSyncError(ctx context.Context, sync *gitops.ModelGitOpsSync, errorMsg string) {
	resourceType := "git_sync"
	_, _ = s.eventService.CreateEvent(ctx, CreateEventRequest{
		Type:         event.EventTypeGitSyncError,
		Severity:     event.EventSeverityError,
		Title:        "Git sync failed",
		Description:  fmt.Sprintf("Failed to sync '%s': %s", sync.Name, errorMsg),
		ResourceType: &resourceType,
		ResourceID:   &sync.ID,
		ResourceName: &sync.Name, UserID: &systemUser.ID,
		Username: &systemUser.Username})
}

func (s *GitOpsSyncService) failSync(ctx context.Context, id string, result *gitops.SyncResult, sync *gitops.ModelGitOpsSync, message, errMsg string) error {
	result.Message = message
	result.Error = &errMsg
	s.updateSyncStatus(ctx, id, "failed", errMsg, "")
	s.logSyncError(ctx, sync, errMsg)
	return fmt.Errorf("%s", errMsg)
}

func (s *GitOpsSyncService) createProjectForSyncInternal(ctx context.Context, sync *gitops.ModelGitOpsSync, id string, composeContent string, envContent *string, result *gitops.SyncResult) (*project.Project, error) {
	project, err := s.projectService.CreateProject(ctx, sync.ProjectName, composeContent, envContent, systemUser)
	if err != nil {
		return nil, s.failSync(ctx, id, result, sync, "Failed to create project", err.Error())
	}

	// Update sync with project ID
	if err := s.store.UpdateGitOpsSyncProjectID(ctx, id, &project.ID); err != nil {
		return nil, s.failSync(ctx, id, result, sync, "Failed to update sync with project ID", err.Error())
	}
	sync.ProjectID = &project.ID

	// Mark project as GitOps-managed
	if err := s.store.SetProjectGitOpsManagedBy(ctx, project.ID, &id); err != nil {
		return nil, s.failSync(ctx, id, result, sync, "Failed to mark project as GitOps-managed", err.Error())
	}

	slog.InfoContext(ctx, "Created project for GitOps sync", "projectName", sync.ProjectName, "projectId", project.ID)

	// Deploy the project immediately after creation
	slog.InfoContext(ctx, "Deploying project after initial Git sync", "projectName", project.Name, "projectId", project.ID)
	if err := s.projectService.DeployProject(ctx, project.ID, systemUser); err != nil {
		slog.ErrorContext(ctx, "Failed to deploy project after initial Git sync", "error", err, "projectId", project.ID)
	}

	return project, nil
}

func (s *GitOpsSyncService) getOrCreateProjectInternal(ctx context.Context, sync *gitops.ModelGitOpsSync, id string, composeContent string, envContent *string, result *gitops.SyncResult) (*project.Project, error) {
	var project *project.Project
	var err error

	if sync.ProjectID != nil && *sync.ProjectID != "" {
		project, err = s.projectService.GetProjectFromDatabaseByID(ctx, *sync.ProjectID)
		if err != nil {
			slog.WarnContext(ctx, "Existing project not found, will create new one", "projectId", *sync.ProjectID, "error", err)
			project = nil
		}
	}

	if project == nil {
		return s.createProjectForSyncInternal(ctx, sync, id, composeContent, envContent, result)
	}

	if err := s.updateProjectForSyncInternal(ctx, sync, id, project, composeContent, envContent, result); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *GitOpsSyncService) updateProjectForSyncInternal(ctx context.Context, sync *gitops.ModelGitOpsSync, id string, proj *project.Project, composeContent string, envContent *string, result *gitops.SyncResult) error {
	// Get current content to see if it changed
	oldCompose, oldEnv, _ := s.projectService.GetProjectContent(ctx, proj.ID)
	contentChanged := oldCompose != composeContent
	if envContent != nil {
		if oldEnv != *envContent {
			contentChanged = true
		}
	} else if oldEnv != "" {
		contentChanged = true
	}

	// Update existing project's compose and env files
	_, err := s.projectService.UpdateProject(ctx, proj.ID, nil, &composeContent, envContent)
	if err != nil {
		return s.failSync(ctx, id, result, sync, "Failed to update project files", err.Error())
	}
	slog.InfoContext(ctx, "Updated project files", "projectName", proj.Name, "projectId", proj.ID)

	// If content changed and project is running, redeploy
	if contentChanged {
		details, err := s.projectService.GetProjectDetails(ctx, proj.ID)
		if err == nil && (details.Status == string(project.ProjectStatusRunning) || details.Status == string(project.ProjectStatusPartiallyRunning)) {
			slog.InfoContext(ctx, "Redeploying project due to content change from Git sync", "projectName", proj.Name, "projectId", proj.ID)
			if err := s.projectService.RedeployProject(ctx, proj.ID, systemUser); err != nil {
				slog.ErrorContext(ctx, "Failed to redeploy project after Git sync", "error", err, "projectId", proj.ID)
			}
		}
	}

	return nil
}
