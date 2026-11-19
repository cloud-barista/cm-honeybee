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
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

type HelmInfoList struct {
	Servers []kubernetes.Helm `json:"servers" validate:"required"`
}
