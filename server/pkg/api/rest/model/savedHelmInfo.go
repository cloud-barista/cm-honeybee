package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"time"
)

type SavedHelmInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	HelmData     string    `gorm:"column:helm_data" json:"helm_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type HelmInfoList struct {
	Servers []kubernetes.Helm `json:"servers" validate:"required"`
}
