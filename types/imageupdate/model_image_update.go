package imageupdate

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

type ImageUpdateRecord struct {
	ID             string    `json:"id"`
	Repository     string    `json:"repository"`
	Tag            string    `json:"tag"`
	HasUpdate      bool      `json:"hasUpdate"`
	UpdateType     string    `json:"updateType"`
	CurrentVersion string    `json:"currentVersion"`
	LatestVersion  *string   `json:"latestVersion,omitempty"`
	CurrentDigest  *string   `json:"currentDigest,omitempty"`
	LatestDigest   *string   `json:"latestDigest,omitempty"`
	CheckTime      time.Time `json:"checkTime"`
	ResponseTimeMs int       `json:"responseTimeMs"`
	LastError      *string   `json:"lastError,omitempty"`

	AuthMethod     *string `json:"authMethod,omitempty"`
	AuthUsername   *string `json:"authUsername,omitempty"`
	AuthRegistry   *string `json:"authRegistry,omitempty"`
	UsedCredential bool    `json:"usedCredential,omitempty"`

	NotificationSent bool `json:"notificationSent"`

	base.BaseModel
}

type ImageUpdate struct {
	HasUpdate      bool   `json:"hasUpdate"`
	UpdateType     string `json:"updateType"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion,omitempty"`
	CheckTime      string `json:"checkTime"`
}

const (
	UpdateTypeDigest = "digest"
	UpdateTypeTag    = "tag"
)

func (i *ImageUpdateRecord) NeedsUpdate() bool {
	return i.HasUpdate
}

func (i *ImageUpdateRecord) IsDigestUpdate() bool {
	return i.UpdateType == UpdateTypeDigest
}

func (i *ImageUpdateRecord) IsTagUpdate() bool {
	return i.UpdateType == UpdateTypeTag
}
