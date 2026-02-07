package database

import "github.com/getarcaneapp/arcane/backend/internal/database/stores"

// Store abstracts database access for incremental migration from GORM to sqlc.
type Store interface {
	stores.UserStore
	stores.EventStore
	stores.SettingsStore
	stores.TemplateStore
	stores.ImageUpdateStore
	stores.ContainerRegistryStore
	stores.EnvironmentStore
	stores.ProjectStore
	stores.ApiKeyStore
	stores.GitRepositoryStore
	stores.GitOpsSyncStore
	stores.NotificationStore
	stores.AppriseStore
	stores.VolumeStore
	stores.UpdaterStore
	stores.SystemStore
	stores.JobStore
}

// Re-export common store interfaces for convenience in services.
type SettingsStore = stores.SettingsStore
type SettingsStoreTx = stores.SettingsStoreTx
type UserStore = stores.UserStore
type EventStore = stores.EventStore
type TemplateStore = stores.TemplateStore
type ImageUpdateStore = stores.ImageUpdateStore
type NotificationStore = stores.NotificationStore
type AppriseStore = stores.AppriseStore
type ApiKeyStore = stores.ApiKeyStore
type ApiKeyCreateInput = stores.ApiKeyCreateInput
type ApiKeyUpdateInput = stores.ApiKeyUpdateInput
type ContainerRegistryStore = stores.ContainerRegistryStore
type ContainerRegistryCreateInput = stores.ContainerRegistryCreateInput
type ContainerRegistryUpdateInput = stores.ContainerRegistryUpdateInput
type EnvironmentStore = stores.EnvironmentStore
type EnvironmentCreateInput = stores.EnvironmentCreateInput
type EnvironmentPatchInput = stores.EnvironmentPatchInput
type ProjectStore = stores.ProjectStore
type GitRepositoryStore = stores.GitRepositoryStore
type GitOpsSyncStore = stores.GitOpsSyncStore
type VolumeStore = stores.VolumeStore
type UpdaterStore = stores.UpdaterStore
