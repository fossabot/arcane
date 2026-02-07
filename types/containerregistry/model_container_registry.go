package containerregistry

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

// ModelContainerRegistry is the persisted container registry model used by the backend data layer.
type ModelContainerRegistry struct {
	URL         string    `json:"url" sortable:"true"`
	Username    string    `json:"username" sortable:"true"`
	Token       string    `json:"token"`
	Description *string   `json:"description,omitempty" sortable:"true"`
	Insecure    bool      `json:"insecure" sortable:"true"`
	Enabled     bool      `json:"enabled" sortable:"true"`
	CreatedAt   time.Time `json:"createdAt" sortable:"true"`
	UpdatedAt   time.Time `json:"updatedAt" sortable:"true"`
	base.BaseModel
}

type CreateContainerRegistryRequest struct {
	URL         string  `json:"url" binding:"required"`
	Username    string  `json:"username" binding:"required"`
	Token       string  `json:"token" binding:"required"`
	Description *string `json:"description"`
	Insecure    *bool   `json:"insecure"`
	Enabled     *bool   `json:"enabled"`
}

type UpdateContainerRegistryRequest struct {
	URL         *string `json:"url"`
	Username    *string `json:"username"`
	Token       *string `json:"token"`
	Description *string `json:"description"`
	Insecure    *bool   `json:"insecure"`
	Enabled     *bool   `json:"enabled"`
}
