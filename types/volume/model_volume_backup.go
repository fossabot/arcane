package volume

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

type VolumeBackup struct {
	base.BaseModel
	VolumeName string    `json:"volumeName"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (b *VolumeBackup) ToDTO() BackupEntry {
	return BackupEntry{
		ID:         b.ID,
		VolumeName: b.VolumeName,
		Size:       b.Size,
		CreatedAt:  b.CreatedAt.Format(time.RFC3339),
	}
}
