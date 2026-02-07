package updater

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

type AutoUpdateStatus string

const (
	AutoUpdateStatusPending   AutoUpdateStatus = "pending"
	AutoUpdateStatusChecking  AutoUpdateStatus = "checking"
	AutoUpdateStatusUpdating  AutoUpdateStatus = "updating"
	AutoUpdateStatusCompleted AutoUpdateStatus = "completed"
	AutoUpdateStatusFailed    AutoUpdateStatus = "failed"
	AutoUpdateStatusSkipped   AutoUpdateStatus = "skipped"
)

type AutoUpdateRecord struct {
	ResourceID       string           `json:"resourceId"`
	ResourceType     string           `json:"resourceType"`
	ResourceName     string           `json:"resourceName"`
	Status           AutoUpdateStatus `json:"status"`
	StartTime        time.Time        `json:"startTime"`
	EndTime          *time.Time       `json:"endTime,omitempty"`
	UpdateAvailable  bool             `json:"updateAvailable"`
	UpdateApplied    bool             `json:"updateApplied"`
	OldImageVersions base.JSON        `json:"oldImageVersions,omitempty"`
	NewImageVersions base.JSON        `json:"newImageVersions,omitempty"`
	Error            *string          `json:"error,omitempty"`
	Details          base.JSON        `json:"details,omitempty"`
	base.BaseModel
}
