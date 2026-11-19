package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"time"
)

type SavedSoftwareInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	SoftwareData string    `gorm:"column:software_data" json:"software_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

type SoftwareInfoList struct {
	Servers []software.Software `json:"servers" validate:"required"`
}
