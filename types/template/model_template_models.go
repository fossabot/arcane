package template

import "github.com/getarcaneapp/arcane/types/base"

// ModelTemplateRegistry is the persisted template registry model used by the backend data layer.
type ModelTemplateRegistry struct {
	base.BaseModel
	Name        string `json:"name"`
	URL         string `json:"url"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

type ComposeTemplate struct {
	base.BaseModel
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Content     string                   `json:"content"`
	EnvContent  *string                  `json:"envContent,omitempty"`
	IsCustom    bool                     `json:"isCustom"`
	IsRemote    bool                     `json:"isRemote"`
	RegistryID  *string                  `json:"registryId,omitempty"`
	Registry    *ModelTemplateRegistry   `json:"registry,omitempty"`
	Metadata    *ComposeTemplateMetadata `json:"metadata,omitempty"`
}

type ComposeTemplateMetadata struct {
	Version          *string  `json:"version,omitempty"`
	Author           *string  `json:"author,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	RemoteURL        *string  `json:"remoteUrl,omitempty"`
	EnvURL           *string  `json:"envUrl,omitempty"`
	DocumentationURL *string  `json:"documentationUrl,omitempty"`
}
