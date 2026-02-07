package apikey

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

// ModelApiKey is the persisted API key model used by the backend data layer.
type ModelApiKey struct {
	Name          string     `json:"name" sortable:"true"`
	Description   *string    `json:"description,omitempty"`
	KeyHash       string     `json:"-"`
	KeyPrefix     string     `json:"keyPrefix"`
	UserID        string     `json:"userId"`
	EnvironmentID *string    `json:"environmentId,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty" sortable:"true"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty" sortable:"true"`
	base.BaseModel
}
