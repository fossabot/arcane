package stores

import (
	"context"
	"time"

	"github.com/getarcaneapp/arcane/types/apikey"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/containerregistry"
	"github.com/getarcaneapp/arcane/types/environment"
	"github.com/getarcaneapp/arcane/types/event"
	"github.com/getarcaneapp/arcane/types/gitops"
	"github.com/getarcaneapp/arcane/types/imageupdate"
	"github.com/getarcaneapp/arcane/types/notification"
	"github.com/getarcaneapp/arcane/types/project"
	"github.com/getarcaneapp/arcane/types/settings"
	"github.com/getarcaneapp/arcane/types/template"
	"github.com/getarcaneapp/arcane/types/updater"
	"github.com/getarcaneapp/arcane/types/user"
	"github.com/getarcaneapp/arcane/types/volume"
)

// Each interface should be kept small and focused on a single domain.

type UserStore interface {
	CreateUser(ctx context.Context, user user.ModelUser) (*user.ModelUser, error)
	GetUserByUsername(ctx context.Context, username string) (*user.ModelUser, error)
	GetUserByID(ctx context.Context, id string) (*user.ModelUser, error)
	GetUserByOidcSubjectID(ctx context.Context, subjectID string) (*user.ModelUser, error)
	GetUserByEmail(ctx context.Context, email string) (*user.ModelUser, error)
	SaveUser(ctx context.Context, user user.ModelUser) (*user.ModelUser, error)
	AttachOidcSubjectTransactional(ctx context.Context, userID string, subject string, updateFn func(u *user.ModelUser)) (*user.ModelUser, error)
	CountUsers(ctx context.Context) (int64, error)
	DeleteUserByID(ctx context.Context, id string) (bool, error)
	UpdateUserPasswordHash(ctx context.Context, id string, passwordHash string, updatedAt time.Time) error
	ListUsers(ctx context.Context) ([]user.ModelUser, error)
}

type EventStore interface {
	CreateEvent(ctx context.Context, event event.ModelEvent) (*event.ModelEvent, error)
	ListEvents(ctx context.Context) ([]event.ModelEvent, error)
	DeleteEventByID(ctx context.Context, id string) (bool, error)
	DeleteEventsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type SettingsStore interface {
	ListSettings(ctx context.Context) ([]settings.SettingVariable, error)
	GetSetting(ctx context.Context, key string) (*settings.SettingVariable, error)
	UpsertSetting(ctx context.Context, key, value string) error
	InsertSettingIfNotExists(ctx context.Context, key, value string) error
	DeleteSetting(ctx context.Context, key string) error
	UpdateSettingKey(ctx context.Context, oldKey, newKey string) error
	DeleteSettingsNotIn(ctx context.Context, keys []string) (int64, error)
	WithSettingsTx(ctx context.Context, fn func(tx SettingsStoreTx) error) error
}

type SettingsStoreTx interface {
	ListSettings(ctx context.Context) ([]settings.SettingVariable, error)
	GetSetting(ctx context.Context, key string) (*settings.SettingVariable, error)
	UpsertSetting(ctx context.Context, key, value string) error
	InsertSettingIfNotExists(ctx context.Context, key, value string) error
	DeleteSetting(ctx context.Context, key string) error
	UpdateSettingKey(ctx context.Context, oldKey, newKey string) error
	DeleteSettingsNotIn(ctx context.Context, keys []string) (int64, error)
}

type TemplateStore interface {
	GetComposeTemplateByID(ctx context.Context, id string) (*template.ComposeTemplate, error)
	ListComposeTemplates(ctx context.Context) ([]template.ComposeTemplate, error)
	FindLocalComposeTemplateByDescriptionOrName(ctx context.Context, description string, name string) (*template.ComposeTemplate, error)
	FindLocalComposeTemplateByDescription(ctx context.Context, description string) (*template.ComposeTemplate, error)
	CreateComposeTemplate(ctx context.Context, template template.ComposeTemplate) (*template.ComposeTemplate, error)
	SaveComposeTemplate(ctx context.Context, template template.ComposeTemplate) (*template.ComposeTemplate, error)
	DeleteComposeTemplateByID(ctx context.Context, id string) (bool, error)
	ListTemplateRegistries(ctx context.Context) ([]template.ModelTemplateRegistry, error)
	GetTemplateRegistryByID(ctx context.Context, id string) (*template.ModelTemplateRegistry, error)
	CreateTemplateRegistry(ctx context.Context, registry template.ModelTemplateRegistry) (*template.ModelTemplateRegistry, error)
	SaveTemplateRegistry(ctx context.Context, registry template.ModelTemplateRegistry) (*template.ModelTemplateRegistry, error)
	DeleteTemplateRegistryByID(ctx context.Context, id string) (bool, error)
}

type ImageUpdateStore interface {
	GetImageUpdateByID(ctx context.Context, id string) (*imageupdate.ImageUpdateRecord, error)
	SaveImageUpdateRecord(ctx context.Context, record imageupdate.ImageUpdateRecord) (*imageupdate.ImageUpdateRecord, error)
	ListImageUpdateRecords(ctx context.Context) ([]imageupdate.ImageUpdateRecord, error)
	ListImageUpdateRecordsByIDs(ctx context.Context, ids []string) ([]imageupdate.ImageUpdateRecord, error)
	ListImageUpdateRecordsWithUpdate(ctx context.Context) ([]imageupdate.ImageUpdateRecord, error)
	ListUnnotifiedImageUpdates(ctx context.Context) ([]imageupdate.ImageUpdateRecord, error)
	MarkImageUpdatesAsNotified(ctx context.Context, ids []string) error
	DeleteImageUpdatesByIDs(ctx context.Context, ids []string) (int64, error)
	UpdateImageUpdateHasUpdateByRepositoryTag(ctx context.Context, repository, tag string, hasUpdate bool) error
	CountImageUpdates(ctx context.Context) (int64, error)
	CountImageUpdatesWithUpdate(ctx context.Context) (int64, error)
	CountImageUpdatesWithUpdateType(ctx context.Context, updateType string) (int64, error)
	CountImageUpdatesWithErrors(ctx context.Context) (int64, error)
}

type ContainerRegistryCreateInput struct {
	ID          string
	URL         string
	Username    string
	Token       string
	Description *string
	Insecure    bool
	Enabled     bool
}

type ContainerRegistryUpdateInput struct {
	ID          string
	URL         string
	Username    string
	Token       string
	Description *string
	Insecure    bool
	Enabled     bool
}

type ContainerRegistryStore interface {
	CreateContainerRegistry(ctx context.Context, input ContainerRegistryCreateInput) (*containerregistry.ModelContainerRegistry, error)
	GetContainerRegistryByID(ctx context.Context, id string) (*containerregistry.ModelContainerRegistry, error)
	ListContainerRegistries(ctx context.Context) ([]containerregistry.ModelContainerRegistry, error)
	ListEnabledContainerRegistries(ctx context.Context) ([]containerregistry.ModelContainerRegistry, error)
	UpdateContainerRegistry(ctx context.Context, input ContainerRegistryUpdateInput) (*containerregistry.ModelContainerRegistry, error)
	DeleteContainerRegistryByID(ctx context.Context, id string) (bool, error)
}

type EnvironmentCreateInput struct {
	ID          string
	Name        string
	APIURL      string
	Status      string
	Enabled     bool
	IsEdge      bool
	LastSeen    *time.Time
	AccessToken *string
	ApiKeyID    *string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

type EnvironmentPatchInput struct {
	ID string

	Name        *string
	APIURL      *string
	Status      *string
	Enabled     *bool
	IsEdge      *bool
	LastSeen    *time.Time
	AccessToken *string
	ApiKeyID    *string
	UpdatedAt   *time.Time

	ClearLastSeen    bool
	ClearAccessToken bool
	ClearApiKeyID    bool
}

type EnvironmentStore interface {
	CreateEnvironment(ctx context.Context, input EnvironmentCreateInput) (*environment.ModelEnvironment, error)
	GetEnvironmentByID(ctx context.Context, id string) (*environment.ModelEnvironment, error)
	ListEnvironments(ctx context.Context) ([]environment.ModelEnvironment, error)
	ListRemoteEnvironments(ctx context.Context) ([]environment.ModelEnvironment, error)
	FindEnvironmentIDByApiKeyHash(ctx context.Context, keyHash string) (string, error)
	PatchEnvironment(ctx context.Context, input EnvironmentPatchInput) (*environment.ModelEnvironment, error)
	DeleteEnvironmentByID(ctx context.Context, id string) (bool, error)
	TouchEnvironmentHeartbeatIfStale(ctx context.Context, id string, now time.Time, staleBefore time.Time) (bool, error)
}

type ProjectStore interface {
	CreateProject(ctx context.Context, project project.Project) (*project.Project, error)
	GetProjectByID(ctx context.Context, id string) (*project.Project, error)
	GetProjectByPathOrDir(ctx context.Context, path string, dirName string) (*project.Project, error)
	ListProjects(ctx context.Context) ([]project.Project, error)
	SaveProject(ctx context.Context, project project.Project) (*project.Project, error)
	DeleteProjectByID(ctx context.Context, id string) (bool, error)
	UpdateProjectStatus(ctx context.Context, id string, status project.ProjectStatus, updatedAt time.Time) error
	UpdateProjectStatusAndCounts(ctx context.Context, id string, status project.ProjectStatus, serviceCount int, runningCount int, updatedAt time.Time) error
	UpdateProjectServiceCount(ctx context.Context, id string, serviceCount int) error
}

type ApiKeyCreateInput struct {
	ID            string
	Name          string
	Description   *string
	KeyHash       string
	KeyPrefix     string
	UserID        string
	EnvironmentID *string
	ExpiresAt     *time.Time
	LastUsedAt    *time.Time
}

type ApiKeyUpdateInput struct {
	ID          string
	Name        string
	Description *string
	ExpiresAt   *time.Time
}

type ApiKeyStore interface {
	CreateApiKey(ctx context.Context, input ApiKeyCreateInput) (*apikey.ModelApiKey, error)
	GetApiKeyByID(ctx context.Context, id string) (*apikey.ModelApiKey, error)
	ListApiKeys(ctx context.Context) ([]apikey.ModelApiKey, error)
	ListApiKeysByPrefix(ctx context.Context, keyPrefix string) ([]apikey.ModelApiKey, error)
	UpdateApiKey(ctx context.Context, input ApiKeyUpdateInput) (*apikey.ModelApiKey, error)
	DeleteApiKeyByID(ctx context.Context, id string) (bool, error)
	TouchApiKeyLastUsed(ctx context.Context, id string, lastUsedAt time.Time) error
}

type GitRepositoryStore interface {
	CreateGitRepository(ctx context.Context, repository gitops.ModelGitRepository) (*gitops.ModelGitRepository, error)
	GetGitRepositoryByID(ctx context.Context, id string) (*gitops.ModelGitRepository, error)
	GetGitRepositoryByName(ctx context.Context, name string) (*gitops.ModelGitRepository, error)
	ListGitRepositories(ctx context.Context) ([]gitops.ModelGitRepository, error)
	SaveGitRepository(ctx context.Context, repository gitops.ModelGitRepository) (*gitops.ModelGitRepository, error)
	DeleteGitRepositoryByID(ctx context.Context, id string) (bool, error)
	CountGitOpsSyncsByRepositoryID(ctx context.Context, repositoryID string) (int64, error)
}

type GitOpsSyncStore interface {
	CreateGitOpsSync(ctx context.Context, sync gitops.ModelGitOpsSync) (*gitops.ModelGitOpsSync, error)
	GetGitOpsSyncByID(ctx context.Context, id string) (*gitops.ModelGitOpsSync, error)
	ListGitOpsSyncs(ctx context.Context) ([]gitops.ModelGitOpsSync, error)
	ListGitOpsSyncsByEnvironment(ctx context.Context, environmentID string) ([]gitops.ModelGitOpsSync, error)
	ListAutoSyncGitOpsSyncs(ctx context.Context) ([]gitops.ModelGitOpsSync, error)
	SaveGitOpsSync(ctx context.Context, sync gitops.ModelGitOpsSync) (*gitops.ModelGitOpsSync, error)
	DeleteGitOpsSyncByID(ctx context.Context, id string) (bool, error)
	UpdateGitOpsSyncInterval(ctx context.Context, id string, minutes int) error
	UpdateGitOpsSyncProjectID(ctx context.Context, id string, projectID *string) error
	UpdateGitOpsSyncStatus(ctx context.Context, id string, lastSyncAt time.Time, status string, errorMsg *string, commitHash *string) error
	SetProjectGitOpsManagedBy(ctx context.Context, projectID string, syncID *string) error
	ClearProjectGitOpsManagedByIfMatches(ctx context.Context, projectID string, syncID string) error
}

type NotificationStore interface {
	ListNotificationSettings(ctx context.Context) ([]notification.NotificationSettings, error)
	GetNotificationSettingByProvider(ctx context.Context, provider notification.NotificationProvider) (*notification.NotificationSettings, error)
	UpsertNotificationSetting(ctx context.Context, provider notification.NotificationProvider, enabled bool, config base.JSON) (*notification.NotificationSettings, error)
	DeleteNotificationSetting(ctx context.Context, provider notification.NotificationProvider) error
	CreateNotificationLog(ctx context.Context, log notification.NotificationLog) error
}

type AppriseStore interface {
	GetAppriseSettings(ctx context.Context) (*notification.AppriseSettings, error)
	UpsertAppriseSettings(ctx context.Context, apiURL string, enabled bool, imageUpdateTag, containerUpdateTag string) (*notification.AppriseSettings, error)
}

type VolumeStore interface {
	CreateVolumeBackup(ctx context.Context, backup volume.VolumeBackup) (*volume.VolumeBackup, error)
	ListVolumeBackupsByVolumeName(ctx context.Context, volumeName string) ([]volume.VolumeBackup, error)
	GetVolumeBackupByID(ctx context.Context, id string) (*volume.VolumeBackup, error)
	DeleteVolumeBackupByID(ctx context.Context, id string) (bool, error)
}

type UpdaterStore interface {
	CreateAutoUpdateRecord(ctx context.Context, record updater.AutoUpdateRecord) error
	ListAutoUpdateRecords(ctx context.Context, limit int) ([]updater.AutoUpdateRecord, error)
}

type SystemStore interface{}

type JobStore interface{}
