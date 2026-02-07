package gitops

import "github.com/getarcaneapp/arcane/types/base"

// ModelGitRepository is the persisted git repository model used by the backend data layer.
type ModelGitRepository struct {
	Name                   string  `json:"name" sortable:"true" search:"git,repository,repo,source,version,control,github,gitlab,bitbucket"`
	URL                    string  `json:"url" sortable:"true" search:"url,git,clone,remote,https,ssh"`
	AuthType               string  `json:"authType" sortable:"true" search:"auth,authentication,credentials,token,ssh,http"`
	Username               string  `json:"username" sortable:"true" search:"username,user,login,account"`
	Token                  string  `json:"token" search:"token,password,credentials,secret,auth"`
	SSHKey                 string  `json:"sshKey" search:"ssh,key,private,public,certificate"`
	SSHHostKeyVerification string  `json:"sshHostKeyVerification"`
	Description            *string `json:"description,omitempty" sortable:"true"`
	Enabled                bool    `json:"enabled" sortable:"true" search:"enabled,active,disabled"`
	base.BaseModel
}

type CreateGitRepositoryRequest struct {
	Name                   string  `json:"name" binding:"required"`
	URL                    string  `json:"url" binding:"required"`
	AuthType               string  `json:"authType" binding:"required,oneof=none http ssh"`
	Username               string  `json:"username,omitempty"`
	Token                  string  `json:"token,omitempty"`
	SSHKey                 string  `json:"sshKey,omitempty"`
	SSHHostKeyVerification string  `json:"sshHostKeyVerification,omitempty" binding:"omitempty,oneof=strict accept_new skip"`
	Description            *string `json:"description,omitempty"`
	Enabled                *bool   `json:"enabled,omitempty"`
}

type UpdateGitRepositoryRequest struct {
	Name                   *string `json:"name,omitempty"`
	URL                    *string `json:"url,omitempty"`
	AuthType               *string `json:"authType,omitempty" binding:"omitempty,oneof=none http ssh"`
	Username               *string `json:"username,omitempty"`
	Token                  *string `json:"token,omitempty"`
	SSHKey                 *string `json:"sshKey,omitempty"`
	SSHHostKeyVerification *string `json:"sshHostKeyVerification,omitempty" binding:"omitempty,oneof=strict accept_new skip"`
	Description            *string `json:"description,omitempty"`
	Enabled                *bool   `json:"enabled,omitempty"`
}
