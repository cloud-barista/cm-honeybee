package model

import (
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
)

// SavedRDBMSInfo stores what was collected for a CSP managed database
// connection (resource_type rdbms). An installed database on a VM is read from
// SavedSoftwareInfo instead, so this table only ever holds CSP results.
type SavedRDBMSInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	RDBMSData    string    `gorm:"column:rdbms_data" json:"rdbms_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
}

// RDBMSInfo is the raw result for one database. A managed database carries the
// instance as the CSP driver reported it and the size of its spec; DBSpec is nil
// when the driver could not resolve the spec, which is not a collection failure.
// A database installed on a VM carries Installed instead.
type RDBMSInfo struct {
	RDBMS     *spider.RDBMSInfo   `json:"rdbms,omitempty"`
	DBSpec    *spider.DBSpecInfo  `json:"db_spec,omitempty"`
	Installed *InstalledRDBMSInfo `json:"installed,omitempty"`
}

// InstalledRDBMSInfo is a database server found in a VM's collected software,
// with the evidence it was judged by. Source tells which software list it came
// from (deb, rpm, legacy, docker, podman); only the fields that list provides
// are set.
type InstalledRDBMSInfo struct {
	Engine  string `json:"engine"`
	Version string `json:"version"`
	Source  string `json:"source"`
	// Ports are the ports the database listens on; empty when it is stopped or
	// the ports could not be read.
	Ports          []int               `json:"ports"`
	Package        string              `json:"package,omitempty"`
	PID            int32               `json:"pid,omitempty"`
	ExecutablePath string              `json:"executable_path,omitempty"`
	VersionRaw     string              `json:"version_raw,omitempty"`
	Syft           *software.SyftMatch `json:"syft,omitempty"`
	Container      string              `json:"container,omitempty"`
	Image          string              `json:"image,omitempty"`
}

// RDBMSInfoList is the raw result for every connection in a source group that
// has database information.
type RDBMSInfoList struct {
	Servers []RDBMSInfo `json:"servers" validate:"required"`
}
