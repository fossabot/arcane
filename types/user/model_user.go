package user

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

// ModelUser is the persisted user model used by the backend data layer.
type ModelUser struct {
	Username               string           `json:"username" sortable:"true"`
	PasswordHash           string           `json:"-"`
	DisplayName            *string          `json:"displayName,omitempty" sortable:"true"`
	Email                  *string          `json:"email,omitempty" sortable:"true"`
	Roles                  base.StringSlice `json:"roles"`
	OidcSubjectId          *string          `json:"oidcSubjectId,omitempty"`
	LastLogin              *time.Time       `json:"lastLogin,omitempty" sortable:"true"`
	Locale                 *string          `json:"locale,omitempty"`
	RequiresPasswordChange bool             `json:"requiresPasswordChange"`

	// OIDC provider tokens
	OidcAccessToken          *string    `json:"-"`
	OidcRefreshToken         *string    `json:"-"`
	OidcAccessTokenExpiresAt *time.Time `json:"-"`
	base.BaseModel
}
