package environment

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

// ModelEnvironment is the persisted environment model used by the backend data layer.
type ModelEnvironment struct {
	Name        string     `json:"name" sortable:"true"`
	ApiUrl      string     `json:"apiUrl" sortable:"true"`
	Status      string     `json:"status" sortable:"true"`
	Enabled     bool       `json:"enabled" sortable:"true"`
	IsEdge      bool       `json:"isEdge"`
	LastSeen    *time.Time `json:"lastSeen"`
	AccessToken *string    `json:"-"`
	ApiKeyID    *string    `json:"-"`

	base.BaseModel
}

type EnvironmentStatus string

const (
	EnvironmentStatusOnline  EnvironmentStatus = "online"
	EnvironmentStatusOffline EnvironmentStatus = "offline"
	EnvironmentStatusError   EnvironmentStatus = "error"
	EnvironmentStatusPending EnvironmentStatus = "pending"
)
