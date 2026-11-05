package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
	"time"
)

type SavedDataInfo struct {
	ConnectionID string `gorm:"primaryKey" json:"connection_id" validate:"required"`
	DataData     string `gorm:"column:data_data" json:"data_data" validate:"required"`
	// CSPData is the bucket as the CSP reported it (CSP source groups only), a
	// storagemodel.SourceObjectStorageProperty in JSON.
	CSPData   string    `gorm:"column:csp_data" json:"csp_data,omitempty"`
	Status    string    `gorm:"column:status" json:"status"`
	SavedTime time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type DataInfoList struct {
	MinIOData []data.MinIOData `json:"minio_data" validate:"required"`
}
