package model

import (
	"time"

	sourcemodel "github.com/cloud-barista/cm-centipede/dmdl/source-model"
)

// SavedObjectStorageInfo statuses. Constants for the same reason as
// SavedFSInfoStatus*: stale is set outside the import controller.
const (
	SavedObjectStorageInfoStatusImporting = "importing"
	SavedObjectStorageInfoStatusSuccess   = "success"
	SavedObjectStorageInfoStatusFailed    = "failed"
	// SavedObjectStorageInfoStatusStale marks a result collected from a bucket
	// or prefix the connection no longer points at.
	SavedObjectStorageInfoStatusStale = "stale"
)

// ImportObjectStorageReq selects how much to collect. The bucket and prefix are
// not part of it: they are the connection's os_scan_bucket and os_scan_prefix.
type ImportObjectStorageReq struct {
	Metric *OSMetricReq `json:"metric,omitempty"`
}

// OSMetricReq picks the optional object storage figures. An omitted field keeps
// the library default (all off).
type OSMetricReq struct {
	TotalSize         *bool `json:"total_size,omitempty"`
	ObjectCount       *bool `json:"object_count,omitempty"`
	ExtensionCount    *bool `json:"extension_count,omitempty"`
	PrefixModTime     *bool `json:"prefix_mod_time,omitempty"`
	PrefixObjectCount *bool `json:"prefix_object_count,omitempty"`
	PrefixObjectSize  *bool `json:"prefix_object_size,omitempty"`
}

// SavedObjectStorageInfo is the last object storage collection of a
// connection, OSData holding the inspect result.
type SavedObjectStorageInfo struct {
	ConnectionID string `gorm:"primaryKey" json:"connection_id" validate:"required"`
	// Bucket is the bucket OSData was collected from. It repeats the
	// connection's os_scan_bucket so that a stale row still says which bucket
	// its result belongs to.
	Bucket    string    `gorm:"column:bucket" json:"bucket"`
	OSData    string    `gorm:"column:os_data" json:"os_data" validate:"required"`
	Status    string    `gorm:"column:status" json:"status"`
	SavedTime time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type ObjectStorageInfoList struct {
	ObjectStorages []sourcemodel.ObjectStorageInfo `json:"object_storages" validate:"required"`
}
