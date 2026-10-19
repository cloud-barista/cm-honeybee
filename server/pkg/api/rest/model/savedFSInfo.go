package model

import (
	"time"

	sourcemodel "github.com/cloud-barista/cm-centipede/dmdl/source-model"
)

// SavedFSInfo statuses. Unlike the other domains these are constants: the
// values are also written and read outside the import controller (stale is set
// when the connection's scan target changes).
const (
	SavedFSInfoStatusImporting = "importing"
	SavedFSInfoStatusSuccess   = "success"
	SavedFSInfoStatusFailed    = "failed"
	// SavedFSInfoStatusStale marks a result collected from a scan path the
	// connection no longer points at.
	SavedFSInfoStatusStale = "stale"
)

// ImportFSReq selects how much to collect. The path is not part of it: it is
// the connection's fs_scan_path.
type ImportFSReq struct {
	// MaxDepth limits how deep folders are walked. 0 → no limit.
	MaxDepth int          `json:"max_depth,omitempty"`
	Metric   *FSMetricReq `json:"metric,omitempty"`
}

// FSMetricReq picks the optional file system figures. An omitted field keeps
// the library default (folder_mod_time and folder_perms are on, the rest off).
type FSMetricReq struct {
	TotalSize       *bool `json:"total_size,omitempty"`
	FileCount       *bool `json:"file_count,omitempty"`
	ExtensionCount  *bool `json:"extension_count,omitempty"`
	FolderModTime   *bool `json:"folder_mod_time,omitempty"`
	FolderPerms     *bool `json:"folder_perms,omitempty"`
	FolderFileCount *bool `json:"folder_file_count,omitempty"`
	FolderFileSize  *bool `json:"folder_file_size,omitempty"`
}

// SavedFSInfo is the last file system collection of a connection, FSData
// holding the inspect result as the agent returned it.
type SavedFSInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	FSData       string    `gorm:"column:fs_data" json:"fs_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type FSInfoList struct {
	FileSystems []sourcemodel.FSInfo `json:"file_systems" validate:"required"`
}
