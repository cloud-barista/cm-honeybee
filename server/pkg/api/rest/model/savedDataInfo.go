package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
	"time"
)

type SavedDataInfo struct {
	ConnectionID string `gorm:"primaryKey" json:"connection_id" validate:"required"`
	DataData     string `gorm:"column:data_data" json:"data_data" validate:"required"`
	// CSPData is the bucket as the CSP reported it (CSP source groups only), a
	// storagemodel.SourceObjectStorageProperty in JSON with the bucket's region
	// added under "region" (empty when the CSP cannot tell it or the lookup
	// failed).
	CSPData   string    `gorm:"column:csp_data" json:"csp_data,omitempty"`
	Status    string    `gorm:"column:status" json:"status"`
	SavedTime time.Time `gorm:"column:saved_time" json:"saved_time"`
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

type DataInfoList struct {
	MinIOData []data.MinIOData `json:"minio_data" validate:"required"`
}
