package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
	"time"
)

type SavedDataInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	DataData     string    `gorm:"column:data_data" json:"data_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type DataInfoList struct {
	MinIOData []data.MinIOData `json:"minio_data" validate:"required"`
}
