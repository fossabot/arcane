package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getarcaneapp/arcane/backend/internal/config"
	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/services"
	"github.com/getarcaneapp/arcane/backend/resources"
)

type Services struct {
	AppImages         *services.ApplicationImagesService
	User              *services.UserService
	Project           *services.ProjectService
	Environment       *services.EnvironmentService
	Settings          *services.SettingsService
	JobSchedule       *services.JobService
	SettingsSearch    *services.SettingsSearchService
	CustomizeSearch   *services.CustomizeSearchService
	Container         *services.ContainerService
	Image             *services.ImageService
	Volume            *services.VolumeService
	Network           *services.NetworkService
	ImageUpdate       *services.ImageUpdateService
	Auth              *services.AuthService
	Oidc              *services.OidcService
	Docker            *services.DockerClientService
	Template          *services.TemplateService
	ContainerRegistry *services.ContainerRegistryService
	System            *services.SystemService
	SystemUpgrade     *services.SystemUpgradeService
	Migration         *services.MigrationService
	Updater           *services.UpdaterService
	Event             *services.EventService
	Version           *services.VersionService
	Notification      *services.NotificationService
	Apprise           *services.AppriseService //nolint:staticcheck // Apprise still functional, deprecated in favor of Shoutrrr
	ApiKey            *services.ApiKeyService
	GitRepository     *services.GitRepositoryService
	GitOpsSync        *services.GitOpsSyncService
	Font              *services.FontService
}

func initializeServices(ctx context.Context, db *database.DB, cfg *config.Config, httpClient *http.Client) (svcs *Services, dockerSrvice *services.DockerClientService, err error) {
	svcs = &Services{}

	store, err := database.NewSqlcStore(db)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize sqlc store: %w", err)
	}

	svcs.Event = services.NewEventService(store)
	svcs.Settings, err = services.NewSettingsService(ctx, store)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to settings service: %w", err)
	}
	svcs.JobSchedule = services.NewJobService(store, svcs.Settings, cfg)
	svcs.SettingsSearch = services.NewSettingsSearchService()
	svcs.CustomizeSearch = services.NewCustomizeSearchService()
	svcs.AppImages = services.NewApplicationImagesService(resources.FS, svcs.Settings)
	svcs.Font = services.NewFontService(resources.FS)
	dockerClient := services.NewDockerClientService(cfg, svcs.Settings)
	svcs.Docker = dockerClient
	svcs.User = services.NewUserService(store)
	svcs.ContainerRegistry = services.NewContainerRegistryService(store)
	svcs.Notification = services.NewNotificationService(store, store, cfg)
	svcs.Apprise = services.NewAppriseService(store, cfg)
	svcs.ImageUpdate = services.NewImageUpdateService(store, svcs.Settings, svcs.ContainerRegistry, svcs.Docker, svcs.Event, svcs.Notification)
	svcs.Image = services.NewImageService(store, svcs.Docker, svcs.ContainerRegistry, svcs.ImageUpdate, svcs.Event)
	svcs.Project = services.NewProjectService(store, svcs.Settings, svcs.Event, svcs.Image, svcs.Docker)
	svcs.Environment = services.NewEnvironmentService(store, httpClient, svcs.Docker, svcs.Event, svcs.Settings)
	svcs.Container = services.NewContainerService(svcs.Event, svcs.Docker, svcs.Image, svcs.Settings)
	svcs.Volume = services.NewVolumeService(store, svcs.Docker, svcs.Event, svcs.Settings, svcs.Container, svcs.Image, cfg.BackupVolumeName)
	svcs.Network = services.NewNetworkService(svcs.Docker, svcs.Event)
	svcs.Template = services.NewTemplateService(ctx, store, httpClient, svcs.Settings)
	svcs.Auth = services.NewAuthService(svcs.User, svcs.Settings, svcs.Event, cfg.JWTSecret, cfg)
	svcs.Oidc = services.NewOidcService(svcs.Auth, cfg, httpClient)
	svcs.ApiKey = services.NewApiKeyService(store, svcs.User)
	svcs.System = services.NewSystemService(svcs.Docker, svcs.Container, svcs.Image, svcs.Volume, svcs.Network, svcs.Settings)
	svcs.Version = services.NewVersionService(httpClient, cfg.UpdateCheckDisabled, config.Version, config.Revision, svcs.ContainerRegistry, svcs.Docker)
	svcs.SystemUpgrade = services.NewSystemUpgradeService(svcs.Docker, svcs.Version, svcs.Event, svcs.Settings)
	svcs.Migration = services.NewMigrationService(db)
	svcs.Updater = services.NewUpdaterService(store, store, svcs.Settings, svcs.Docker, svcs.Project, svcs.ImageUpdate, svcs.ContainerRegistry, svcs.Event, svcs.Image, svcs.Notification, svcs.SystemUpgrade)
	svcs.GitRepository = services.NewGitRepositoryService(store, cfg.GitWorkDir, svcs.Event, svcs.Settings)
	svcs.GitOpsSync = services.NewGitOpsSyncService(store, svcs.GitRepository, svcs.Project, svcs.Event)

	return svcs, dockerClient, nil
}
